# Completion Audit

This audit maps the productization goal to concrete artifacts and the current
evidence. It is intentionally strict: local gates and historical remote success
do not complete requirements that need a fresh two-host run.

## Current Gate State

| Gate | Current result |
| --- | --- |
| Local gates | PASS: `go test ./...`, `go vet ./...`, `bash -n scripts/*.sh`, `go test ./x/network/nwpacket`, `go vet ./x/network/nwpacket`, and `go test ./...`/`go vet ./...` in `../apple-pion` passed on 2026-05-19. |
| Workspace modules | PASS: checked-in `go.work` uses `.`, `../apple`, and `../apple-pion`; the matrix binary was built from that workspace. |
| Tag policy | PASS: no new tags were published for the current proof. |
| Current Pion backend boundary | PASS: `../apple-pion` commits `2cca84d` and `15e13e4` are pushed. Native Network.framework ownership is restricted to numeric UDP endpoints and configured wildcard listeners; named UDP endpoints, DNS, TCP, and unsupported families stay on the fallback `transport.Net`. |
| Last successful remote reachability | PASS for the workspace matrix run: `ssh -o BatchMode=yes -o ConnectTimeout=5 tmc2@10.0.18.249 'echo ok'` returned `ok`, and the successful matrix completed without SSH drops. Earlier SSH-control-plane failures remain recorded in [matrix-try-again-20260519.md](matrix-try-again-20260519.md). |
| Current peer reachability | BLOCKED on 2026-05-19: later Bonjour and soak attempts could not reach `tmc2@10.0.18.249`; route/ping/TCP/22 diagnostics are preserved in `/tmp/awdl-webrtc-bonjour-unreachable-20260519.txt` and `/tmp/awdl-webrtc-dry-run-test.txt`. |
| Remote matrix | PASS: `WEBRTC_TRACE=0 WEBRTC_ATTEMPTS=3 WEBRTC_RETRY_DELAY=3 REMOTE_READY_TIMEOUT=30 REMOTE_STEP_READY_TIMEOUT=30 CANDIDATE_POLICY=auto REQUIRE_PATHS=1 DURATION=5s TRIALS=3 WINDOW=8 STREAMS=2 TIMEOUT=90s LISTEN_IDLE_TIMEOUT=3s SSH_TARGET=tmc2@10.0.18.249 REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin-workspace OUTPUT=/tmp/awdl-webrtc-matrix-workspace2-20260519.txt SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-workspace2-20260519.md scripts/remote-matrix-bundle.sh` passed all LAN, Thunderbolt, and AWDL probes. Pion-native WebRTC opened datachannels on all three profiles, and UDP perf, latency, and callback probes passed with Network.framework path evidence. |

## Latest Successful Retry Evidence

Command:

```sh
WEBRTC_TRACE=0 WEBRTC_ATTEMPTS=3 WEBRTC_RETRY_DELAY=3 \
  REMOTE_READY_TIMEOUT=30 REMOTE_STEP_READY_TIMEOUT=30 \
  CANDIDATE_POLICY=auto REQUIRE_PATHS=1 \
  DURATION=5s TRIALS=3 WINDOW=8 STREAMS=2 TIMEOUT=90s \
  LISTEN_IDLE_TIMEOUT=3s \
  SSH_TARGET=tmc2@10.0.18.249 \
  REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin-workspace \
  OUTPUT=/tmp/awdl-webrtc-matrix-workspace2-20260519.txt \
  SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-workspace2-20260519.md \
  scripts/remote-matrix-bundle.sh
```

Result:

The full table is in [matrix-workspace-20260519.md](matrix-workspace-20260519.md).
The bundle passed `profiles=lan thunderbolt awdl` from the `go.work` workspace.
Thunderbolt and LAN use IPv4 candidates; AWDL uses explicit `fe80::...`
candidates while synthetic `fd00::1` SDP candidates are stripped from the wire
signal.

## Prompt-to-Artifact Checklist

This checklist maps the original requests to concrete files, commands, and
current evidence. A passing local gate is not counted as proof for requirements
that need two Macs.

