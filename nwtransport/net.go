//go:build darwin

package nwtransport

import (
	"context"
	"fmt"
	"net"

	"github.com/pion/transport/v4"
	"github.com/pion/transport/v4/stdnet"
	"github.com/tmc/apple/network/nwpacket"
)

// Config configures a Pion transport.Net backed by Network.framework UDP
// listeners.
type Config struct {
	// Packet is copied for each UDP listener created through ListenPacket.
	Packet nwpacket.Config

	// Fallback handles non-UDP operations and UDP operations that should remain
	// on Go's standard network stack.
	Fallback transport.Net
}

// Net implements transport.Net.
type Net struct {
	config       Config
	fallback     transport.Net
	listenPacket func(context.Context, nwpacket.Config) (net.PacketConn, error)
}

var _ transport.Net = (*Net)(nil)
var _ transport.UDPConn = (*udpConn)(nil)

// New creates a Network.framework-backed transport.Net.
func New(config Config) (*Net, error) {
	fallback := config.Fallback
	if fallback == nil {
		var err error
		fallback, err = stdnet.NewNet()
		if err != nil {
			return nil, fmt.Errorf("create fallback net: %w", err)
		}
	}
	return &Net{config: config, fallback: fallback}, nil
}

// ListenPacket announces on the local network address.
func (n *Net) ListenPacket(network string, address string) (net.PacketConn, error) {
	addr, ok, err := nativePacketAddress(network, address)
	if err != nil {
		return nil, err
	}
	if !ok {
		return n.fallback.ListenPacket(network, address)
	}
	return n.listenNativePacket(addr)
}

func (n *Net) listenNativePacket(addr *net.UDPAddr) (net.PacketConn, error) {
	return n.listenNativePacketContext(context.Background(), addr)
}

func (n *Net) listenNativePacketContext(ctx context.Context, addr *net.UDPAddr) (net.PacketConn, error) {
	config := n.config.Packet
	config.LocalAddr = addr
	if config.InterfaceName != "" && config.LocalAddr.Zone == "" && config.LocalAddr.IP.To4() == nil {
		config.LocalAddr.Zone = config.InterfaceName
	}
	if n.listenPacket != nil {
		return n.listenPacket(ctx, config)
	}
	return nwpacket.ListenPacketContext(ctx, config)
}

// ListenUDP acts like ListenPacket for UDP networks.
func (n *Net) ListenUDP(network string, locAddr *net.UDPAddr) (transport.UDPConn, error) {
	addr, ok := nativeUDPAddr(network, locAddr)
	if ok {
		conn, err := n.listenNativePacket(addr)
		if err != nil {
			return nil, err
		}
		return &udpConn{PacketConn: conn}, nil
	}
	return n.fallback.ListenUDP(network, locAddr)
}

// ListenTCP acts like Listen for TCP networks.
func (n *Net) ListenTCP(network string, laddr *net.TCPAddr) (transport.TCPListener, error) {
	return n.fallback.ListenTCP(network, laddr)
}

// Dial connects to the address on the named network.
func (n *Net) Dial(network, address string) (net.Conn, error) {
	conn, ok, err := n.dialNativeUDPAddress(network, address)
	if err != nil {
		return nil, err
	}
	if ok {
		return conn, nil
	}
	return n.fallback.Dial(network, address)
}

// DialUDP acts like Dial for UDP networks.
func (n *Net) DialUDP(network string, laddr, raddr *net.UDPAddr) (transport.UDPConn, error) {
	conn, ok, err := n.dialNativeUDP(network, laddr, raddr)
	if err != nil {
		return nil, err
	}
	if ok {
		return conn, nil
	}
	return n.fallback.DialUDP(network, laddr, raddr)
}

func (n *Net) dialNativeUDP(network string, laddr, raddr *net.UDPAddr) (*udpConn, bool, error) {
	local, remote, ok := n.nativeDialUDPAddrs(network, laddr, raddr)
	if !ok {
		return nil, false, nil
	}
	conn, err := n.listenNativePacket(local)
	if err != nil {
		return nil, false, err
	}
	return &udpConn{PacketConn: conn, remote: remote}, true, nil
}

func (n *Net) dialNativeUDPAddress(network, address string) (*udpConn, bool, error) {
	if !isUDPNetwork(network) {
		return nil, false, nil
	}
	raddr, err := net.ResolveUDPAddr(network, address)
	if err != nil {
		return nil, false, fmt.Errorf("resolve udp dial address %q: %w", address, err)
	}
	return n.dialNativeUDP(network, nil, raddr)
}

// DialTCP acts like Dial for TCP networks.
func (n *Net) DialTCP(network string, laddr, raddr *net.TCPAddr) (transport.TCPConn, error) {
	return n.fallback.DialTCP(network, laddr, raddr)
}

