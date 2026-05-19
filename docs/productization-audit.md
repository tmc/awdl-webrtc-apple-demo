# Productization Audit

This checklist maps the productization request to concrete artifacts and current
evidence.

| Requirement | Status | Evidence | Remaining work |
| --- | --- | --- | --- |
| Full Pion-native backend closer to `transport.Net` | PASS for the selected UDP surface | `github.com/tmc/apple-pion/nwtransport` provides the Pion `transport.Net` UDP surface. The demo now uses `go.work` to build against local sibling checkouts while avoiding new tags. Current two-host workspace matrix: LAN, Thunderbolt, and AWDL `-pion-net -mdns disabled -candidate-policy auto` WebRTC pass. `../apple-pion` now tests that native Network.framework ownership is limited to numeric UDP endpoints and configured wildcard listeners, while named UDP endpoints fall back without native DNS resolution. | A broader all-Network.framework DNS/TCP/TURN/STUN backend remains a separate package design. |
| Durable AWDL/link-local candidate handling | PASS | `github.com/tmc/apple-pion/icepolicy` owns the AWDL raw-host-candidate policy, with package tests and examples. The demo publishes explicit `ICECandidateInit` records with the selected link-local IP and strips synthetic SDP candidate lines from the signaled SDP so Pion does not try `fd00::1`; `-candidate-policy auto` enables this only for disabled-mDNS AWDL/link-local cases. Current AWDL `-pion-net` matrix leg opens the datachannel. | mDNS candidate exchange remains a separate boundary; use disabled mDNS plus `auto` for link-local AWDL proof. |
| Direction/asymmetry cleanup | PASS | The current workspace matrix and live soak prove sequential and bidirectional Network.framework UDP perf, latency, and callback probes in both directions on LAN, Thunderbolt, and AWDL with path evidence. `scripts/remote-matrix.sh` records route, ping, TCP/22, `scutil --nwi`, listener-side route, `lsof`, `netstat`, sender-side route checks, simultaneous bidirectional perf, stale-process cleanup, WebRTC retry attempts, and optional discovery-sourced peer addresses through `USE_DISCOVERY=1` or `DISCOVERY_FILE`. | Remaining work is tuning and UI polish, not basic direction/asymmetry. |
| Non-SSH control plane | PASS | `answer-stdio`/`offer-stdio` keep the WebRTC wire signal independent of SSH. `answer-bonjour`/`offer-bonjour` add a Network.framework Bonjour service `_awdl-webrtc-signal._tcp` carrying the same `OFFER`/`ANSWER` payload, with version/commit/mode TXT metadata. The two-host Bonjour harness passed LAN, Thunderbolt, and AWDL in both `signal` and `full` phases; see [bonjour-live-20260519.md](bonjour-live-20260519.md). SSH was used only to install/start the peer process. | None for the two-host Bonjour proof. |
| SwiftUI/link-health UI | PARTIAL | `-mode ui` renders the SwiftUI link monitor; `-mode discover` and `-mode discover-wait` expose the same Bonjour/TXT discovery state as JSON; `scripts/run-ui-harness.sh` builds the workspace binary, installs it on the remote Mac, starts the remote headless publisher, runs local discovery preflight, and opens the local UI. `LAUNCH_UI=0` passed as a non-visual harness smoke from clean commit `d7da303`, matching the current remote publisher and seeing LAN, Thunderbolt, and AWDL ready. `TestLinkHealthSamplePreferredFallsBackToAWDL` and `TestLinkHealthSamplePreferredSkipsUnavailableThunderbolt` cover Thunderbolt-to-AWDL fallback selection. Two-host headless discovery and the discovery-fed matrix found live LAN, Thunderbolt, and AWDL peer addresses with version/commit/modes metadata; see [discovery-matrix-20260519.md](discovery-matrix-20260519.md). A local visible UI run with a remote headless publisher showed readable rows, Thunderbolt selected as active path, all three possible paths, and updating bandwidth history after the text-layout fix. | A two-live-Mac visual UI session and physical Thunderbolt removal observation are still required; use [ui-two-host.md](ui-two-host.md) for the harness and manual validation steps. |
| Performance hardening | PASS for this proof | `udp-perf` and `udp-perf-send` support fixed-count runs, fixed-duration runs with `-duration`, `-trials`, aggregate trial summaries, bounded in-flight `-window`, and concurrent `-streams`; `udp-latency` and `udp-latency-send` add explicit ping-pong latency output and JSON; stale late replies are ignored until the per-packet deadline instead of aborting; output includes `Lost` and `Loss`; `-perf-json` emits per-trial and aggregate machine-readable result records including `window`, `streams`, `duration_ns`, and Network.framework `paths` when available; `cmd/matrix-summary` renders saved transcripts into compact Markdown tables. [soak-live-20260519.md](soak-live-20260519.md) records the 30s x 5-trial x 2-stream live soak. | The long discovery-fed probe exposed a publisher lifetime bug; the script default is fixed and a LAN-only 70s discovery-fed smoke passed. |
| Reusable package | PASS | `../apple` provides `github.com/tmc/apple/x/network/nwpacket` with `PathReporter`, outbound readiness retry knobs, and AWDL-scoped listener-source binding; `../apple-pion` provides `github.com/tmc/apple-pion/nwtransport` and `github.com/tmc/apple-pion/icepolicy`; the demo consumes both through `go.work`. The latest `apple-pion` backend-boundary commits are pushed without publishing tags. | Publish a later release only when the local workspace changes are ready; no new tag was created for this proof. |
| Workspace module graph | PASS | `go.work` includes `.`, `../apple`, and `../apple-pion`; local gates passed in all three modules before the workspace matrix. | Release preflight remains a separate published-module gate. |
| Repo hygiene | PASS | Demo changes are split into atomic code commits with docs layered separately; the untracked `awdl-webrtc-apple-demo` binary was removed; unrelated untracked files in `../apple` are preserved. | None for owned files. |

