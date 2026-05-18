//go:build darwin

package main

import (
	"context"
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

func TestDefaultThunderboltInterface(t *testing.T) {
	tests := []struct {
		name   string
		ifaces map[string]linkInterface
		want   string
	}{
		{
			name: "prefers bridge with address",
			ifaces: map[string]linkInterface{
				"bridge0": {Name: "bridge0", IPs: []net.IP{net.ParseIP("169.254.61.91")}},
				"en1":     {Name: "en1", IPs: []net.IP{net.ParseIP("172.31.253.1")}},
			},
			want: "bridge0",
		},
		{
			name: "falls back to member with address",
			ifaces: map[string]linkInterface{
				"bridge0": {Name: "bridge0"},
				"en1":     {Name: "en1", IPs: []net.IP{net.ParseIP("172.31.253.1")}},
			},
			want: "en1",
		},
		{
			name: "keeps first existing without address",
			ifaces: map[string]linkInterface{
				"bridge0": {Name: "bridge0"},
				"en1":     {Name: "en1"},
			},
			want: "bridge0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := defaultThunderboltInterface(func(name string) (linkInterface, error) {
				iface, ok := tt.ifaces[name]
				if !ok {
					return linkInterface{}, net.UnknownNetworkError(name)
				}
				return iface, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("defaultThunderboltInterface() = %q, want %q", got, tt.want)
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

func TestRunUDPEchoPerfCountsReadTimeoutsAsLoss(t *testing.T) {
	conn := &timeoutPacketConn{}
	result, err := runUDPEchoPerf(context.Background(), conn, &net.UDPAddr{IP: net.ParseIP("192.0.2.1"), Port: 9}, 3, 16, 0, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if result.Lost != 3 || len(result.RTT) != 0 {
		t.Fatalf("result lost=%d rtt=%d, want 3 lost and 0 rtt", result.Lost, len(result.RTT))
	}
	if conn.writes != 3 {
		t.Fatalf("writes = %d, want 3", conn.writes)
	}
}

func TestPacketDeadlineUsesEarlierContextDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	got := packetDeadline(ctx, time.Second)
	if time.Until(got) > 100*time.Millisecond {
		t.Fatalf("packetDeadline ignored context deadline: %s", got)
	}
}

type timeoutPacketConn struct {
	writes int
}

func (c *timeoutPacketConn) ReadFrom([]byte) (int, net.Addr, error) {
	return 0, nil, timeoutError{}
}

func (c *timeoutPacketConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	c.writes++
	return len(b), nil
}

func (c *timeoutPacketConn) Close() error                     { return nil }
func (c *timeoutPacketConn) LocalAddr() net.Addr              { return &net.UDPAddr{} }
func (c *timeoutPacketConn) SetDeadline(time.Time) error      { return nil }
func (c *timeoutPacketConn) SetReadDeadline(time.Time) error  { return nil }
func (c *timeoutPacketConn) SetWriteDeadline(time.Time) error { return nil }

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

var _ net.Error = timeoutError{}
