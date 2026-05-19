//go:build darwin

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/tmc/apple/dispatch"
	applenetwork "github.com/tmc/apple/network"
	"github.com/tmc/apple/objc"
	"github.com/tmc/apple/objectivec"
	"github.com/tmc/macgo"
	"github.com/tmc/swiftui"
)

const (
	linkHealthServiceType = "_awdl-webrtc._tcp"
	linkHealthDomain      = "local."
	linkHealthModes       = "answer-bonjour,discover,discover-wait,offer-bonjour,ui,udp,udp-perf,udp-latency,webrtc"
)

var linkHealthProfileOrder = []string{"thunderbolt", "awdl", "lan"}

var (
	linkHealthGitCommitOnce sync.Once
	linkHealthGitCommit     string
)

type linkHealthConfig struct {
	Backend       udpBackend
	Interval      time.Duration
	Count         int
	Size          int
	Window        int
	PacketTimeout time.Duration
}

type linkHealthApp struct {
	mu       sync.Mutex
	snapshot linkHealthSnapshot

	status  *swiftui.StringState
	active  *swiftui.StringState
	rate    *swiftui.StringState
	peer    *swiftui.StringState
	version *swiftui.IntState
}

type linkHealthSnapshot struct {
	ServiceName string
	Active      string
	Peer        linkHealthPeer
	Links       []linkHealthLink
	Samples     []linkHealthSample
	Status      string
	Updated     time.Time
}

type linkHealthLink struct {
	Profile    string
	Interface  string
	LocalAddr  string
	RemoteAddr string
	State      string
	LastRate   string
	Error      string
}

type linkHealthSample struct {
	Time       time.Time
	Profile    string
	Peer       string
	BitrateBPS float64
	Datagrams  int
	Lost       int
	Loss       float64
	RTTAvg     time.Duration
	Error      string
}

type linkHealthEndpoint struct {
	profile linkProfile
	iface   linkInterface
	link    *linkPacketConn
	cancel  context.CancelFunc
	err     string
}

type linkHealthPeer struct {
	ID          string
	Name        string
	ServiceName string
	Addrs       map[string]string
	Meta        map[string]string
	LastSeen    time.Time
}

type linkHealthAgent struct {
	cfg         linkHealthConfig
	serviceName string

	mu          sync.Mutex
	endpoints   map[string]*linkHealthEndpoint
	advertiser  *linkHealthAdvertiser
	browser     *linkHealthBrowser
	advertised  string
	lastSamples map[string]linkHealthSample
	samples     []linkHealthSample
	sampleLink  func(context.Context, *linkHealthEndpoint, linkHealthPeer, string) linkHealthSample
}

func runLinkHealthUI(ctx context.Context, cfg linkHealthConfig) error {
	runtime.LockOSThread()
	cfg = normalizeLinkHealthConfig(cfg)
	if err := macgo.Start(macgo.NewConfig().
		WithAppName("AWDL WebRTC Link Monitor").
		WithBundleID("com.github.tmc.awdl-webrtc-apple-demo").
		WithLocalNetworkUsage("AWDL WebRTC Link Monitor discovers nearby Macs and measures LAN, Thunderbolt, and AWDL link health.").
		WithBonjourServices(linkHealthServiceType).
		WithPermissions(macgo.Network).
		WithCustom("com.apple.security.network.server").
		WithAdHocSign().
		WithUIMode(macgo.UIModeRegular)); err != nil {
		return err
	}

	app := &linkHealthApp{
		status:  swiftui.NewStringState("Starting"),
		active:  swiftui.NewStringState("none"),
		rate:    swiftui.NewStringState("-"),
		peer:    swiftui.NewStringState("waiting for peer"),
		version: swiftui.NewIntState(0),
	}
	agent := newLinkHealthAgent(cfg)
	go agent.Run(ctx, app.apply)
	swiftui.RunWithMenuBar(
		swiftui.AppConfig{Title: "AWDL WebRTC Link Monitor", Width: 820, Height: 620},
		app.window(),
		swiftui.MenuBarConfig{
			Label:        "Link -",
			SystemImage:  "antenna.radiowaves.left.and.right",
			Width:        380,
			Height:       260,
			OpenOnLaunch: false,
		},
		app.menu(),
	)
	return nil
}

func normalizeLinkHealthConfig(cfg linkHealthConfig) linkHealthConfig {
	if cfg.Interval <= 0 {
		cfg.Interval = 3 * time.Second
	}
	if cfg.Count <= 0 {
		cfg.Count = 20
	}
	if cfg.Size <= 0 {
		cfg.Size = 1200
	}
	if cfg.Window <= 0 {
		cfg.Window = 4
	}
	if cfg.PacketTimeout <= 0 {
		cfg.PacketTimeout = time.Second
	}
	return cfg
}

