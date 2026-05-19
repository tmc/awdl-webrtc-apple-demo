//go:build darwin

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pion/ice/v4"
	"github.com/pion/webrtc/v4"
	"github.com/tmc/apple/dispatch"
	"github.com/tmc/apple/foundation"
	applenetwork "github.com/tmc/apple/network"
	"github.com/tmc/apple/objc"
	"github.com/tmc/apple/objectivec"
)

const (
	webRTCSignalServiceType = "_awdl-webrtc-signal._tcp"
	nwSignalDialAttempt     = 4 * time.Second
)

type nwSignalListener struct {
	name     string
	listener applenetwork.NWListener
	queue    dispatch.Queue
	conns    chan nwSignalPendingConn
}

type nwSignalConn struct {
	conn  applenetwork.NWConnection
	queue dispatch.Queue
}

type nwSignalPendingConn struct {
	conn  applenetwork.NWConnection
	queue dispatch.Queue
	ready <-chan error
	state <-chan string
}

func answerBonjour(ctx context.Context, profile linkProfile, iface linkInterface, backend udpBackend, usePionNet bool, mdnsMode ice.MulticastDNSMode, candidatePolicy candidatePolicyConfig, signalName string, signalOnly bool) error {
	listener, err := newNWSignalListener(ctx, profile, signalName)
	if err != nil {
		return err
	}
	defer listener.Close()
	fmt.Printf("signal service=%s type=%s domain=%s\n", listener.name, webRTCSignalServiceType, linkHealthDomain)

	link, err := newLinkWebRTCNet(profile, iface, backend, usePionNet)
	if err != nil {
		return err
	}
	defer link.Close()
	link.print("answer", iface, mdnsMode, candidatePolicy)

	pc, err := newPeer(iface, link, mdnsMode, candidatePolicy.Policy)
	if err != nil {
		return err
	}
	defer pc.Close()
	diag := newWebRTCDiagnostics("answer", pc)

	received := make(chan string, 1)
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		name := "remote:" + dc.Label()
		diag.setDataChannel(name, dc)
		dc.OnOpen(func() {
			diag.dataChannelState(name, "open")
		})
		dc.OnClose(func() {
			diag.dataChannelState(name, "closed")
		})
		dc.OnError(func(err error) {
			diag.dataChannelError(name, err)
		})
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			received <- string(msg.Data)
			_ = dc.SendText("pong")
		})
	})

	signal, err := listener.Accept(ctx)
	if err != nil {
		return err
	}
	defer signal.Close()

	offer, err := signal.ReadWireSignal(ctx, "OFFER")
	if err != nil {
		return err
	}
	if err := setRemoteWireSignal(pc, offer, "offer"); err != nil {
		return err
	}
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("create answer: %w", err)
	}
	gathered := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("set local answer: %w", err)
	}
	if err := wait(ctx, gathered, "answer gather"); err != nil {
		return err
	}
	wire := newWireSignal(*pc.LocalDescription(), candidatePolicy.Policy, link.ip)
	if err := signal.WriteWireSignal(ctx, "ANSWER", wire); err != nil {
		return err
	}
	encoded, err := encodeWireSignal(wire)
	if err != nil {
		return err
	}
	fmt.Printf("ANSWER %s\n", encoded)
	if signalOnly {
		fmt.Printf("bonjour signal exchanged offer/answer as %s over %s\n", listener.name, iface.Name)
		time.Sleep(200 * time.Millisecond)
		return nil
	}

	select {
	case got := <-received:
		fmt.Printf("webrtc answer received %q and sent pong over %s-constrained ICE\n", got, iface.Name)
		time.Sleep(500 * time.Millisecond)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for answer datachannel message over %s: %w; %s", iface.Name, ctx.Err(), diag.snapshot())
	}
}

