//go:build darwin

package main

import (
	"context"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/pion/ice/v4"
	"github.com/pion/webrtc/v4"
	"github.com/tmc/apple/foundation"
	applenetwork "github.com/tmc/apple/network"
	privatenetwork "github.com/tmc/apple/private/network"
	"golang.org/x/sys/unix"
)

type linkProfile struct {
	Name                  string
	DefaultInterface      string
	RequiredInterfaceType applenetwork.NWInterfaceType
	IncludePeerToPeer     bool
	UseAWDL               bool
	UseP2P                bool
}

type linkInterface struct {
	Name  string
	Index int
	Flags net.Flags
	IPs   []net.IP
}

type applePolicy struct {
	IncludePeerToPeer     bool
	RequiredInterfaceType applenetwork.NWInterfaceType
}

type privatePolicy struct {
	RequiredInterfaceName string
	RequiredInterfaceType int64
	UseAWDL               bool
	UseP2P                bool
	AllowSocketAccess     bool
	ReuseLocalAddress     bool
	ProhibitFallback      bool
	Valid                 bool
	InterfaceType         int64
	InterfaceTypeString   string
	InterfaceMTU          int64
	InterfaceMulticast    bool
}

type linkUDPMux struct {
	conn    *net.UDPConn
	mux     *ice.UDPMuxDefault
	ip      net.IP
	network string
	bound   bool
}

func main() {
	profileName := flag.String("profile", "awdl", "link profile: awdl, thunderbolt, or lan")
	ifaceName := flag.String("iface", "", "network interface to constrain ICE candidates to; default depends on profile")
	mode := flag.String("mode", "check", "mode: check, gather, pair, udp, udp-listen, udp-send, udp-perf, udp-perf-listen, or udp-perf-send")
	timeout := flag.Duration("timeout", 8*time.Second, "timeout for WebRTC and UDP modes")
	peerAddr := flag.String("peer", "", "remote UDP address for udp-send, such as [fe80::1%awdl0]:12345")
	message := flag.String("message", "ping", "UDP payload for udp and udp-send")
	count := flag.Int("count", 1000, "UDP perf datagram count")
	size := flag.Int("size", 1200, "UDP perf payload size in bytes")
	warmup := flag.Int("warmup", 5, "UDP perf warm-up datagrams to omit")
	flag.Parse()

	profile, err := profileByName(*profileName)
	if err != nil {
		fail(err)
	}
	if *ifaceName != "" {
		profile.DefaultInterface = *ifaceName
	}
	if profile.DefaultInterface == "" {
		profile.DefaultInterface, err = defaultInterface(profile)
		if err != nil {
			fail(err)
		}
	}

	iface, err := inspectInterface(profile.DefaultInterface)
	if err != nil {
		fail(err)
	}
	fmt.Printf("profile=%s interface=%s index=%d flags=%s ips=%s\n",
		profile.Name,
		iface.Name,
		iface.Index,
		iface.Flags,
		ipList(iface.IPs),
	)
	if len(iface.IPs) == 0 {
		fmt.Printf("warning: %s has no usable IP addresses; WebRTC ICE gathering cannot produce host candidates\n", iface.Name)
	}

	publicPolicy, err := configurePublicPolicy(profile)
	if err != nil {
		fail(err)
	}
	fmt.Printf("apple.network include_peer_to_peer=%t required_interface_type=%s\n",
		publicPolicy.IncludePeerToPeer,
		publicPolicy.RequiredInterfaceType,
	)

	privatePolicy, err := configurePrivatePolicy(profile, iface)
	if err != nil {
		fail(err)
	}
	fmt.Printf("apple.private.network required_interface=%s required_interface_type=%d use_awdl=%t use_p2p=%t allow_socket_access=%t reuse_local_address=%t prohibit_fallback=%t valid=%t\n",
		privatePolicy.RequiredInterfaceName,
		privatePolicy.RequiredInterfaceType,
		privatePolicy.UseAWDL,
		privatePolicy.UseP2P,
		privatePolicy.AllowSocketAccess,
		privatePolicy.ReuseLocalAddress,
		privatePolicy.ProhibitFallback,
		privatePolicy.Valid,
	)
	fmt.Printf("apple.private.network interface type=%d type_string=%q mtu=%d supports_multicast=%t\n",
		privatePolicy.InterfaceType,
		privatePolicy.InterfaceTypeString,
		privatePolicy.InterfaceMTU,
		privatePolicy.InterfaceMulticast,
	)

	switch *mode {
	case "check":
		fmt.Printf("pion webrtc interface_filter=%s network_types=udp4,udp6 mdns=query-and-gather\n", iface.Name)
	case "gather":
		if err := gather(ctxWithTimeout(*timeout), iface); err != nil {
			fail(err)
		}
	case "pair":
		if err := pair(ctxWithTimeout(*timeout), iface); err != nil {
			fail(err)
		}
	case "udp":
		if err := udpEcho(ctxWithTimeout(*timeout), iface, *message); err != nil {
			fail(err)
		}
	case "udp-listen":
		if err := udpListen(ctxWithTimeout(*timeout), iface); err != nil {
			fail(err)
		}
	case "udp-send":
		if err := udpSend(ctxWithTimeout(*timeout), iface, *peerAddr, *message); err != nil {
			fail(err)
		}
	case "udp-perf":
		if err := udpPerf(ctxWithTimeout(*timeout), iface, *count, *size, *warmup); err != nil {
			fail(err)
		}
	case "udp-perf-listen":
		if err := udpPerfListen(ctxWithTimeout(*timeout), iface, *count+*warmup); err != nil {
			fail(err)
		}
	case "udp-perf-send":
		if err := udpPerfSend(ctxWithTimeout(*timeout), iface, *peerAddr, *count, *size, *warmup); err != nil {
			fail(err)
		}
	default:
		fail(fmt.Errorf("unknown -mode %q", *mode))
	}
}