type linkHealthDiscoveryRecord struct {
	Kind        string                    `json:"kind"`
	ServiceName string                    `json:"service_name"`
	Status      string                    `json:"status"`
	Updated     string                    `json:"updated,omitempty"`
	Peer        linkHealthDiscoveryPeer   `json:"peer,omitempty"`
	Links       []linkHealthDiscoveryLink `json:"links"`
}

type linkHealthDiscoveryPeer struct {
	ID          string            `json:"id,omitempty"`
	Name        string            `json:"name,omitempty"`
	ServiceName string            `json:"service_name,omitempty"`
	Addrs       map[string]string `json:"addrs,omitempty"`
	Meta        map[string]string `json:"meta,omitempty"`
}

type linkHealthDiscoveryLink struct {
	Profile    string `json:"profile"`
	Interface  string `json:"interface,omitempty"`
	LocalAddr  string `json:"local_addr,omitempty"`
	RemoteAddr string `json:"remote_addr,omitempty"`
	State      string `json:"state"`
	Error      string `json:"error,omitempty"`
}

func runLinkDiscovery(ctx context.Context, cfg linkHealthConfig) error {
	agent, cleanup, err := startLinkHealthAgent(cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	enc := json.NewEncoder(os.Stdout)
	tick := time.NewTicker(agent.cfg.Interval)
	defer tick.Stop()
	for {
		agent.refresh(ctx)
		agent.publish()
		peer := agent.browser.FirstPeer()
		status := "waiting for peer"
		if peer.ID != "" {
			status = "peer found"
		}
		if err := enc.Encode(linkHealthDiscoveryRecordFromSnapshot(agent.snapshot(status, peer))); err != nil {
			return fmt.Errorf("write discovery record: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
		}
	}
}

func runLinkDiscoverWait(ctx context.Context, cfg linkHealthConfig, peerName string) error {
	agent, cleanup, err := startLinkHealthAgent(cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	enc := json.NewEncoder(os.Stdout)
	tick := time.NewTicker(agent.cfg.Interval)
	defer tick.Stop()
	for {
		agent.refresh(ctx)
		agent.publish()
		peer := agent.browser.FirstMatchingPeer(peerName)
		if peer.ID != "" {
			return enc.Encode(linkHealthDiscoveryRecordFromSnapshot(agent.snapshot("peer found", peer)))
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("discover wait for %s: %w", linkHealthPeerMatchLabel(peerName), ctx.Err())
		case <-tick.C:
		}
	}
}

func startLinkHealthAgent(cfg linkHealthConfig) (*linkHealthAgent, func(), error) {
	cfg = normalizeLinkHealthConfig(cfg)
	agent := newLinkHealthAgent(cfg)
	agent.browser = newLinkHealthBrowser(agent.serviceName)
	if err := agent.browser.Start(); err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		agent.browser.Stop()
		agent.Close()
	}
	return agent, cleanup, nil
}

func linkHealthDiscoveryRecordFromSnapshot(snapshot linkHealthSnapshot) linkHealthDiscoveryRecord {
	record := linkHealthDiscoveryRecord{
		Kind:        "link_health_discovery",
		ServiceName: snapshot.ServiceName,
		Status:      snapshot.Status,
		Peer: linkHealthDiscoveryPeer{
			ID:          snapshot.Peer.ID,
			Name:        snapshot.Peer.Name,
			ServiceName: snapshot.Peer.ServiceName,
			Addrs:       snapshot.Peer.Addrs,
			Meta:        snapshot.Peer.Meta,
		},
		Links: make([]linkHealthDiscoveryLink, 0, len(snapshot.Links)),
	}
	if !snapshot.Updated.IsZero() {
		record.Updated = snapshot.Updated.Format(time.RFC3339Nano)
	}
	for _, link := range snapshot.Links {
		record.Links = append(record.Links, linkHealthDiscoveryLink{
			Profile:    link.Profile,
			Interface:  link.Interface,
			LocalAddr:  link.LocalAddr,
			RemoteAddr: link.RemoteAddr,
			State:      link.State,
			Error:      link.Error,
		})
	}
	return record
}

func linkHealthPeerMatchLabel(name string) string {
	if strings.TrimSpace(name) == "" {
		return "any peer"
	}
	return name
}

func linkHealthPeerMatches(peer linkHealthPeer, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return true
	}
	return peer.ID == name || peer.Name == name || peer.ServiceName == name
}

func (a *linkHealthApp) apply(snapshot linkHealthSnapshot) {
	a.mu.Lock()
	a.snapshot = snapshot
	a.mu.Unlock()
	a.status.Set(snapshot.Status)
	if snapshot.Active == "" {
		a.active.Set("none")
		a.rate.Set("-")
	} else {
		a.active.Set(snapshot.Active)
		if len(snapshot.Samples) != 0 {
			a.rate.Set(formatLinkRate(snapshot.Samples[len(snapshot.Samples)-1].BitrateBPS))
		}
	}
	if snapshot.Peer.ID == "" {
		a.peer.Set("waiting for peer")
	} else {
		a.peer.Set(snapshot.Peer.Name)
	}
	a.version.Set(a.version.Get() + 1)
	swiftui.UpdateMenuBarLabelStyled("Link "+a.active.Get()+" "+a.rate.Get(), swiftui.MenuBarLabelStyle{MonospacedDigits: true, Animate: true})
}

func (a *linkHealthApp) window() swiftui.View {
	return swiftui.ScrollView(swiftui.VStackSpaced(14,
		swiftui.HStack(
			swiftui.Label("AWDL WebRTC Link Monitor", "antenna.radiowaves.left.and.right").
				Font(swiftui.FontTitle).
				FontWeight(swiftui.WeightBold),
			swiftui.Spacer(),
			swiftui.DynamicView(a.version, func(int) swiftui.View {
				return swiftui.Text(time.Now().Format("15:04:05")).
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary").
					AsView()
			}),
		),
		swiftui.GroupBox("Active Path", swiftui.HStackSpaced(28,
			metricBlock("Path", a.active),
			metricBlock("Bandwidth", a.rate),
			metricBlock("Peer", a.peer),
			swiftui.Spacer(),
		).Padding(10)).MaxFrame(-1, 0),
		swiftui.GroupBox("Possible Paths", swiftui.DynamicView(a.version, func(int) swiftui.View {
			return a.linksView().Padding(8)
		})).MaxFrame(-1, 0),
		swiftui.GroupBox("Bandwidth Over Time", swiftui.DynamicView(a.version, func(int) swiftui.View {
			return a.samplesView().Padding(8)
		})).MaxFrame(-1, 0),
		swiftui.GroupBox("Status", swiftui.HStackSpaced(10,
			swiftui.Image("checkmark.circle.fill").ForegroundStyle(0.2, 0.65, 0.35, 1),
			swiftui.TextFromString(a.status).Font(swiftui.FontCallout).AsView(),
			swiftui.Spacer(),
		).Padding(8)).MaxFrame(-1, 0),
	).Padding(20))
}

func (a *linkHealthApp) menu() swiftui.View {
	return swiftui.VStackSpaced(12,
		swiftui.HStack(
			swiftui.Label("Link Monitor", "antenna.radiowaves.left.and.right").
				Font(swiftui.FontHeadline),
			swiftui.Spacer(),
			swiftui.TextFromString(a.active).Font(swiftui.FontCaption).ForegroundStyleNamed("secondary"),
		),
		swiftui.Divider(),
		swiftui.HStackSpaced(10,
			metricBlock("Path", a.active),
			metricBlock("Rate", a.rate),
		),
		swiftui.TextFromString(a.status).Font(swiftui.FontCaption).ForegroundStyleNamed("secondary"),
	).Padding(12)
}

func (a *linkHealthApp) linksView() swiftui.View {
	snapshot := a.current()
	rows := make([]swiftui.Viewable, 0, len(snapshot.Links)+1)
	rows = append(rows, swiftui.HStackSpaced(12,
		headerText("Profile").Frame(110, 0),
		headerText("Interface").Frame(100, 0),
		headerText("Local").Frame(210, 0),
		headerText("Remote").Frame(210, 0),
		headerText("State").Frame(120, 0),
	))
	for _, link := range snapshot.Links {
		state := link.State
		if link.Error != "" {
			state = link.Error
		}
		rows = append(rows, swiftui.HStackSpaced(12,
			swiftui.Text(link.Profile).Font(swiftui.FontCallout).FontWeight(swiftui.WeightSemibold).AsView().Frame(110, 0),
			monoSmall(link.Interface).Frame(100, 0),
			monoSmall(link.LocalAddr).Frame(210, 0),
			monoSmall(link.RemoteAddr).Frame(210, 0),
			swiftui.Text(state).Font(swiftui.FontCaption).ForegroundStyleNamed("secondary").AsView().Frame(120, 0),
		))
	}
	if len(snapshot.Links) == 0 {
		rows = append(rows, swiftui.Text("No local link listeners are available yet.").Font(swiftui.FontCaption).ForegroundStyleNamed("secondary"))
	}
	return swiftui.VStackSpaced(8, rows...)
}

func (a *linkHealthApp) samplesView() swiftui.View {
	snapshot := a.current()
	if len(snapshot.Samples) == 0 {
		return swiftui.Text("Waiting for a peer sample. Run this command on both Macs.").
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary").
			AsView()
	}
	rows := make([]swiftui.Viewable, 0, len(snapshot.Samples)+1)
	rows = append(rows, swiftui.HStackSpaced(12,
		headerText("Time").Frame(80, 0),
		headerText("Path").Frame(100, 0),
		headerText("Bandwidth").Frame(110, 0),
		headerText("Loss").Frame(90, 0),
		headerText("RTT avg").Frame(100, 0),
		headerText("Result").Frame(270, 0),
	))
	for i := len(snapshot.Samples) - 1; i >= 0 && len(rows) < 13; i-- {
		sample := snapshot.Samples[i]
		result := "ok"
		if sample.Error != "" {
			result = sample.Error
		}
		rows = append(rows, swiftui.HStackSpaced(12,
			monoSmall(sample.Time.Format("15:04:05")).Frame(80, 0),
			swiftui.Text(sample.Profile).Font(swiftui.FontCallout).AsView().Frame(100, 0),
			monoSmall(formatLinkRate(sample.BitrateBPS)).Frame(110, 0),
			monoSmall(fmt.Sprintf("%.1f%%", sample.Loss)).Frame(90, 0),
			monoSmall(formatDuration(sample.RTTAvg)).Frame(100, 0),
			swiftui.Text(result).Font(swiftui.FontCaption).ForegroundStyleNamed("secondary").AsView().Frame(270, 0),
		))
	}
	return swiftui.VStackSpaced(8, rows...)
}

func (a *linkHealthApp) current() linkHealthSnapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.snapshot
}