func offerBonjour(ctx context.Context, profile linkProfile, iface linkInterface, backend udpBackend, usePionNet bool, mdnsMode ice.MulticastDNSMode, candidatePolicy candidatePolicyConfig, signalPeer string, signalOnly bool) error {
	if strings.TrimSpace(signalPeer) == "" {
		return errors.New("missing -signal-peer for offer-bonjour")
	}
	link, err := newLinkWebRTCNet(profile, iface, backend, usePionNet)
	if err != nil {
		return err
	}
	defer link.Close()
	link.print("offer", iface, mdnsMode, candidatePolicy)

	pc, err := newPeer(iface, link, mdnsMode, candidatePolicy.Policy)
	if err != nil {
		return err
	}
	defer pc.Close()
	diag := newWebRTCDiagnostics("offer", pc)

	opened := make(chan struct{})
	received := make(chan string, 1)
	dc, err := pc.CreateDataChannel("link-bonjour", nil)
	if err != nil {
		return fmt.Errorf("create data channel: %w", err)
	}
	diag.setDataChannel("local:link-bonjour", dc)
	dc.OnOpen(func() {
		diag.dataChannelState("local:link-bonjour", "open")
		close(opened)
		_ = dc.SendText("ping")
	})
	dc.OnClose(func() {
		diag.dataChannelState("local:link-bonjour", "closed")
	})
	dc.OnError(func(err error) {
		diag.dataChannelError("local:link-bonjour", err)
	})
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		received <- string(msg.Data)
	})

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("create offer: %w", err)
	}
	gathered := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("set local offer: %w", err)
	}
	if err := wait(ctx, gathered, "offer gather"); err != nil {
		return err
	}
	wireOffer := newWireSignal(*pc.LocalDescription(), candidatePolicy.Policy, link.ip)

	signal, err := dialNWSignal(ctx, profile, strings.TrimSpace(signalPeer))
	if err != nil {
		return err
	}
	defer signal.Close()
	if err := signal.WriteWireSignal(ctx, "OFFER", wireOffer); err != nil {
		return err
	}
	answer, err := signal.ReadWireSignal(ctx, "ANSWER")
	if err != nil {
		return err
	}
	if err := setRemoteWireSignal(pc, answer, "answer"); err != nil {
		return err
	}
	if signalOnly {
		fmt.Printf("bonjour signal exchanged offer/answer with %s over %s\n", signalPeer, iface.Name)
		time.Sleep(200 * time.Millisecond)
		return nil
	}
	select {
	case <-opened:
	case <-ctx.Done():
		return fmt.Errorf("wait for data channel open over %s: %w; %s", iface.Name, ctx.Err(), diag.snapshot())
	}
	select {
	case got := <-received:
		if got != "pong" {
			return fmt.Errorf("received %q, want pong", got)
		}
		fmt.Printf("webrtc datachannel opened and exchanged payload with %s over %s-constrained ICE\n", signalPeer, iface.Name)
	case <-ctx.Done():
		return fmt.Errorf("wait for datachannel pong over %s: %w; %s", iface.Name, ctx.Err(), diag.snapshot())
	}
	return nil
}

func newNWSignalListener(ctx context.Context, profile linkProfile, name string) (*nwSignalListener, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = linkHealthServiceName()
	}
	listener := applenetwork.NWListenerCreateWithPort("0", signalTCPParams(profile))
	if listener.ID == 0 {
		return nil, errors.New("signal listen: nil listener")
	}
	queue := dispatch.QueueCreate("awdl-webrtc.signal.listen")
	desc := applenetwork.NWAdvertiseDescriptorCreateBonjourService(name, webRTCSignalServiceType, linkHealthDomain)
	meta := map[string]string{
		"id":      name,
		"name":    name,
		"modes":   linkHealthModes,
		"version": linkHealthVersion(),
	}
	if commit := linkHealthCommit(); commit != "" {
		meta["commit"] = commit
	}
	applenetwork.NWAdvertiseDescriptorSetTXTRecordObject(desc, linkHealthTXTRecord(meta))
	applenetwork.NWListenerSetAdvertiseDescriptor(listener, desc)
	applenetwork.NWListenerSetQueue(listener, queue)

	ready := make(chan error, 1)
	conns := make(chan nwSignalPendingConn, 1)
	applenetwork.NWListenerSetNewConnectionHandler(listener, func(obj objectivec.Object) {
		if obj.ID == 0 {
			return
		}
		conn := applenetwork.NWConnectionFromID(obj.ID)
		pending := startPendingNWSignalConn(conn, queue)
		select {
		case conns <- pending:
		default:
			applenetwork.NWConnectionCancel(conn)
		}
	})
	applenetwork.NWListenerSetStateChangedHandler(listener, func(state applenetwork.NWListenerState, nwErr applenetwork.NWError) {
		switch state {
		case applenetwork.NWListenerStateReady:
			signalError(ready, nil)
		case applenetwork.NWListenerStateFailed, applenetwork.NWListenerStateCancelled:
			err := fmt.Errorf("signal listener %s", state)
			if !nwErr.IsZero() {
				err = fmt.Errorf("%s: %w", err, nwErr)
			}
			signalError(ready, err)
		}
	})
	applenetwork.NWListenerStart(listener)
	select {
	case err := <-ready:
		if err != nil {
			applenetwork.NWListenerCancel(listener)
			return nil, err
		}
	case <-ctx.Done():
		applenetwork.NWListenerCancel(listener)
		return nil, ctx.Err()
	}
	return &nwSignalListener{name: name, listener: listener, queue: queue, conns: conns}, nil
}