## Verification Commands

```sh
go test ./...
go vet ./...
bash -n scripts/remote-matrix.sh
bash -n scripts/remote-matrix-bundle.sh
bash -n scripts/remote-diagnostics.sh
bash -n scripts/remote-bonjour.sh
bash -n scripts/remote-soak.sh
bash -n scripts/release-preflight.sh
go test ./cmd/matrix-summary
go run . -profile lan -backend network -mode gather -timeout 8s
go run . -profile thunderbolt -backend network -mode gather -timeout 5s
go run . -profile awdl -backend network -mdns disabled -mode gather -timeout 12s
go run . -profile awdl -backend network -pion-net -mdns disabled -mode gather -timeout 15s
go run . -profile lan -backend network -mode udp-perf -count 2 -warmup 0 -trials 2 -perf-json -timeout 15s
go run . -profile lan -backend network -mode udp-perf -count 4 -warmup 0 -window 2 -perf-json -timeout 10s
go run . -profile awdl -backend network -mode udp-perf-send -peer '[fe80::peer%awdl0]:12345' -count 20 -warmup 0 -perf-json -require-path-interface awdl0 -forbid-loopback-path -timeout 30s
go run . -profile lan -iface lo0 -backend go -mode udp-perf -count 2 -warmup 0 -trials 1 -streams 2 -window 1 -packet-timeout 100ms -perf-json -timeout 5s
go run . -profile lan -iface lo0 -backend go -mode udp-latency -count 3 -warmup 0 -streams 2 -size 64 -packet-timeout 100ms -perf-json -timeout 5s
# callback smoke: run udp-callback-listen on lo0, then udp-callback-request to the printed address
go run . -profile thunderbolt -backend go -mode udp-perf -duration 20ms -warmup 0 -window 1 -packet-timeout 20ms -perf-json -timeout 5s
go run . -profile lan -backend go -mode udp-perf-send -peer 10.0.199.147:9 -count 2 -warmup 0 -packet-timeout 100ms -perf-json -timeout 5s
go run . -profile awdl -backend network -pion-net -mdns disabled -candidate-policy auto -mode answer-bonjour -signal-name awdl-webrtc-peer-a -signal-only -timeout 90s
go run . -profile awdl -backend network -pion-net -mdns disabled -candidate-policy auto -mode offer-bonjour -signal-peer awdl-webrtc-peer-a -signal-only -timeout 90s
REMOTE_READY_TIMEOUT=30 PROFILES="lan thunderbolt awdl" PHASES="signal full" \
SSH_TARGET=tmc2@10.0.18.249 OUTPUT=/tmp/awdl-webrtc-bonjour.txt \
scripts/remote-bonjour.sh
SOAK_LABEL=workspace-soak SSH_TARGET=tmc2@10.0.18.249 scripts/remote-soak.sh
CANDIDATE_POLICY=auto \
REQUIRE_PATHS=1 \
WEBRTC_TRACE=0 \
WEBRTC_ATTEMPTS=3 \
WEBRTC_RETRY_DELAY=3 \
REMOTE_READY_TIMEOUT=30 \
REMOTE_STEP_READY_TIMEOUT=30 \
DURATION=5s \
TRIALS=3 \
WINDOW=8 \
STREAMS=2 \
TIMEOUT=90s \
LISTEN_IDLE_TIMEOUT=3s \
SSH_TARGET=tmc2@10.0.18.249 \
REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin-workspace \
OUTPUT=/tmp/awdl-webrtc-matrix.txt \
SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-summary.md \
scripts/remote-matrix-bundle.sh
go run . -profile lan -backend network -pion-net -mdns disabled -candidate-policy auto -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 35s
go run . -profile thunderbolt -backend network -pion-net -mdns disabled -candidate-policy auto -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 40s
go run . -profile awdl -backend network -pion-net -mdns disabled -candidate-policy auto -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 45s
SSH_TARGET=tmc2@10.0.18.249 OUTPUT=/tmp/awdl-webrtc-diagnostics.txt scripts/remote-diagnostics.sh
```

`scripts/release-preflight.sh` remains the published-module release gate. It is
not part of the current `go.work` proof path.

## Stop Condition

This pass productizes the proof enough to expose reusable package boundaries,
better measurement/candidate-policy surfaces, a Pion `transport.Net` adapter,
two-host Bonjour signaling, long selected-link soak data, and a readable local
SwiftUI active-path view. LAN, Thunderbolt, and AWDL Pion-native WebRTC now pass
from the local `go.work` workspace, and raw UDP passes in both directions with
path evidence. The remaining product work is the two-live-Mac SwiftUI visual
observation and physical Thunderbolt-removal fallback proof; the smoke-tested
harness now reduces that proof to a repeatable launch plus human cable removal.