func profileByName(name string) (linkProfile, error) {
	switch strings.ToLower(name) {
	case "awdl":
		return linkProfile{
			Name:                  "awdl",
			DefaultInterface:      "awdl0",
			RequiredInterfaceType: applenetwork.NWInterfaceTypeWifi,
			IncludePeerToPeer:     true,
			UseAWDL:               true,
			UseP2P:                true,
		}, nil
	case "thunderbolt", "tb":
		return linkProfile{
			Name:                  "thunderbolt",
			RequiredInterfaceType: applenetwork.NWInterfaceTypeWired,
		}, nil
	case "lan":
		return linkProfile{
			Name:                  "lan",
			DefaultInterface:      "en0",
			RequiredInterfaceType: applenetwork.NWInterfaceTypeWifi,
		}, nil
	default:
		return linkProfile{}, fmt.Errorf("unknown -profile %q", name)
	}
}

func defaultInterface(profile linkProfile) (string, error) {
	if profile.DefaultInterface != "" {
		return profile.DefaultInterface, nil
	}
	if profile.Name != "thunderbolt" {
		return "", fmt.Errorf("profile %s has no default interface", profile.Name)
	}
	for _, name := range []string{"bridge0", "en1", "en2", "en3"} {
		if _, err := net.InterfaceByName(name); err == nil {
			return name, nil
		}
	}
	return "", errors.New("could not find Thunderbolt Bridge interface; pass -iface explicitly")
}

func configurePublicPolicy(profile linkProfile) (policy applePolicy, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("configure public Network.framework policy: %v", r)
		}
	}()
	params := applenetwork.NWParametersCreateSecureUDP(nil, nil)
	if params.ID == 0 {
		return applePolicy{}, errors.New("NWParametersCreateSecureUDP returned nil")
	}
	applenetwork.NWParametersSetIncludePeerToPeer(params, profile.IncludePeerToPeer)
	applenetwork.NWParametersSetRequiredInterfaceType(params, profile.RequiredInterfaceType)
	return applePolicy{
		IncludePeerToPeer:     applenetwork.NWParametersGetIncludePeerToPeer(params),
		RequiredInterfaceType: applenetwork.NWParametersGetRequiredInterfaceType(params),
	}, nil
}

func configurePrivatePolicy(profile linkProfile, iface linkInterface) (policy privatePolicy, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("configure private Network.framework policy: %v", r)
		}
	}()
	params := privatenetwork.NewNWParameters()
	if params.GetID() == 0 {
		return privatePolicy{}, errors.New("private NWParameters returned nil")
	}
	privateIface := privatenetwork.NewNWInterfaceWithInterfaceName(foundation.NewStringWithString(iface.Name))
	if privateIface.GetID() == 0 {
		return privatePolicy{}, fmt.Errorf("private NWInterface(%s) returned nil", iface.Name)
	}
	params.SetRequiredInterface(privateIface)
	params.SetRequiredInterfaceType(int64(profile.RequiredInterfaceType))
	params.SetUseAWDL(profile.UseAWDL)
	params.SetUseP2P(profile.UseP2P)
	params.SetAllowSocketAccess(true)
	params.SetReuseLocalAddress(true)
	params.SetProhibitFallback(true)

	requiredIface := params.RequiredInterface()
	requiredName := ""
	if requiredIface.GetID() != 0 {
		requiredName = requiredIface.InterfaceName()
	}
	return privatePolicy{
		RequiredInterfaceName: requiredName,
		RequiredInterfaceType: params.RequiredInterfaceType(),
		UseAWDL:               params.UseAWDL(),
		UseP2P:                params.UseP2P(),
		AllowSocketAccess:     params.AllowSocketAccess(),
		ReuseLocalAddress:     params.ReuseLocalAddress(),
		ProhibitFallback:      params.ProhibitFallback(),
		Valid:                 params.IsValid(),
		InterfaceType:         privateIface.Type(),
		InterfaceTypeString:   privateIface.TypeString(),
		InterfaceMTU:          privateIface.Mtu(),
		InterfaceMulticast:    privateIface.SupportsMulticast(),
	}, nil
}

