//go:build darwin

package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tmc/apple/dispatch"
	applenetwork "github.com/tmc/apple/network"
	"github.com/tmc/apple/objc"
	"github.com/tmc/apple/objectivec"
)

var networkTrace = os.Getenv("AWDL_DEMO_NETWORK_TRACE") != ""

type nwPacket struct {
	data []byte
	addr *net.UDPAddr
}

type nwPeerConn struct {
	conn  applenetwork.NWConnection
	addr  *net.UDPAddr
	ready chan error
	once  sync.Once
}

type nwPacketConn struct {
	profile linkProfile
	iface   linkInterface

	queue    dispatch.Queue
	listener applenetwork.NWListener
	local    *net.UDPAddr

	packets chan nwPacket
	closed  chan struct{}
	once    sync.Once

	mu                   sync.Mutex
	conns                map[string]*nwPeerConn
	sendSeq              atomic.Uint64
	readDeadline         time.Time
	writeDeadline        time.Time
	readDeadlineChanged  chan struct{}
	writeDeadlineChanged chan struct{}
}

func newNWPacketConn(profile linkProfile, iface linkInterface, ip net.IP, zone string, port int) (*nwPacketConn, error) {
	local := &net.UDPAddr{IP: append(net.IP(nil), ip...), Port: port, Zone: zone}
	params, err := newNWParameters(profile, iface, local)
	if err != nil {
		return nil, err
	}
	listener := applenetwork.NWListenerCreate(params)
	if listener.ID == 0 {
		return nil, errors.New("nw_listener_create returned nil")
	}

	c := &nwPacketConn{
		profile:              profile,
		iface:                iface,
		queue:                dispatch.QueueCreate("com.github.tmc.awdl-webrtc-apple-demo.network-packetconn"),
		listener:             listener,
		local:                local,
		packets:              make(chan nwPacket, 16384),
		closed:               make(chan struct{}),
		conns:                make(map[string]*nwPeerConn),
		readDeadlineChanged:  make(chan struct{}),
		writeDeadlineChanged: make(chan struct{}),
	}

	ready := make(chan error, 1)
	applenetwork.NWListenerSetQueue(listener, c.queue)
	applenetwork.NWListenerSetStateChangedHandler(listener, func(state applenetwork.NWListenerState, nwErr applenetwork.NWError) {
		traceNW("listener %s state=%s err=%s", c.local, state, nwErrorString(nwErr))
		switch state {
		case applenetwork.NWListenerStateReady:
			select {
			case ready <- nil:
			default:
			}
		case applenetwork.NWListenerStateFailed, applenetwork.NWListenerStateCancelled:
			err := fmt.Errorf("nw listener %s", state)
			if !nwErr.IsZero() {
				err = fmt.Errorf("%s: %w", err, nwErr)
			}
			select {
			case ready <- err:
			default:
			}
		}
	})
	applenetwork.NWListenerSetNewConnectionHandler(listener, func(obj objectivec.Object) {
		c.accept(applenetwork.NWConnectionFromID(obj.ID))
	})
	applenetwork.NWListenerStart(listener)

	select {
	case err := <-ready:
		if err != nil {
			_ = c.Close()
			return nil, err
		}
	case <-time.After(5 * time.Second):
		_ = c.Close()
		return nil, errors.New("timed out waiting for nw listener readiness")
	}
	if got := applenetwork.NWListenerGetPort(listener); got != 0 {
		c.local.Port = int(got)
	}
	if c.local.Port == 0 {
		_ = c.Close()
		return nil, errors.New("nw listener returned port 0")
	}
	return c, nil
}

func newNWParameters(profile linkProfile, iface linkInterface, local *net.UDPAddr) (params applenetwork.NWParameters, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("configure nw parameters: %v", r)
		}
	}()
	params = applenetwork.NWParametersCreate()
	if params.ID == 0 {
		return applenetwork.NWParameters{}, errors.New("nw_parameters_create returned nil")
	}
	stack := applenetwork.NWParametersCopyDefaultProtocolStack(params)
	if stack.ID == 0 {
		return applenetwork.NWParameters{}, errors.New("nw_parameters_copy_default_protocol_stack returned nil")
	}
	udpOptions := applenetwork.NWUDPCreateOptions()
	if udpOptions.ID == 0 {
		return applenetwork.NWParameters{}, errors.New("nw_udp_create_options returned nil")
	}
	applenetwork.NWProtocolStackClearApplicationProtocols(stack)
	applenetwork.NWProtocolStackSetTransportProtocol(stack, udpOptions)
	applenetwork.NWParametersSetIncludePeerToPeer(params, profile.IncludePeerToPeer)
	if profile.Name != "thunderbolt" {
		applenetwork.NWParametersSetRequiredInterfaceType(params, profile.RequiredInterfaceType)
	}
	applenetwork.NWParametersSetReuseLocalAddress(params, true)
	if local != nil {
		applenetwork.NWParametersSetLocalEndpoint(params, nwEndpointForUDPAddr(local))
	}
	if networkBackendRequireInterface || profile.Name == "awdl" {
		if err := requireNWInterface(params, iface); err != nil {
			return applenetwork.NWParameters{}, err
		}
	}
	return params, nil
}

