//go:build darwin

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/tmc/apple-pion/icepolicy"
	applenetwork "github.com/tmc/apple/network"
	"github.com/tmc/apple/x/network/nwpacket"
)

func TestValidateNetworkConnectPolicy(t *testing.T) {
	tests := []struct {
		name    string
		policy  networkConnectPolicy
		wantErr bool
	}{
		{
			name:   "defaults",
			policy: networkConnectPolicy{Timeout: defaultNetworkConnectTimeout, Retries: defaultNetworkConnectRetries},
		},
		{
			name:    "zero timeout",
			policy:  networkConnectPolicy{Timeout: 0, Retries: 0},
			wantErr: true,
		},
		{
			name:    "negative timeout",
			policy:  networkConnectPolicy{Timeout: -time.Nanosecond, Retries: 0},
			wantErr: true,
		},
		{
			name:    "negative retries",
			policy:  networkConnectPolicy{Timeout: time.Second, Retries: -1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNetworkConnectPolicy(tt.policy)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateNetworkConnectPolicy err = %v, wantErr %t", err, tt.wantErr)
			}
		})
	}
}

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

func TestNewWireSignalPublishesRawCandidates(t *testing.T) {
	desc := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP: strings.Join([]string{
			"v=0",
			"m=application 9 UDP/DTLS/SCTP webrtc-datachannel",
			"a=mid:0",
			"a=candidate:1 1 udp 2130706431 fd00::1 12345 typ host ufrag test",
			"a=end-of-candidates",
		}, "\n"),
	}
	signal := newWireSignal(desc, icepolicy.Policy{RawHostCandidates: true}, net.ParseIP("fe80::1"))
	if signal.Description.SDP != desc.SDP {
		t.Fatalf("description SDP changed:\n%s", signal.Description.SDP)
	}
	if len(signal.Candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(signal.Candidates))
	}
	if !strings.Contains(signal.Candidates[0].Candidate, " fe80::1 12345 typ host ") {
		t.Fatalf("candidate was not published with link-local IP: %q", signal.Candidates[0].Candidate)
	}
	if signal.Candidates[0].SDPMid == nil || *signal.Candidates[0].SDPMid != "0" {
		t.Fatalf("candidate mid = %v, want 0", signal.Candidates[0].SDPMid)
	}
	if signal.Candidates[0].SDPMLineIndex == nil || *signal.Candidates[0].SDPMLineIndex != 0 {
		t.Fatalf("candidate m-line = %v, want 0", signal.Candidates[0].SDPMLineIndex)
	}
}

func TestDecodeWireSignalAcceptsLegacyDescription(t *testing.T) {
	desc := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: "v=0"}
	data, err := json.Marshal(desc)
	if err != nil {
		t.Fatal(err)
	}
	signal, err := decodeWireSignal(base64.StdEncoding.EncodeToString(data))
	if err != nil {
		t.Fatal(err)
	}
	if signal.Description.SDP != desc.SDP || signal.Description.Type != desc.Type {
		t.Fatalf("legacy signal = %#v, want %#v", signal.Description, desc)
	}
}

func TestUDPCallbackWireRequest(t *testing.T) {
	data, err := marshalUDPCallbackRequest("[fe80::1%awdl0]:12345", "ping")
	if err != nil {
		t.Fatal(err)
	}
	req, err := parseUDPCallbackRequest(data)
	if err != nil {
		t.Fatal(err)
	}
	if req.Callback != "[fe80::1%awdl0]:12345" || req.Message != "ping" {
		t.Fatalf("callback request = %#v", req)
	}
	if _, err := marshalUDPCallbackRequest("", "ping"); err == nil {
		t.Fatal("marshal empty callback succeeded")
	}
	if _, err := parseUDPCallbackRequest([]byte(`{"message":"ping"}`)); err == nil {
		t.Fatal("parse missing callback succeeded")
	}
}