| Request | Artifact / command | Current evidence | Gap |
| --- | --- | --- | --- |
| Demonstrate WebRTC support over Apple links | `main.go` modes `gather`, `pair`, `answer-stdio`, `offer-stdio`, and `offer-ssh`; Pion WebRTC in `newPeer`; [RESULTS.md](../RESULTS.md). | LAN, Thunderbolt, and AWDL Pion-native WebRTC passed in the current two-host matrix. | None for the selected-link UDP WebRTC proof. |
| Keep backends pluggable | `-backend go\|network`; `-pion-net` switch from `SetICEUDPMux` to `SettingEngine.SetNet`; `github.com/tmc/apple-pion/nwtransport`; checked-in `go.work`. | Workspace `go test ./...` passes; LAN, Thunderbolt, and AWDL `-pion-net` WebRTC pass in the current matrix; `../apple-pion` tests assert native numeric UDP ownership and fallback for named UDP addresses. | Broader all-Network.framework TCP/TURN/STUN ownership remains outside this demo. |
| Prefer `go.work` over new tags | `go.work`; local sibling `../apple` and `../apple-pion`; no new tags. | Current matrix binary was built from the workspace, and no tags were created after the direction change. | Publish later only when these local changes are ready. |
| Prove UDP over AWDL | `udp`, `udp-perf`, `udp-latency`, `udp-callback-*`; [matrix-workspace-20260519.md](matrix-workspace-20260519.md). | Current matrix shows AWDL raw Network.framework UDP perf, latency, and callback probes pass both directions over `awdl0`. | Longer sweeps remain useful for product throughput claims. |
| Cover Thunderbolt, AWDL, and LAN tests | `scripts/remote-matrix.sh`; `scripts/remote-matrix-bundle.sh`; path requirements. | Current matrix covers all three profiles and passes WebRTC, UDP perf, latency, and callback probes with path evidence. | None for the short-run proof. |
| Add iperf-like output and result tables | UDP summary printers in `main.go`; `cmd/matrix-summary`; [RESULTS.md](../RESULTS.md). | Output includes transfer, bitrate, datagrams, loss, RTT, JSON, path records, summaries, and Markdown tables. | Longer comparative runs still need live peer access. |
| Run longer soak sweeps | [../scripts/remote-soak.sh](../scripts/remote-soak.sh); [soak-sweeps.md](soak-sweeps.md); `cmd/matrix-summary`. | The soak wrapper sets 30s x 5-trial defaults, path requirements, WebRTC retries, timestamped transcript/summary outputs, and fail-closed remote readiness. A short remote-readiness exercise wrote `/tmp/awdl-webrtc-dry-run-test.{txt,md}` and stopped at the unreachable peer. | Full LAN/Thunderbolt/AWDL long-soak proof still needs live peer access. |
| Package reusable pieces out of the temporary demo | `github.com/tmc/apple/x/network/nwpacket`; `github.com/tmc/apple-pion/nwtransport`; `github.com/tmc/apple-pion/icepolicy`. | Reusable packages have tests/examples and are consumed by the demo through the checked-in workspace. | None for the package split. |
| Add SwiftUI link-health view with fallback | `ui.go`; `ui_test.go`; [ui-two-host.md](ui-two-host.md); `-mode ui`; `-mode discover`; `-mode discover-wait`; README UI and discovery commands. | UI advertises Bonjour TXT metadata, samples Thunderbolt, AWDL, then LAN, and falls back when a higher-priority path returns no replies. `TestLinkHealthSamplePreferredFallsBackToAWDL` and `TestLinkHealthSamplePreferredSkipsUnavailableThunderbolt` cover the fallback rule. Headless discovery emitted JSON records with local Thunderbolt, AWDL, and LAN listener addresses; `discover-wait` emitted one matching peer record when a local publisher was present. TXT metadata includes advertised version, commit, and supported modes when available. | It measures UDP link health, not WebRTC datachannel health. A two-live-Mac UI session is still required for visual proof and cable-removal observation. |
| Avoid SSH as the only control plane | `offer-stdio`, `answer-stdio`, `offer-bonjour`, `answer-bonjour`, [../scripts/remote-bonjour.sh](../scripts/remote-bonjour.sh), and README manual/Bonjour workflows. | Parser tests pass; local stdio smoke exchanged `OFFER` and `ANSWER` lines. Bonjour signaling advertises `_awdl-webrtc-signal._tcp`, falls back from Bonjour endpoint dialing to `NSNetService` host/port resolution, and `-signal-only` exchanged local `OFFER`/`ANSWER` lines with exit 0 on both sides. `remote-bonjour.sh` now launches a two-host signal/full Bonjour proof while using SSH only for process control. | Same-host stdio and full Bonjour datachannels did not open; both reached ICE checking with no same-host candidate-pair responses. The current peer was unreachable on 2026-05-19, so the two-host Bonjour proof remains blocked on peer availability. |
| Keep repo hygiene and atomic history | Git commits and notes; `git status`; workspace gates. | Demo changes are split into atomic code commits with docs layered separately; `apple-pion` is clean; unrelated untracked files in `tmc/apple` are preserved. | Push/publish policy is separate from this local workspace proof. |

## Requirement Checklist