func metricBlock(label string, value *swiftui.StringState) swiftui.View {
	return swiftui.VStackSpaced(4,
		swiftui.Text(label).Font(swiftui.FontCaption).ForegroundStyleNamed("secondary"),
		swiftui.TextFromString(value).Font(swiftui.FontTitle3).FontWeight(swiftui.WeightSemibold).MonospacedDigit(),
	)
}

func headerText(text string) swiftui.View {
	return swiftui.Text(text).Font(swiftui.FontCaption).FontWeight(swiftui.WeightSemibold).ForegroundStyleNamed("secondary").AsView()
}

func monoSmall(text string) swiftui.View {
	if text == "" {
		text = "-"
	}
	return swiftui.Text(text).Font(swiftui.FontCaption).MonospacedDigit().AsView()
}

func newLinkHealthAgent(cfg linkHealthConfig) *linkHealthAgent {
	return &linkHealthAgent{
		cfg:         cfg,
		serviceName: linkHealthServiceName(),
		endpoints:   make(map[string]*linkHealthEndpoint),
		lastSamples: make(map[string]linkHealthSample),
	}
}

func (a *linkHealthAgent) Run(ctx context.Context, update func(linkHealthSnapshot)) {
	a.browser = newLinkHealthBrowser(a.serviceName)
	if err := a.browser.Start(); err != nil {
		update(linkHealthSnapshot{ServiceName: a.serviceName, Status: err.Error(), Updated: time.Now()})
		return
	}
	defer a.browser.Stop()
	defer a.Close()
	a.refresh(ctx)
	a.publish()
	update(a.snapshot("waiting for peer", linkHealthPeer{}))

	tick := time.NewTicker(a.cfg.Interval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		a.refresh(ctx)
		a.publish()
		peer := a.browser.FirstPeer()
		if peer.ID == "" {
			update(a.snapshot("waiting for peer", peer))
			continue
		}
		sample := a.samplePreferred(ctx, peer)
		a.remember(sample)
		if sample.Error != "" {
			update(a.snapshot("peer found; no path completed", peer))
			continue
		}
		update(a.snapshot("sampling "+sample.Profile, peer))
	}
}

