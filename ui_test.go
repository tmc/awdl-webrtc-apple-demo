//go:build darwin

package main

import "testing"

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

func TestLinkHealthPerfError(t *testing.T) {
	if got := linkHealthPerfError(udpPerfRecord{Count: 20, Datagrams: 0}); got != "no replies" {
		t.Fatalf("linkHealthPerfError(all lost) = %q, want no replies", got)
	}
	if got := linkHealthPerfError(udpPerfRecord{Count: 20, Datagrams: 1}); got != "" {
		t.Fatalf("linkHealthPerfError(partial success) = %q, want empty", got)
	}
}
