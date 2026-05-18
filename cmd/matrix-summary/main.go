//go:build darwin

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type resultRecord struct {
	Kind          string       `json:"kind"`
	Trial         int          `json:"trial"`
	Trials        int          `json:"trials"`
	Datagrams     int          `json:"datagrams"`
	Expected      int64        `json:"expected"`
	Lost          int          `json:"lost"`
	LossPercent   float64      `json:"loss_percent"`
	TransferBytes int64        `json:"transfer_bytes"`
	BitrateBPS    float64      `json:"bitrate_bps"`
	ElapsedNS     int64        `json:"elapsed_ns"`
	RTTAvgNS      int64        `json:"rtt_avg_ns"`
	RTTP95NS      int64        `json:"rtt_p95_ns"`
	Paths         []pathRecord `json:"paths"`
}

type pathRecord struct {
	Status     string            `json:"status"`
	Interfaces []interfaceRecord `json:"interfaces"`
}

type interfaceRecord struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type tableRow struct {
	Section string
	Record  resultRecord
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "matrix-summary: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) > 1 {
		return errors.New("usage: matrix-summary [transcript]")
	}
	var in io.Reader = os.Stdin
	if len(args) == 1 && args[0] != "-" {
		f, err := os.Open(args[0])
		if err != nil {
			return fmt.Errorf("open %s: %w", args[0], err)
		}
		defer f.Close()
		in = f
	}
	rows, err := parseRows(in)
	if err != nil {
		return err
	}
	printTable(out, rows)
	return nil
}

func parseRows(in io.Reader) ([]tableRow, error) {
	var rows []tableRow
	section := ""
	scanner := bufio.NewScanner(in)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "## ") {
			section = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			continue
		}
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var record resultRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			continue
		}
		if record.Kind == "" {
			continue
		}
		rows = append(rows, tableRow{Section: section, Record: record})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read transcript: %w", err)
	}
	return rows, nil
}

func printTable(out io.Writer, rows []tableRow) {
	fmt.Fprintln(out, "| Section | Kind | Trial | Datagrams | Lost | Loss | Transfer | Bitrate | Elapsed | RTT avg | RTT p95 | Path |")
	fmt.Fprintln(out, "| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |")
	for _, row := range rows {
		record := row.Record
		fmt.Fprintf(out, "| %s | %s | %s | %d%s | %d | %.2f%% | %s | %s | %s | %s | %s | %s |\n",
			escapeCell(row.Section),
			escapeCell(record.Kind),
			escapeCell(formatTrial(record)),
			record.Datagrams,
			formatExpected(record.Expected),
			record.Lost,
			record.LossPercent,
			formatBytes(record.TransferBytes),
			formatBitrate(record.BitrateBPS),
			formatDurationNS(record.ElapsedNS),
			formatDurationNS(record.RTTAvgNS),
			formatDurationNS(record.RTTP95NS),
			escapeCell(formatPaths(record.Paths)),
		)
	}
}

func formatTrial(record resultRecord) string {
	if record.Trials == 0 {
		return "-"
	}
	if record.Trial == 0 {
		return fmt.Sprintf("summary/%d", record.Trials)
	}
	return fmt.Sprintf("%d/%d", record.Trial, record.Trials)
}

func formatExpected(expected int64) string {
	if expected <= 0 {
		return ""
	}
	return fmt.Sprintf("/%d", expected)
}

func formatBytes(bytes int64) string {
	const unit = 1024
	value := float64(bytes)
	for _, suffix := range []string{"B", "KiB", "MiB", "GiB"} {
		if value < unit || suffix == "GiB" {
			return fmt.Sprintf("%.2f %s", value, suffix)
		}
		value /= unit
	}
	return fmt.Sprintf("%d B", bytes)
}

func formatBitrate(bps float64) string {
	for _, suffix := range []string{"bps", "Kbps", "Mbps", "Gbps"} {
		if bps < 1000 || suffix == "Gbps" {
			return fmt.Sprintf("%.2f %s", bps, suffix)
		}
		bps /= 1000
	}
	return "0 bps"
}

func formatDurationNS(ns int64) string {
	if ns == 0 {
		return "-"
	}
	switch {
	case ns >= 1e9:
		return fmt.Sprintf("%.3fs", float64(ns)/1e9)
	case ns >= 1e6:
		return fmt.Sprintf("%.3fms", float64(ns)/1e6)
	case ns >= 1e3:
		return fmt.Sprintf("%.3fus", float64(ns)/1e3)
	default:
		return fmt.Sprintf("%dns", ns)
	}
}

func formatPaths(paths []pathRecord) string {
	if len(paths) == 0 {
		return "-"
	}
	var parts []string
	for _, path := range paths {
		var ifaces []string
		for _, iface := range path.Interfaces {
			if iface.Name == "" {
				continue
			}
			if iface.Type == "" {
				ifaces = append(ifaces, iface.Name)
			} else {
				ifaces = append(ifaces, iface.Name+":"+iface.Type)
			}
		}
		if len(ifaces) == 0 {
			parts = append(parts, path.Status)
		} else {
			parts = append(parts, path.Status+" "+strings.Join(ifaces, ","))
		}
	}
	return strings.Join(parts, "; ")
}

func escapeCell(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	if s == "" {
		return "-"
	}
	return s
}