func (a *linkHealthAgent) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, endpoint := range a.endpoints {
		endpoint.Close()
	}
	a.endpoints = nil
	if a.advertiser != nil {
		a.advertiser.Stop()
		a.advertiser = nil
	}
}

func (e *linkHealthEndpoint) Close() {
	if e.cancel != nil {
		e.cancel()
	}
	if e.link != nil && e.link.conn != nil {
		_ = e.link.conn.Close()
	}
}

func (a *linkHealthAgent) refresh(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, name := range linkHealthProfileOrder {
		if endpoint := a.endpoints[name]; endpoint != nil && endpoint.err == "" {
			continue
		}
		endpoint := a.openEndpoint(ctx, name)
		old := a.endpoints[name]
		if old != nil {
			old.Close()
		}
		a.endpoints[name] = endpoint
	}
}

func (a *linkHealthAgent) openEndpoint(ctx context.Context, name string) *linkHealthEndpoint {
	profile, err := profileByName(name)
	if err != nil {
		return &linkHealthEndpoint{err: err.Error()}
	}
	if profile.DefaultInterface == "" {
		profile.DefaultInterface, err = defaultInterface(profile)
		if err != nil {
			return &linkHealthEndpoint{profile: profile, err: err.Error()}
		}
	}
	iface, err := inspectInterface(profile.DefaultInterface)
	if err != nil {
		return &linkHealthEndpoint{profile: profile, err: err.Error()}
	}
	if len(iface.IPs) == 0 {
		return &linkHealthEndpoint{profile: profile, iface: iface, err: "no address"}
	}
	link, err := newLinkPacketConn(profile, iface, a.cfg.Backend, "")
	if err != nil {
		return &linkHealthEndpoint{profile: profile, iface: iface, err: err.Error()}
	}
	echoCtx, cancel := context.WithCancel(ctx)
	go echoUDPPacketsForever(echoCtx, link.conn)
	return &linkHealthEndpoint{profile: profile, iface: iface, link: link, cancel: cancel}
}

