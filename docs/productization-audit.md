# Productization Audit

This checklist maps the productization request to concrete artifacts and current
evidence.

| Requirement | Status | Evidence | Remaining work |
| --- | --- | --- | --- |
| Full Pion-native backend closer to `transport.Net` | PARTIAL | `github.com/tmc/apple-pion/nwtransport` provides the Pion `transport.Net` UDP surface; `v0.1.1` reports only the configured selected-interface address so Pion does not gather unrelated `en0` addresses. Current two-host matrix: LAN and Thunderbolt `-pion-net -mdns disabled -candidate-policy auto` WebRTC pass. | AWDL `-pion-net` WebRTC timed out in the current matrix, despite raw AWDL UDP passing. A broader all-Network.framework TCP/TURN/STUN backend remains a separate package design. |
| Durable AWDL/link-local candidate handling | PARTIAL | `github.com/tmc/apple-pion/icepolicy` owns the AWDL raw-host-candidate policy, with package tests and examples. The demo keeps SDP unmodified, publishes explicit `ICECandidateInit` records with the selected link-local IP, and exposes `-candidate-policy auto\|mdns\|raw`; `auto` enables raw publication only for disabled-mDNS AWDL/link-local cases. `TestNewCandidatePolicyConfig` covers the selector, and AWDL `-mdns disabled` gather passes without the legacy `-raw-candidates` flag. `-webrtc-trace` now reports SDP candidates separately from explicit `ICECandidateInit` records, and timeout snapshots include Pion candidate-pair stats. The current matrix uses auto policy. | Candidate publication is no longer the obvious blocker; the next two-host AWDL run should use the trace to isolate candidate installation, connectivity checks, DTLS/SCTP, or Network.framework reads. |
| Direction/asymmetry cleanup | PASS | The `v0.1.1` matrix proves sequential Network.framework UDP perf, latency, and callback probes in both directions on LAN, Thunderbolt, and AWDL with path evidence. `scripts/remote-matrix.sh` records route, ping, TCP/22, `scutil --nwi`, listener-side route, `lsof`, `netstat`, sender-side route checks, and simultaneous bidirectional perf. `scripts/remote-diagnostics.sh` now runs the remote body under Bash so list splitting is correct on macOS zsh accounts. | Bidirectional pressure still shows occasional 5% loss; continue tuning under longer runs. |
| Performance hardening | PARTIAL | `udp-perf` and `udp-perf-send` support fixed-count runs, fixed-duration runs with `-duration`, `-trials`, aggregate trial summaries, bounded in-flight `-window`, and concurrent `-streams`; `udp-latency` and `udp-latency-send` add explicit ping-pong latency output and JSON; output includes `Lost` and `Loss`; `-perf-json` emits per-trial and aggregate machine-readable result records including `window`, `streams`, `duration_ns`, and Network.framework `paths` when available; `-packet-timeout` counts echo timeouts as lost datagrams; `-listen-idle-timeout` lets duration-mode listeners stop after traffic goes idle, and `remote-matrix.sh` forwards `LISTEN_IDLE_TIMEOUT`; the matrix starts listeners on both hosts and runs both perf senders concurrently for a bidirectional pressure sample; `cmd/matrix-summary` renders saved transcripts into compact Markdown tables including de-duplicated `FAIL:` rows; `docs/matrix-v0.1.1.md` records the current short-run matrix. | Run longer repeated remote trials and tune window/backpressure behavior after the AWDL WebRTC path is instrumented. |
| Reusable package | PASS | `tmc/apple` release `v0.6.7` provides `github.com/tmc/apple/x/network/nwpacket` with `PathReporter` and outbound readiness retry knobs; `tmc/apple-pion` release `v0.1.1` provides `github.com/tmc/apple-pion/nwtransport` and `github.com/tmc/apple-pion/icepolicy`; the demo resolves both from the module cache with no local replace. | None for the package split. |
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
CANDIDATE_POLICY=auto \
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
and current two-host direction checks. LAN and Thunderbolt Pion-native WebRTC
now pass on the published backend, and raw AWDL UDP passes in both directions.
The remaining productization boundary is AWDL Pion-native WebRTC reliability:
the current matrix times out waiting for the datachannel even though candidates
and raw UDP path evidence are correct.
