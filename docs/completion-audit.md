# Completion Audit

This audit maps the productization goal to concrete artifacts and the current
evidence. It is intentionally strict: local gates and historical remote success
do not complete requirements that need a fresh two-host run.

## Current Gate State

| Gate | Current result |
| --- | --- |
| Release preflight | PASS: `GITHUB_RESOLVE_IP=140.82.116.3 scripts/release-preflight.sh` passed on 2026-05-19; use `GITHUB_RESOLVE_IP` when local DNS cannot resolve GitHub. |
| Published modules | PASS: the demo resolves `github.com/tmc/apple v0.6.7` and `github.com/tmc/apple-pion v0.1.3` from the module cache with no local replace. |
| Published repos | PASS: demo and `apple-pion` HEADs are published at `origin/main`; `apple-pion` is tagged `v0.1.3`. |
| Remote reachability | PASS for the current run: `ssh -o BatchMode=yes -o ConnectTimeout=5 tmc2@10.0.18.249 'echo ok'` returned `ok`, and the successful matrix completed without SSH drops. Earlier SSH-control-plane failures remain recorded in [matrix-try-again-20260519.md](matrix-try-again-20260519.md). |
| Remote matrix | PASS: `WEBRTC_TRACE=1 REMOTE_READY_TIMEOUT=30 REMOTE_STEP_READY_TIMEOUT=30 CANDIDATE_POLICY=auto REQUIRE_PATHS=1 SSH_TARGET=tmc2@10.0.18.249 REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin OUTPUT=/tmp/awdl-webrtc-matrix-signalonly-followup.txt SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-signalonly-followup.md scripts/remote-matrix-bundle.sh` passed all LAN, Thunderbolt, and AWDL probes. Pion-native WebRTC opened datachannels on all three profiles, and UDP perf, latency, and callback probes passed with Network.framework path evidence. |

## Latest Successful Retry Evidence

Command:

```sh
WEBRTC_TRACE=1 REMOTE_READY_TIMEOUT=30 REMOTE_STEP_READY_TIMEOUT=30 \
  CANDIDATE_POLICY=auto REQUIRE_PATHS=1 \
  SSH_TARGET=tmc2@10.0.18.249 \
  REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin \
  OUTPUT=/tmp/awdl-webrtc-matrix-signalonly-followup.txt \
  SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-signalonly-followup.md \
  scripts/remote-matrix-bundle.sh
```

Result:

The full table is in [matrix-20260519-success.md](matrix-20260519-success.md).
The bundle passed `profiles=lan thunderbolt awdl`. Before the successful run,
stale demo listener processes from an interrupted older matrix were killed on
both hosts. The matrix harness now performs that stale-process cleanup by
default for commands whose process line starts with `LOCAL_BIN` or `REMOTE_BIN`.

## Prompt-to-Artifact Checklist

This checklist maps the original requests to concrete files, commands, and
current evidence. A passing local gate is not counted as proof for requirements
that need two Macs.

| Request | Artifact / command | Current evidence | Gap |
| --- | --- | --- | --- |
| Demonstrate WebRTC support over Apple links | `main.go` modes `gather`, `pair`, `answer-stdio`, `offer-stdio`, and `offer-ssh`; Pion WebRTC in `newPeer`; [RESULTS.md](../RESULTS.md). | LAN, Thunderbolt, and AWDL Pion-native WebRTC passed in the current two-host matrix. | None for the selected-link UDP WebRTC proof. |
| Keep backends pluggable | `-backend go\|network`; `-pion-net` switch from `SetICEUDPMux` to `SettingEngine.SetNet`; `github.com/tmc/apple-pion/nwtransport`. | `GOWORK=off go test ./...` and release preflight pass; `apple-pion v0.1.3` is imported without a local replace. | Broader all-Network.framework TCP/TURN/STUN ownership remains outside this demo. |
| Use published `tmc/apple` bindings | `go.mod`; `scripts/release-preflight.sh`; module cache resolution. | Release preflight reports `github.com/tmc/apple v0.6.7` and `github.com/tmc/apple-pion v0.1.3` from the module cache. | None for the published module pins. |
| Prove UDP over AWDL | `udp`, `udp-perf`, `udp-latency`, `udp-callback-*`; [matrix-20260519-success.md](matrix-20260519-success.md). | Current matrix shows AWDL raw Network.framework UDP perf, latency, and callback probes pass both directions over `awdl0`. | Longer duration and multi-stream sweeps are still useful for throughput claims. |
| Cover Thunderbolt, AWDL, and LAN tests | `scripts/remote-matrix.sh`; `scripts/remote-matrix-bundle.sh`; path requirements. | Current matrix covers all three profiles and passes WebRTC, UDP perf, latency, and callback probes with path evidence. | None for the short-run proof. |
| Add iperf-like output and result tables | UDP summary printers in `main.go`; `cmd/matrix-summary`; [RESULTS.md](../RESULTS.md). | Output includes transfer, bitrate, datagrams, loss, RTT, JSON, path records, summaries, and Markdown tables. | Longer comparative runs still need live peer access. |
| Package reusable pieces out of the temporary demo | `github.com/tmc/apple/x/network/nwpacket`; `github.com/tmc/apple-pion/nwtransport`; `github.com/tmc/apple-pion/icepolicy`. | Published packages have tests/examples and are consumed by the demo as releases. | None for the package split. |
| Add SwiftUI link-health view with fallback | `ui.go`; `ui_test.go`; `-mode ui`; `-mode discover`; `-mode discover-wait`; README UI and discovery commands. | UI advertises Bonjour TXT metadata, samples Thunderbolt, AWDL, then LAN, and falls back when a higher-priority path returns no replies. Headless discovery emitted JSON records with local Thunderbolt, AWDL, and LAN listener addresses; `discover-wait` emitted one matching peer record when a local publisher was present. TXT metadata includes advertised version, commit, and supported modes when available. | It measures UDP link health, not WebRTC datachannel health. A two-live-Mac UI session is still worth doing. |
| Avoid SSH as the only control plane | `offer-stdio`, `answer-stdio`, `offer-bonjour`, and `answer-bonjour`; README manual and Bonjour signaling workflows. | Parser tests pass; local stdio smoke exchanged `OFFER` and `ANSWER` lines. Bonjour signaling advertises `_awdl-webrtc-signal._tcp`, falls back from Bonjour endpoint dialing to `NSNetService` host/port resolution, and `-signal-only` exchanged local `OFFER`/`ANSWER` lines with exit 0 on both sides. | Same-host stdio and full Bonjour datachannels did not open; both reached ICE checking with no same-host candidate-pair responses. Two-host Bonjour signaling still needs a reachable peer. |
| Keep repo hygiene and atomic history | Git commits and notes; `git status`; release preflight. | Demo and `apple-pion` worktrees are clean and pushed; notes are pushed; `tmc/apple` unrelated untracked files are preserved. | None for owned files; continue ignoring unrelated `tmc/apple` worktree state. |