func (a *linkHealthAgent) publish() {
	meta := a.metadata()
	sig := linkHealthMetadataSignature(meta)
	a.mu.Lock()
	defer a.mu.Unlock()
	if sig == a.advertised {
		return
	}
	if a.advertiser != nil {
		a.advertiser.Stop()
	}
	a.advertiser = newLinkHealthAdvertiser(a.serviceName, meta)
	if err := a.advertiser.Start(); err != nil {
		a.advertised = "error:" + err.Error()
		return
	}
	a.advertised = sig
}

func (a *linkHealthAgent) metadata() map[string]string {
	host, _ := os.Hostname()
	meta := map[string]string{
		"id":      a.serviceName,
		"name":    host,
		"modes":   linkHealthModes,
		"version": linkHealthVersion(),
	}
	if commit := linkHealthCommit(); commit != "" {
		meta["commit"] = commit
	}
	if modified := linkHealthVCSModified(); modified != "" {
		meta["vcs_modified"] = modified
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, name := range linkHealthProfileOrder {
		endpoint := a.endpoints[name]
		if endpoint == nil || endpoint.err != "" || endpoint.link == nil {
			continue
		}
		meta[name] = endpoint.link.conn.LocalAddr().String()
		meta[name+"_if"] = endpoint.iface.Name
	}
	return meta
}

func linkHealthVersion() string {
	if strings.TrimSpace(buildVersion) != "" && buildVersion != "dev" {
		return buildVersion
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	if strings.TrimSpace(buildVersion) == "" {
		return "dev"
	}
	return buildVersion
}

func linkHealthCommit() string {
	if strings.TrimSpace(buildCommit) != "" {
		return buildCommit
	}
	if value := linkHealthBuildSetting("vcs.revision"); value != "" {
		return value
	}
	return linkHealthGitRevision()
}

func linkHealthVCSModified() string {
	return linkHealthBuildSetting("vcs.modified")
}

func linkHealthBuildSetting(key string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, setting := range info.Settings {
		if setting.Key == key {
			return setting.Value
		}
	}
	return ""
}

func linkHealthGitRevision() string {
	linkHealthGitCommitOnce.Do(func() {
		out, err := exec.Command("git", "rev-parse", "HEAD").Output()
		if err == nil {
			linkHealthGitCommit = strings.TrimSpace(string(out))
		}
	})
	return linkHealthGitCommit
}

func (a *linkHealthAgent) samplePreferred(ctx context.Context, peer linkHealthPeer) linkHealthSample {
	for _, name := range linkHealthProfileOrder {
		remote := peer.Addrs[name]
		if remote == "" {
			continue
		}
		endpoint := a.endpoint(name)
		if endpoint == nil || endpoint.err != "" {
			a.remember(linkHealthSample{Time: time.Now(), Profile: name, Peer: peer.Name, Error: "local unavailable"})
			continue
		}
		sample := a.sampleEndpoint(ctx, endpoint, peer, remote)
		if sample.Error == "" {
			return sample
		}
		a.remember(sample)
	}
	return linkHealthSample{Time: time.Now(), Peer: peer.Name, Error: "no shared path"}
}

func (a *linkHealthAgent) sampleEndpoint(ctx context.Context, endpoint *linkHealthEndpoint, peer linkHealthPeer, remote string) linkHealthSample {
	if a.sampleLink != nil {
		return a.sampleLink(ctx, endpoint, peer, remote)
	}
	return a.sample(ctx, endpoint, peer, remote)
}

func (a *linkHealthAgent) sample(ctx context.Context, endpoint *linkHealthEndpoint, peer linkHealthPeer, remote string) linkHealthSample {
	sample := linkHealthSample{Time: time.Now(), Profile: endpoint.profile.Name, Peer: peer.Name}
	networkName, err := udpNetworkForPeer(remote)
	if err != nil {
		sample.Error = err.Error()
		return sample
	}
	link, err := newLinkPacketConn(endpoint.profile, endpoint.iface, a.cfg.Backend, networkName)
	if err != nil {
		sample.Error = err.Error()
		return sample
	}
	defer link.conn.Close()
	addr, err := net.ResolveUDPAddr(networkName, remote)
	if err != nil {
		sample.Error = err.Error()
		return sample
	}
	timeout := maxDuration(2*a.cfg.PacketTimeout*time.Duration(a.cfg.Count/a.cfg.Window+1), a.cfg.Interval)
	sampleCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	result, err := runUDPEchoPerf(sampleCtx, link.conn, addr, a.cfg.Count, a.cfg.Size, 0, a.cfg.Window, a.cfg.PacketTimeout, 0)
	if err != nil {
		sample.Error = err.Error()
		return sample
	}
	record := udpPerfRecordForTrial(result, 0, 0)
	if err := linkHealthPerfError(record); err != "" {
		sample.Datagrams = record.Datagrams
		sample.Lost = record.Lost
		sample.Loss = record.LossPercent
		sample.Error = err
		return sample
	}
	sample.BitrateBPS = record.BitrateBPS
	sample.Datagrams = record.Datagrams
	sample.Lost = record.Lost
	sample.Loss = record.LossPercent
	sample.RTTAvg = time.Duration(record.RTTAvgNS)
	return sample
}

func linkHealthPerfError(record udpPerfRecord) string {
	if record.Count > 0 && record.Datagrams == 0 {
		return "no replies"
	}
	return ""
}

func (a *linkHealthAgent) endpoint(name string) *linkHealthEndpoint {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.endpoints[name]
}

func (a *linkHealthAgent) remember(sample linkHealthSample) {
	if sample.Time.IsZero() {
		sample.Time = time.Now()
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if sample.Profile != "" {
		a.lastSamples[sample.Profile] = sample
	}
	a.samples = append(a.samples, sample)
	if len(a.samples) > 60 {
		a.samples = append([]linkHealthSample(nil), a.samples[len(a.samples)-60:]...)
	}
}

func (a *linkHealthAgent) snapshot(status string, peer linkHealthPeer) linkHealthSnapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	links := make([]linkHealthLink, 0, len(linkHealthProfileOrder))
	active := ""
	for _, name := range linkHealthProfileOrder {
		endpoint := a.endpoints[name]
		link := linkHealthLink{Profile: name, State: "unavailable"}
		if endpoint != nil {
			link.Interface = endpoint.iface.Name
			link.Error = endpoint.err
			if endpoint.err == "" && endpoint.link != nil {
				link.State = "ready"
				link.LocalAddr = endpoint.link.conn.LocalAddr().String()
			}
		}
		if peer.Addrs != nil {
			link.RemoteAddr = peer.Addrs[name]
		}
		if sample, ok := a.lastSamples[name]; ok {
			link.LastRate = formatLinkRate(sample.BitrateBPS)
			if sample.Error == "" && active == "" {
				active = name
			}
		}
		links = append(links, link)
	}
	return linkHealthSnapshot{
		ServiceName: a.serviceName,
		Active:      active,
		Peer:        peer,
		Links:       links,
		Samples:     append([]linkHealthSample(nil), a.samples...),
		Status:      status,
		Updated:     time.Now(),
	}
}

func echoUDPPacketsForever(ctx context.Context, conn net.PacketConn) {
	buf := make([]byte, 64*1024)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if isTimeout(err) {
				continue
			}
			continue
		}
		_ = conn.SetWriteDeadline(deadline(ctx))
		_, _ = conn.WriteTo(buf[:n], addr)
	}
}

