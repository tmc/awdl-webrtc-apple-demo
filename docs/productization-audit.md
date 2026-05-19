# Productization Audit

This checklist maps the productization request to concrete artifacts and current
evidence.

| Requirement | Status | Evidence | Remaining work |
| --- | --- | --- | --- |
| Full Pion-native backend closer to `transport.Net` | PASS for the selected UDP surface | `github.com/tmc/apple-pion/nwtransport` provides the Pion `transport.Net` UDP surface; `v0.1.1` reports only the configured selected-interface address so Pion does not gather unrelated `en0` addresses, and the demo now imports published `v0.1.3`. Current two-host matrix: LAN, Thunderbolt, and AWDL `-pion-net -mdns disabled -candidate-policy auto` WebRTC pass. | A broader all-Network.framework DNS/TCP/TURN/STUN backend remains a separate package design. |
| Durable AWDL/link-local candidate handling | PASS | `github.com/tmc/apple-pion/icepolicy` owns the AWDL raw-host-candidate policy, with package tests and examples. The demo keeps SDP unmodified, publishes explicit `ICECandidateInit` records with the selected link-local IP, and exposes `-candidate-policy auto\|mdns\|raw`; `auto` enables raw publication only for disabled-mDNS AWDL/link-local cases. `apple-pion v0.1.3` uses Pion address rewrite rules instead of the deprecated NAT1To1 shim. `TestNewCandidatePolicyConfig` covers the selector, AWDL `-mdns disabled` gather passes without the legacy `-raw-candidates` flag, and the current AWDL `-pion-net` matrix leg opens the datachannel. | mDNS candidate exchange remains a separate boundary; use disabled mDNS plus `auto` for link-local AWDL proof. |
| Direction/asymmetry cleanup | PASS | The current matrix proves sequential and bidirectional Network.framework UDP perf, latency, and callback probes in both directions on LAN, Thunderbolt, and AWDL with path evidence. `scripts/remote-matrix.sh` records route, ping, TCP/22, `scutil --nwi`, listener-side route, `lsof`, `netstat`, sender-side route checks, and simultaneous bidirectional perf. It can use `REMOTE_STEP_READY_TIMEOUT` to wait for SSH before each later remote sender/listener command and now cleans stale local/remote demo binary processes before and after a run unless `CLEAN_STALE_PROCESSES=0`. `scripts/remote-diagnostics.sh` runs the remote body under Bash so list splitting is correct on macOS zsh accounts. | One LAN local-to-remote aggregate saw 5% loss; continue tuning under longer runs. |
| Non-SSH control plane | PARTIAL | `answer-stdio`/`offer-stdio` keep the WebRTC wire signal independent of SSH. `answer-bonjour`/`offer-bonjour` add a Network.framework Bonjour service `_awdl-webrtc-signal._tcp` carrying the same `OFFER`/`ANSWER` payload, with version/commit/mode TXT metadata. The offer side falls back from Bonjour endpoint dialing to `NSNetService` host/port resolution plus Network.framework TCP, and local `-signal-only` signaling exchanged `OFFER`/`ANSWER` with exit 0 on both sides. | Same-host ICE still timed out after full signaling; two-host Bonjour signaling still needs a stable peer. |
| Performance hardening | PARTIAL | `udp-perf` and `udp-perf-send` support fixed-count runs, fixed-duration runs with `-duration`, `-trials`, aggregate trial summaries, bounded in-flight `-window`, and concurrent `-streams`; `udp-latency` and `udp-latency-send` add explicit ping-pong latency output and JSON; output includes `Lost` and `Loss`; `-perf-json` emits per-trial and aggregate machine-readable result records including `window`, `streams`, `duration_ns`, and Network.framework `paths` when available; `-packet-timeout` counts echo timeouts as lost datagrams; `-listen-idle-timeout` lets duration-mode listeners stop after traffic goes idle, and `remote-matrix.sh` forwards `LISTEN_IDLE_TIMEOUT`; the matrix starts listeners on both hosts and runs both perf senders concurrently for a bidirectional pressure sample; `cmd/matrix-summary` renders saved transcripts into compact Markdown tables including de-duplicated `FAIL:` rows and `webrtc_trace` candidate-pair summaries; `docs/matrix-20260519-success.md` records the current short-run matrix. | Run longer repeated remote trials and tune window/backpressure behavior before making stable throughput claims. |
| Reusable package | PASS | `tmc/apple` release `v0.6.7` provides `github.com/tmc/apple/x/network/nwpacket` with `PathReporter` and outbound readiness retry knobs; `tmc/apple-pion` release `v0.1.3` provides `github.com/tmc/apple-pion/nwtransport` and `github.com/tmc/apple-pion/icepolicy`; the demo resolves both from the module cache with no local replace. | None for the package split. |
| Published module graph | PASS | `scripts/release-preflight.sh` verifies local tests, vet, script syntax, published module pins for `tmc/apple` and `apple-pion`, clean owned worktrees, configured remotes, demo and `apple-pion` published HEADs, and absence of local replaces. It warns about unrelated sibling `tmc/apple` changes, fails if owned `x/network/nwpacket` is dirty, and can enforce the sibling Apple HEAD with `REQUIRE_APPLE_HEAD=1`. | This host's default DNS is currently flaky; set `GITHUB_RESOLVE_IP` for GitHub remote checks when needed. |
| Repo hygiene | PASS | The untracked `awdl-webrtc-apple-demo` binary was removed; `git status --short` is clean after commits. | None. |

## Verification Commands

```sh
go test ./...
go vet ./...
bash -n scripts/remote-matrix.sh
bash -n scripts/remote-matrix-bundle.sh
bash -n scripts/remote-diagnostics.sh
bash -n scripts/release-preflight.sh
scripts/release-preflight.sh
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
CANDIDATE_POLICY=auto \
WEBRTC_TRACE=1 \
REMOTE_READY_TIMEOUT=30 \
REMOTE_STEP_READY_TIMEOUT=30 \
SSH_TARGET=tmc2@10.0.18.249 \
REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin \
OUTPUT=/tmp/awdl-webrtc-matrix.txt \
SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-summary.md \
scripts/remote-matrix-bundle.sh
go run . -profile lan -backend network -pion-net -mdns disabled -candidate-policy auto -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 35s
go run . -profile thunderbolt -backend network -pion-net -mdns disabled -candidate-policy auto -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 40s
go run . -profile awdl -backend network -pion-net -mdns disabled -candidate-policy auto -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 45s
SSH_TARGET=tmc2@10.0.18.249 OUTPUT=/tmp/awdl-webrtc-diagnostics.txt scripts/remote-diagnostics.sh
```

## Stop Condition

This pass productizes the proof enough to expose reusable package boundaries,
better measurement/candidate-policy surfaces, a Pion `transport.Net` adapter,
and current two-host direction checks. LAN, Thunderbolt, and AWDL Pion-native
WebRTC now pass on the published backend, and raw UDP passes in both directions
with path evidence. The remaining productization work is two-host Bonjour
signaling, two-live-Mac SwiftUI observation, and longer performance sweeps.
