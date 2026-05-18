//go:build darwin

package nwtransport

import (
	"testing"

	"github.com/pion/transport/v4"
)

func TestNetImplementsTransportNet(t *testing.T) {
	var _ transport.Net = (*Net)(nil)
}

func TestNativePacketAddress(t *testing.T) {
	tests := []struct {
		name    string
		network string
		address string
		want    bool
	}{
		{"udp4 host", "udp4", "192.0.2.1:0", true},
		{"udp6 scoped host", "udp6", "[fe80::1%awdl0]:9", true},
		{"unspecified", "udp4", "0.0.0.0:0", false},
		{"multicast", "udp4", "224.0.0.251:5353", false},
		{"tcp", "tcp4", "192.0.2.1:0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got, err := nativePacketAddress(tt.network, tt.address)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("nativePacketAddress(%q, %q) native = %t, want %t", tt.network, tt.address, got, tt.want)
			}
		})
	}
}