const networkBackendRequireInterface = false

func requireNWInterface(params applenetwork.NWParameters, iface linkInterface) error {
	privateIface := newPrivateNWInterfaceWithName(iface.Name)
	if privateIface.ID == 0 {
		return fmt.Errorf("private NWInterface(%s) returned nil", iface.Name)
	}
	ciface := privateGetObject(privateIface, "cInterface")
	if ciface.ID == 0 {
		return fmt.Errorf("private NWInterface(%s).cInterface returned nil", iface.Name)
	}
	applenetwork.NWParametersRequireInterface(params, applenetwork.NWInterfaceFromID(ciface.ID))
	return nil
}

func (c *nwPacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	for {
		deadline, changed := c.readDeadlineState()
		var timer *time.Timer
		var timeout <-chan time.Time
		if !deadline.IsZero() {
			wait := time.Until(deadline)
			if wait <= 0 {
				return 0, nil, nwTimeoutError{op: "read"}
			}
			timer = time.NewTimer(wait)
			timeout = timer.C
		}

		select {
		case pkt := <-c.packets:
			if timer != nil {
				timer.Stop()
			}
			n := copy(b, pkt.data)
			return n, copyUDPAddr(pkt.addr), nil
		case <-c.closed:
			if timer != nil {
				timer.Stop()
			}
			return 0, nil, net.ErrClosed
		case <-changed:
			if timer != nil {
				timer.Stop()
			}
			continue
		case <-timeout:
			return 0, nil, nwTimeoutError{op: "read"}
		}
	}
}

func (c *nwPacketConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	select {
	case <-c.closed:
		return 0, net.ErrClosed
	default:
	}
	deadline, _ := c.writeDeadlineState()
	if !deadline.IsZero() && time.Until(deadline) <= 0 {
		return 0, nwTimeoutError{op: "write"}
	}
	peer, err := c.peerConn(addr)
	if err != nil {
		return 0, err
	}
	if err := peer.waitReady(deadline); err != nil {
		return 0, err
	}
	_ = c.connectionPath(peer.conn)
	data := dispatch.DataCreate(b)
	context := applenetwork.NWContentContextCreate(fmt.Sprintf("awdl-webrtc-apple-demo-%d", c.sendSeq.Add(1)))
	done := make(chan error, 1)
	applenetwork.NWConnectionSend(peer.conn, data, context, true, func(nwErr applenetwork.NWError) {
		traceNW("send %s bytes=%d err=%s", peer.addr, len(b), nwErrorString(nwErr))
		data.Release()
		if context.ID != 0 {
			context.Release()
		}
		if nwErr.IsZero() {
			done <- nil
			return
		}
		done <- nwErr
	})
	if deadline.IsZero() {
		return len(b), nil
	}
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	select {
	case err := <-done:
		if err != nil {
			return 0, fmt.Errorf("nw send %s: %w", peer.addr, err)
		}
		return len(b), nil
	case <-timer.C:
		return 0, nwTimeoutError{op: "write"}
	case <-c.closed:
		return 0, net.ErrClosed
	}
}

func (c *nwPacketConn) Close() error {
	c.once.Do(func() {
		close(c.closed)
		applenetwork.NWListenerCancel(c.listener)
		c.mu.Lock()
		defer c.mu.Unlock()
		for _, peer := range c.conns {
			applenetwork.NWConnectionCancel(peer.conn)
		}
		c.conns = nil
	})
	return nil
}

func (c *nwPacketConn) LocalAddr() net.Addr {
	c.mu.Lock()
	defer c.mu.Unlock()
	return copyUDPAddr(c.local)
}

func (c *nwPacketConn) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}

func (c *nwPacketConn) SetReadDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readDeadline = t
	close(c.readDeadlineChanged)
	c.readDeadlineChanged = make(chan struct{})
	return nil
}

func (c *nwPacketConn) SetWriteDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writeDeadline = t
	close(c.writeDeadlineChanged)
	c.writeDeadlineChanged = make(chan struct{})
	return nil
}

