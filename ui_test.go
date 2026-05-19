//go:build darwin

package main

import (
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
