//go:build darwin

package main

import (
	"encoding/json"
	"net"
	"testing"
	"time"
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

func TestUDPPerfRecordForTrial(t *testing.T) {
	result := udpPerfResult{
		Count:   3,
		Size:    100,
		Warmup:  1,
		Lost:    1,
		Elapsed: 10 * time.Millisecond,
		RTT: []time.Duration{
			3 * time.Millisecond,
			1 * time.Millisecond,
		},
	}
	record := udpPerfRecordForTrial(result, 2, 4)
	if record.Kind != "udp_perf" || record.Trial != 2 || record.Trials != 4 {
		t.Fatalf("record identity = %#v", record)
	}
	if record.Datagrams != 2 || record.Lost != 1 || record.TransferBytes != 400 {
		t.Fatalf("record counts = %#v", record)
	}
	if record.LossPercent != 100.0/3.0 {
		t.Fatalf("loss percent = %v", record.LossPercent)
	}
	if record.RTTMinNS != int64(time.Millisecond) || record.RTTMaxNS != int64(3*time.Millisecond) {
		t.Fatalf("rtt bounds = %#v", record)
	}
	if _, err := json.Marshal(record); err != nil {
		t.Fatal(err)
	}
}

func TestUDPPerfListenRecord(t *testing.T) {
	record := udpPerfListenRecord(8, 9600, 2*time.Second, 10)
	if record.Kind != "udp_perf_listen" || record.Datagrams != 8 || record.Expected != 10 || record.Lost != 2 {
		t.Fatalf("listen record = %#v", record)
	}
	if record.LossPercent != 20 {
		t.Fatalf("loss percent = %v", record.LossPercent)
	}
	if _, err := json.Marshal(record); err != nil {
		t.Fatal(err)
	}
}