func (l *nwSignalListener) Accept(ctx context.Context) (*nwSignalConn, error) {
	select {
	case pending := <-l.conns:
		return waitNWSignalConn(ctx, pending)
	case <-ctx.Done():
		return nil, fmt.Errorf("wait for bonjour signal connection: %w", ctx.Err())
	}
}

func (l *nwSignalListener) Close() {
	if l.listener.ID != 0 {
		applenetwork.NWListenerCancel(l.listener)
	}
}

func dialNWSignal(ctx context.Context, profile linkProfile, serviceName string) (*nwSignalConn, error) {
	endpoint, err := browseNWSignalEndpoint(ctx, profile, serviceName)
	if err != nil {
		return nil, err
	}
	if endpoint.ID == 0 {
		return nil, fmt.Errorf("signal dial %q: nil endpoint", serviceName)
	}
	conn := applenetwork.NWConnectionCreate(endpoint, signalTCPParams(profile))
	if conn.ID == 0 {
		return nil, fmt.Errorf("signal dial %q: nil connection", serviceName)
	}
	pending := startPendingNWSignalConn(conn, dispatch.QueueCreate("awdl-webrtc.signal.dial"))
	attemptCtx, cancel := signalDialAttemptContext(ctx)
	signal, err := waitNWSignalConn(attemptCtx, pending)
	cancel()
	if err == nil {
		return signal, nil
	}
	directErr := err

	host, port, err := resolveNWSignalHostPort(ctx, profile, serviceName)
	if err != nil {
		return nil, fmt.Errorf("signal dial %q via bonjour endpoint: %w; resolve host endpoint: %w", serviceName, directErr, err)
	}
	hostEndpoint := applenetwork.NWEndpointCreateHost(host, port)
	if hostEndpoint.ID == 0 {
		return nil, fmt.Errorf("signal dial %q host %s:%s: nil endpoint", serviceName, host, port)
	}
	hostConn := applenetwork.NWConnectionCreate(hostEndpoint, signalTCPParams(profile))
	if hostConn.ID == 0 {
		return nil, fmt.Errorf("signal dial %q host %s:%s: nil connection", serviceName, host, port)
	}
	pending = startPendingNWSignalConn(hostConn, dispatch.QueueCreate("awdl-webrtc.signal.dial.host"))
	signal, err = waitNWSignalConn(ctx, pending)
	if err != nil {
		return nil, fmt.Errorf("signal dial %q via bonjour endpoint: %w; host endpoint %s:%s: %w", serviceName, directErr, host, port, err)
	}
	return signal, nil
}

func browseNWSignalEndpoint(ctx context.Context, profile linkProfile, serviceName string) (applenetwork.NWEndpoint, error) {
	desc := applenetwork.NWBrowseDescriptorCreateBonjourService(webRTCSignalServiceType, linkHealthDomain)
	applenetwork.NWBrowseDescriptorSetIncludeTXTRecord(desc, true)
	browser := applenetwork.NWBrowserCreate(desc, signalTCPParams(profile))
	if browser.ID == 0 {
		return applenetwork.NWEndpoint{}, errors.New("signal browse: nil browser")
	}
	defer applenetwork.NWBrowserCancel(browser)

	queue := dispatch.QueueCreate("awdl-webrtc.signal.browse")
	found := make(chan applenetwork.NWEndpoint, 1)
	failed := make(chan error, 1)
	applenetwork.NWBrowserSetQueue(browser, queue)
	applenetwork.NWBrowserSetBrowseResultsChangedHandler(browser, func(_, result objectivec.Object, _ bool) {
		if result.ID == 0 {
			return
		}
		endpoint := applenetwork.NWBrowseResultCopyEndpoint(result)
		if endpoint.ID == 0 {
			return
		}
		if objc.GoString(applenetwork.NWEndpointGetBonjourServiceName(endpoint)) != serviceName {
			return
		}
		select {
		case found <- endpoint:
		default:
		}
	})
	applenetwork.NWBrowserSetStateChangedHandler(browser, func(state applenetwork.NWBrowserState, nwErr applenetwork.NWError) {
		switch state {
		case applenetwork.NWBrowserStateFailed, applenetwork.NWBrowserStateCancelled:
			err := fmt.Errorf("signal browse %s", state)
			if !nwErr.IsZero() {
				err = fmt.Errorf("%s: %w", err, nwErr)
			}
			signalError(failed, err)
		}
	})
	applenetwork.NWBrowserStart(browser)
	select {
	case endpoint := <-found:
		return endpoint, nil
	case err := <-failed:
		return applenetwork.NWEndpoint{}, err
	case <-ctx.Done():
		return applenetwork.NWEndpoint{}, fmt.Errorf("browse bonjour signal %q: %w", serviceName, ctx.Err())
	}
}