func inspectInterface(name string) (linkInterface, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return linkInterface{}, fmt.Errorf("find %s: %w", name, err)
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return linkInterface{}, fmt.Errorf("list %s addresses: %w", name, err)
	}
	var ips []net.IP
	for _, addr := range addrs {
		ip := addrIP(addr)
		if ip == nil || ip.IsUnspecified() || ip.IsLoopback() {
			continue
		}
		ips = append(ips, ip)
	}
	sort.Slice(ips, func(i, j int) bool { return ips[i].String() < ips[j].String() })
	return linkInterface{Name: iface.Name, Index: iface.Index, Flags: iface.Flags, IPs: ips}, nil
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

func gather(ctx context.Context, iface linkInterface) error {
	mux, err := newLinkUDPMux(iface)
	if err != nil {
		return err
	}
	defer mux.Close()
	fmt.Printf("udp mux listen=%s network=%s bound_if=%s mdns=query-and-gather\n",
		mux.conn.LocalAddr(), mux.network, boundIfString(iface, mux.bound))
	pc, err := newPeer(iface, mux)
	if err != nil {
		return err
	}
	defer pc.Close()
	if _, err := pc.CreateDataChannel("link-check", nil); err != nil {
		return fmt.Errorf("create data channel: %w", err)
	}
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("create offer: %w", err)
	}
	complete := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("set local description: %w", err)
	}
	select {
	case <-complete:
	case <-ctx.Done():
		return fmt.Errorf("gather %s candidates: %w", iface.Name, ctx.Err())
	}
	desc := pc.LocalDescription()
	if desc == nil {
		return errors.New("missing local description after gather")
	}
	candidates := candidateLines(desc.SDP)
	if len(candidates) == 0 {
		return fmt.Errorf("no ICE candidates gathered for %s", iface.Name)
	}
	for _, candidate := range candidates {
		fmt.Println(candidate)
	}
	if len(iface.IPs) != 0 && !candidatesUseInterface(candidates, iface.IPs) {
		return fmt.Errorf("gathered candidate outside %s IP set %s or mDNS publication", iface.Name, ipList(iface.IPs))
	}
	fmt.Printf("gathered %d host candidate(s) from %s-bound UDP mux\n", len(candidates), iface.Name)
	return nil
}

func pair(ctx context.Context, iface linkInterface) error {
	leftMux, err := newLinkUDPMux(iface)
	if err != nil {
		return err
	}
	defer leftMux.Close()
	rightMux, err := newLinkUDPMux(iface)
	if err != nil {
		return err
	}
	defer rightMux.Close()
	fmt.Printf("left udp mux listen=%s network=%s bound_if=%s\n",
		leftMux.conn.LocalAddr(), leftMux.network, boundIfString(iface, leftMux.bound))
	fmt.Printf("right udp mux listen=%s network=%s bound_if=%s\n",
		rightMux.conn.LocalAddr(), rightMux.network, boundIfString(iface, rightMux.bound))

	left, err := newPeer(iface, leftMux)
	if err != nil {
		return err
	}
	defer left.Close()
	right, err := newPeer(iface, rightMux)
	if err != nil {
		return err
	}
	defer right.Close()

	opened := make(chan struct{})
	received := make(chan string, 1)
	right.OnDataChannel(func(dc *webrtc.DataChannel) {
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			received <- string(msg.Data)
			_ = dc.SendText("pong")
		})
	})
	dc, err := left.CreateDataChannel("link-pair", nil)
	if err != nil {
		return fmt.Errorf("create data channel: %w", err)
	}
	dc.OnOpen(func() {
		close(opened)
		_ = dc.SendText("ping")
	})

	if err := signalNonTrickle(left, right, ctx); err != nil {
		return err
	}
	select {
	case <-opened:
	case <-ctx.Done():
		return fmt.Errorf("wait for data channel open over %s: %w", iface.Name, ctx.Err())
	}
	select {
	case got := <-received:
		if got != "ping" {
			return fmt.Errorf("received %q, want ping", got)
		}
		fmt.Printf("webrtc datachannel opened and exchanged payload over %s-constrained ICE\n", iface.Name)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for datachannel message over %s: %w", iface.Name, ctx.Err())
	}
}

