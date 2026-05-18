//go:build darwin

package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/pion/ice/v4"
	piontransport "github.com/pion/transport/v4"
	"github.com/pion/webrtc/v4"
	"github.com/tmc/apple-pion/nwtransport"
	applenetwork "github.com/tmc/apple/network"
	"github.com/tmc/apple/objc"
	"github.com/tmc/apple/objectivec"
	"github.com/tmc/apple/x/network/nwpacket"
	"github.com/tmc/awdl-webrtc-apple-demo/internal/icepolicy"
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
	conn    net.PacketConn
	mux     *ice.UDPMuxDefault
	ip      net.IP
	network string
	bound   bool
	backend udpBackend
}

type linkPacketConn struct {
	conn    net.PacketConn
	ip      net.IP
	network string
	bound   bool
	backend udpBackend
}

type linkWebRTCNet struct {
	mux     *linkUDPMux
	net     piontransport.Net
	ip      net.IP
	network string
	bound   bool
	backend udpBackend
}

type udpBackend string

const (
	udpBackendGo      udpBackend = "go"
	udpBackendNetwork udpBackend = "network"
)

func main() {
	profileName := flag.String("profile", "awdl", "link profile: awdl, thunderbolt, or lan")
	ifaceName := flag.String("iface", "", "network interface to constrain ICE candidates to; default depends on profile")
	backendName := flag.String("backend", string(udpBackendGo), "UDP backend: go or network")
	pionNet := flag.Bool("pion-net", false, "use Network.framework as Pion transport.Net instead of ICE UDP mux")
	mdnsName := flag.String("mdns", "query-and-gather", "ICE mDNS mode: query-and-gather, query-only, or disabled")
	mode := flag.String("mode", "check", "mode: check, gather, pair, answer-stdio, offer-ssh, udp, udp-listen, udp-send, udp-callback-listen, udp-callback-request, udp-perf, udp-perf-listen, udp-perf-send, or ui")
	timeout := flag.Duration("timeout", 8*time.Second, "timeout for WebRTC and UDP modes")
	peerAddr := flag.String("peer", "", "remote UDP address for udp-send, such as [fe80::1%awdl0]:12345")
	sshTarget := flag.String("ssh", "", "ssh target for offer-ssh, such as tmc2@10.0.18.249")
	remoteBin := flag.String("remote-bin", "/tmp/awdl-webrtc-apple-demo-bin", "remote binary path for offer-ssh")
	rawCandidates := flag.Bool("raw-candidates", false, "rewrite gathered host candidates to the selected interface IP during explicit signaling")
	message := flag.String("message", "ping", "UDP payload for udp and udp-send")
	count := flag.Int("count", 1000, "UDP perf datagram count")
	duration := flag.Duration("duration", 0, "UDP perf trial duration; when set, run each trial for this long instead of using -count")
	size := flag.Int("size", 1200, "UDP perf payload size in bytes")
	warmup := flag.Int("warmup", 5, "UDP perf warm-up datagrams to omit")
	trials := flag.Int("trials", 1, "UDP perf trial count")
	window := flag.Int("window", 1, "UDP perf maximum in-flight echo requests")
	packetTimeout := flag.Duration("packet-timeout", time.Second, "UDP perf per-datagram echo timeout")
	perfJSON := flag.Bool("perf-json", false, "also print UDP perf result records as JSON lines")
	uiInterval := flag.Duration("ui-interval", 3*time.Second, "SwiftUI link monitor sample interval")
	uiCount := flag.Int("ui-count", 20, "SwiftUI link monitor datagrams per sample")
	uiWindow := flag.Int("ui-window", 4, "SwiftUI link monitor maximum in-flight datagrams per sample")
	flag.Parse()

	profile, err := profileByName(*profileName)
	if err != nil {
		fail(err)
	}
	backend, err := parseUDPBackend(*backendName)
	if err != nil {
		fail(err)
	}
	if *pionNet && backend != udpBackendNetwork {
		fail(errors.New("-pion-net requires -backend network"))
	}
	mdnsMode, err := parseMDNSMode(*mdnsName)
	if err != nil {
		fail(err)
	}
	candidatePolicy := icepolicy.Policy{RawHostCandidates: *rawCandidates}
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
		fmt.Printf("pion webrtc interface_filter=%s network_types=udp4,udp6 mdns=%s udp_backend=%s pion_net=%t\n", iface.Name, mdnsModeString(mdnsMode), backend, *pionNet)
	case "gather":
		runWithTimeout(*timeout, func(ctx context.Context) error {
			return gather(ctx, profile, iface, backend, *pionNet, mdnsMode, candidatePolicy)
		})
	case "pair":
		runWithTimeout(*timeout, func(ctx context.Context) error {
			return pair(ctx, profile, iface, backend, *pionNet, mdnsMode, candidatePolicy)
		})
	case "answer-stdio":
		runWithTimeout(*timeout, func(ctx context.Context) error {
			return answerStdio(ctx, profile, iface, backend, *pionNet, mdnsMode, candidatePolicy)
		})
	case "offer-ssh":
		runWithTimeout(*timeout, func(ctx context.Context) error {
			return offerSSH(ctx, profile, iface, backend, *pionNet, mdnsMode, candidatePolicy, *timeout, *sshTarget, *remoteBin)
		})
	case "udp":
		runWithTimeout(*timeout, func(ctx context.Context) error {
			return udpEcho(ctx, profile, iface, backend, *message)
		})
	case "udp-listen":
		runWithTimeout(*timeout, func(ctx context.Context) error {
			return udpListen(ctx, profile, iface, backend)
		})
	case "udp-send":
		runWithTimeout(*timeout, func(ctx context.Context) error {
			return udpSend(ctx, profile, iface, backend, *peerAddr, *message)
		})
	case "udp-callback-listen":
		runWithTimeout(*timeout, func(ctx context.Context) error {
			return udpCallbackListen(ctx, profile, iface, backend)
		})
	case "udp-callback-request":
		runWithTimeout(*timeout, func(ctx context.Context) error {
			return udpCallbackRequest(ctx, profile, iface, backend, *peerAddr, *message)
		})
	case "udp-perf":
		runWithTimeout(*timeout, func(ctx context.Context) error {
			return udpPerf(ctx, profile, iface, backend, *count, *size, *warmup, *trials, *window, *packetTimeout, *duration, *perfJSON)
		})
	case "udp-perf-listen":
		runWithTimeout(*timeout, func(ctx context.Context) error {
			return udpPerfListen(ctx, profile, iface, backend, *count+*warmup, *perfJSON)
		})
	case "udp-perf-send":
		runWithTimeout(*timeout, func(ctx context.Context) error {
			return udpPerfSend(ctx, profile, iface, backend, *peerAddr, *count, *size, *warmup, *trials, *window, *packetTimeout, *duration, *perfJSON)
		})
	case "ui":
		if err := runLinkHealthUI(context.Background(), linkHealthConfig{
			Backend:       backend,
			Interval:      *uiInterval,
			Count:         *uiCount,
			Size:          *size,
			Window:        *uiWindow,
			PacketTimeout: *packetTimeout,
		}); err != nil {
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

func parseUDPBackend(name string) (udpBackend, error) {
	switch udpBackend(strings.ToLower(name)) {
	case udpBackendGo:
		return udpBackendGo, nil
	case udpBackendNetwork:
		return udpBackendNetwork, nil
	default:
		return "", fmt.Errorf("unknown -backend %q", name)
	}
}

func parseMDNSMode(name string) (ice.MulticastDNSMode, error) {
	switch strings.ToLower(name) {
	case "query-and-gather", "queryandgather", "gather", "":
		return ice.MulticastDNSModeQueryAndGather, nil
	case "query-only", "queryonly":
		return ice.MulticastDNSModeQueryOnly, nil
	case "disabled", "disable", "off":
		return ice.MulticastDNSModeDisabled, nil
	default:
		return 0, fmt.Errorf("unknown -mdns %q", name)
	}
}

func mdnsModeString(mode ice.MulticastDNSMode) string {
	switch mode {
	case ice.MulticastDNSModeQueryAndGather:
		return "query-and-gather"
	case ice.MulticastDNSModeQueryOnly:
		return "query-only"
	case ice.MulticastDNSModeDisabled:
		return "disabled"
	default:
		return fmt.Sprintf("unknown(%d)", mode)
	}
}

func defaultInterface(profile linkProfile) (string, error) {
	if profile.DefaultInterface != "" {
		return profile.DefaultInterface, nil
	}
	if profile.Name != "thunderbolt" {
		return "", fmt.Errorf("profile %s has no default interface", profile.Name)
	}
	return defaultThunderboltInterface(inspectInterface)
}

func defaultThunderboltInterface(inspect func(string) (linkInterface, error)) (string, error) {
	var firstExisting string
	for _, name := range []string{"bridge0", "en1", "en2", "en3"} {
		iface, err := inspect(name)
		if err != nil {
			continue
		}
		if firstExisting == "" {
			firstExisting = name
		}
		if len(iface.IPs) != 0 {
			return name, nil
		}
	}
	if firstExisting != "" {
		return firstExisting, nil
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
	params := newPrivateNWParameters()
	if params.ID == 0 {
		return privatePolicy{}, errors.New("private NWParameters returned nil")
	}
	privateIface := newPrivateNWInterfaceWithName(iface.Name)
	if privateIface.ID == 0 {
		return privatePolicy{}, fmt.Errorf("private NWInterface(%s) returned nil", iface.Name)
	}
	privateSetObject(params, "setRequiredInterface:", privateIface)
	privateSetInt64(params, "setRequiredInterfaceType:", int64(profile.RequiredInterfaceType))
	privateSetBool(params, "setUseAWDL:", profile.UseAWDL)
	privateSetBool(params, "setUseP2P:", profile.UseP2P)
	privateSetBool(params, "setAllowSocketAccess:", true)
	privateSetBool(params, "setReuseLocalAddress:", true)
	privateSetBool(params, "setProhibitFallback:", true)

	requiredIface := privateGetObject(params, "requiredInterface")
	requiredName := ""
	if requiredIface.ID != 0 {
		requiredName = privateString(requiredIface, "interfaceName")
	}
	return privatePolicy{
		RequiredInterfaceName: requiredName,
		RequiredInterfaceType: privateInt64(params, "requiredInterfaceType"),
		UseAWDL:               privateBool(params, "useAWDL"),
		UseP2P:                privateBool(params, "useP2P"),
		AllowSocketAccess:     privateBool(params, "allowSocketAccess"),
		ReuseLocalAddress:     privateBool(params, "reuseLocalAddress"),
		ProhibitFallback:      privateBool(params, "prohibitFallback"),
		Valid:                 privateBool(params, "isValid"),
		InterfaceType:         privateInt64(privateIface, "type"),
		InterfaceTypeString:   privateString(privateIface, "typeString"),
		InterfaceMTU:          privateInt64(privateIface, "mtu"),
		InterfaceMulticast:    privateBool(privateIface, "supportsMulticast"),
	}, nil
}

func newPrivateNWParameters() objectivec.Object {
	return objectivec.ObjectFromID(objc.Send[objc.ID](objc.ID(objc.GetClass("NWParameters")), objc.Sel("new")))
}

func newPrivateNWInterfaceWithName(name string) objectivec.Object {
	instance := objc.Send[objc.ID](objc.ID(objc.GetClass("NWInterface")), objc.Sel("alloc"))
	return objectivec.ObjectFromID(objc.Send[objc.ID](instance, objc.Sel("initWithInterfaceName:"), objectivec.ObjectFromID(objc.String(name))))
}

func privateSetObject(obj objectivec.Object, selector string, value objectivec.Object) {
	objc.Send[struct{}](obj.ID, objc.Sel(selector), value)
}

func privateSetBool(obj objectivec.Object, selector string, value bool) {
	objc.Send[struct{}](obj.ID, objc.Sel(selector), value)
}

func privateSetInt64(obj objectivec.Object, selector string, value int64) {
	objc.Send[struct{}](obj.ID, objc.Sel(selector), value)
}

func privateGetObject(obj objectivec.Object, selector string) objectivec.Object {
	return objectivec.ObjectFromID(objc.Send[objc.ID](obj.ID, objc.Sel(selector)))
}

func privateBool(obj objectivec.Object, selector string) bool {
	return objc.Send[bool](obj.ID, objc.Sel(selector))
}

func privateInt64(obj objectivec.Object, selector string) int64 {
	return objc.Send[int64](obj.ID, objc.Sel(selector))
}

func privateString(obj objectivec.Object, selector string) string {
	id := objc.Send[objc.ID](obj.ID, objc.Sel(selector))
	if id == 0 {
		return ""
	}
	return objc.IDToString(id)
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

func gather(ctx context.Context, profile linkProfile, iface linkInterface, backend udpBackend, usePionNet bool, mdnsMode ice.MulticastDNSMode, candidatePolicy icepolicy.Policy) error {
	link, err := newLinkWebRTCNet(profile, iface, backend, usePionNet)
	if err != nil {
		return err
	}
	defer link.Close()
	link.print("gather", iface, mdnsMode, candidatePolicy)
	pc, err := newPeer(iface, link, mdnsMode, candidatePolicy)
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
	candidates := publishedCandidateLines(desc.SDP, candidatePolicy, link.ip)
	if len(candidates) == 0 {
		return fmt.Errorf("no ICE candidates gathered for %s", iface.Name)
	}
	for _, candidate := range candidates {
		fmt.Println(candidate)
	}
	if len(iface.IPs) != 0 && !candidatesUseInterface(candidates, iface.IPs) {
		return fmt.Errorf("gathered candidate outside %s IP set %s or mDNS publication", iface.Name, ipList(iface.IPs))
	}
	if link.net != nil {
		fmt.Printf("gathered %d host candidate(s) from %s-bound Pion transport.Net\n", len(candidates), iface.Name)
		return nil
	}
	fmt.Printf("gathered %d host candidate(s) from %s-bound UDP mux\n", len(candidates), iface.Name)
	return nil
}

func pair(ctx context.Context, profile linkProfile, iface linkInterface, backend udpBackend, usePionNet bool, mdnsMode ice.MulticastDNSMode, candidatePolicy icepolicy.Policy) error {
	leftLink, err := newLinkWebRTCNet(profile, iface, backend, usePionNet)
	if err != nil {
		return err
	}
	defer leftLink.Close()
	rightLink, err := newLinkWebRTCNet(profile, iface, backend, usePionNet)
	if err != nil {
		return err
	}
	defer rightLink.Close()
	leftLink.print("left", iface, mdnsMode, candidatePolicy)
	rightLink.print("right", iface, mdnsMode, candidatePolicy)

	left, err := newPeer(iface, leftLink, mdnsMode, candidatePolicy)
	if err != nil {
		return err
	}
	defer left.Close()
	right, err := newPeer(iface, rightLink, mdnsMode, candidatePolicy)
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

	if err := signalNonTrickle(left, right, ctx, signalOptions{
		CandidatePolicy: candidatePolicy,
		LeftIP:          leftLink.ip,
		RightIP:         rightLink.ip,
	}); err != nil {
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

func answerStdio(ctx context.Context, profile linkProfile, iface linkInterface, backend udpBackend, usePionNet bool, mdnsMode ice.MulticastDNSMode, candidatePolicy icepolicy.Policy) error {
	link, err := newLinkWebRTCNet(profile, iface, backend, usePionNet)
	if err != nil {
		return err
	}
	defer link.Close()
	link.print("answer", iface, mdnsMode, candidatePolicy)

	pc, err := newPeer(iface, link, mdnsMode, candidatePolicy)
	if err != nil {
		return err
	}
	defer pc.Close()

	received := make(chan string, 1)
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			received <- string(msg.Data)
			_ = dc.SendText("pong")
		})
	})

	offer, err := readWireSignal(bufio.NewScanner(os.Stdin), "OFFER")
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
	wire, err := encodeWireSignal(newWireSignal(*pc.LocalDescription(), candidatePolicy, link.ip))
	if err != nil {
		return err
	}
	fmt.Printf("ANSWER %s\n", wire)

	select {
	case got := <-received:
		fmt.Printf("webrtc answer received %q and sent pong over %s-constrained ICE\n", got, iface.Name)
		time.Sleep(500 * time.Millisecond)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for answer datachannel message over %s: %w", iface.Name, ctx.Err())
	}
}

func offerSSH(ctx context.Context, profile linkProfile, iface linkInterface, backend udpBackend, usePionNet bool, mdnsMode ice.MulticastDNSMode, candidatePolicy icepolicy.Policy, timeout time.Duration, sshTarget, remoteBin string) error {
	if sshTarget == "" {
		return errors.New("missing -ssh for offer-ssh")
	}
	link, err := newLinkWebRTCNet(profile, iface, backend, usePionNet)
	if err != nil {
		return err
	}
	defer link.Close()
	link.print("offer", iface, mdnsMode, candidatePolicy)

	pc, err := newPeer(iface, link, mdnsMode, candidatePolicy)
	if err != nil {
		return err
	}
	defer pc.Close()

	opened := make(chan struct{})
	received := make(chan string, 1)
	dc, err := pc.CreateDataChannel("link-ssh", nil)
	if err != nil {
		return fmt.Errorf("create data channel: %w", err)
	}
	dc.OnOpen(func() {
		close(opened)
		_ = dc.SendText("ping")
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
	wireOffer, err := encodeWireSignal(newWireSignal(*pc.LocalDescription(), candidatePolicy, link.ip))
	if err != nil {
		return err
	}

	cmdArgs := []string{
		sshTarget,
		remoteBin,
		"-profile", profile.Name,
		"-iface", iface.Name,
		"-backend", string(backend),
		"-mdns", mdnsModeString(mdnsMode),
		"-mode", "answer-stdio",
		"-timeout", timeout.String(),
	}
	if candidatePolicy.RawHostCandidates {
		cmdArgs = append(cmdArgs, "-raw-candidates")
	}
	if usePionNet {
		cmdArgs = append(cmdArgs, "-pion-net")
	}
	cmd := exec.CommandContext(ctx, "ssh", cmdArgs...)
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("ssh stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("ssh stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ssh: %w", err)
	}
	waitc := make(chan error, 1)
	go func() { waitc <- cmd.Wait() }()

	answerc := make(chan wireSignal, 1)
	scanErrc := make(chan error, 1)
	go scanRemoteAnswer(stdout, answerc, scanErrc)

	if _, err := fmt.Fprintf(stdin, "OFFER %s\n", wireOffer); err != nil {
		return fmt.Errorf("write offer to ssh: %w", err)
	}
	if err := stdin.Close(); err != nil {
		return fmt.Errorf("close ssh stdin: %w", err)
	}

	var answer wireSignal
	select {
	case answer = <-answerc:
	case err := <-scanErrc:
		return err
	case <-ctx.Done():
		return fmt.Errorf("wait for ssh answer: %w", ctx.Err())
	}
	if err := setRemoteWireSignal(pc, answer, "answer"); err != nil {
		return err
	}
	select {
	case <-opened:
	case <-ctx.Done():
		return fmt.Errorf("wait for data channel open over %s: %w", iface.Name, ctx.Err())
	}
	select {
	case got := <-received:
		if got != "pong" {
			return fmt.Errorf("received %q, want pong", got)
		}
		fmt.Printf("webrtc datachannel opened and exchanged payload with %s over %s-constrained ICE\n", sshTarget, iface.Name)
	case <-ctx.Done():
		return fmt.Errorf("wait for datachannel pong over %s: %w", iface.Name, ctx.Err())
	}
	select {
	case err := <-waitc:
		if err != nil {
			return fmt.Errorf("ssh answer process: %w", err)
		}
	case <-time.After(5 * time.Second):
		return errors.New("ssh answer process did not exit after datachannel exchange")
	}
	return nil
}

func udpEcho(ctx context.Context, profile linkProfile, iface linkInterface, backend udpBackend, message string) error {
	serverLink, err := newLinkPacketConn(profile, iface, backend, "")
	if err != nil {
		return err
	}
	defer serverLink.conn.Close()
	clientLink, err := newLinkPacketConn(profile, iface, backend, "")
	if err != nil {
		return err
	}
	defer clientLink.conn.Close()
	server := serverLink.conn
	client := clientLink.conn

	errc := make(chan error, 1)
	go func() {
		buf := make([]byte, 2048)
		_ = server.SetReadDeadline(deadline(ctx))
		n, addr, err := server.ReadFrom(buf)
		if err != nil {
			errc <- fmt.Errorf("server read: %w", err)
			return
		}
		reply := append([]byte("echo:"), buf[:n]...)
		_ = server.SetWriteDeadline(deadline(ctx))
		if _, err := server.WriteTo(reply, addr); err != nil {
			errc <- fmt.Errorf("server write %s: %w", addr, err)
			return
		}
		errc <- nil
	}()

	serverAddr := server.LocalAddr()
	clientAddr := client.LocalAddr()
	fmt.Printf("udp server=%s client=%s network=%s backend=%s server_bound_if=%s client_bound_if=%s\n",
		serverAddr, clientAddr, serverLink.network, backend, boundIfString(iface, serverLink.bound), boundIfString(iface, clientLink.bound))
	if _, err := client.WriteTo([]byte(message), serverAddr); err != nil {
		return fmt.Errorf("client write %s: %w", serverAddr, err)
	}
	buf := make([]byte, 2048)
	_ = client.SetReadDeadline(deadline(ctx))
	n, from, err := client.ReadFrom(buf)
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

func udpListen(ctx context.Context, profile linkProfile, iface linkInterface, backend udpBackend) error {
	link, err := newLinkPacketConn(profile, iface, backend, "")
	if err != nil {
		return err
	}
	conn := link.conn
	defer conn.Close()
	fmt.Printf("udp listen=%s network=%s backend=%s bound_if=%s\n",
		conn.LocalAddr(), link.network, link.backend, boundIfString(iface, link.bound))
	buf := make([]byte, 2048)
	_ = conn.SetReadDeadline(deadline(ctx))
	n, addr, err := conn.ReadFrom(buf)
	if err != nil {
		return fmt.Errorf("udp listen read: %w", err)
	}
	reply := append([]byte("echo:"), buf[:n]...)
	_ = conn.SetWriteDeadline(deadline(ctx))
	if _, err := conn.WriteTo(reply, addr); err != nil {
		return fmt.Errorf("udp listen write %s: %w", addr, err)
	}
	fmt.Printf("udp received %q from %s and echoed reply\n", string(buf[:n]), addr)
	return nil
}

func udpSend(ctx context.Context, profile linkProfile, iface linkInterface, backend udpBackend, peer, message string) error {
	if peer == "" {
		return errors.New("missing -peer for udp-send")
	}
	networkName, err := udpNetworkForPeer(peer)
	if err != nil {
		return err
	}
	link, err := newLinkPacketConn(profile, iface, backend, networkName)
	if err != nil {
		return err
	}
	conn := link.conn
	defer conn.Close()
	addr, err := net.ResolveUDPAddr(networkName, peer)
	if err != nil {
		return fmt.Errorf("resolve peer %q: %w", peer, err)
	}
	fmt.Printf("udp local=%s peer=%s network=%s backend=%s bound_if=%s\n",
		conn.LocalAddr(), addr, networkName, link.backend, boundIfString(iface, link.bound))
	if _, err := conn.WriteTo([]byte(message), addr); err != nil {
		return fmt.Errorf("udp send %s: %w", addr, err)
	}
	buf := make([]byte, 2048)
	_ = conn.SetReadDeadline(deadline(ctx))
	n, from, err := conn.ReadFrom(buf)
	if err != nil {
		return fmt.Errorf("udp receive echo: %w", err)
	}
	fmt.Printf("udp received %q from %s\n", string(buf[:n]), from)
	return nil
}

type udpCallbackWireRequest struct {
	Callback string `json:"callback"`
	Message  string `json:"message,omitempty"`
}

func udpCallbackListen(ctx context.Context, profile linkProfile, iface linkInterface, backend udpBackend) error {
	link, err := newLinkPacketConn(profile, iface, backend, "")
	if err != nil {
		return err
	}
	defer link.conn.Close()
	conn := link.conn
	fmt.Printf("udp callback listen=%s network=%s backend=%s bound_if=%s\n",
		conn.LocalAddr(), link.network, link.backend, boundIfString(iface, link.bound))

	buf := make([]byte, 4096)
	_ = conn.SetReadDeadline(deadline(ctx))
	n, from, err := conn.ReadFrom(buf)
	if err != nil {
		return fmt.Errorf("udp callback listen read: %w", err)
	}
	req, err := parseUDPCallbackRequest(buf[:n])
	if err != nil {
		return fmt.Errorf("udp callback request from %s: %w", from, err)
	}
	if err := sendUDPCallback(ctx, profile, iface, backend, link, req); err != nil {
		return err
	}
	fmt.Printf("udp callback received request from %s callback=%s sent=%q\n",
		from, req.Callback, callbackPayload(req.Message))
	return nil
}

func udpCallbackRequest(ctx context.Context, profile linkProfile, iface linkInterface, backend udpBackend, peer, message string) error {
	if peer == "" {
		return errors.New("missing -peer for udp-callback-request")
	}
	networkName, err := udpNetworkForPeer(peer)
	if err != nil {
		return err
	}
	link, err := newLinkPacketConn(profile, iface, backend, networkName)
	if err != nil {
		return err
	}
	defer link.conn.Close()
	conn := link.conn
	addr, err := net.ResolveUDPAddr(networkName, peer)
	if err != nil {
		return fmt.Errorf("resolve peer %q: %w", peer, err)
	}
	req, err := marshalUDPCallbackRequest(conn.LocalAddr().String(), message)
	if err != nil {
		return err
	}
	fmt.Printf("udp callback local=%s peer=%s network=%s backend=%s bound_if=%s\n",
		conn.LocalAddr(), addr, networkName, link.backend, boundIfString(iface, link.bound))
	_ = conn.SetWriteDeadline(deadline(ctx))
	if _, err := conn.WriteTo(req, addr); err != nil {
		return fmt.Errorf("udp callback request %s: %w", addr, err)
	}
	buf := make([]byte, 4096)
	_ = conn.SetReadDeadline(deadline(ctx))
	n, from, err := conn.ReadFrom(buf)
	if err != nil {
		return fmt.Errorf("udp callback receive: %w", err)
	}
	want := callbackPayload(message)
	if got := string(buf[:n]); got != want {
		return fmt.Errorf("udp callback = %q, want %q", got, want)
	}
	fmt.Printf("udp callback received %q from %s\n", string(buf[:n]), from)
	return nil
}

func sendUDPCallback(ctx context.Context, profile linkProfile, iface linkInterface, backend udpBackend, link *linkPacketConn, req udpCallbackWireRequest) error {
	networkName, err := udpNetworkForPeer(req.Callback)
	if err != nil {
		return err
	}
	conn := link.conn
	if networkName != link.network {
		sendLink, err := newLinkPacketConn(profile, iface, backend, networkName)
		if err != nil {
			return err
		}
		defer sendLink.conn.Close()
		conn = sendLink.conn
	}
	addr, err := net.ResolveUDPAddr(networkName, req.Callback)
	if err != nil {
		return fmt.Errorf("resolve callback %q: %w", req.Callback, err)
	}
	_ = conn.SetWriteDeadline(deadline(ctx))
	if _, err := conn.WriteTo([]byte(callbackPayload(req.Message)), addr); err != nil {
		return fmt.Errorf("udp callback write %s: %w", addr, err)
	}
	return nil
}

func marshalUDPCallbackRequest(callback, message string) ([]byte, error) {
	if strings.TrimSpace(callback) == "" {
		return nil, errors.New("missing callback address")
	}
	return json.Marshal(udpCallbackWireRequest{Callback: callback, Message: message})
}

func parseUDPCallbackRequest(data []byte) (udpCallbackWireRequest, error) {
	var req udpCallbackWireRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return udpCallbackWireRequest{}, err
	}
	if strings.TrimSpace(req.Callback) == "" {
		return udpCallbackWireRequest{}, errors.New("missing callback address")
	}
	return req, nil
}

func callbackPayload(message string) string {
	return "callback:" + message
}

type udpPerfResult struct {
	Count    int
	Duration time.Duration
	Size     int
	Warmup   int
	Window   int
	Lost     int
	Elapsed  time.Duration
	RTT      []time.Duration
}

type udpPerfRecord struct {
	Kind          string  `json:"kind"`
	Trial         int     `json:"trial,omitempty"`
	Trials        int     `json:"trials,omitempty"`
	Count         int     `json:"count,omitempty"`
	Size          int     `json:"size,omitempty"`
	DurationNS    int64   `json:"duration_ns,omitempty"`
	Warmup        int     `json:"warmup,omitempty"`
	Window        int     `json:"window,omitempty"`
	Datagrams     int     `json:"datagrams"`
	Lost          int     `json:"lost"`
	LossPercent   float64 `json:"loss_percent"`
	TransferBytes int64   `json:"transfer_bytes"`
	BitrateBPS    float64 `json:"bitrate_bps"`
	ElapsedNS     int64   `json:"elapsed_ns"`
	RTTMinNS      int64   `json:"rtt_min_ns"`
	RTTAvgNS      int64   `json:"rtt_avg_ns"`
	RTTP50NS      int64   `json:"rtt_p50_ns"`
	RTTP95NS      int64   `json:"rtt_p95_ns"`
	RTTMaxNS      int64   `json:"rtt_max_ns"`
	Expected      int64   `json:"expected,omitempty"`
}

func udpPerf(ctx context.Context, profile linkProfile, iface linkInterface, backend udpBackend, count, size, warmup, trials, window int, packetTimeout, duration time.Duration, jsonOut bool) error {
	if trials <= 0 {
		return errors.New("udp perf -trials must be positive")
	}
	if window <= 0 {
		return errors.New("udp perf -window must be positive")
	}
	serverLink, err := newLinkPacketConn(profile, iface, backend, "")
	if err != nil {
		return err
	}
	defer serverLink.conn.Close()
	clientLink, err := newLinkPacketConn(profile, iface, backend, "")
	if err != nil {
		return err
	}
	defer clientLink.conn.Close()
	server := serverLink.conn
	client := clientLink.conn

	errc := make(chan error, 1)
	serverCtx := ctx
	cancelServer := func() {}
	expected := (count + warmup) * trials
	if duration > 0 {
		serverCtx, cancelServer = context.WithCancel(ctx)
		expected = 0
	}
	defer cancelServer()
	go echoUDPPackets(serverCtx, server, expected, errc)

	serverAddr := server.LocalAddr()
	clientAddr := client.LocalAddr()
	fmt.Printf("udp perf server=%s client=%s network=%s backend=%s server_bound_if=%s client_bound_if=%s window=%d%s\n",
		serverAddr, clientAddr, serverLink.network, backend, boundIfString(iface, serverLink.bound), boundIfString(iface, clientLink.bound), window, durationSuffix(duration))
	results := make([]udpPerfResult, 0, trials)
	for trial := 1; trial <= trials; trial++ {
		if trials > 1 {
			fmt.Printf("udp perf trial=%d/%d\n", trial, trials)
		}
		result, err := runUDPEchoPerf(ctx, client, serverAddr, count, size, warmup, window, packetTimeout, duration)
		if err != nil {
			return err
		}
		printUDPPerf(result)
		if jsonOut {
			printJSON(udpPerfRecordForTrial(result, trial, trials))
		}
		results = append(results, result)
	}
	if duration > 0 {
		cancelServer()
	}
	if err := <-errc; err != nil {
		return err
	}
	printUDPPerfSummary(results, jsonOut)
	return nil
}

func udpPerfListen(ctx context.Context, profile linkProfile, iface linkInterface, backend udpBackend, expected int, jsonOut bool) error {
	link, err := newLinkPacketConn(profile, iface, backend, "")
	if err != nil {
		return err
	}
	conn := link.conn
	defer conn.Close()
	fmt.Printf("udp perf listen=%s network=%s backend=%s bound_if=%s\n",
		conn.LocalAddr(), link.network, link.backend, boundIfString(iface, link.bound))

	var packets, bytes int64
	start := time.Now()
	buf := make([]byte, 64*1024)
	for {
		_ = conn.SetReadDeadline(deadline(ctx))
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			if ctx.Err() != nil || isTimeout(err) {
				elapsed := time.Since(start)
				printUDPPerfListen(packets, bytes, elapsed, int64(expected))
				if jsonOut {
					printJSON(udpPerfListenRecord(packets, bytes, elapsed, int64(expected)))
				}
				return nil
			}
			return fmt.Errorf("udp perf listen read: %w", err)
		}
		packets++
		bytes += int64(n)
		_ = conn.SetWriteDeadline(deadline(ctx))
		if _, err := conn.WriteTo(buf[:n], addr); err != nil {
			return fmt.Errorf("udp perf listen write %s: %w", addr, err)
		}
		if expected > 0 && packets >= int64(expected) {
			elapsed := time.Since(start)
			printUDPPerfListen(packets, bytes, elapsed, int64(expected))
			if jsonOut {
				printJSON(udpPerfListenRecord(packets, bytes, elapsed, int64(expected)))
			}
			return nil
		}
	}
}

func udpPerfSend(ctx context.Context, profile linkProfile, iface linkInterface, backend udpBackend, peer string, count, size, warmup, trials, window int, packetTimeout, duration time.Duration, jsonOut bool) error {
	if peer == "" {
		return errors.New("missing -peer for udp-perf-send")
	}
	if trials <= 0 {
		return errors.New("udp perf -trials must be positive")
	}
	if window <= 0 {
		return errors.New("udp perf -window must be positive")
	}
	networkName, err := udpNetworkForPeer(peer)
	if err != nil {
		return err
	}
	link, err := newLinkPacketConn(profile, iface, backend, networkName)
	if err != nil {
		return err
	}
	conn := link.conn
	defer conn.Close()
	addr, err := net.ResolveUDPAddr(networkName, peer)
	if err != nil {
		return fmt.Errorf("resolve peer %q: %w", peer, err)
	}
	fmt.Printf("udp perf local=%s peer=%s network=%s backend=%s bound_if=%s window=%d%s\n",
		conn.LocalAddr(), addr, networkName, link.backend, boundIfString(iface, link.bound), window, durationSuffix(duration))
	results := make([]udpPerfResult, 0, trials)
	for trial := 1; trial <= trials; trial++ {
		if trials > 1 {
			fmt.Printf("udp perf trial=%d/%d\n", trial, trials)
		}
		result, err := runUDPEchoPerf(ctx, conn, addr, count, size, warmup, window, packetTimeout, duration)
		if err != nil {
			return err
		}
		printUDPPerf(result)
		if jsonOut {
			printJSON(udpPerfRecordForTrial(result, trial, trials))
		}
		results = append(results, result)
	}
	printUDPPerfSummary(results, jsonOut)
	return nil
}

func echoUDPPackets(ctx context.Context, conn net.PacketConn, count int, errc chan<- error) {
	buf := make([]byte, 64*1024)
	for packets := 0; count <= 0 || packets < count; {
		if count <= 0 {
			_ = conn.SetReadDeadline(packetDeadline(ctx, 250*time.Millisecond))
		} else {
			_ = conn.SetReadDeadline(deadline(ctx))
		}
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			if ctx.Err() != nil {
				errc <- nil
				return
			}
			if count <= 0 && isTimeout(err) {
				continue
			}
			errc <- fmt.Errorf("udp perf server read: %w", err)
			return
		}
		packets++
		_ = conn.SetWriteDeadline(deadline(ctx))
		if _, err := conn.WriteTo(buf[:n], addr); err != nil {
			errc <- fmt.Errorf("udp perf server write %s: %w", addr, err)
			return
		}
	}
	errc <- nil
}

func runUDPEchoPerf(ctx context.Context, conn net.PacketConn, addr net.Addr, count, size, warmup, window int, packetTimeout, duration time.Duration) (udpPerfResult, error) {
	if count <= 0 && duration <= 0 {
		return udpPerfResult{}, errors.New("udp perf -count must be positive")
	}
	if size < 8 {
		return udpPerfResult{}, errors.New("udp perf -size must be at least 8")
	}
	if warmup < 0 {
		return udpPerfResult{}, errors.New("udp perf -warmup must be non-negative")
	}
	if window <= 0 {
		return udpPerfResult{}, errors.New("udp perf -window must be positive")
	}
	if packetTimeout < 0 {
		return udpPerfResult{}, errors.New("udp perf -packet-timeout must be non-negative")
	}
	if duration < 0 {
		return udpPerfResult{}, errors.New("udp perf -duration must be non-negative")
	}
	send := make([]byte, size)
	recv := make([]byte, max(size, 64*1024))
	for i := 0; i < warmup; i++ {
		if _, err := udpEchoOnce(ctx, conn, addr, send, recv, i, packetTimeout); err != nil {
			return udpPerfResult{}, fmt.Errorf("udp perf warmup %d: %w", i, err)
		}
	}
	rttCap := count
	if rttCap < 0 {
		rttCap = 0
	}
	rtt := make([]time.Duration, 0, rttCap)
	start := time.Now()
	lost := 0
	var err error
	if duration > 0 {
		if window == 1 {
			count, lost, rtt, err = runUDPEchoPerfDuration(ctx, conn, addr, send, recv, warmup, packetTimeout, duration)
		} else {
			count, rtt, lost, err = runUDPEchoPerfWindowDuration(ctx, conn, addr, size, warmup, window, packetTimeout, duration)
		}
		if err != nil {
			return udpPerfResult{}, err
		}
	} else if window == 1 {
		for i := 0; i < count; i++ {
			before := time.Now()
			ok, err := udpEchoOnce(ctx, conn, addr, send, recv, warmup+i, packetTimeout)
			if err != nil {
				return udpPerfResult{}, fmt.Errorf("udp perf echo %d: %w", i, err)
			}
			if !ok {
				lost++
				continue
			}
			rtt = append(rtt, time.Since(before))
		}
	} else {
		windowRTT, windowLost, err := runUDPEchoPerfWindow(ctx, conn, addr, count, size, warmup, window, packetTimeout)
		if err != nil {
			return udpPerfResult{}, err
		}
		rtt = append(rtt, windowRTT...)
		lost = windowLost
	}
	return udpPerfResult{
		Count:    count,
		Duration: duration,
		Size:     size,
		Warmup:   warmup,
		Window:   window,
		Lost:     lost,
		Elapsed:  time.Since(start),
		RTT:      rtt,
	}, nil
}

func runUDPEchoPerfDuration(ctx context.Context, conn net.PacketConn, addr net.Addr, send, recv []byte, seqBase int, packetTimeout, duration time.Duration) (int, int, []time.Duration, error) {
	end := time.Now().Add(duration)
	rtt := make([]time.Duration, 0, 1024)
	count := 0
	lost := 0
	for count == 0 || time.Now().Before(end) {
		before := time.Now()
		ok, err := udpEchoOnce(ctx, conn, addr, send, recv, seqBase+count, packetTimeout)
		if err != nil {
			return 0, 0, nil, fmt.Errorf("udp perf echo %d: %w", count, err)
		}
		count++
		if !ok {
			lost++
			continue
		}
		rtt = append(rtt, time.Since(before))
	}
	return count, lost, rtt, nil
}

type udpPendingPacket struct {
	sent time.Time
}

func runUDPEchoPerfWindow(ctx context.Context, conn net.PacketConn, addr net.Addr, count, size, seqBase, window int, packetTimeout time.Duration) ([]time.Duration, int, error) {
	recv := make([]byte, max(size, 64*1024))
	pending := make(map[uint64]udpPendingPacket)
	rtt := make([]time.Duration, 0, count)
	lost := 0
	nextSeq := seqBase
	limit := seqBase + count
	for nextSeq < limit || len(pending) != 0 {
		for nextSeq < limit && len(pending) < window {
			if err := writeUDPEchoPacket(ctx, conn, addr, size, nextSeq); err != nil {
				return nil, 0, fmt.Errorf("udp perf echo %d: %w", nextSeq-seqBase, err)
			}
			pending[uint64(nextSeq)] = udpPendingPacket{sent: time.Now()}
			nextSeq++
		}
		if len(pending) == 0 {
			continue
		}
		_ = conn.SetReadDeadline(windowReadDeadline(ctx, pending, packetTimeout))
		n, from, err := conn.ReadFrom(recv)
		now := time.Now()
		if err != nil {
			if isTimeout(err) && ctx.Err() == nil && packetTimeout > 0 {
				lost += expirePending(pending, now, packetTimeout)
				continue
			}
			return nil, 0, fmt.Errorf("read: %w", err)
		}
		if n != size {
			return nil, 0, fmt.Errorf("reply from %s has size %d, want %d", from, n, size)
		}
		seq := binary.BigEndian.Uint64(recv[:8])
		packet, ok := pending[seq]
		if !ok {
			continue
		}
		delete(pending, seq)
		rtt = append(rtt, now.Sub(packet.sent))
	}
	return rtt, lost, nil
}

func runUDPEchoPerfWindowDuration(ctx context.Context, conn net.PacketConn, addr net.Addr, size, seqBase, window int, packetTimeout, duration time.Duration) (int, []time.Duration, int, error) {
	recv := make([]byte, max(size, 64*1024))
	pending := make(map[uint64]udpPendingPacket)
	rtt := make([]time.Duration, 0, 1024)
	lost := 0
	nextSeq := seqBase
	count := 0
	end := time.Now().Add(duration)
	for count == 0 || time.Now().Before(end) || len(pending) != 0 {
		for (count == 0 || time.Now().Before(end)) && len(pending) < window {
			if err := writeUDPEchoPacket(ctx, conn, addr, size, nextSeq); err != nil {
				return 0, nil, 0, fmt.Errorf("udp perf echo %d: %w", count, err)
			}
			pending[uint64(nextSeq)] = udpPendingPacket{sent: time.Now()}
			nextSeq++
			count++
		}
		if len(pending) == 0 {
			continue
		}
		_ = conn.SetReadDeadline(windowReadDeadline(ctx, pending, packetTimeout))
		n, from, err := conn.ReadFrom(recv)
		now := time.Now()
		if err != nil {
			if isTimeout(err) && ctx.Err() == nil && packetTimeout > 0 {
				lost += expirePending(pending, now, packetTimeout)
				continue
			}
			return 0, nil, 0, fmt.Errorf("read: %w", err)
		}
		if n != size {
			return 0, nil, 0, fmt.Errorf("reply from %s has size %d, want %d", from, n, size)
		}
		seq := binary.BigEndian.Uint64(recv[:8])
		packet, ok := pending[seq]
		if !ok {
			continue
		}
		delete(pending, seq)
		rtt = append(rtt, now.Sub(packet.sent))
	}
	return count, rtt, lost, nil
}

func udpEchoOnce(ctx context.Context, conn net.PacketConn, addr net.Addr, send, recv []byte, seq int, packetTimeout time.Duration) (bool, error) {
	binary.BigEndian.PutUint64(send[:8], uint64(seq))
	fillPayload(send[8:], byte(seq))
	_ = conn.SetWriteDeadline(deadline(ctx))
	if _, err := conn.WriteTo(send, addr); err != nil {
		return false, fmt.Errorf("write %s: %w", addr, err)
	}
	_ = conn.SetReadDeadline(packetDeadline(ctx, packetTimeout))
	n, from, err := conn.ReadFrom(recv)
	if err != nil {
		if isTimeout(err) && ctx.Err() == nil {
			return false, nil
		}
		return false, fmt.Errorf("read: %w", err)
	}
	if n != len(send) {
		return false, fmt.Errorf("reply from %s has size %d, want %d", from, n, len(send))
	}
	if got := binary.BigEndian.Uint64(recv[:8]); got != uint64(seq) {
		return false, fmt.Errorf("reply sequence = %d, want %d", got, seq)
	}
	return true, nil
}

func writeUDPEchoPacket(ctx context.Context, conn net.PacketConn, addr net.Addr, size, seq int) error {
	send := make([]byte, size)
	binary.BigEndian.PutUint64(send[:8], uint64(seq))
	fillPayload(send[8:], byte(seq))
	_ = conn.SetWriteDeadline(deadline(ctx))
	if n, err := conn.WriteTo(send, addr); err != nil {
		return fmt.Errorf("write %s: %w", addr, err)
	} else if n != len(send) {
		return fmt.Errorf("write %s: wrote %d bytes, want %d", addr, n, len(send))
	}
	return nil
}

func windowReadDeadline(ctx context.Context, pending map[uint64]udpPendingPacket, timeout time.Duration) time.Time {
	if timeout <= 0 {
		return deadline(ctx)
	}
	var earliest time.Time
	for _, packet := range pending {
		d := packet.sent.Add(timeout)
		if earliest.IsZero() || d.Before(earliest) {
			earliest = d
		}
	}
	if ctxDeadline, ok := ctx.Deadline(); ok && (earliest.IsZero() || ctxDeadline.Before(earliest)) {
		return ctxDeadline
	}
	return earliest
}

func expirePending(pending map[uint64]udpPendingPacket, now time.Time, timeout time.Duration) int {
	var lost int
	for seq, packet := range pending {
		if !packet.sent.Add(timeout).After(now) {
			delete(pending, seq)
			lost++
		}
	}
	return lost
}

func fillPayload(buf []byte, seed byte) {
	for i := range buf {
		buf[i] = seed + byte(i)
	}
}

func printUDPPerf(result udpPerfResult) {
	record := udpPerfRecordForTrial(result, 0, 0)
	fmt.Println("[ ID] Interval           Transfer     Bitrate         Datagrams  Lost  Loss    Omit  RTT min/avg/p50/p95/max")
	if record.Datagrams == 0 {
		fmt.Printf("[  5] 0.00-%-7.2f sec  %10s  %13s  %9d  %4d  %6s  %4d  -/-/-/-/-\n",
			result.Elapsed.Seconds(),
			formatBytes(record.TransferBytes),
			formatBitrate(record.TransferBytes, result.Elapsed),
			result.Count,
			result.Lost,
			formatLoss(result.Lost, result.Count),
			result.Warmup,
		)
		return
	}
	fmt.Printf("[  5] 0.00-%-7.2f sec  %10s  %13s  %9d  %4d  %6s  %4d  %s/%s/%s/%s/%s\n",
		result.Elapsed.Seconds(),
		formatBytes(record.TransferBytes),
		formatBitrate(record.TransferBytes, result.Elapsed),
		result.Count,
		result.Lost,
		formatLoss(result.Lost, result.Count),
		result.Warmup,
		formatDuration(time.Duration(record.RTTMinNS)),
		formatDuration(time.Duration(record.RTTAvgNS)),
		formatDuration(time.Duration(record.RTTP50NS)),
		formatDuration(time.Duration(record.RTTP95NS)),
		formatDuration(time.Duration(record.RTTMaxNS)),
	)
}

func printUDPPerfSummary(results []udpPerfResult, jsonOut bool) {
	if len(results) <= 1 {
		return
	}
	result := aggregateUDPPerfResults(results)
	fmt.Printf("udp perf summary trials=%d\n", len(results))
	printUDPPerf(result)
	if jsonOut {
		printJSON(udpPerfRecordForSummary(result, len(results)))
	}
}

func printUDPPerfListen(packets, bytes int64, elapsed time.Duration, expected int64) {
	lost := int64(0)
	if expected > packets {
		lost = expected - packets
	}
	fmt.Println("[ ID] Interval           Transfer     Bitrate         Datagrams  Lost  Loss")
	fmt.Printf("[  5] 0.00-%-7.2f sec  %10s  %13s  %9d  %4d  %6s\n",
		elapsed.Seconds(), formatBytes(bytes), formatBitrate(bytes, elapsed), packets, lost, formatLoss64(lost, expected))
}

func udpPerfRecordForTrial(result udpPerfResult, trial, trials int) udpPerfRecord {
	rtt := append([]time.Duration(nil), result.RTT...)
	sort.Slice(rtt, func(i, j int) bool { return rtt[i] < rtt[j] })
	success := len(rtt)
	bytes := int64(success * result.Size * 2)
	record := udpPerfRecord{
		Kind:          "udp_perf",
		Trial:         trial,
		Trials:        trials,
		Count:         result.Count,
		Size:          result.Size,
		DurationNS:    result.Duration.Nanoseconds(),
		Warmup:        result.Warmup,
		Window:        result.Window,
		Datagrams:     success,
		Lost:          result.Lost,
		LossPercent:   lossPercent(int64(result.Lost), int64(result.Count)),
		TransferBytes: bytes,
		BitrateBPS:    bitrateBPS(bytes, result.Elapsed),
		ElapsedNS:     result.Elapsed.Nanoseconds(),
	}
	if success == 0 {
		return record
	}
	total := time.Duration(0)
	for _, d := range rtt {
		total += d
	}
	record.RTTMinNS = rtt[0].Nanoseconds()
	record.RTTAvgNS = (total / time.Duration(success)).Nanoseconds()
	record.RTTP50NS = percentileDuration(rtt, 50).Nanoseconds()
	record.RTTP95NS = percentileDuration(rtt, 95).Nanoseconds()
	record.RTTMaxNS = rtt[success-1].Nanoseconds()
	return record
}

func aggregateUDPPerfResults(results []udpPerfResult) udpPerfResult {
	if len(results) == 0 {
		return udpPerfResult{}
	}
	out := udpPerfResult{
		Size:   results[0].Size,
		Window: results[0].Window,
	}
	for _, result := range results {
		out.Count += result.Count
		out.Duration += result.Duration
		out.Warmup += result.Warmup
		out.Lost += result.Lost
		out.Elapsed += result.Elapsed
		out.RTT = append(out.RTT, result.RTT...)
	}
	return out
}

func durationSuffix(duration time.Duration) string {
	if duration <= 0 {
		return ""
	}
	return " duration=" + duration.String()
}

func udpPerfRecordForSummary(result udpPerfResult, trials int) udpPerfRecord {
	record := udpPerfRecordForTrial(result, 0, trials)
	record.Kind = "udp_perf_summary"
	return record
}

func udpPerfListenRecord(packets, bytes int64, elapsed time.Duration, expected int64) udpPerfRecord {
	lost := int64(0)
	if expected > packets {
		lost = expected - packets
	}
	return udpPerfRecord{
		Kind:          "udp_perf_listen",
		Datagrams:     int(packets),
		Expected:      expected,
		Lost:          int(lost),
		LossPercent:   lossPercent(lost, expected),
		TransferBytes: bytes,
		BitrateBPS:    bitrateBPS(bytes, elapsed),
		ElapsedNS:     elapsed.Nanoseconds(),
	}
}

func printJSON(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		fmt.Printf("json error: %v\n", err)
		return
	}
	fmt.Println(string(data))
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
	bitsPerSecond := bitrateBPS(bytes, elapsed)
	for _, suffix := range []string{"bits/sec", "Kbits/sec", "Mbits/sec", "Gbits/sec"} {
		if bitsPerSecond < 1000 || suffix == "Gbits/sec" {
			return fmt.Sprintf("%.2f %s", bitsPerSecond, suffix)
		}
		bitsPerSecond /= 1000
	}
	return "0 bits/sec"
}

func bitrateBPS(bytes int64, elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	return float64(bytes*8) / elapsed.Seconds()
}

func formatLoss(lost, count int) string {
	return formatLoss64(int64(lost), int64(count))
}

func formatLoss64(lost, count int64) string {
	if count <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.2f%%", float64(lost)*100/float64(count))
}

func lossPercent(lost, count int64) float64 {
	if count <= 0 {
		return 0
	}
	return float64(lost) * 100 / float64(count)
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

func newLinkUDPMux(profile linkProfile, iface linkInterface, backend udpBackend) (*linkUDPMux, error) {
	link, err := newLinkPacketConn(profile, iface, backend, "")
	if err != nil {
		return nil, err
	}
	mux := ice.NewUDPMuxDefault(ice.UDPMuxParams{UDPConn: link.conn})
	return &linkUDPMux{
		conn:    link.conn,
		mux:     mux,
		ip:      link.ip,
		network: link.network,
		bound:   link.bound,
		backend: link.backend,
	}, nil
}

func newLinkWebRTCNet(profile linkProfile, iface linkInterface, backend udpBackend, usePionNet bool) (*linkWebRTCNet, error) {
	if !usePionNet {
		mux, err := newLinkUDPMux(profile, iface, backend)
		if err != nil {
			return nil, err
		}
		return &linkWebRTCNet{
			mux:     mux,
			ip:      mux.ip,
			network: mux.network,
			bound:   mux.bound,
			backend: mux.backend,
		}, nil
	}
	netTransport, ip, networkName, bound, err := newNetworkLinkTransportNet(profile, iface)
	if err != nil {
		return nil, err
	}
	return &linkWebRTCNet{
		net:     netTransport,
		ip:      ip,
		network: networkName,
		bound:   bound,
		backend: backend,
	}, nil
}

func newNetworkLinkTransportNet(profile linkProfile, iface linkInterface) (piontransport.Net, net.IP, string, bool, error) {
	ip, networkName, zone, err := listenIP(iface)
	if err != nil {
		return nil, nil, "", false, err
	}
	netTransport, err := nwtransport.New(nwtransport.Config{
		Packet: nwpacket.Config{
			InterfaceName:         iface.Name,
			LocalAddr:             &net.UDPAddr{IP: ip, Zone: zone},
			RequiredInterfaceType: profile.RequiredInterfaceType,
			SetRequiredInterface:  profile.Name != "thunderbolt",
			IncludePeerToPeer:     profile.IncludePeerToPeer,
			RequireInterface:      profile.Name == "awdl",
			ReuseLocalAddress:     true,
			QueueLabel:            "com.github.tmc.awdl-webrtc-apple-demo.network-transport",
			Tracef:                networkTracef,
		},
	})
	if err != nil {
		return nil, nil, "", false, err
	}
	return netTransport, ip, networkName, true, nil
}

func (l *linkWebRTCNet) Close() {
	if l == nil || l.mux == nil {
		return
	}
	l.mux.Close()
}

func (l *linkWebRTCNet) print(prefix string, iface linkInterface, mdnsMode ice.MulticastDNSMode, candidatePolicy icepolicy.Policy) {
	if l.net != nil {
		fmt.Printf("%s pion net local_ip=%s network=%s backend=%s bound_if=%s mdns=%s raw_candidates=%t\n",
			prefix, l.ip, l.network, l.backend, boundIfString(iface, l.bound), mdnsModeString(mdnsMode), candidatePolicy.RawHostCandidates)
		return
	}
	fmt.Printf("%s udp mux listen=%s network=%s backend=%s bound_if=%s mdns=%s raw_candidates=%t\n",
		prefix, l.mux.conn.LocalAddr(), l.network, l.backend, boundIfString(iface, l.bound), mdnsModeString(mdnsMode), candidatePolicy.RawHostCandidates)
}

func newLinkPacketConn(profile linkProfile, iface linkInterface, backend udpBackend, networkName string) (*linkPacketConn, error) {
	switch backend {
	case udpBackendGo:
		return newGoLinkPacketConn(iface, networkName)
	case udpBackendNetwork:
		return newNetworkLinkPacketConn(profile, iface, networkName)
	default:
		return nil, fmt.Errorf("unsupported UDP backend %q", backend)
	}
}

func newGoLinkPacketConn(iface linkInterface, networkName string) (*linkPacketConn, error) {
	if networkName == "" {
		ip, resolvedNetwork, zone, err := listenIP(iface)
		if err != nil {
			return nil, err
		}
		bindIf := shouldBindUDPToInterface(iface, resolvedNetwork, ip)
		conn, ip, networkName, bound, err := listenUDP(iface, ip, resolvedNetwork, zone, bindIf)
		if err != nil {
			return nil, err
		}
		return &linkPacketConn{conn: conn, ip: ip, network: networkName, bound: bound, backend: udpBackendGo}, nil
	}
	ip, zone, err := listenIPForNetwork(iface, networkName)
	if err != nil {
		return nil, err
	}
	bindIf := shouldBindUDPToInterface(iface, networkName, ip)
	conn, ip, _, bound, err := listenUDP(iface, ip, networkName, zone, bindIf)
	if err != nil {
		return nil, err
	}
	return &linkPacketConn{conn: conn, ip: ip, network: networkName, bound: bound, backend: udpBackendGo}, nil
}

func newNetworkLinkPacketConn(profile linkProfile, iface linkInterface, networkName string) (*linkPacketConn, error) {
	ip, resolvedNetwork, zone, err := listenIP(iface)
	if networkName != "" {
		ip, zone, err = listenIPForNetwork(iface, networkName)
		resolvedNetwork = networkName
	}
	if err != nil {
		return nil, err
	}
	conn, err := nwpacket.ListenPacket(nwpacket.Config{
		InterfaceName:         iface.Name,
		LocalAddr:             &net.UDPAddr{IP: ip, Zone: zone},
		RequiredInterfaceType: profile.RequiredInterfaceType,
		SetRequiredInterface:  profile.Name != "thunderbolt",
		IncludePeerToPeer:     profile.IncludePeerToPeer,
		RequireInterface:      profile.Name == "awdl",
		ReuseLocalAddress:     true,
		QueueLabel:            "com.github.tmc.awdl-webrtc-apple-demo.network-packetconn",
		Tracef:                networkTracef,
	})
	if err != nil {
		return nil, err
	}
	return &linkPacketConn{conn: conn, ip: ip, network: resolvedNetwork, bound: true, backend: udpBackendNetwork}, nil
}

func networkTracef(format string, args ...any) {
	if os.Getenv("AWDL_DEMO_NETWORK_TRACE") == "" {
		return
	}
	fmt.Fprintf(os.Stderr, "nwtrace: "+format+"\n", args...)
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

func shouldBindUDPToInterface(iface linkInterface, networkName string, ip net.IP) bool {
	if strings.HasPrefix(iface.Name, "awdl") {
		return true
	}
	if networkName == "udp6" {
		return true
	}
	return ip != nil && ip.IsLinkLocalUnicast()
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

func newPeer(iface linkInterface, link *linkWebRTCNet, mdnsMode ice.MulticastDNSMode, candidatePolicy icepolicy.Policy) (*webrtc.PeerConnection, error) {
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
	se.SetICEMulticastDNSMode(mdnsMode)
	if link != nil {
		candidatePolicy.Configure(&se, mdnsMode, link.ip)
	}
	if link != nil && link.mux != nil {
		se.SetICEUDPMux(link.mux.mux)
	}
	if link != nil && link.net != nil {
		se.SetNet(link.net)
	}
	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))
	return api.NewPeerConnection(webrtc.Configuration{})
}

type signalOptions struct {
	CandidatePolicy icepolicy.Policy
	LeftIP          net.IP
	RightIP         net.IP
}

type wireSignal struct {
	Description webrtc.SessionDescription `json:"description"`
	Candidates  []webrtc.ICECandidateInit `json:"candidates,omitempty"`
}

func signalNonTrickle(left, right *webrtc.PeerConnection, ctx context.Context, opts signalOptions) error {
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
	leftSignal := newWireSignal(*left.LocalDescription(), opts.CandidatePolicy, opts.LeftIP)
	if err := setRemoteWireSignal(right, leftSignal, "right offer"); err != nil {
		return err
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
	rightSignal := newWireSignal(*right.LocalDescription(), opts.CandidatePolicy, opts.RightIP)
	if err := setRemoteWireSignal(left, rightSignal, "left answer"); err != nil {
		return err
	}
	return nil
}

func newWireSignal(desc webrtc.SessionDescription, policy icepolicy.Policy, localIP net.IP) wireSignal {
	if !policy.UsesSyntheticHostCandidate(localIP) {
		return wireSignal{Description: desc}
	}
	candidates := candidateInitsFromSDP(desc.SDP, policy, localIP)
	desc.SDP = icepolicy.StripSDPCandidates(desc.SDP)
	return wireSignal{Description: desc, Candidates: candidates}
}

func setRemoteWireSignal(pc *webrtc.PeerConnection, signal wireSignal, label string) error {
	if err := pc.SetRemoteDescription(signal.Description); err != nil {
		return fmt.Errorf("set remote %s: %w", label, err)
	}
	for i, candidate := range signal.Candidates {
		if err := pc.AddICECandidate(candidate); err != nil {
			return fmt.Errorf("add remote %s candidate %d: %w", label, i, err)
		}
	}
	return nil
}

func encodeWireSignal(signal wireSignal) (string, error) {
	data, err := json.Marshal(signal)
	if err != nil {
		return "", fmt.Errorf("marshal wire signal: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func readWireSignal(scanner *bufio.Scanner, prefix string) (wireSignal, error) {
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, prefix+" ") {
			continue
		}
		return decodeWireSignal(strings.TrimSpace(strings.TrimPrefix(line, prefix+" ")))
	}
	if err := scanner.Err(); err != nil {
		return wireSignal{}, fmt.Errorf("read %s: %w", strings.ToLower(prefix), err)
	}
	return wireSignal{}, fmt.Errorf("missing %s line", prefix)
}

func decodeWireSignal(value string) (wireSignal, error) {
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return wireSignal{}, fmt.Errorf("decode wire signal: %w", err)
	}
	var signal wireSignal
	if err := json.Unmarshal(data, &signal); err == nil && signal.Description.SDP != "" {
		return signal, nil
	}
	var desc webrtc.SessionDescription
	if err := json.Unmarshal(data, &desc); err != nil {
		return wireSignal{}, fmt.Errorf("unmarshal wire signal: %w", err)
	}
	if desc.SDP == "" {
		return wireSignal{}, errors.New("wire signal missing session description")
	}
	return wireSignal{Description: desc}, nil
}

func scanRemoteAnswer(r io.Reader, answerc chan<- wireSignal, errc chan<- error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	sentAnswer := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "ANSWER ") {
			answer, err := decodeWireSignal(strings.TrimSpace(strings.TrimPrefix(line, "ANSWER ")))
			if err != nil {
				errc <- err
				return
			}
			answerc <- answer
			sentAnswer = true
			continue
		}
		fmt.Fprintf(os.Stderr, "remote: %s\n", line)
	}
	if err := scanner.Err(); err != nil {
		errc <- fmt.Errorf("scan ssh answer: %w", err)
		return
	}
	if sentAnswer {
		return
	}
	errc <- errors.New("ssh answer ended before ANSWER line")
}

