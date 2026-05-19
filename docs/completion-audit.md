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
| Remote reachability | BLOCKED: the 2026-05-19 retry reached the host only intermittently by ICMP, while `ssh -o BatchMode=yes -o ConnectTimeout=20 ... tmc2@10.0.18.249` and `nc -vz -G 5 10.0.18.249 22` both timed out on TCP/22. `REMOTE_READY_TIMEOUT` can now wait for SSH before matrix setup, but the two-host trace run still needs the peer reachable. |
| Remote matrix | PARTIAL: `CANDIDATE_POLICY=auto REQUIRE_PATHS=1 SSH_TARGET=tmc2@10.0.18.249 REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin OUTPUT=/tmp/awdl-webrtc-matrix-v011.txt SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-v011.md scripts/remote-matrix-bundle.sh` completed all probes and failed only `awdl Pion transport.Net WebRTC`. LAN and Thunderbolt Pion-native WebRTC passed; LAN/Thunderbolt/AWDL raw Network.framework UDP perf, latency, and callback probes passed with path evidence. |

## Latest Retry Evidence

Command:

```sh
WEBRTC_TRACE=1 REMOTE_READY_TIMEOUT=30 CANDIDATE_POLICY=auto REQUIRE_PATHS=1 \
  SSH_TARGET=tmc2@10.0.18.249 \
  REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin \
  OUTPUT=/tmp/awdl-webrtc-matrix-trace-retry.txt \
  SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-trace-retry.md \
  scripts/remote-matrix-bundle.sh
```

Result:

| Section | Kind | Trial | Datagrams | Lost | Loss | Transfer | Bitrate | Elapsed | RTT avg | RTT p95 | Path |
| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| remote reachability | failure | - | - | - | - | - | - | - | - | - | remote reachability exit=255 |

The bundle ran local diagnostics, reached `10.0.18.249` once by ICMP over `en0`
with 321.947 ms latency, then timed out on TCP/22 for `nc` and four SSH
readiness attempts. It did not reach binary upload or LAN/Thunderbolt/AWDL
test execution.

## Prompt-to-Artifact Checklist

This checklist maps the original requests to concrete files, commands, and
current evidence. A passing local gate is not counted as proof for requirements
that need two Macs.

| Request | Artifact / command | Current evidence | Gap |
| --- | --- | --- | --- |
| Demonstrate WebRTC support over Apple links | `main.go` modes `gather`, `pair`, `answer-stdio`, `offer-stdio`, and `offer-ssh`; Pion WebRTC in `newPeer`; [RESULTS.md](../RESULTS.md). | LAN and Thunderbolt Pion-native WebRTC passed in the recorded two-host matrix; UDP-mux AWDL WebRTC passed historically. | Fresh AWDL Pion-native WebRTC still fails or is blocked by peer reachability. |
| Keep backends pluggable | `-backend go\|network`; `-pion-net` switch from `SetICEUDPMux` to `SettingEngine.SetNet`; `github.com/tmc/apple-pion/nwtransport`. | `GOWORK=off go test ./...` and release preflight pass; `apple-pion v0.1.3` is imported without a local replace. | Broader all-Network.framework TCP/TURN/STUN ownership remains outside this demo. |
| Use published `tmc/apple` bindings | `go.mod`; `scripts/release-preflight.sh`; module cache resolution. | Release preflight reports `github.com/tmc/apple v0.6.7` and `github.com/tmc/apple-pion v0.1.3` from the module cache. | None for the published module pins. |
| Prove UDP over AWDL | `udp`, `udp-perf`, `udp-latency`, `udp-callback-*`; [matrix-v0.1.1.md](matrix-v0.1.1.md). | Recorded matrix shows AWDL raw Network.framework UDP perf, latency, and callback probes pass both directions over `awdl0`. | Longer duration and multi-stream sweeps still need a reachable peer. |
| Cover Thunderbolt, AWDL, and LAN tests | `scripts/remote-matrix.sh`; `scripts/remote-matrix-bundle.sh`; path requirements. | Recorded matrix covers all three profiles; latest retry writes a failure summary when SSH setup is blocked. | Fresh matrix cannot run while TCP/22 to `tmc2@10.0.18.249` times out. |
| Add iperf-like output and result tables | UDP summary printers in `main.go`; `cmd/matrix-summary`; [RESULTS.md](../RESULTS.md). | Output includes transfer, bitrate, datagrams, loss, RTT, JSON, path records, summaries, and Markdown tables. | Longer comparative runs still need live peer access. |
| Package reusable pieces out of the temporary demo | `github.com/tmc/apple/x/network/nwpacket`; `github.com/tmc/apple-pion/nwtransport`; `github.com/tmc/apple-pion/icepolicy`. | Published packages have tests/examples and are consumed by the demo as releases. | None for the package split. |
| Add SwiftUI link-health view with fallback | `ui.go`; `ui_test.go`; `-mode ui`; `-mode discover`; README UI and discovery commands. | UI advertises Bonjour TXT metadata, samples Thunderbolt, AWDL, then LAN, and falls back when a higher-priority path returns no replies. Headless discovery emitted JSON records with local Thunderbolt, AWDL, and LAN listener addresses. | It measures UDP link health, not WebRTC datachannel health, and has not been verified on two live Macs in this retry. |
| Avoid SSH as the only control plane | `offer-stdio` and `answer-stdio`; README manual signaling workflow. | Parser tests pass; local two-process smoke exchanged `OFFER` and `ANSWER` lines. | Same-host datachannel did not open; two-host manual signaling still needs a peer or another transport for the two signal lines. |
| Keep repo hygiene and atomic history | Git commits and notes; `git status`; release preflight. | Demo and `apple-pion` worktrees are clean and pushed; notes are pushed; `tmc/apple` unrelated untracked files are preserved. | None for owned files; continue ignoring unrelated `tmc/apple` worktree state. |