func udpEcho(ctx context.Context, iface linkInterface, message string) error {
	bindIf := shouldBindUDPToInterface(iface, "")
	server, _, networkName, serverBound, err := listenUDPOnInterface(iface, bindIf)
	if err != nil {
		return err
	}
	defer server.Close()
	client, _, _, clientBound, err := listenUDPOnInterface(iface, bindIf)
	if err != nil {
		return err
	}
	defer client.Close()

	errc := make(chan error, 1)
	go func() {
		buf := make([]byte, 2048)
		_ = server.SetReadDeadline(deadline(ctx))
		n, addr, err := server.ReadFromUDP(buf)
		if err != nil {
			errc <- fmt.Errorf("server read: %w", err)
			return
		}
		reply := append([]byte("echo:"), buf[:n]...)
		if _, err := server.WriteToUDP(reply, addr); err != nil {
			errc <- fmt.Errorf("server write %s: %w", addr, err)
			return
		}
		errc <- nil
	}()

	serverAddr := server.LocalAddr().(*net.UDPAddr)
	clientAddr := client.LocalAddr().(*net.UDPAddr)
	fmt.Printf("udp server=%s client=%s network=%s server_bound_if=%s client_bound_if=%s\n",
		serverAddr, clientAddr, networkName, boundIfString(iface, serverBound), boundIfString(iface, clientBound))
	if _, err := client.WriteToUDP([]byte(message), serverAddr); err != nil {
		return fmt.Errorf("client write %s: %w", serverAddr, err)
	}
	buf := make([]byte, 2048)
	_ = client.SetReadDeadline(deadline(ctx))
	n, from, err := client.ReadFromUDP(buf)
	if err != nil {
		return fmt.Errorf("client read: %w", err)
	}
	if err := <-errc; err != nil {
		return err
	}
	want := "echo:" + message
	if got := string(buf[:n]); got != want {
		return fmt.Errorf("udp echo = %q, want %q", got, want)
	}
	fmt.Printf("udp datagram echoed over %s-bound sockets from %s payload=%q\n", iface.Name, from, message)
	return nil
}

func udpListen(ctx context.Context, iface linkInterface) error {
	bindIf := shouldBindUDPToInterface(iface, "")
	conn, _, networkName, bound, err := listenUDPOnInterface(iface, bindIf)
	if err != nil {
		return err
	}
	defer conn.Close()
	fmt.Printf("udp listen=%s network=%s bound_if=%s\n",
		conn.LocalAddr(), networkName, boundIfString(iface, bound))
	buf := make([]byte, 2048)
	_ = conn.SetReadDeadline(deadline(ctx))
	n, addr, err := conn.ReadFromUDP(buf)
	if err != nil {
		return fmt.Errorf("udp listen read: %w", err)
	}
	reply := append([]byte("echo:"), buf[:n]...)
	if _, err := conn.WriteToUDP(reply, addr); err != nil {
		return fmt.Errorf("udp listen write %s: %w", addr, err)
	}
	fmt.Printf("udp received %q from %s and echoed reply\n", string(buf[:n]), addr)
	return nil
}

func udpSend(ctx context.Context, iface linkInterface, peer, message string) error {
	if peer == "" {
		return errors.New("missing -peer for udp-send")
	}
	networkName, err := udpNetworkForPeer(peer)
	if err != nil {
		return err
	}
	bindIf := shouldBindUDPToInterface(iface, networkName)
	conn, _, bound, err := listenUDPOnInterfaceNetwork(iface, networkName, bindIf)
	if err != nil {
		return err
	}
	defer conn.Close()
	addr, err := net.ResolveUDPAddr(networkName, peer)
	if err != nil {
		return fmt.Errorf("resolve peer %q: %w", peer, err)
	}
	fmt.Printf("udp local=%s peer=%s network=%s bound_if=%s\n",
		conn.LocalAddr(), addr, networkName, boundIfString(iface, bound))
	if _, err := conn.WriteToUDP([]byte(message), addr); err != nil {
		return fmt.Errorf("udp send %s: %w", addr, err)
	}
	buf := make([]byte, 2048)
	_ = conn.SetReadDeadline(deadline(ctx))
	n, from, err := conn.ReadFromUDP(buf)
	if err != nil {
		return fmt.Errorf("udp receive echo: %w", err)
	}
	fmt.Printf("udp received %q from %s\n", string(buf[:n]), from)
	return nil
}