// ResolveIPAddr returns an address of IP end point.
func (n *Net) ResolveIPAddr(network, address string) (*net.IPAddr, error) {
	return n.fallback.ResolveIPAddr(network, address)
}

// ResolveUDPAddr returns an address of UDP end point.
func (n *Net) ResolveUDPAddr(network, address string) (*net.UDPAddr, error) {
	return n.fallback.ResolveUDPAddr(network, address)
}

// ResolveTCPAddr returns an address of TCP end point.
func (n *Net) ResolveTCPAddr(network, address string) (*net.TCPAddr, error) {
	return n.fallback.ResolveTCPAddr(network, address)
}

// Interfaces returns a list of the system's network interfaces.
func (n *Net) Interfaces() ([]*transport.Interface, error) {
	ifaces, err := n.fallback.Interfaces()
	if err != nil {
		return nil, err
	}
	if n.config.Packet.InterfaceName == "" {
		return ifaces, nil
	}
	out := make([]*transport.Interface, 0, len(ifaces))
	for _, iface := range ifaces {
		if iface.Name != n.config.Packet.InterfaceName {
			out = append(out, iface)
			continue
		}
		clone := transport.NewInterface(iface.Interface)
		addrs, err := iface.Addrs()
		if err != nil {
			out = append(out, clone)
			continue
		}
		for _, addr := range addrs {
			clone.AddAddress(unzoneLinkLocalAddr(addr))
		}
		out = append(out, clone)
	}
	return out, nil
}

// InterfaceByIndex returns the interface specified by index.
func (n *Net) InterfaceByIndex(index int) (*transport.Interface, error) {
	return n.fallback.InterfaceByIndex(index)
}

// InterfaceByName returns the interface specified by name.
func (n *Net) InterfaceByName(name string) (*transport.Interface, error) {
	return n.fallback.InterfaceByName(name)
}

// CreateDialer creates a dialer backed by this network.
func (n *Net) CreateDialer(dialer *net.Dialer) transport.Dialer {
	return dialerNet{
		net:      n,
		fallback: n.fallback.CreateDialer(dialer),
	}
}

type dialerNet struct {
	net      *Net
	fallback transport.Dialer
}

func (d dialerNet) Dial(network, address string) (net.Conn, error) {
	conn, ok, err := d.net.dialNativeUDPAddress(network, address)
	if err != nil {
		return nil, err
	}
	if ok {
		return conn, nil
	}
	return d.fallback.Dial(network, address)
}

// CreateListenConfig creates a listen config backed by this network.
func (n *Net) CreateListenConfig(listenerConfig *net.ListenConfig) transport.ListenConfig {
	return listenConfig{
		net:      n,
		fallback: n.fallback.CreateListenConfig(listenerConfig),
	}
}

type listenConfig struct {
	net      *Net
	fallback transport.ListenConfig
}

func (l listenConfig) Listen(ctx context.Context, network, address string) (net.Listener, error) {
	return l.fallback.Listen(ctx, network, address)
}

func (l listenConfig) ListenPacket(ctx context.Context, network, address string) (net.PacketConn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	addr, ok, err := nativePacketAddress(network, address)
	if err != nil {
		return nil, err
	}
	if !ok {
		return l.fallback.ListenPacket(ctx, network, address)
	}
	return l.net.listenNativePacketContext(ctx, addr)
}

type udpConn struct {
	net.PacketConn
	remote *net.UDPAddr
}

func (c *udpConn) RemoteAddr() net.Addr {
	if c.remote == nil {
		return nil
	}
	return copyUDPAddr(c.remote)
}

func (c *udpConn) SetReadBuffer(int) error {
	return nil
}

func (c *udpConn) SetWriteBuffer(int) error {
	return nil
}

func (c *udpConn) Read(b []byte) (int, error) {
	n, _, err := c.ReadFrom(b)
	return n, err
}

func (c *udpConn) ReadFrom(b []byte) (int, net.Addr, error) {
	for {
		n, addr, err := c.PacketConn.ReadFrom(b)
		if err != nil || c.remote == nil {
			return n, addr, err
		}
		udpAddr, ok := addr.(*net.UDPAddr)
		if !ok {
			return n, nil, transport.ErrNotUDPAddress
		}
		if sameUDPAddr(udpAddr, c.remote) {
			return n, copyUDPAddr(udpAddr), nil
		}
	}
}

func (c *udpConn) ReadFromUDP(b []byte) (int, *net.UDPAddr, error) {
	n, addr, err := c.ReadFrom(b)
	if err != nil {
		return n, nil, err
	}
	udpAddr, ok := addr.(*net.UDPAddr)
	if !ok {
		return n, nil, transport.ErrNotUDPAddress
	}
	return n, copyUDPAddr(udpAddr), nil
}

