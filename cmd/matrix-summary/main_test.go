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
		"not json",
		"## matrix summary",
		`{"kind":"udp_perf_listen","datagrams":18,"expected":20,"lost":2,"loss_percent":10}`,
	}, "\n"))
	rows, err := parseRows(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0].Section != "lan remote-to-local UDP perf" || rows[0].Record.Kind != "udp_perf" {
		t.Fatalf("first row = %#v", rows[0])
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
	}
	var out bytes.Buffer
	printTable(&out, rows)
	got := out.String()
	for _, want := range []string{"| Section | Kind |", "lan \\| section", "2.00 KiB", "2.00 Mbps", "25.00%"} {
		if !strings.Contains(got, want) {
			t.Fatalf("table missing %q:\n%s", want, got)
		}
	}
}