type udpPerfResult struct {
	Count   int
	Size    int
	Warmup  int
	Elapsed time.Duration
	RTT     []time.Duration
}

func udpPerf(ctx context.Context, iface linkInterface, count, size, warmup int) error {
	bindIf := shouldBindUDPToInterface(iface, "")
	server, _, networkName, serverBound, err := listenUDPOnInterface(iface, bindIf)
	if err != nil {
		return err
	}
	defer server.Close()
	client, _, _, clientBound, err := listenUDPOnInterface(iface, bindIf)
	if err != nil {
		return err
	}
	defer client.Close()

	errc := make(chan error, 1)
	go echoUDPPackets(ctx, server, count+warmup, errc)

	serverAddr := server.LocalAddr().(*net.UDPAddr)
	clientAddr := client.LocalAddr().(*net.UDPAddr)
	fmt.Printf("udp perf server=%s client=%s network=%s server_bound_if=%s client_bound_if=%s\n",
		serverAddr, clientAddr, networkName, boundIfString(iface, serverBound), boundIfString(iface, clientBound))
	result, err := runUDPEchoPerf(ctx, client, serverAddr, count, size, warmup)
	if err != nil {
		return err
	}
	if err := <-errc; err != nil {
		return err
	}
	printUDPPerf(result)
	return nil
}

func udpPerfListen(ctx context.Context, iface linkInterface, expected int) error {
	bindIf := shouldBindUDPToInterface(iface, "")
	conn, _, networkName, bound, err := listenUDPOnInterface(iface, bindIf)
	if err != nil {
		return err
	}
	defer conn.Close()
	fmt.Printf("udp perf listen=%s network=%s bound_if=%s\n",
		conn.LocalAddr(), networkName, boundIfString(iface, bound))

	var packets, bytes int64
	start := time.Now()
	buf := make([]byte, 64*1024)
	for {
		_ = conn.SetReadDeadline(deadline(ctx))
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil || isTimeout(err) {
				elapsed := time.Since(start)
				printUDPPerfListen(packets, bytes, elapsed)
				return nil
			}
			return fmt.Errorf("udp perf listen read: %w", err)
		}
		packets++
		bytes += int64(n)
		if _, err := conn.WriteToUDP(buf[:n], addr); err != nil {
			return fmt.Errorf("udp perf listen write %s: %w", addr, err)
		}
		if expected > 0 && packets >= int64(expected) {
			printUDPPerfListen(packets, bytes, time.Since(start))
			return nil
		}
	}
}

func udpPerfSend(ctx context.Context, iface linkInterface, peer string, count, size, warmup int) error {
	if peer == "" {
		return errors.New("missing -peer for udp-perf-send")
	}
	networkName, err := udpNetworkForPeer(peer)
	if err != nil {
		return err
	}
	bindIf := shouldBindUDPToInterface(iface, networkName)
	conn, _, bound, err := listenUDPOnInterfaceNetwork(iface, networkName, bindIf)
	if err != nil {
		return err
	}
	defer conn.Close()
	addr, err := net.ResolveUDPAddr(networkName, peer)
	if err != nil {
		return fmt.Errorf("resolve peer %q: %w", peer, err)
	}
	fmt.Printf("udp perf local=%s peer=%s network=%s bound_if=%s\n",
		conn.LocalAddr(), addr, networkName, boundIfString(iface, bound))
	result, err := runUDPEchoPerf(ctx, conn, addr, count, size, warmup)
	if err != nil {
		return err
	}
	printUDPPerf(result)
	return nil
}

func echoUDPPackets(ctx context.Context, conn *net.UDPConn, count int, errc chan<- error) {
	buf := make([]byte, 64*1024)
	for i := 0; i < count; i++ {
		_ = conn.SetReadDeadline(deadline(ctx))
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			errc <- fmt.Errorf("udp perf server read: %w", err)
			return
		}
		if _, err := conn.WriteToUDP(buf[:n], addr); err != nil {
			errc <- fmt.Errorf("udp perf server write %s: %w", addr, err)
			return
		}
	}
	errc <- nil
}

