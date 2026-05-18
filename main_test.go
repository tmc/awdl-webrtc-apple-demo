//go:build darwin

package main

import (
	"net"
	"testing"
)

func TestShouldBindUDPToInterface(t *testing.T) {
	tests := []struct {
		name    string
		iface   linkInterface
		network string
		ip      net.IP
		want    bool
	}{
		{
			name:    "awdl",
			iface:   linkInterface{Name: "awdl0"},
			network: "udp6",
			ip:      net.ParseIP("fe80::1"),
			want:    true,
		},
		{
			name:    "ipv6",
			iface:   linkInterface{Name: "en0"},
			network: "udp6",
			ip:      net.ParseIP("2606:4700:4700::1111"),
			want:    true,
		},
		{
			name:    "ipv4 link local",
			iface:   linkInterface{Name: "bridge0"},
			network: "udp4",
			ip:      net.ParseIP("169.254.61.91"),
			want:    true,
		},
		{
			name:    "ipv4 lan",
			iface:   linkInterface{Name: "en0"},
			network: "udp4",
			ip:      net.ParseIP("10.0.0.1"),
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldBindUDPToInterface(tt.iface, tt.network, tt.ip)
			if got != tt.want {
				t.Fatalf("shouldBindUDPToInterface(%s, %s, %s) = %t, want %t", tt.iface.Name, tt.network, tt.ip, got, tt.want)
			}
		})
	}
}