type linkHealthAdvertiser struct {
	name     string
	meta     map[string]string
	listener applenetwork.NWListener
	queue    dispatch.Queue
}

func newLinkHealthAdvertiser(name string, meta map[string]string) *linkHealthAdvertiser {
	return &linkHealthAdvertiser{name: name, meta: meta}
}

func (a *linkHealthAdvertiser) Start() error {
	listener := applenetwork.NWListenerCreateWithPort("0", linkHealthTCPParams())
	if listener.ID == 0 {
		return fmt.Errorf("link health advertise: nil listener")
	}
	queue := dispatch.QueueCreate("awdl-webrtc.link-health.advertise")
	desc := applenetwork.NWAdvertiseDescriptorCreateBonjourService(a.name, linkHealthServiceType, linkHealthDomain)
	applenetwork.NWAdvertiseDescriptorSetTXTRecordObject(desc, linkHealthTXTRecord(a.meta))
	applenetwork.NWListenerSetAdvertiseDescriptor(listener, desc)
	applenetwork.NWListenerSetQueue(listener, queue)
	applenetwork.NWListenerSetNewConnectionHandler(listener, func(objectivec.Object) {})
	ready := make(chan error, 1)
	applenetwork.NWListenerSetStateChangedHandler(listener, func(state applenetwork.NWListenerState, nwErr applenetwork.NWError) {
		switch state {
		case applenetwork.NWListenerStateReady:
			signalError(ready, nil)
		case applenetwork.NWListenerStateFailed:
			signalError(ready, fmt.Errorf("link health advertise failed: %s", nwErr.Description()))
		}
	})
	applenetwork.NWListenerStart(listener)
	select {
	case err := <-ready:
		if err != nil {
			applenetwork.NWListenerCancel(listener)
			return err
		}
	case <-time.After(2 * time.Second):
		// Some releases deliver browse results even when listener readiness is slow.
	}
	a.listener = listener
	a.queue = queue
	return nil
}