func runUDPEchoPerf(ctx context.Context, conn *net.UDPConn, addr *net.UDPAddr, count, size, warmup int) (udpPerfResult, error) {
	if count <= 0 {
		return udpPerfResult{}, errors.New("udp perf -count must be positive")
	}
	if size < 8 {
		return udpPerfResult{}, errors.New("udp perf -size must be at least 8")
	}
	if warmup < 0 {
		return udpPerfResult{}, errors.New("udp perf -warmup must be non-negative")
	}
	send := make([]byte, size)
	recv := make([]byte, max(size, 64*1024))
	rtt := make([]time.Duration, 0, count)
	for i := 0; i < warmup; i++ {
		if err := udpEchoOnce(ctx, conn, addr, send, recv, i); err != nil {
			return udpPerfResult{}, fmt.Errorf("udp perf warmup %d: %w", i, err)
		}
	}
	start := time.Now()
	for i := 0; i < count; i++ {
		before := time.Now()
		if err := udpEchoOnce(ctx, conn, addr, send, recv, warmup+i); err != nil {
			return udpPerfResult{}, fmt.Errorf("udp perf echo %d: %w", i, err)
		}
		rtt = append(rtt, time.Since(before))
	}
	return udpPerfResult{
		Count:   count,
		Size:    size,
		Warmup:  warmup,
		Elapsed: time.Since(start),
		RTT:     rtt,
	}, nil
}

func udpEchoOnce(ctx context.Context, conn *net.UDPConn, addr *net.UDPAddr, send, recv []byte, seq int) error {
	binary.BigEndian.PutUint64(send[:8], uint64(seq))
	fillPayload(send[8:], byte(seq))
	if _, err := conn.WriteToUDP(send, addr); err != nil {
		return fmt.Errorf("write %s: %w", addr, err)
	}
	_ = conn.SetReadDeadline(deadline(ctx))
	n, from, err := conn.ReadFromUDP(recv)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if n != len(send) {
		return fmt.Errorf("reply from %s has size %d, want %d", from, n, len(send))
	}
	if got := binary.BigEndian.Uint64(recv[:8]); got != uint64(seq) {
		return fmt.Errorf("reply sequence = %d, want %d", got, seq)
	}
	return nil
}

func fillPayload(buf []byte, seed byte) {
	for i := range buf {
		buf[i] = seed + byte(i)
	}
}

func printUDPPerf(result udpPerfResult) {
	rtt := append([]time.Duration(nil), result.RTT...)
	sort.Slice(rtt, func(i, j int) bool { return rtt[i] < rtt[j] })
	total := time.Duration(0)
	for _, d := range rtt {
		total += d
	}
	bytes := int64(result.Count * result.Size * 2)
	fmt.Println("[ ID] Interval           Transfer     Bitrate         Datagrams  Omit  RTT min/avg/p50/p95/max")
	fmt.Printf("[  5] 0.00-%-7.2f sec  %10s  %13s  %9d  %4d  %s/%s/%s/%s/%s\n",
		result.Elapsed.Seconds(),
		formatBytes(bytes),
		formatBitrate(bytes, result.Elapsed),
		result.Count,
		result.Warmup,
		formatDuration(rtt[0]),
		formatDuration(total/time.Duration(len(rtt))),
		formatDuration(percentileDuration(rtt, 50)),
		formatDuration(percentileDuration(rtt, 95)),
		formatDuration(rtt[len(rtt)-1]),
	)
}

func printUDPPerfListen(packets, bytes int64, elapsed time.Duration) {
	fmt.Println("[ ID] Interval           Transfer     Bitrate         Datagrams")
	fmt.Printf("[  5] 0.00-%-7.2f sec  %10s  %13s  %9d\n",
		elapsed.Seconds(), formatBytes(bytes), formatBitrate(bytes, elapsed), packets)
}

func percentileDuration(values []time.Duration, percentile int) time.Duration {
	if len(values) == 0 {
		return 0
	}
	i := (len(values)*percentile + 99) / 100
	if i <= 0 {
		i = 1
	}
	if i > len(values) {
		i = len(values)
	}
	return values[i-1]
}

func roundDuration(d time.Duration) time.Duration {
	if d > time.Second {
		return d.Round(time.Millisecond)
	}
	if d > time.Millisecond {
		return d.Round(time.Microsecond)
	}
	return d.Round(time.Nanosecond)
}

func formatBytes(bytes int64) string {
	const unit = 1024
	value := float64(bytes)
	for _, suffix := range []string{"Bytes", "KBytes", "MBytes", "GBytes"} {
		if value < unit || suffix == "GBytes" {
			return fmt.Sprintf("%.2f %s", value, suffix)
		}
		value /= unit
	}
	return fmt.Sprintf("%d Bytes", bytes)
}