| Requirement | Artifact / evidence | Status | Remaining proof |
| --- | --- | --- | --- |
| Full Pion-native backend | `github.com/tmc/apple-pion/nwtransport` implements Pion `transport.Net`; demo uses `-pion-net`; workspace `../apple-pion` filters selected interfaces to the configured address, leaves named UDP endpoints to the fallback `transport.Net`, and uses Pion address rewrite rules for link-local host publication. | PASS for the selected UDP surface | LAN, Thunderbolt, and AWDL Pion-native WebRTC pass in the current matrix. Broader TCP/TURN/STUN Network.framework ownership is explicitly out of this demo slice. |
| Durable AWDL/link-local candidates | `github.com/tmc/apple-pion/icepolicy`; demo publishes explicit `ICECandidateInit` records and strips synthetic SDP candidate lines for explicit link-local signaling; `-candidate-policy auto\|mdns\|raw`; tests cover raw candidate extraction and auto-policy selection; the current matrix used auto policy. | PASS | Candidate publication is durable enough for the current AWDL Pion-native datachannel proof. |
| Direction/asymmetry cleanup | `udp-callback-listen`, `udp-callback-request`, path policy checks, `nwpacket` readiness retries, `remote-diagnostics.sh`, matrix-local reachability diagnostics, per-listener UDP socket/route diagnostics, and a simultaneous bidirectional UDP matrix step are implemented. | PASS | Current LAN/Thunderbolt/AWDL sequential raw UDP perf, latency, and callback probes pass both directions with path evidence. Remaining work is performance tuning, not basic reachability. |
| Performance hardening | `udp-perf` supports `-duration`, `-trials`, `-window`, `-streams`, `-packet-timeout`, `-listen-idle-timeout`, loss columns, JSON, latency mode, stale-reply handling, bidirectional matrix probing, Network.framework path records, and `cmd/matrix-summary` Markdown summaries. | PASS for this proof | Current 5s x 3-trial x 2-stream matrix is in [matrix-workspace-20260519.md](matrix-workspace-20260519.md). Run longer sweeps before making stable product throughput claims. |
| Reusable package split | `github.com/tmc/apple/x/network/nwpacket`; `github.com/tmc/apple-pion/nwtransport`; `github.com/tmc/apple-pion/icepolicy`; demo has checked-in `go.work`. | PASS | None for the package split. |
| Repo hygiene | Demo changes are split into atomic commits before docs; owned `tmc/apple/x/network/nwpacket` is committed locally; unrelated untracked files in `tmc/apple` are preserved. | PASS | Push/publish policy is separate from this local workspace proof. |

## Required Remote Commands

Use these for follow-up runs:

```sh
SSH_TARGET=tmc2@10.0.18.249 \
  THUNDERBOLT_PEER=169.254.88.35 \
  AWDL_PEER='fe80::9477:6dff:fe11:6a55%awdl0' \
  OUTPUT=/tmp/awdl-webrtc-diagnostics.txt \
  scripts/remote-diagnostics.sh

WEBRTC_ATTEMPTS=3 \
WEBRTC_RETRY_DELAY=3 \
CANDIDATE_POLICY=auto \
REQUIRE_PATHS=1 \
DURATION=5s \
TRIALS=3 \
WINDOW=8 \
STREAMS=2 \
TIMEOUT=90s \
LISTEN_IDLE_TIMEOUT=3s \
REMOTE_READY_TIMEOUT=30 \
REMOTE_STEP_READY_TIMEOUT=30 \
SSH_TARGET=tmc2@10.0.18.249 \
REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin-workspace \
OUTPUT=/tmp/awdl-webrtc-matrix.txt \
SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-summary.md \
scripts/remote-matrix-bundle.sh

REMOTE_READY_TIMEOUT=30 \
PROFILES="lan thunderbolt awdl" \
PHASES="signal full" \
SSH_TARGET=tmc2@10.0.18.249 \
OUTPUT=/tmp/awdl-webrtc-bonjour.txt \
scripts/remote-bonjour.sh

SOAK_LABEL=workspace-soak \
SSH_TARGET=tmc2@10.0.18.249 \
scripts/remote-soak.sh

DURATION=10s TRIALS=5 WINDOW=8 STREAMS=2 CANDIDATE_POLICY=auto REQUIRE_PATHS=1 \
REMOTE_READY_TIMEOUT=30 REMOTE_STEP_READY_TIMEOUT=30 \
SSH_TARGET=tmc2@10.0.18.249 \
REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin-workspace \
OUTPUT=/tmp/awdl-webrtc-matrix-long.txt \
SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-long-summary.md \
scripts/remote-matrix-bundle.sh
```

The core goal items are complete for the selected-link UDP/WebRTC proof. The
remaining product work is outside this completion boundary: two-host Bonjour
signaling, two-live-Mac SwiftUI observation, and longer soak/performance sweeps.