func TestUDPPerfRecordForTrial(t *testing.T) {
	result := udpPerfResult{
		Count:    3,
		Duration: 5 * time.Second,
		Size:     100,
		Warmup:   1,
		Window:   2,
		Streams:  4,
		Lost:     1,
		Elapsed:  10 * time.Millisecond,
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
	if record.Window != 2 {
		t.Fatalf("record window = %d, want 2", record.Window)
	}
	if record.Streams != 4 {
		t.Fatalf("record streams = %d, want 4", record.Streams)
	}
	if record.DurationNS != int64(5*time.Second) {
		t.Fatalf("record duration = %d, want %d", record.DurationNS, int64(5*time.Second))
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

func TestUDPLatencyRecordForTrial(t *testing.T) {
	result := udpPerfResult{
		Count:   3,
		Size:    64,
		Warmup:  1,
		Streams: 2,
		Lost:    1,
		Elapsed: 10 * time.Millisecond,
		RTT: []time.Duration{
			3 * time.Millisecond,
			time.Millisecond,
		},
	}
	record := udpLatencyRecordForTrial(result, 2, 4)
	if record.Kind != "udp_latency" || record.Trial != 2 || record.Trials != 4 {
		t.Fatalf("record identity = %#v", record)
	}
	if record.Datagrams != 2 || record.Lost != 1 || record.Streams != 2 {
		t.Fatalf("record counts = %#v", record)
	}
	if record.Size != 64 || record.Warmup != 1 {
		t.Fatalf("record shape = %#v", record)
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

func TestUDPPathRecords(t *testing.T) {
	result := udpPerfResult{
		Count:   1,
		Size:    64,
		Streams: 1,
		Elapsed: time.Millisecond,
		RTT:     []time.Duration{time.Millisecond},
		Paths: []nwpacket.Path{
			{
				Status: applenetwork.NWPathStatusSatisfied,
				Interfaces: []nwpacket.PathInterface{
					{Name: "awdl0", Index: 16, Type: applenetwork.NWInterfaceTypeWifi},
				},
			},
		},
	}
	record := udpPerfRecordForTrial(result, 1, 1)
	if len(record.Paths) != 1 || record.Paths[0].Status != "NWPathStatusSatisfied" {
		t.Fatalf("paths = %#v", record.Paths)
	}
	if len(record.Paths[0].Interfaces) != 1 || record.Paths[0].Interfaces[0].Name != "awdl0" {
		t.Fatalf("path interfaces = %#v", record.Paths)
	}
	if err := checkUDPPathPolicy(result, udpPathPolicy{RequireInterface: "awdl0"}); err != nil {
		t.Fatal(err)
	}
	if err := checkUDPPathPolicy(result, udpPathPolicy{RequireInterface: "en0"}); err == nil {
		t.Fatal("checkUDPPathPolicy accepted missing interface")
	}
}

func TestUDPPerfRecordForSummary(t *testing.T) {
	results := []udpPerfResult{
		{
			Count:   3,
			Size:    100,
			Warmup:  1,
			Window:  2,
			Streams: 2,
			Lost:    1,
			Elapsed: 10 * time.Millisecond,
			RTT: []time.Duration{
				time.Millisecond,
				3 * time.Millisecond,
			},
		},
		{
			Count:   3,
			Size:    100,
			Warmup:  1,
			Window:  2,
			Streams: 2,
			Lost:    0,
			Elapsed: 20 * time.Millisecond,
			RTT: []time.Duration{
				2 * time.Millisecond,
				4 * time.Millisecond,
				5 * time.Millisecond,
			},
		},
	}
	summary := aggregateUDPPerfResults(results)
	if summary.Count != 6 || summary.Warmup != 2 || summary.Lost != 1 || summary.Elapsed != 30*time.Millisecond {
		t.Fatalf("summary counts = %#v", summary)
	}
	if len(summary.RTT) != 5 || summary.Window != 2 || summary.Streams != 2 || summary.Size != 100 {
		t.Fatalf("summary shape = %#v", summary)
	}
	record := udpPerfRecordForSummary(summary, len(results))
	if record.Kind != "udp_perf_summary" || record.Trials != 2 {
		t.Fatalf("record identity = %#v", record)
	}
	if record.Datagrams != 5 || record.Lost != 1 || record.TransferBytes != 1000 {
		t.Fatalf("record counts = %#v", record)
	}
	if record.LossPercent != 100.0/6.0 {
		t.Fatalf("loss percent = %v", record.LossPercent)
	}
	if record.RTTMinNS != int64(time.Millisecond) || record.RTTMaxNS != int64(5*time.Millisecond) {
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

func TestUDPPerfExpected(t *testing.T) {
	got, err := udpPerfExpected(10, 2, 3, 4, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got != 144 {
		t.Fatalf("udpPerfExpected = %d, want 144", got)
	}
	got, err = udpPerfExpected(10, 2, 3, 4, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Fatalf("duration udpPerfExpected = %d, want 0", got)
	}
}

func TestRunUDPEchoPerfCountsReadTimeoutsAsLoss(t *testing.T) {
	conn := &timeoutPacketConn{}
	result, err := runUDPEchoPerf(context.Background(), conn, &net.UDPAddr{IP: net.ParseIP("192.0.2.1"), Port: 9}, 3, 16, 0, 1, time.Millisecond, 0)
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

func TestRunUDPEchoPerfWindowPipelines(t *testing.T) {
	conn := &echoPacketConn{addr: &net.UDPAddr{IP: net.ParseIP("192.0.2.1"), Port: 9}}
	result, err := runUDPEchoPerf(context.Background(), conn, conn.addr, 5, 16, 0, 3, time.Second, 0)
	if err != nil {
		t.Fatal(err)
	}
	if result.Lost != 0 || len(result.RTT) != 5 || result.Window != 3 {
		t.Fatalf("result lost=%d rtt=%d window=%d, want 0, 5, 3", result.Lost, len(result.RTT), result.Window)
	}
	if conn.writes != 5 {
		t.Fatalf("writes = %d, want 5", conn.writes)
	}
	if conn.maxQueued != 3 {
		t.Fatalf("max queued = %d, want 3", conn.maxQueued)
	}
}

func TestRunUDPEchoPerfStreamsAggregates(t *testing.T) {
	conns := []net.PacketConn{
		&timeoutPacketConn{},
		&timeoutPacketConn{},
	}
	result, err := runUDPEchoPerfStreams(context.Background(), conns, &net.UDPAddr{IP: net.ParseIP("192.0.2.1"), Port: 9}, 3, 16, 0, 1, time.Millisecond, 0)
	if err != nil {
		t.Fatal(err)
	}
	if result.Streams != 2 || result.Count != 6 || result.Lost != 6 || len(result.RTT) != 0 {
		t.Fatalf("result streams=%d count=%d lost=%d rtt=%d, want 2, 6, 6, 0", result.Streams, result.Count, result.Lost, len(result.RTT))
	}
}

func TestRunUDPEchoPerfDuration(t *testing.T) {
	conn := &echoPacketConn{addr: &net.UDPAddr{IP: net.ParseIP("192.0.2.1"), Port: 9}}
	result, err := runUDPEchoPerf(context.Background(), conn, conn.addr, 0, 16, 0, 4, time.Second, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if result.Duration != time.Millisecond {
		t.Fatalf("duration = %s, want 1ms", result.Duration)
	}
	if result.Count == 0 || len(result.RTT) != result.Count || result.Lost != 0 {
		t.Fatalf("result count=%d rtt=%d lost=%d, want successful duration run", result.Count, len(result.RTT), result.Lost)
	}
	if conn.maxQueued != 4 {
		t.Fatalf("max queued = %d, want 4", conn.maxQueued)
	}
}

func TestRunUDPPerfListenStopsAfterIdle(t *testing.T) {
	conn := &listenPacketConn{
		packets: [][]byte{make([]byte, 16)},
		addr:    &net.UDPAddr{IP: net.ParseIP("192.0.2.1"), Port: 9},
	}
	result, err := runUDPPerfListen(context.Background(), conn, 0, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if result.Packets != 1 || result.Bytes != 16 {
		t.Fatalf("result packets=%d bytes=%d, want 1 and 16", result.Packets, result.Bytes)
	}
	if conn.writes != 1 {
		t.Fatalf("writes = %d, want 1", conn.writes)
	}
}

func TestRunUDPPerfListenRejectsNegativeIdleTimeout(t *testing.T) {
	_, err := runUDPPerfListen(context.Background(), &timeoutPacketConn{}, 0, -time.Nanosecond)
	if err == nil {
		t.Fatal("runUDPPerfListen accepted negative idle timeout")
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

type echoPacketConn struct {
	packets   [][]byte
	addr      net.Addr
	writes    int
	maxQueued int
}

func (c *echoPacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	if len(c.packets) == 0 {
		return 0, nil, timeoutError{}
	}
	packet := c.packets[0]
	c.packets = c.packets[1:]
	return copy(b, packet), c.addr, nil
}

func (c *echoPacketConn) WriteTo(b []byte, _ net.Addr) (int, error) {
	c.writes++
	c.packets = append(c.packets, append([]byte(nil), b...))
	if len(c.packets) > c.maxQueued {
		c.maxQueued = len(c.packets)
	}
	return len(b), nil
}

func (c *echoPacketConn) Close() error                     { return nil }
func (c *echoPacketConn) LocalAddr() net.Addr              { return &net.UDPAddr{} }
func (c *echoPacketConn) SetDeadline(time.Time) error      { return nil }
func (c *echoPacketConn) SetReadDeadline(time.Time) error  { return nil }
func (c *echoPacketConn) SetWriteDeadline(time.Time) error { return nil }

type listenPacketConn struct {
	packets [][]byte
	addr    net.Addr
	writes  int
}

func (c *listenPacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	if len(c.packets) == 0 {
		return 0, nil, timeoutError{}
	}
	packet := c.packets[0]
	c.packets = c.packets[1:]
	return copy(b, packet), c.addr, nil
}

func (c *listenPacketConn) WriteTo(b []byte, _ net.Addr) (int, error) {
	c.writes++
	return len(b), nil
}

func (c *listenPacketConn) Close() error                     { return nil }
func (c *listenPacketConn) LocalAddr() net.Addr              { return &net.UDPAddr{} }
func (c *listenPacketConn) SetDeadline(time.Time) error      { return nil }
func (c *listenPacketConn) SetReadDeadline(time.Time) error  { return nil }
func (c *listenPacketConn) SetWriteDeadline(time.Time) error { return nil }

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

var _ net.Error = timeoutError{}