func wait(ctx context.Context, ch <-chan struct{}, name string) error {
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("%s: %w", name, ctx.Err())
	}
}

func runWithTimeout(timeout time.Duration, fn func(context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := fn(ctx); err != nil {
		fail(err)
	}
}

func deadline(ctx context.Context) time.Time {
	if d, ok := ctx.Deadline(); ok {
		return d
	}
	return time.Now().Add(30 * time.Second)
}

func packetDeadline(ctx context.Context, timeout time.Duration) time.Time {
	if timeout <= 0 {
		return deadline(ctx)
	}
	d := time.Now().Add(timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(d) {
		return ctxDeadline
	}
	return d
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

func publishedCandidateLines(sdp string, policy icepolicy.Policy, localIP net.IP) []string {
	lines := candidateLines(sdp)
	if !policy.UsesSyntheticHostCandidate(localIP) {
		return lines
	}
	for i, line := range lines {
		candidate := strings.TrimPrefix(line, "a=")
		lines[i] = "a=" + policy.PublishCandidate(candidate, localIP)
	}
	return lines
}

func candidateInitsFromSDP(sdp string, policy icepolicy.Policy, localIP net.IP) []webrtc.ICECandidateInit {
	var candidates []webrtc.ICECandidateInit
	var mid string
	var mline int
	for _, line := range strings.Split(sdp, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "m="):
			mline++
			mid = ""
		case strings.HasPrefix(line, "a=mid:"):
			mid = strings.TrimPrefix(line, "a=mid:")
		case strings.HasPrefix(line, "a=candidate:"):
			candidate := strings.TrimPrefix(line, "a=")
			init := webrtc.ICECandidateInit{Candidate: policy.PublishCandidate(candidate, localIP)}
			if mid != "" {
				midCopy := mid
				init.SDPMid = &midCopy
			}
			if mline > 0 {
				index := uint16(mline - 1)
				init.SDPMLineIndex = &index
			}
			candidates = append(candidates, init)
		}
	}
	return candidates
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