## Requirement Checklist

| Requirement | Artifact / evidence | Status | Remaining proof |
| --- | --- | --- | --- |
| Full Pion-native backend | `github.com/tmc/apple-pion/nwtransport` implements Pion `transport.Net`; demo uses `-pion-net`; `apple-pion v0.1.3` filters selected interfaces to the configured address and uses Pion address rewrite rules for link-local host publication. | PASS for the selected UDP surface | LAN, Thunderbolt, and AWDL Pion-native WebRTC pass in the current matrix. Broader TCP/TURN/STUN Network.framework ownership is explicitly out of this demo slice. |
| Durable AWDL/link-local candidates | `github.com/tmc/apple-pion/icepolicy`; demo keeps SDP unmodified and publishes explicit `ICECandidateInit` records; `-candidate-policy auto\|mdns\|raw`; tests cover raw candidate extraction and auto-policy selection; local AWDL `-pion-net -mdns disabled` gather prints `candidate_policy=auto raw_candidates=true`; the current matrix used auto policy. | PASS | Candidate publication is durable enough for the current AWDL Pion-native datachannel proof. |
| Direction/asymmetry cleanup | `udp-callback-listen`, `udp-callback-request`, path policy checks, `nwpacket` readiness retries, `remote-diagnostics.sh`, matrix-local reachability diagnostics, per-listener UDP socket/route diagnostics, and a simultaneous bidirectional UDP matrix step are implemented. | PASS | Current LAN/Thunderbolt/AWDL sequential raw UDP perf, latency, and callback probes pass both directions with path evidence. Remaining work is performance tuning, not basic reachability. |
| Performance hardening | `udp-perf` supports `-duration`, `-trials`, `-window`, `-streams`, `-packet-timeout`, `-listen-idle-timeout`, loss columns, JSON, latency mode, bidirectional matrix probing, Network.framework path records, and `cmd/matrix-summary` Markdown summaries with failure and `webrtc_trace` candidate-pair rows. | PARTIAL | Current short matrix is in [matrix-20260519-success.md](matrix-20260519-success.md). Run longer `DURATION`, `TRIALS`, `WINDOW`, and `STREAMS` sweeps before making stable throughput claims. |
| Reusable package split | `github.com/tmc/apple/x/network/nwpacket v0.6.7`; `github.com/tmc/apple-pion v0.1.3`; demo has no local replaces. | PASS | None. |
| Repo hygiene | Demo and `apple-pion` worktrees are clean; owned `tmc/apple/x/network/nwpacket` is clean; release preflight passes. | PASS | Preserve unrelated dirty generated/private files in `tmc/apple`; they are outside this slice. |

## Required Remote Commands

Use these for follow-up runs:

```sh
SSH_TARGET=tmc2@10.0.18.249 \
  THUNDERBOLT_PEER=169.254.88.35 \
  AWDL_PEER='fe80::9477:6dff:fe11:6a55%awdl0' \
  OUTPUT=/tmp/awdl-webrtc-diagnostics.txt \
  scripts/remote-diagnostics.sh

CANDIDATE_POLICY=auto \
REQUIRE_PATHS=1 \
REMOTE_READY_TIMEOUT=30 \
REMOTE_STEP_READY_TIMEOUT=30 \
SSH_TARGET=tmc2@10.0.18.249 \
REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin \
OUTPUT=/tmp/awdl-webrtc-matrix.txt \
SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-summary.md \
scripts/remote-matrix-bundle.sh

DURATION=10s TRIALS=5 WINDOW=8 STREAMS=2 CANDIDATE_POLICY=auto REQUIRE_PATHS=1 \
REMOTE_READY_TIMEOUT=30 REMOTE_STEP_READY_TIMEOUT=30 \
SSH_TARGET=tmc2@10.0.18.249 \
REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin \
OUTPUT=/tmp/awdl-webrtc-matrix-long.txt \
SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-long-summary.md \
scripts/remote-matrix-bundle.sh
```

The remaining incomplete items are product-hardening items rather than the core
proof: two-host Bonjour signaling, two-live-Mac SwiftUI observation, and longer
duration/multi-stream performance sweeps. AWDL Pion-native WebRTC now passes in
the current matrix when using `-mdns disabled -candidate-policy auto`.