func (a *linkHealthAdvertiser) Stop() {
	if a.listener.ID != 0 {
		applenetwork.NWListenerCancel(a.listener)
	}
}

type linkHealthBrowser struct {
	self    string
	browser applenetwork.NWBrowser
	queue   dispatch.Queue
	mu      sync.Mutex
	peers   map[string]linkHealthPeer
}

func newLinkHealthBrowser(self string) *linkHealthBrowser {
	return &linkHealthBrowser{self: self, peers: make(map[string]linkHealthPeer)}
}

func (b *linkHealthBrowser) Start() error {
	desc := applenetwork.NWBrowseDescriptorCreateBonjourService(linkHealthServiceType, linkHealthDomain)
	applenetwork.NWBrowseDescriptorSetIncludeTXTRecord(desc, true)
	browser := applenetwork.NWBrowserCreate(desc, linkHealthTCPParams())
	if browser.ID == 0 {
		return fmt.Errorf("link health browse: nil browser")
	}
	queue := dispatch.QueueCreate("awdl-webrtc.link-health.browse")
	applenetwork.NWBrowserSetQueue(browser, queue)
	applenetwork.NWBrowserSetBrowseResultsChangedHandler(browser, b.handleResults)
	applenetwork.NWBrowserStart(browser)
	b.browser = browser
	b.queue = queue
	return nil
}

func (b *linkHealthBrowser) Stop() {
	if b.browser.ID != 0 {
		applenetwork.NWBrowserCancel(b.browser)
	}
}

func (b *linkHealthBrowser) FirstPeer() linkHealthPeer {
	return b.FirstMatchingPeer("")
}

func (b *linkHealthBrowser) FirstMatchingPeer(name string) linkHealthPeer {
	b.mu.Lock()
	defer b.mu.Unlock()
	peers := make([]linkHealthPeer, 0, len(b.peers))
	for _, peer := range b.peers {
		peers = append(peers, peer)
	}
	sort.Slice(peers, func(i, j int) bool {
		return peers[i].LastSeen.After(peers[j].LastSeen)
	})
	if len(peers) == 0 {
		return linkHealthPeer{}
	}
	for _, peer := range peers {
		if linkHealthPeerMatches(peer, name) {
			return peer
		}
	}
	return linkHealthPeer{}
}

func (b *linkHealthBrowser) handleResults(oldResult, newResult objectivec.Object, _ bool) {
	if oldResult.ID != 0 {
		name := linkHealthResultName(oldResult)
		if name != "" {
			b.mu.Lock()
			delete(b.peers, name)
			b.mu.Unlock()
		}
	}
	if newResult.ID == 0 {
		return
	}
	peer := linkHealthPeerFromResult(newResult)
	if peer.ID == "" || peer.ID == b.self {
		return
	}
	b.mu.Lock()
	b.peers[peer.ServiceName] = peer
	b.mu.Unlock()
}