func (c *nwPacketConn) accept(conn applenetwork.NWConnection) {
	addr := c.connectionEndpoint(conn)
	peer := &nwPeerConn{conn: conn, addr: addr, ready: make(chan error, 1)}
	c.mu.Lock()
	c.conns[addr.String()] = peer
	c.mu.Unlock()

	applenetwork.NWConnectionSetQueue(conn, c.queue)
	applenetwork.NWConnectionSetStateChangedHandler(conn, func(state applenetwork.NWConnectionState, nwErr applenetwork.NWError) {
		traceNW("accepted %s state=%s err=%s", addr, state, nwErrorString(nwErr))
		switch state {
		case applenetwork.NWConnectionStateReady:
			peer.markReady(nil)
		case applenetwork.NWConnectionStateFailed, applenetwork.NWConnectionStateCancelled:
			err := fmt.Errorf("nw connection %s %s", addr, state)
			if !nwErr.IsZero() {
				err = fmt.Errorf("%s: %w", err, nwErr)
			}
			peer.markReady(err)
		}
	})
	c.receive(conn, addr)
	applenetwork.NWConnectionStart(conn)
}

func (c *nwPacketConn) peerConn(addr net.Addr) (*nwPeerConn, error) {
	udpAddr, err := netAddrToUDP(addr)
	if err != nil {
		return nil, err
	}
	key := udpAddr.String()
	c.mu.Lock()
	if peer := c.conns[key]; peer != nil {
		c.mu.Unlock()
		return peer, nil
	}
	c.mu.Unlock()

	params, err := newNWParameters(c.profile, c.iface, nil)
	if err != nil {
		return nil, err
	}
	endpoint := nwEndpointForUDPAddr(udpAddr)
	conn := applenetwork.NWConnectionCreate(endpoint, params)
	if conn.ID == 0 {
		return nil, fmt.Errorf("nw_connection_create %s returned nil", udpAddr)
	}
	peer := &nwPeerConn{conn: conn, addr: udpAddr, ready: make(chan error, 1)}
	applenetwork.NWConnectionSetQueue(conn, c.queue)
	applenetwork.NWConnectionSetStateChangedHandler(conn, func(state applenetwork.NWConnectionState, nwErr applenetwork.NWError) {
		traceNW("outbound %s state=%s err=%s", udpAddr, state, nwErrorString(nwErr))
		if state == applenetwork.NWConnectionStateReady {
			traceNW("outbound %s path=%s", udpAddr, c.connectionPath(conn))
		}
		switch state {
		case applenetwork.NWConnectionStateReady:
			peer.markReady(nil)
		case applenetwork.NWConnectionStateFailed, applenetwork.NWConnectionStateCancelled:
			err := fmt.Errorf("nw connection %s %s", udpAddr, state)
			if !nwErr.IsZero() {
				err = fmt.Errorf("%s: %w", err, nwErr)
			}
			peer.markReady(err)
		}
	})
	c.receive(conn, udpAddr)
	applenetwork.NWConnectionStart(conn)

	c.mu.Lock()
	if existing := c.conns[key]; existing != nil {
		c.mu.Unlock()
		applenetwork.NWConnectionCancel(conn)
		return existing, nil
	}
	c.conns[key] = peer
	c.mu.Unlock()
	return peer, nil
}

func (p *nwPeerConn) markReady(err error) {
	p.once.Do(func() {
		p.ready <- err
		close(p.ready)
	})
}

func (p *nwPeerConn) waitReady(deadline time.Time) error {
	var timeout <-chan time.Time
	var timer *time.Timer
	if !deadline.IsZero() {
		wait := time.Until(deadline)
		if wait <= 0 {
			return nwTimeoutError{op: "write"}
		}
		timer = time.NewTimer(wait)
		timeout = timer.C
	} else {
		timer = time.NewTimer(5 * time.Second)
		timeout = timer.C
	}
	defer timer.Stop()

	select {
	case err := <-p.ready:
		if err == nil {
			time.Sleep(100 * time.Millisecond)
		}
		return err
	case <-timeout:
		return fmt.Errorf("nw connection %s readiness: %w", p.addr, nwTimeoutError{op: "write"})
	}
}