func (c *udpConn) ReadMsgUDP([]byte, []byte) (int, int, int, *net.UDPAddr, error) {
	return 0, 0, 0, nil, transport.ErrNotSupported
}

func (c *udpConn) Write(b []byte) (int, error) {
	if c.remote == nil {
		return 0, transport.ErrNoAddressAssigned
	}
	return c.WriteTo(b, c.remote)
}

func (c *udpConn) WriteToUDP(b []byte, addr *net.UDPAddr) (int, error) {
	return c.WriteTo(b, addr)
}

func (c *udpConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	if c.remote == nil {
		return c.PacketConn.WriteTo(b, addr)
	}
	if addr == nil {
		return c.PacketConn.WriteTo(b, c.remote)
	}
	udpAddr, ok := addr.(*net.UDPAddr)
	if !ok {
		return 0, transport.ErrNotUDPAddress
	}
	if !sameUDPAddr(udpAddr, c.remote) {
		return 0, transport.ErrNoAddressAssigned
	}
	return c.PacketConn.WriteTo(b, c.remote)
}

func (c *udpConn) WriteMsgUDP(b, oob []byte, addr *net.UDPAddr) (int, int, error) {
	if len(oob) != 0 {
		return 0, 0, transport.ErrNotSupported
	}
	if addr == nil {
		n, err := c.Write(b)
		return n, 0, err
	}
	n, err := c.WriteToUDP(b, addr)
	return n, 0, err
}

func nativePacketAddress(network string, address string) (*net.UDPAddr, bool, error) {
	switch network {
	case "udp", "udp4", "udp6":
	default:
		return nil, false, nil
	}
	addr, err := net.ResolveUDPAddr(network, address)
	if err != nil {
		return nil, false, fmt.Errorf("resolve udp listen address %q: %w", address, err)
	}
	addr, ok := nativeUDPAddr(network, addr)
	return addr, ok, nil
}

func nativeUDPAddr(network string, addr *net.UDPAddr) (*net.UDPAddr, bool) {
	if !isUDPNetwork(network) {
		return nil, false
	}
	if addr == nil || addr.IP == nil || addr.IP.IsUnspecified() || addr.IP.IsMulticast() {
		return nil, false
	}
	if !networkAllowsUDPAddr(network, addr) {
		return nil, false
	}
	return copyUDPAddr(addr), true
}

func (n *Net) nativeDialUDPAddrs(network string, laddr, raddr *net.UDPAddr) (*net.UDPAddr, *net.UDPAddr, bool) {
	remote, ok := nativeUDPAddr(network, raddr)
	if !ok {
		return nil, nil, false
	}
	local := laddr
	if local == nil {
		local = n.config.Packet.LocalAddr
	}
	local, ok = nativeUDPAddr(network, local)
	if !ok {
		return nil, nil, false
	}
	if local.Zone == "" && local.IP.To4() == nil && n.config.Packet.InterfaceName != "" {
		local.Zone = n.config.Packet.InterfaceName
	}
	if remote.Zone == "" && remote.IP.To4() == nil && remote.IP.IsLinkLocalUnicast() && n.config.Packet.InterfaceName != "" {
		remote.Zone = n.config.Packet.InterfaceName
	}
	return local, remote, true
}

func isUDPNetwork(network string) bool {
	switch network {
	case "udp", "udp4", "udp6":
		return true
	default:
		return false
	}
}

func networkAllowsUDPAddr(network string, addr *net.UDPAddr) bool {
	switch network {
	case "udp4":
		return addr.IP.To4() != nil
	case "udp6":
		return addr.IP.To4() == nil
	default:
		return true
	}
}

func sameUDPAddr(a, b *net.UDPAddr) bool {
	if a == nil || b == nil {
		return false
	}
	if a.Port != b.Port || !a.IP.Equal(b.IP) {
		return false
	}
	return a.Zone == b.Zone || a.Zone == "" || b.Zone == ""
}

func copyUDPAddr(addr *net.UDPAddr) *net.UDPAddr {
	if addr == nil {
		return nil
	}
	ip := append(net.IP(nil), addr.IP...)
	return &net.UDPAddr{IP: ip, Port: addr.Port, Zone: addr.Zone}
}

func unzoneLinkLocalAddr(addr net.Addr) net.Addr {
	ip := addrIP(addr)
	if ip == nil || ip.To4() != nil || !ip.IsLinkLocalUnicast() {
		return addr
	}
	return &net.IPAddr{IP: append(net.IP(nil), ip...)}
}

func addrIP(addr net.Addr) net.IP {
	switch a := addr.(type) {
	case *net.IPNet:
		return a.IP
	case *net.IPAddr:
		return a.IP
	default:
		return nil
	}
}
