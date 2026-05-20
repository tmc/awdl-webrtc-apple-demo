//go:build darwin

package main

import (
	"context"
	"testing"
	"time"
)

func TestLinkHealthMetadataSignatureStable(t *testing.T) {
	a := linkHealthMetadataSignature(map[string]string{
		"awdl": "fe80::1%awdl0",
		"id":   "peer",
	})
	b := linkHealthMetadataSignature(map[string]string{
		"id":   "peer",
		"awdl": "fe80::1%awdl0",
	})
	if a != b {
		t.Fatalf("metadata signatures differ:\n%s\n%s", a, b)
	}
}

func TestFormatLinkRate(t *testing.T) {
	tests := []struct {
		bps  float64
		want string
	}{
		{0, "-"},
		{900, "900 bit/s"},
		{1200, "1.20 Kbit/s"},
		{2.5e6, "2.50 Mbit/s"},
		{1.5e9, "1.50 Gbit/s"},
	}
	for _, tt := range tests {
		if got := formatLinkRate(tt.bps); got != tt.want {
			t.Fatalf("formatLinkRate(%v) = %q, want %q", tt.bps, got, tt.want)
		}
	}
}

func TestShortEndpoint(t *testing.T) {
	tests := []struct {
		addr string
		want string
	}{
		{"", "-"},
		{"10.0.199.147:57718", "10.0.199.147:57718"},
		{"169.254.61.91:51052", "169.254.61.91:51052"},
		{"[fe80::c814:f7ff:fe87:2c83%awdl0]:52714", "fe80::c814...2c83%awdl0:52714"},
		{"[fe80::4c41:acff:fec5:96f1%awdl0]", "fe80::4c41...96f1%awdl0"},
	}
	for _, tt := range tests {
		if got := shortEndpoint(tt.addr); got != tt.want {
			t.Fatalf("shortEndpoint(%q) = %q, want %q", tt.addr, got, tt.want)
		}
	}
}

func TestProfileLabel(t *testing.T) {
	tests := []struct {
		profile string
		want    string
	}{
		{"thunderbolt", "Thunderbolt"},
		{"awdl", "AWDL"},
		{"lan", "LAN"},
		{"", "-"},
		{"other", "other"},
	}
	for _, tt := range tests {
		if got := profileLabel(tt.profile); got != tt.want {
			t.Fatalf("profileLabel(%q) = %q, want %q", tt.profile, got, tt.want)
		}
	}
}

func TestNormalizeLinkHealthConfig(t *testing.T) {
	cfg := normalizeLinkHealthConfig(linkHealthConfig{})
	if cfg.Interval != 3*time.Second {
		t.Fatalf("interval = %s, want 3s", cfg.Interval)
	}
	if cfg.Count != 20 {
		t.Fatalf("count = %d, want 20", cfg.Count)
	}
	if cfg.Size != 1200 {
		t.Fatalf("size = %d, want 1200", cfg.Size)
	}
	if cfg.Window != 4 {
		t.Fatalf("window = %d, want 4", cfg.Window)
	}
	if cfg.PacketTimeout != time.Second {
		t.Fatalf("packet timeout = %s, want 1s", cfg.PacketTimeout)
	}
}

