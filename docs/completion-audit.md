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
| Remote reachability | PASS for the current proof set: SSH process control stayed available during the workspace matrix, two-host Bonjour proof, discovery-fed matrix, long soak, and discovery-lifetime smoke. Earlier SSH-control-plane failures remain recorded in [matrix-try-again-20260519.md](matrix-try-again-20260519.md) as historical setup diagnostics. |
| Remote matrix | PASS: `WEBRTC_TRACE=0 WEBRTC_ATTEMPTS=3 WEBRTC_RETRY_DELAY=3 REMOTE_READY_TIMEOUT=30 REMOTE_STEP_READY_TIMEOUT=30 CANDIDATE_POLICY=auto REQUIRE_PATHS=1 DURATION=5s TRIALS=3 WINDOW=8 STREAMS=2 TIMEOUT=90s LISTEN_IDLE_TIMEOUT=3s SSH_TARGET=tmc2@10.0.18.249 REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin-workspace OUTPUT=/tmp/awdl-webrtc-matrix-workspace2-20260519.txt SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-workspace2-20260519.md scripts/remote-matrix-bundle.sh` passed all LAN, Thunderbolt, and AWDL probes. Pion-native WebRTC opened datachannels on all three profiles, and UDP perf, latency, and callback probes passed with Network.framework path evidence. |
| Bonjour control plane | PASS: `REMOTE_READY_TIMEOUT=30 REMOTE_STEP_READY_TIMEOUT=30 PROFILES="lan thunderbolt awdl" PHASES="signal full" SSH_TARGET=tmc2@10.0.18.249 OUTPUT=/tmp/awdl-webrtc-bonjour-live2-20260519.txt scripts/remote-bonjour.sh` passed LAN, Thunderbolt, and AWDL `signal` and `full` phases. See [bonjour-live-20260519.md](bonjour-live-20260519.md). |
| Discovery-fed matrix | PASS: `USE_DISCOVERY=1 ... OUTPUT=/tmp/awdl-webrtc-discovery-matrix-20260519.txt SUMMARY_OUTPUT=/tmp/awdl-webrtc-discovery-matrix-20260519.md scripts/remote-matrix-bundle.sh` discovered live LAN, Thunderbolt, and AWDL peer addresses and passed WebRTC on all three profiles. See [discovery-matrix-20260519.md](discovery-matrix-20260519.md). |
| Long selected-link soak | PASS: `USE_DISCOVERY=1 SOAK_LABEL=workspace-soak-live ... scripts/remote-soak.sh` passed LAN, Thunderbolt, and AWDL selected-link WebRTC, perf, bidirectional perf, latency, and callback sweeps. See [soak-live-20260519.md](soak-live-20260519.md). |
| Long discovery-fed probe | FIXED after failure: the long soak exposed that runtime discovery publishers expired after the old 60s default. `scripts/remote-matrix.sh` now defaults `DISCOVERY_PUBLISH_TIMEOUT=2h`, and a LAN-only 70s discovery-fed smoke passed with `matrix_exit=0`. |

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
signal. Follow-up proof artifacts are [bonjour-live-20260519.md](bonjour-live-20260519.md),
[discovery-matrix-20260519.md](discovery-matrix-20260519.md), and
[soak-live-20260519.md](soak-live-20260519.md).

## Prompt-to-Artifact Checklist

This checklist maps the original requests to concrete files, commands, and
current evidence. A passing local gate is not counted as proof for requirements
that need two Macs.

