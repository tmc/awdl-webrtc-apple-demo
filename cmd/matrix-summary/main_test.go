//go:build darwin

package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseRows(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{
		"## lan remote-to-local UDP perf",
		`{"kind":"udp_perf","trial":1,"trials":3,"datagrams":20,"lost":0,"loss_percent":0,"transfer_bytes":48000,"bitrate_bps":1200000,"elapsed_ns":320000000,"rtt_avg_ns":1000000,"rtt_p95_ns":2000000,"paths":[{"status":"NWPathStatusSatisfied","interfaces":[{"name":"en0","type":"NWInterfaceTypeWifi"}]}]}`,
		"link-webrtc-demo: wait for data channel open over awdl0: context deadline exceeded; left webrtc_state label=left signaling=stable ice_gathering=complete ice_connection=checking peer_connection=connecting datachannel=local:link-pair:connecting datachannel_error=- webrtc_stats local_candidates=[candidate:aaa:host/udp/fd00::1:12345] remote_candidates=[candidate:bbb:host/udp/fe80::1:23456] candidate_pairs=[candidate:aaa>candidate:bbb:in-progress:nominated=false:req=2/0:resp=0/0:pkts=0/0]",
		"not json",
		"FAIL: lan local-to-remote UDP perf exit=1",
		"## matrix summary",
		"FAIL: lan local-to-remote UDP perf exit=1",
		`{"kind":"udp_perf_listen","datagrams":18,"expected":20,"lost":2,"loss_percent":10}`,
	}, "\n"))
	rows, err := parseRows(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 {
		t.Fatalf("rows = %d, want 4", len(rows))
	}
	if rows[0].Section != "lan remote-to-local UDP perf" || rows[0].Record.Kind != "udp_perf" {
		t.Fatalf("first row = %#v", rows[0])
	}
	if rows[1].Section != "lan remote-to-local UDP perf" || !strings.Contains(rows[1].Diagnostic, "pairs=[in-progress nominated=false:req=2/0:resp=0/0:pkts=0/0]") {
		t.Fatalf("diagnostic row = %#v", rows[1])
	}
	if rows[2].Section != "lan remote-to-local UDP perf" || rows[2].Failure != "lan local-to-remote UDP perf exit=1" {
		t.Fatalf("failure row = %#v", rows[2])
	}
	if got := formatPaths(rows[0].Record.Paths); got != "NWPathStatusSatisfied en0:NWInterfaceTypeWifi" {
		t.Fatalf("paths = %q", got)
	}
}

func TestPrintTable(t *testing.T) {
	rows := []tableRow{
		{
			Section: "lan | section",
			Record: resultRecord{
				Kind:          "udp_perf",
				Trial:         1,
				Trials:        2,
				Datagrams:     3,
				Lost:          1,
				LossPercent:   25,
				TransferBytes: 2048,
				BitrateBPS:    2_000_000,
				ElapsedNS:     2_000_000,
				RTTAvgNS:      1000,
				RTTP95NS:      2000,
			},
		},
		{
			Section: "matrix summary",
			Failure: "awdl local-to-remote UDP perf exit=1",
		},
		{
			Section:    "awdl Pion transport.Net WebRTC",
			Diagnostic: "left ice=checking peer=connecting pairs=[in-progress nominated=false:req=2/0:resp=0/0:pkts=0/0]",
		},
	}
	var out bytes.Buffer
	printTable(&out, rows)
	got := out.String()
	for _, want := range []string{"| Section | Kind |", "lan \\| section", "2.00 KiB", "2.00 Mbps", "25.00%", "| matrix summary | failure |", "awdl local-to-remote UDP perf exit=1", "| awdl Pion transport.Net WebRTC | webrtc_trace |"} {
		if !strings.Contains(got, want) {
			t.Fatalf("table missing %q:\n%s", want, got)
		}
	}
}

func TestSummarizeWebRTCTrace(t *testing.T) {
	line := "link-webrtc-demo: wait; left webrtc_state label=left signaling=stable ice_gathering=complete ice_connection=checking peer_connection=connecting datachannel=local:link-pair:connecting datachannel_error=- webrtc_stats local_candidates=[candidate:aaa:host/udp/fd00::1:12345] remote_candidates=[candidate:bbb:host/udp/fe80::1:23456] candidate_pairs=[candidate:aaa>candidate:bbb:in-progress:nominated=false:req=2/0:resp=0/0:pkts=0/0]; right webrtc_state label=right signaling=stable ice_gathering=complete ice_connection=failed peer_connection=failed datachannel=-:- datachannel_error=- webrtc_stats local_candidates=[candidate:ccc:host/udp/fe80::2:34567] remote_candidates=[candidate:ddd:host/udp/fe80::1:12345] candidate_pairs=[candidate:ccc>candidate:ddd:failed:nominated=false:req=3/1:resp=0/2:pkts=0/0]"
	got := summarizeWebRTCTrace(line)
	for _, want := range []string{
		"left ice=checking peer=connecting dc=local:link-pair:connecting",
		"local=[host/udp/fd00::1:12345]",
		"remote=[host/udp/fe80::1:23456]",
		"pairs=[in-progress nominated=false:req=2/0:resp=0/0:pkts=0/0]",
		"right ice=failed peer=failed",
		"pairs=[failed nominated=false:req=3/1:resp=0/2:pkts=0/0]",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary missing %q:\n%s", want, got)
		}
	}
}