func formatBitrate(bytes int64, elapsed time.Duration) string {
	if elapsed <= 0 {
		return "0 bits/sec"
	}
	bitsPerSecond := float64(bytes*8) / elapsed.Seconds()
	for _, suffix := range []string{"bits/sec", "Kbits/sec", "Mbits/sec", "Gbits/sec"} {
		if bitsPerSecond < 1000 || suffix == "Gbits/sec" {
			return fmt.Sprintf("%.2f %s", bitsPerSecond, suffix)
		}
		bitsPerSecond /= 1000
	}
	return "0 bits/sec"
}

func formatDuration(d time.Duration) string {
	d = roundDuration(d)
	switch {
	case d >= time.Second:
		return fmt.Sprintf("%.3fs", d.Seconds())
	case d >= time.Millisecond:
		return fmt.Sprintf("%.3fms", float64(d)/float64(time.Millisecond))
	case d >= time.Microsecond:
		return fmt.Sprintf("%.3fus", float64(d)/float64(time.Microsecond))
	default:
		return d.String()
	}
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func newLinkUDPMux(iface linkInterface) (*linkUDPMux, error) {
	bindIf := shouldBindUDPToInterface(iface, "")
	conn, ip, networkName, bound, err := listenUDPOnInterface(iface, bindIf)
	if err != nil {
		return nil, err
	}
	mux := ice.NewUDPMuxDefault(ice.UDPMuxParams{UDPConn: conn})
	return &linkUDPMux{conn: conn, mux: mux, ip: ip, network: networkName, bound: bound}, nil
}

func listenUDPOnInterface(iface linkInterface, bindIf bool) (*net.UDPConn, net.IP, string, bool, error) {
	ip, networkName, zone, err := listenIP(iface)
	if err != nil {
		return nil, nil, "", false, err
	}
	return listenUDP(iface, ip, networkName, zone, bindIf)
}

func listenUDPOnInterfaceNetwork(iface linkInterface, networkName string, bindIf bool) (*net.UDPConn, net.IP, bool, error) {
	ip, zone, err := listenIPForNetwork(iface, networkName)
	if err != nil {
		return nil, nil, false, err
	}
	conn, ip, _, bound, err := listenUDP(iface, ip, networkName, zone, bindIf)
	return conn, ip, bound, err
}

func listenUDP(iface linkInterface, ip net.IP, networkName, zone string, bindIf bool) (*net.UDPConn, net.IP, string, bool, error) {
	conn, err := net.ListenUDP(networkName, &net.UDPAddr{IP: ip, Port: 0, Zone: zone})
	if err != nil {
		return nil, nil, "", false, fmt.Errorf("listen %s on %s%%%s: %w", networkName, ip, zone, err)
	}
	if bindIf {
		if err := bindUDPConnToInterface(conn, networkName, iface); err != nil {
			_ = conn.Close()
			return nil, nil, "", false, err
		}
	}
	return conn, ip, networkName, bindIf, nil
}

func shouldBindUDPToInterface(iface linkInterface, networkName string) bool {
	if strings.HasPrefix(iface.Name, "awdl") {
		return true
	}
	return networkName == "udp6"
}

func boundIfString(iface linkInterface, bound bool) string {
	if !bound {
		return "none"
	}
	return fmt.Sprintf("%s(%d)", iface.Name, iface.Index)
}

func bindUDPConnToInterface(conn *net.UDPConn, networkName string, iface linkInterface) error {
	raw, err := conn.SyscallConn()
	if err != nil {
		return fmt.Errorf("udp raw conn: %w", err)
	}
	var sockErr error
	if err := raw.Control(func(fd uintptr) {
		if strings.HasSuffix(networkName, "4") {
			sockErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_BOUND_IF, iface.Index)
			return
		}
		sockErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IPV6, unix.IPV6_BOUND_IF, iface.Index)
	}); err != nil {
		return fmt.Errorf("udp control: %w", err)
	}
	if sockErr != nil {
		return fmt.Errorf("bind udp to %s ifindex %d: %w", iface.Name, iface.Index, sockErr)
	}
	return nil
}

func (m *linkUDPMux) Close() {
	if m == nil {
		return
	}
	if m.mux != nil {
		_ = m.mux.Close()
		return
	}
	if m.conn != nil {
		_ = m.conn.Close()
	}
}