func signalTCPParams(profile linkProfile) applenetwork.NWParameters {
	params := applenetwork.NWParametersCreatePlainTCP(nil)
	stack := applenetwork.NWParametersCopyDefaultProtocolStack(params)
	tcpOpts := applenetwork.NWProtocolStackCopyTransportProtocol(stack)
	applenetwork.NWTCPOptionsSetNoDelay(tcpOpts, true)
	applenetwork.NWParametersSetIncludePeerToPeer(params, profile.IncludePeerToPeer)
	if profile.Name != "lan" {
		applenetwork.NWParametersSetRequiredInterfaceType(params, profile.RequiredInterfaceType)
	}
	if profile.UseAWDL {
		linkHealthSetPrivateBool(params, "setUseAWDL:", true)
	}
	if profile.UseP2P {
		linkHealthSetPrivateBool(params, "setUseP2P:", true)
	}
	return params
}

func signalDialAttemptContext(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := nwSignalDialAttempt
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			timeout = remaining
		}
	}
	if timeout <= 0 {
		timeout = time.Nanosecond
	}
	return context.WithTimeout(ctx, timeout)
}

func resolveNWSignalHostPort(ctx context.Context, profile linkProfile, serviceName string) (string, string, error) {
	service := foundation.NewNetServiceWithDomainTypeName(linkHealthDomain, webRTCSignalServiceType, serviceName)
	if service.ID == 0 {
		return "", "", fmt.Errorf("resolve bonjour signal %q: nil service", serviceName)
	}
	service.SetIncludesPeerToPeer(profile.IncludePeerToPeer || profile.UseAWDL || profile.UseP2P)

	type result struct {
		host string
		port string
		err  error
	}
	done := make(chan result, 1)
	delegate := foundation.NewNSNetServiceDelegate(foundation.NSNetServiceDelegateConfig{
		NetServiceDidResolveAddress: func(sender foundation.NSNetService) {
			host := strings.TrimSpace(sender.HostName())
			port := sender.Port()
			if host == "" || port <= 0 {
				done <- result{err: fmt.Errorf("resolved empty host or port: host=%q port=%d", host, port)}
				return
			}
			done <- result{host: host, port: strconv.Itoa(port)}
		},
	})
	service.SetDelegate(delegate)
	resolveTimeout := nwSignalDialAttempt.Seconds()
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline).Seconds()
		if remaining < resolveTimeout {
			resolveTimeout = remaining
		}
	}
	if resolveTimeout <= 0 {
		resolveTimeout = 0.001
	}
	objc.Send[struct{}](service.ID, objc.Sel("resolveWithTimeout:"), resolveTimeout)
	defer objc.Send[struct{}](service.ID, objc.Sel("stop"))

	loop := foundation.GetRunLoopClass().CurrentRunLoop()
	for {
		select {
		case got := <-done:
			runtime.KeepAlive(delegate)
			runtime.KeepAlive(service)
			return got.host, got.port, got.err
		case <-ctx.Done():
			runtime.KeepAlive(delegate)
			runtime.KeepAlive(service)
			return "", "", fmt.Errorf("resolve bonjour signal %q: %w", serviceName, ctx.Err())
		default:
			loop.RunModeBeforeDate(foundation.RunLoopDefaultMode, foundation.NewDateWithTimeIntervalSinceNow(0.05))
		}
	}
}