| Request | Artifact / command | Current evidence | Gap |
| --- | --- | --- | --- |
| Demonstrate WebRTC support over Apple links | `main.go` modes `gather`, `pair`, `answer-stdio`, `offer-stdio`, and `offer-ssh`; Pion WebRTC in `newPeer`; [RESULTS.md](../RESULTS.md). | LAN, Thunderbolt, and AWDL Pion-native WebRTC passed in the current two-host matrix. | None for the selected-link UDP WebRTC proof. |
| Keep backends pluggable | `-backend go\|network`; `-pion-net` switch from `SetICEUDPMux` to `SettingEngine.SetNet`; `github.com/tmc/apple-pion/nwtransport`; checked-in `go.work`. | Workspace `go test ./...` passes; LAN, Thunderbolt, and AWDL `-pion-net` WebRTC pass in the current matrix; `../apple-pion` tests assert native numeric UDP ownership and fallback for named UDP addresses. | Broader all-Network.framework TCP/TURN/STUN ownership remains outside this demo. |
| Prefer `go.work` over new tags | `go.work`; local sibling `../apple` and `../apple-pion`; no new tags. | Current matrix binary was built from the workspace, and no tags were created after the direction change. | Publish later only when these local changes are ready. |
| Prove UDP over AWDL | `udp`, `udp-perf`, `udp-latency`, `udp-callback-*`; [matrix-workspace-20260519.md](matrix-workspace-20260519.md); [soak-live-20260519.md](soak-live-20260519.md). | Current matrix shows AWDL raw Network.framework UDP perf, latency, and callback probes pass both directions over `awdl0`. The long soak shows AWDL at 28.30 Mbps remote-to-local, 28.31 Mbps local-to-remote, 26.48/26.77 Mbps bidirectional, and 0.00% loss in those long rows. | None for selected-link AWDL UDP proof. |
| Cover Thunderbolt, AWDL, and LAN tests | `scripts/remote-matrix.sh`; `scripts/remote-matrix-bundle.sh`; path requirements; optional `USE_DISCOVERY=1` or `DISCOVERY_FILE` peer addresses. | Current matrix, discovery-fed matrix, Bonjour proof, and long selected-link soak all cover LAN, Thunderbolt, and AWDL with path evidence. | None for headless two-host link proof. |
| Add iperf-like output and result tables | UDP summary printers in `main.go`; `cmd/matrix-summary`; [RESULTS.md](../RESULTS.md); [soak-live-20260519.md](soak-live-20260519.md). | Output includes transfer, bitrate, datagrams, loss, RTT, JSON, path records, summaries, and Markdown tables. The live soak doc contains compact throughput and latency tables. | None for the requested at-a-glance result tables. |
| Run longer soak sweeps | [../scripts/remote-soak.sh](../scripts/remote-soak.sh); [soak-sweeps.md](soak-sweeps.md); `cmd/matrix-summary`; [soak-live-20260519.md](soak-live-20260519.md). | The soak wrapper ran with 30s x 5-trial x 2-stream defaults and passed selected-link LAN, Thunderbolt, and AWDL WebRTC, perf, bidirectional perf, latency, and callback sweeps. | The discovery-fed long probe failed in the first run because of the old 60s publisher timeout; the default is now 2h and a 70s LAN discovery-fed smoke passed. |
| Package reusable pieces out of the temporary demo | `github.com/tmc/apple/x/network/nwpacket`; `github.com/tmc/apple-pion/nwtransport`; `github.com/tmc/apple-pion/icepolicy`. | Reusable packages have tests/examples and are consumed by the demo through the checked-in workspace. | None for the package split. |
| Add SwiftUI link-health view with fallback | `ui.go`; `ui_test.go`; [ui-two-host.md](ui-two-host.md); `-mode ui`; `-mode discover`; `-mode discover-wait`; README UI and discovery commands; [discovery-matrix-20260519.md](discovery-matrix-20260519.md). | UI advertises Bonjour TXT metadata, samples Thunderbolt, AWDL, then LAN, and falls back when a higher-priority path returns no replies. `TestLinkHealthSamplePreferredFallsBackToAWDL` and `TestLinkHealthSamplePreferredSkipsUnavailableThunderbolt` cover the fallback rule. Live headless discovery fed LAN, Thunderbolt, and AWDL peer addresses into the two-host matrix. | It measures UDP link health, not WebRTC datachannel health. A two-live-Mac UI session is still required for visual proof and physical Thunderbolt-removal observation. |
| Avoid SSH as the only control plane | `offer-stdio`, `answer-stdio`, `offer-bonjour`, `answer-bonjour`, [../scripts/remote-bonjour.sh](../scripts/remote-bonjour.sh), and README manual/Bonjour workflows. | Parser tests pass; local stdio smoke exchanged `OFFER` and `ANSWER` lines. Bonjour signaling advertises `_awdl-webrtc-signal._tcp`, falls back from Bonjour endpoint dialing to `NSNetService` host/port resolution, and [bonjour-live-20260519.md](bonjour-live-20260519.md) records two-host LAN, Thunderbolt, and AWDL `signal` and `full` phases passing while SSH was used only for process control. | None for the two-host Bonjour WebRTC control-plane proof. |
| Use NotebookLM-assisted self-prompting | `/Users/tmc/go/src/github.com/tmc/skills/skills/notebooklm-assisted-self-prompting/SKILL.md`; `nlm source sync`; `nlm generate-chat`. | The repo state was synced to notebook `awdl-webrtc-apple-demo productization self-prompting 2026-05-19`, and the drift check confirmed the current plan while flagging stale blocked-doc claims to correct before commit. | Continue using the anchor before any further broad design changes. |
| Keep repo hygiene and atomic history | Git commits and notes; `git status`; workspace gates. | Demo changes are split into atomic code commits with docs layered separately; `apple-pion` is clean; unrelated untracked files in `tmc/apple` are preserved. | Push/publish policy is separate from this local workspace proof. |