func linkHealthPeerFromResult(result objectivec.Object) linkHealthPeer {
	endpoint := applenetwork.NWBrowseResultCopyEndpoint(result)
	service := objc.GoString(applenetwork.NWEndpointGetBonjourServiceName(endpoint))
	meta := linkHealthTXTRecordMap(applenetwork.NWBrowseResultCopyTXTRecordObject(result))
	name := meta["name"]
	if name == "" {
		name = service
	}
	id := meta["id"]
	if id == "" {
		id = service
	}
	addrs := make(map[string]string)
	for _, profile := range linkHealthProfileOrder {
		if meta[profile] != "" {
			addrs[profile] = meta[profile]
		}
	}
	return linkHealthPeer{
		ID:          id,
		Name:        name,
		ServiceName: service,
		Addrs:       addrs,
		Meta:        meta,
		LastSeen:    time.Now(),
	}
}

func linkHealthResultName(result objectivec.Object) string {
	endpoint := applenetwork.NWBrowseResultCopyEndpoint(result)
	return objc.GoString(applenetwork.NWEndpointGetBonjourServiceName(endpoint))
}

func linkHealthTCPParams() applenetwork.NWParameters {
	params := applenetwork.NWParametersCreatePlainTCP(nil)
	stack := applenetwork.NWParametersCopyDefaultProtocolStack(params)
	tcpOpts := applenetwork.NWProtocolStackCopyTransportProtocol(stack)
	applenetwork.NWTCPOptionsSetNoDelay(tcpOpts, true)
	applenetwork.NWParametersSetIncludePeerToPeer(params, true)
	linkHealthSetPrivateBool(params, "setUseAWDL:", true)
	linkHealthSetPrivateBool(params, "setUseP2P:", true)
	return params
}

func linkHealthSetPrivateBool(obj objectivec.Object, sel string, value bool) {
	if obj.ID == 0 {
		return
	}
	selector := objc.Sel(sel)
	if !obj.RespondsToSelector(selector) {
		return
	}
	objc.Send[struct{}](obj.ID, selector, value)
}

func linkHealthTXTRecord(meta map[string]string) applenetwork.NWTXTRecord {
	txt := applenetwork.NWTXTRecordCreateDictionary()
	keys := make([]string, 0, len(meta))
	for key := range meta {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := meta[key]
		b := []byte(value)
		applenetwork.NWTXTRecordSetKey(txt, key, b, uintptr(len(b)))
	}
	return txt
}

func linkHealthTXTRecordMap(txt applenetwork.NWTXTRecord) map[string]string {
	meta := make(map[string]string)
	if txt.ID == 0 {
		return meta
	}
	applenetwork.NWTXTRecordAccessBytesFunc(txt, func(b *uint8, n uint32) bool {
		if b == nil || n == 0 {
			return true
		}
		raw := unsafe.Slice(b, int(n))
		for len(raw) > 0 {
			itemLen := int(raw[0])
			raw = raw[1:]
			if itemLen > len(raw) {
				break
			}
			item := raw[:itemLen]
			raw = raw[itemLen:]
			key, value, ok := strings.Cut(string(item), "=")
			if ok && key != "" {
				meta[key] = value
			}
		}
		return true
	})
	return meta
}

func linkHealthMetadataSignature(meta map[string]string) string {
	keys := make([]string, 0, len(meta))
	for key := range meta {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, key := range keys {
		fmt.Fprintf(&b, "%s=%s\n", key, meta[key])
	}
	return b.String()
}

func linkHealthServiceName() string {
	host, _ := os.Hostname()
	name := strings.TrimSpace(host)
	if name == "" {
		name = "awdl-webrtc"
	}
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		default:
			return '-'
		}
	}, name)
	name = strings.Trim(name, "-")
	if name == "" {
		name = "awdl-webrtc"
	}
	suffix := fmt.Sprintf("-%d", os.Getpid())
	if len(name)+len(suffix) > 48 {
		name = name[:48-len(suffix)]
	}
	return name + suffix
}

func formatLinkRate(bps float64) string {
	switch {
	case bps <= 0:
		return "-"
	case bps >= 1e9:
		return fmt.Sprintf("%.2f Gbit/s", bps/1e9)
	case bps >= 1e6:
		return fmt.Sprintf("%.2f Mbit/s", bps/1e6)
	case bps >= 1e3:
		return fmt.Sprintf("%.2f Kbit/s", bps/1e3)
	default:
		return fmt.Sprintf("%.0f bit/s", bps)
	}
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func signalError(ch chan<- error, err error) {
	select {
	case ch <- err:
	default:
	}
}