func TestLinkHealthWaitingStatus(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"", "waiting for peer"},
		{" remote-service ", "waiting for remote-service"},
	}
	for _, tt := range tests {
		agent := newLinkHealthAgent(linkHealthConfig{PeerName: tt.name})
		if got := agent.waitingStatus(); got != tt.want {
			t.Fatalf("waitingStatus(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestLinkHealthDiscoveryRecordFromSnapshot(t *testing.T) {
	updated := time.Date(2026, 5, 19, 5, 0, 0, 123, time.UTC)
	record := linkHealthDiscoveryRecordFromSnapshot(linkHealthSnapshot{
		ServiceName: "local-1",
		Status:      "peer found",
		Updated:     updated,
		Peer: linkHealthPeer{
			ID:          "peer-id",
			Name:        "peer",
			ServiceName: "peer-service",
			Addrs:       map[string]string{"awdl": "[fe80::1%awdl0]:1234"},
			Meta:        map[string]string{"version": "vtest", "commit": "abcdef"},
		},
		Links: []linkHealthLink{
			{
				Profile:    "awdl",
				Interface:  "awdl0",
				LocalAddr:  "[fe80::2%awdl0]:5678",
				RemoteAddr: "[fe80::1%awdl0]:1234",
				State:      "ready",
			},
			{
				Profile: "thunderbolt",
				State:   "unavailable",
				Error:   "no address",
			},
		},
	})
	if record.Kind != "link_health_discovery" {
		t.Fatalf("kind = %q, want link_health_discovery", record.Kind)
	}
	if record.ServiceName != "local-1" || record.Status != "peer found" {
		t.Fatalf("record header = %#v", record)
	}
	if record.Updated != updated.Format(time.RFC3339Nano) {
		t.Fatalf("updated = %q", record.Updated)
	}
	if record.Peer.ID != "peer-id" || record.Peer.Addrs["awdl"] == "" {
		t.Fatalf("peer = %#v", record.Peer)
	}
	if record.Peer.Meta["version"] != "vtest" || record.Peer.Meta["commit"] != "abcdef" {
		t.Fatalf("peer metadata = %#v", record.Peer.Meta)
	}
	if len(record.Links) != 2 {
		t.Fatalf("links = %d, want 2", len(record.Links))
	}
	if record.Links[0].Profile != "awdl" || record.Links[0].State != "ready" {
		t.Fatalf("link 0 = %#v", record.Links[0])
	}
	if record.Links[1].Error != "no address" {
		t.Fatalf("link 1 = %#v", record.Links[1])
	}
}

func TestLinkHealthMetadataIncludesBuildFields(t *testing.T) {
	oldVersion, oldCommit := buildVersion, buildCommit
	buildVersion, buildCommit = "vtest", "abcdef"
	t.Cleanup(func() {
		buildVersion, buildCommit = oldVersion, oldCommit
	})
	agent := newLinkHealthAgent(linkHealthConfig{})
	meta := agent.metadata()
	if meta["id"] != agent.serviceName {
		t.Fatalf("id = %q, want %q", meta["id"], agent.serviceName)
	}
	if meta["version"] != "vtest" {
		t.Fatalf("version = %q, want vtest", meta["version"])
	}
	if meta["commit"] != "abcdef" {
		t.Fatalf("commit = %q, want abcdef", meta["commit"])
	}
	if meta["modes"] != linkHealthModes {
		t.Fatalf("modes = %q, want %q", meta["modes"], linkHealthModes)
	}
}

func TestLinkHealthPeerMatches(t *testing.T) {
	peer := linkHealthPeer{
		ID:          "peer-id",
		Name:        "peer-name",
		ServiceName: "peer-service",
	}
	tests := []struct {
		name string
		want bool
	}{
		{"", true},
		{" peer-id ", true},
		{"peer-name", true},
		{"peer-service", true},
		{"missing", false},
	}
	for _, tt := range tests {
		if got := linkHealthPeerMatches(peer, tt.name); got != tt.want {
			t.Fatalf("linkHealthPeerMatches(%q) = %t, want %t", tt.name, got, tt.want)
		}
	}
}

func TestLinkHealthBrowserFirstMatchingPeer(t *testing.T) {
	browser := newLinkHealthBrowser("self")
	now := time.Now()
	browser.peers["old"] = linkHealthPeer{ID: "old-id", Name: "old", ServiceName: "old-service", LastSeen: now.Add(-time.Minute)}
	browser.peers["new"] = linkHealthPeer{ID: "new-id", Name: "new", ServiceName: "new-service", LastSeen: now}
	if got := browser.FirstMatchingPeer(""); got.ID != "new-id" {
		t.Fatalf("FirstMatchingPeer(empty) = %#v, want new-id", got)
	}
	if got := browser.FirstMatchingPeer("old-service"); got.ID != "old-id" {
		t.Fatalf("FirstMatchingPeer(old-service) = %#v, want old-id", got)
	}
	if got := browser.FirstMatchingPeer("missing"); got.ID != "" {
		t.Fatalf("FirstMatchingPeer(missing) = %#v, want empty", got)
	}
}

func TestLinkHealthPerfError(t *testing.T) {
	if got := linkHealthPerfError(udpPerfRecord{Count: 20, Datagrams: 0}); got != "no replies" {
		t.Fatalf("linkHealthPerfError(all lost) = %q, want no replies", got)
	}
	if got := linkHealthPerfError(udpPerfRecord{Count: 20, Datagrams: 1}); got != "" {
		t.Fatalf("linkHealthPerfError(partial success) = %q, want empty", got)
	}
}

func TestLinkHealthSamplePreferredFallsBackToAWDL(t *testing.T) {
	agent := newLinkHealthAgent(linkHealthConfig{})
	agent.endpoints = map[string]*linkHealthEndpoint{
		"thunderbolt": {profile: linkProfile{Name: "thunderbolt"}},
		"awdl":        {profile: linkProfile{Name: "awdl"}},
		"lan":         {profile: linkProfile{Name: "lan"}},
	}
	var sampled []string
	agent.sampleLink = func(_ context.Context, endpoint *linkHealthEndpoint, peer linkHealthPeer, remote string) linkHealthSample {
		sampled = append(sampled, endpoint.profile.Name+"="+remote)
		switch endpoint.profile.Name {
		case "thunderbolt":
			return linkHealthSample{Profile: "thunderbolt", Peer: peer.Name, Error: "no replies"}
		case "awdl":
			return linkHealthSample{Profile: "awdl", Peer: peer.Name, BitrateBPS: 25e6, Datagrams: 20}
		default:
			t.Fatalf("unexpected sample on %s", endpoint.profile.Name)
			return linkHealthSample{}
		}
	}

	sample := agent.samplePreferred(context.Background(), linkHealthPeer{
		Name: "peer",
		Addrs: map[string]string{
			"thunderbolt": "169.254.88.35:1234",
			"awdl":        "[fe80::1%awdl0]:1234",
			"lan":         "10.0.18.249:1234",
		},
	})
	if sample.Profile != "awdl" || sample.Error != "" {
		t.Fatalf("sample = %#v, want awdl success", sample)
	}
	if len(sampled) != 2 || sampled[0] != "thunderbolt=169.254.88.35:1234" || sampled[1] != "awdl=[fe80::1%awdl0]:1234" {
		t.Fatalf("sampled = %#v", sampled)
	}
	if got := agent.lastSamples["thunderbolt"].Error; got != "no replies" {
		t.Fatalf("thunderbolt remembered error = %q, want no replies", got)
	}
}

func TestLinkHealthSamplePreferredSkipsUnavailableThunderbolt(t *testing.T) {
	agent := newLinkHealthAgent(linkHealthConfig{})
	agent.endpoints = map[string]*linkHealthEndpoint{
		"thunderbolt": {profile: linkProfile{Name: "thunderbolt"}, err: "no address"},
		"awdl":        {profile: linkProfile{Name: "awdl"}},
	}
	agent.sampleLink = func(_ context.Context, endpoint *linkHealthEndpoint, peer linkHealthPeer, remote string) linkHealthSample {
		if endpoint.profile.Name != "awdl" {
			t.Fatalf("unexpected sample on %s", endpoint.profile.Name)
		}
		return linkHealthSample{Profile: "awdl", Peer: peer.Name, BitrateBPS: 20e6, Datagrams: 20}
	}

	sample := agent.samplePreferred(context.Background(), linkHealthPeer{
		Name: "peer",
		Addrs: map[string]string{
			"thunderbolt": "169.254.88.35:1234",
			"awdl":        "[fe80::1%awdl0]:1234",
		},
	})
	if sample.Profile != "awdl" || sample.Error != "" {
		t.Fatalf("sample = %#v, want awdl success", sample)
	}
	if got := agent.lastSamples["thunderbolt"].Error; got != "local unavailable" {
		t.Fatalf("thunderbolt remembered error = %q, want local unavailable", got)
	}
}