## Requirement Checklist

| Requirement | Artifact / evidence | Status | Remaining proof |
| --- | --- | --- | --- |
| Full Pion-native backend | `github.com/tmc/apple-pion/nwtransport` implements Pion `transport.Net`; demo uses `-pion-net`; workspace `../apple-pion` filters selected interfaces to the configured address, leaves named UDP endpoints to the fallback `transport.Net`, and uses Pion address rewrite rules for link-local host publication. | PASS for the selected UDP surface | LAN, Thunderbolt, and AWDL Pion-native WebRTC pass in the current matrix. Broader TCP/TURN/STUN Network.framework ownership is explicitly out of this demo slice. |
| Durable AWDL/link-local candidates | `github.com/tmc/apple-pion/icepolicy`; demo publishes explicit `ICECandidateInit` records and strips synthetic SDP candidate lines for explicit link-local signaling; `-candidate-policy auto\|mdns\|raw`; tests cover raw candidate extraction and auto-policy selection; the current matrix used auto policy. | PASS | Candidate publication is durable enough for the current AWDL Pion-native datachannel proof. |
| Direction/asymmetry cleanup | `udp-callback-listen`, `udp-callback-request`, path policy checks, `nwpacket` readiness retries, `remote-diagnostics.sh`, matrix-local reachability diagnostics, per-listener UDP socket/route diagnostics, optional discovery-fed peer addresses, and a simultaneous bidirectional UDP matrix step are implemented. | PASS | Current LAN/Thunderbolt/AWDL sequential raw UDP perf, latency, and callback probes pass both directions with path evidence. Remaining work is performance tuning, not basic reachability. |
| Performance hardening | `udp-perf` supports `-duration`, `-trials`, `-window`, `-streams`, `-packet-timeout`, `-listen-idle-timeout`, loss columns, JSON, latency mode, stale-reply handling, bidirectional matrix probing, Network.framework path records, and `cmd/matrix-summary` Markdown summaries. | PASS for this proof | Current 5s x 3-trial x 2-stream matrix is in [matrix-workspace-20260519.md](matrix-workspace-20260519.md), and the 30s x 5-trial x 2-stream selected-link soak is in [soak-live-20260519.md](soak-live-20260519.md). The long discovery-fed timeout bug is fixed with a 2h publisher default and 70s LAN smoke. |
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

The core goal items are complete for the selected-link UDP/WebRTC proof, the
two-host Bonjour control plane, live discovery-fed headless matrix, and long
selected-link soak. The remaining product proof is the two-live-Mac SwiftUI
visual observation with physical Thunderbolt-removal fallback.