func (c *nwPacketConn) receive(conn applenetwork.NWConnection, addr *net.UDPAddr) {
	select {
	case <-c.closed:
		return
	default:
	}
	applenetwork.NWConnectionReceiveMessage(conn, func(content objectivec.Object, _ objectivec.Object, _ bool, nwErr applenetwork.NWError) {
		if !nwErr.IsZero() {
			c.enqueueErrorPacket(addr, nwErr)
			return
		}
		if content.ID != 0 {
			data := dispatch.DataFromHandle(uintptr(content.ID))
			c.enqueue(nwPacket{data: data.Bytes(), addr: addr})
		}
		c.receive(conn, addr)
	})
}

func (c *nwPacketConn) enqueue(pkt nwPacket) {
	select {
	case <-c.closed:
	case c.packets <- pkt:
	default:
	}
}

func (c *nwPacketConn) enqueueErrorPacket(addr *net.UDPAddr, err error) {
	_ = addr
	_ = err
}

func (c *nwPacketConn) connectionEndpoint(conn applenetwork.NWConnection) *net.UDPAddr {
	endpoint := applenetwork.NWConnectionCopyEndpoint(conn)
	if endpoint.ID == 0 {
		return &net.UDPAddr{}
	}
	addr, err := nwEndpointToUDPAddr(endpoint, c.iface.Name)
	if err != nil {
		return &net.UDPAddr{}
	}
	return addr
}

func (c *nwPacketConn) connectionPath(conn applenetwork.NWConnection) string {
	path := applenetwork.NWConnectionCopyCurrentPath(conn)
	if path.ID == 0 {
		return "none"
	}
	var parts []string
	applenetwork.NWPathEnumerateInterfaces(path, func(obj objectivec.Object) bool {
		iface := applenetwork.NWInterfaceFromID(obj.ID)
		name := objc.GoString(applenetwork.NWInterfaceGetName(iface))
		parts = append(parts, fmt.Sprintf("%s/%s", name, applenetwork.NWInterfaceGetType(iface)))
		return true
	})
	if len(parts) == 0 {
		return "interfaces=none"
	}
	return strings.Join(parts, ",")
}

func (c *nwPacketConn) readDeadlineState() (time.Time, <-chan struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.readDeadline, c.readDeadlineChanged
}

func (c *nwPacketConn) writeDeadlineState() (time.Time, <-chan struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writeDeadline, c.writeDeadlineChanged
}

func nwEndpointForUDPAddr(addr *net.UDPAddr) applenetwork.NWEndpoint {
	return applenetwork.NWEndpointCreateHost(nwEndpointHost(addr), strconv.Itoa(addr.Port))
}

func nwEndpointToUDPAddr(endpoint applenetwork.NWEndpoint, defaultZone string) (*net.UDPAddr, error) {
	host := objc.GoString(applenetwork.NWEndpointGetHostname(endpoint))
	port := int(applenetwork.NWEndpointGetPort(endpoint))
	host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	zone := ""
	if i := strings.LastIndex(host, "%"); i >= 0 {
		zone = host[i+1:]
		host = host[:i]
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return nil, fmt.Errorf("nw endpoint host %q is not an IP address", host)
	}
	if zone == "" && ip.To4() == nil && ip.IsLinkLocalUnicast() {
		zone = defaultZone
	}
	return &net.UDPAddr{IP: ip, Port: port, Zone: zone}, nil
}

func nwEndpointHost(addr *net.UDPAddr) string {
	host := addr.IP.String()
	if addr.Zone != "" {
		host += "%" + addr.Zone
	}
	return host
}

func netAddrToUDP(addr net.Addr) (*net.UDPAddr, error) {
	switch a := addr.(type) {
	case *net.UDPAddr:
		return copyUDPAddr(a), nil
	default:
		udpAddr, err := net.ResolveUDPAddr("udp", addr.String())
		if err != nil {
			return nil, fmt.Errorf("resolve UDP addr %s: %w", addr, err)
		}
		return udpAddr, nil
	}
}

func copyUDPAddr(addr *net.UDPAddr) *net.UDPAddr {
	if addr == nil {
		return nil
	}
	cp := *addr
	cp.IP = append(net.IP(nil), addr.IP...)
	return &cp
}

type nwTimeoutError struct {
	op string
}

func (e nwTimeoutError) Error() string {
	return e.op + " i/o timeout"
}

func (e nwTimeoutError) Timeout() bool {
	return true
}

func (e nwTimeoutError) Temporary() bool {
	return true
}

func traceNW(format string, args ...any) {
	if !networkTrace {
		return
	}
	fmt.Fprintf(os.Stderr, "nwtrace: "+format+"\n", args...)
}

func nwErrorString(err applenetwork.NWError) string {
	if err.IsZero() {
		return "-"
	}
	return fmt.Sprintf("%s domain=%s code=%d", err.Error(), err.DomainString(), err.Code())
}