func startPendingNWSignalConn(conn applenetwork.NWConnection, queue dispatch.Queue) nwSignalPendingConn {
	ready := make(chan error, 1)
	statec := make(chan string, 8)
	applenetwork.NWConnectionSetQueue(conn, queue)
	applenetwork.NWConnectionSetStateChangedHandler(conn, func(state applenetwork.NWConnectionState, nwErr applenetwork.NWError) {
		stateText := state.String()
		if !nwErr.IsZero() {
			stateText += ": " + nwErr.Description()
		}
		select {
		case statec <- stateText:
		default:
		}
		switch state {
		case applenetwork.NWConnectionStateReady:
			signalError(ready, nil)
		case applenetwork.NWConnectionStateFailed, applenetwork.NWConnectionStateCancelled:
			err := fmt.Errorf("signal connection %s", state)
			if !nwErr.IsZero() {
				err = fmt.Errorf("%s: %w", err, nwErr)
			}
			signalError(ready, err)
		}
	})
	applenetwork.NWConnectionStart(conn)
	return nwSignalPendingConn{conn: conn, queue: queue, ready: ready, state: statec}
}

func waitNWSignalConn(ctx context.Context, pending nwSignalPendingConn) (*nwSignalConn, error) {
	lastState := "none"
	for {
		select {
		case state := <-pending.state:
			lastState = state
		case err := <-pending.ready:
			if err != nil {
				applenetwork.NWConnectionCancel(pending.conn)
				return nil, err
			}
			return &nwSignalConn{conn: pending.conn, queue: pending.queue}, nil
		case <-ctx.Done():
			applenetwork.NWConnectionCancel(pending.conn)
			return nil, fmt.Errorf("wait for bonjour signal connection: %w; last_state=%s", ctx.Err(), lastState)
		}
	}
}

func (c *nwSignalConn) ReadWireSignal(ctx context.Context, prefix string) (wireSignal, error) {
	var data []byte
	for {
		chunk, err := c.receive(ctx)
		if err != nil {
			return wireSignal{}, err
		}
		data = append(data, chunk...)
		if len(data) > 1024*1024 {
			return wireSignal{}, fmt.Errorf("signal %s line too large", prefix)
		}
		lines := strings.Split(string(data), "\n")
		complete := lines
		if !strings.HasSuffix(string(data), "\n") {
			complete = lines[:len(lines)-1]
		}
		for _, line := range complete {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, prefix+" ") {
				continue
			}
			return decodeWireSignal(strings.TrimSpace(strings.TrimPrefix(line, prefix+" ")))
		}
		if !strings.HasSuffix(string(data), "\n") {
			data = []byte(lines[len(lines)-1])
		} else {
			data = data[:0]
		}
	}
}

func (c *nwSignalConn) WriteWireSignal(ctx context.Context, prefix string, signal wireSignal) error {
	value, err := encodeWireSignal(signal)
	if err != nil {
		return err
	}
	return c.writeLine(ctx, prefix+" "+value)
}

func (c *nwSignalConn) receive(ctx context.Context) ([]byte, error) {
	type result struct {
		data     []byte
		complete bool
		err      error
	}
	done := make(chan result, 1)
	applenetwork.NWConnectionReceive(c.conn, 1, 64*1024, func(content objectivec.Object, _ objectivec.Object, complete bool, nwErr applenetwork.NWError) {
		if !nwErr.IsZero() {
			done <- result{err: nwErr}
			return
		}
		if content.ID == 0 {
			done <- result{complete: complete}
			return
		}
		data := dispatch.DataFromHandle(uintptr(content.ID)).Bytes()
		done <- result{data: append([]byte(nil), data...), complete: complete}
	})
	select {
	case got := <-done:
		if got.err != nil {
			return nil, fmt.Errorf("receive bonjour signal: %w", got.err)
		}
		if len(got.data) == 0 && got.complete {
			return nil, io.EOF
		}
		return got.data, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("receive bonjour signal: %w", ctx.Err())
	}
}

func (c *nwSignalConn) writeLine(ctx context.Context, line string) error {
	data := dispatch.DataCreate([]byte(line + "\n"))
	content := applenetwork.NWContentContextCreate("awdl-webrtc-signal")
	done := make(chan error, 1)
	applenetwork.NWConnectionSend(c.conn, data, content, false, func(nwErr applenetwork.NWError) {
		data.Release()
		if content.ID != 0 {
			content.Release()
		}
		if nwErr.IsZero() {
			done <- nil
			return
		}
		done <- nwErr
	})
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("send bonjour signal: %w", err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("send bonjour signal: %w", ctx.Err())
	}
}

func (c *nwSignalConn) Close() {
	if c.conn.ID != 0 {
		applenetwork.NWConnectionCancel(c.conn)
	}
}