func listenIP(iface linkInterface) (net.IP, string, string, error) {
	for _, ip := range iface.IPs {
		if ip4 := ip.To4(); ip4 != nil {
			return ip4, "udp4", "", nil
		}
	}
	for _, ip := range iface.IPs {
		if ip.To16() != nil {
			return ip, "udp6", iface.Name, nil
		}
	}
	return nil, "", "", fmt.Errorf("%s has no usable IP address for UDP mux", iface.Name)
}

func listenIPForNetwork(iface linkInterface, networkName string) (net.IP, string, error) {
	switch networkName {
	case "udp4":
		for _, ip := range iface.IPs {
			if ip4 := ip.To4(); ip4 != nil {
				return ip4, "", nil
			}
		}
	case "udp6":
		for _, ip := range iface.IPs {
			if ip.To4() == nil && ip.To16() != nil {
				return ip, iface.Name, nil
			}
		}
	default:
		return nil, "", fmt.Errorf("unsupported udp network %q", networkName)
	}
	return nil, "", fmt.Errorf("%s has no usable %s address", iface.Name, networkName)
}

func udpNetworkForPeer(peer string) (string, error) {
	host, _, err := net.SplitHostPort(peer)
	if err != nil {
		return "", fmt.Errorf("split peer %q: %w", peer, err)
	}
	if i := strings.LastIndex(host, "%"); i >= 0 {
		host = host[:i]
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "", fmt.Errorf("peer %q has non-IP host %q", peer, host)
	}
	if ip.To4() != nil {
		return "udp4", nil
	}
	return "udp6", nil
}

func newPeer(iface linkInterface, udpMux *linkUDPMux) (*webrtc.PeerConnection, error) {
	var se webrtc.SettingEngine
	se.SetInterfaceFilter(func(name string) bool {
		return name == iface.Name
	})
	se.SetIPFilter(func(ip net.IP) bool {
		if len(iface.IPs) == 0 {
			return true
		}
		for _, allowed := range iface.IPs {
			if ip.Equal(allowed) {
				return true
			}
		}
		return false
	})
	se.SetNetworkTypes([]webrtc.NetworkType{
		webrtc.NetworkTypeUDP4,
		webrtc.NetworkTypeUDP6,
	})
	se.SetIncludeLoopbackCandidate(false)
	se.SetICEMulticastDNSMode(ice.MulticastDNSModeQueryAndGather)
	if udpMux != nil {
		se.SetICEUDPMux(udpMux.mux)
	}
	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))
	return api.NewPeerConnection(webrtc.Configuration{})
}

func signalNonTrickle(left, right *webrtc.PeerConnection, ctx context.Context) error {
	offer, err := left.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("create offer: %w", err)
	}
	leftGathered := webrtc.GatheringCompletePromise(left)
	if err := left.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("left set local offer: %w", err)
	}
	if err := wait(ctx, leftGathered, "left gather"); err != nil {
		return err
	}
	if err := right.SetRemoteDescription(*left.LocalDescription()); err != nil {
		return fmt.Errorf("right set remote offer: %w", err)
	}
	answer, err := right.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("create answer: %w", err)
	}
	rightGathered := webrtc.GatheringCompletePromise(right)
	if err := right.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("right set local answer: %w", err)
	}
	if err := wait(ctx, rightGathered, "right gather"); err != nil {
		return err
	}
	if err := left.SetRemoteDescription(*right.LocalDescription()); err != nil {
		return fmt.Errorf("left set remote answer: %w", err)
	}
	return nil
}

func wait(ctx context.Context, ch <-chan struct{}, name string) error {
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("%s: %w", name, ctx.Err())
	}
}

func ctxWithTimeout(timeout time.Duration) context.Context {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	return ctx
}

func deadline(ctx context.Context) time.Time {
	if d, ok := ctx.Deadline(); ok {
		return d
	}
	return time.Now().Add(30 * time.Second)
}

func candidateLines(sdp string) []string {
	var out []string
	for _, line := range strings.Split(sdp, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "a=candidate:") {
			out = append(out, line)
		}
	}
	return out
}

func candidatesUseInterface(candidates []string, ips []net.IP) bool {
	for _, candidate := range candidates {
		var matched bool
		for _, ip := range ips {
			if strings.Contains(candidate, " "+ip.String()+" ") {
				matched = true
				break
			}
		}
		if !matched && strings.Contains(candidate, ".local ") {
			matched = true
		}
		if !matched {
			return false
		}
	}
	return true
}

func ipList(ips []net.IP) string {
	if len(ips) == 0 {
		return "-"
	}
	values := make([]string, 0, len(ips))
	for _, ip := range ips {
		values = append(values, ip.String())
	}
	return strings.Join(values, ",")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "link-webrtc-demo:", err)
	os.Exit(1)
}