## Requirement Checklist

| Requirement | Artifact / evidence | Status | Remaining proof |
| --- | --- | --- | --- |
| Full Pion-native backend | `github.com/tmc/apple-pion/nwtransport` implements Pion `transport.Net`; demo uses `-pion-net`; `apple-pion v0.1.3` filters selected interfaces to the configured address and uses Pion address rewrite rules for link-local host publication. | PARTIAL | LAN and Thunderbolt Pion-native WebRTC pass in the current matrix. AWDL Pion-native WebRTC timed out and needs deeper instrumentation. Broader TCP/TURN/STUN Network.framework ownership is explicitly out of this demo slice. |
| Durable AWDL/link-local candidates | `github.com/tmc/apple-pion/icepolicy`; demo keeps SDP unmodified and publishes explicit `ICECandidateInit` records; `-candidate-policy auto\|mdns\|raw`; tests cover raw candidate extraction and auto-policy selection; local AWDL `-pion-net -mdns disabled` gather prints `candidate_policy=auto raw_candidates=true`; the current matrix used auto policy. | PARTIAL | Candidate publication is durable; AWDL Pion-native datachannel open is still intermittent after candidate exchange. |
| Direction/asymmetry cleanup | `udp-callback-listen`, `udp-callback-request`, path policy checks, `nwpacket` readiness retries, `remote-diagnostics.sh`, matrix-local reachability diagnostics, per-listener UDP socket/route diagnostics, and a simultaneous bidirectional UDP matrix step are implemented. | PASS | Current LAN/Thunderbolt/AWDL sequential raw UDP perf, latency, and callback probes pass both directions with path evidence. Remaining work is performance tuning, not basic reachability. |
| Performance hardening | `udp-perf` supports `-duration`, `-trials`, `-window`, `-streams`, `-packet-timeout`, `-listen-idle-timeout`, loss columns, JSON, latency mode, bidirectional matrix probing, Network.framework path records, and `cmd/matrix-summary` Markdown summaries with failure and `webrtc_trace` candidate-pair rows. | PARTIAL | Current short matrix is in [matrix-v0.1.1.md](matrix-v0.1.1.md). Run longer `DURATION`, `TRIALS`, `WINDOW`, and `STREAMS` sweeps after the AWDL WebRTC issue is instrumented. |
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
SSH_TARGET=tmc2@10.0.18.249 \
REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin \
OUTPUT=/tmp/awdl-webrtc-matrix.txt \
SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-summary.md \
scripts/remote-matrix-bundle.sh

DURATION=10s TRIALS=5 WINDOW=8 STREAMS=2 CANDIDATE_POLICY=auto REQUIRE_PATHS=1 \
SSH_TARGET=tmc2@10.0.18.249 \
REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin \
OUTPUT=/tmp/awdl-webrtc-matrix-long.txt \
SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-long-summary.md \
scripts/remote-matrix-bundle.sh
```

The remaining incomplete item is AWDL Pion-native WebRTC reliability. Raw AWDL
UDP now passes both directions with Network.framework path evidence. The demo
now has opt-in `-webrtc-trace` instrumentation for Pion signaling, ICE
gathering, ICE connection, peer connection, datachannel transitions, and the
wire-signaling split between SDP candidates and explicit `ICECandidateInit`
records. Timeout snapshots also include local-candidate, remote-candidate, and
candidate-pair stats from Pion `GetStats`. The next remote run should use that
trace while isolating whether the AWDL timeout is in candidate installation,
connectivity checks, DTLS/SCTP, or Network.framework reads.
