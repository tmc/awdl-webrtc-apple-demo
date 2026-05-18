# Productization Audit

This checklist maps the productization request to concrete artifacts and current
evidence.

| Requirement | Status | Evidence | Remaining work |
| --- | --- | --- | --- |
| Full Pion-native backend closer to `transport.Net` | PASS | `tmc/apple-pion` commits `d2698d5`, `167087c`, `3ea6faf`, and `7d4aced` add `github.com/tmc/apple-pion/nwtransport`; concrete UDP listeners, configured wildcard UDP listeners, and UDP dials route through Network.framework using `github.com/tmc/apple/x/network/nwpacket`; DNS, TCP, unconstrained wildcard UDP, TURN/STUN helper traffic outside the selected UDP surface, and unsupported UDP cases are documented and test-covered fallback paths; `-pion-net -mdns disabled -raw-candidates` remote WebRTC passes on LAN, Thunderbolt, and AWDL. | None for the demo backend; a broader all-Network.framework TCP/TURN/STUN backend would be a separate package design. |
| Durable AWDL/link-local candidate handling | PARTIAL | `github.com/tmc/apple-pion/icepolicy` now owns the AWDL raw-host-candidate policy, with package tests and examples. The demo strips candidates from SDP and publishes explicit `ICECandidateInit` records with the selected link-local IP. AWDL raw-candidate gather passes. A direct no-SDP-rewrite `SetNAT1To1IPs(fe80/fe80)` experiment gathered zero AWDL candidates. | Re-run two-host raw-candidate WebRTC with `tmc2@10.0.18.249` once SSH is reachable again. |
| Direction/asymmetry cleanup | PARTIAL | Remote-to-local Network.framework UDP listener tests pass on Thunderbolt and AWDL; LAN local-to-remote passes. Local route inspection showed `169.254.88.35` would route over `en0`, so the Go UDP backend now binds IPv4 link-local sockets to the selected interface. The Thunderbolt default selector now falls back from an addressless `bridge0` to a usable member interface, and local Network.framework gather passed on `en1`. `udp-callback-listen` and `udp-callback-request` add a request/callback probe that can identify whether the request reached the listener and whether the reverse callback returned. `nwpacket.PathReporter` plus `-require-path-interface` and `-forbid-loopback-path` make actual Network.framework path checks fail closed when a perf sender reaches readiness. `tmc/apple v0.6.7` adds outbound readiness timeout/retry knobs, the demo exposes `-nw-connect-timeout` and `-nw-connect-retries`, and the remote matrix forwards `NW_CONNECT_TIMEOUT` and `NW_CONNECT_RETRIES` to both hosts. `scripts/remote-diagnostics.sh` captures both hosts' interface, route, Network Extension, and UDP socket state for asymmetry triage. | Re-run two-host Thunderbolt/AWDL/LAN when `tmc2@10.0.18.249` is reachable again; current clean local-to-remote Network.framework remote-listener tests for Thunderbolt and AWDL still need deeper tracing. |
| Performance hardening | PARTIAL | `udp-perf` and `udp-perf-send` support fixed-count runs, fixed-duration runs with `-duration`, `-trials`, aggregate trial summaries, bounded in-flight `-window`, and concurrent `-streams`; `udp-latency` and `udp-latency-send` add explicit ping-pong latency output and JSON; output includes `Lost` and `Loss`; `-perf-json` emits per-trial and aggregate machine-readable result records including `window`, `streams`, `duration_ns`, and Network.framework `paths` when available; `-packet-timeout` counts echo timeouts as lost datagrams; local and two-host Network.framework smoke tests pass. | Run longer repeated remote trials and tune the window/backpressure behavior with the second host once reachable; full-duplex mode is still future work. |
| Reusable package | PASS | `tmc/apple` release `v0.6.7` provides `github.com/tmc/apple/x/network/nwpacket` with `PathReporter` and outbound readiness retry knobs; `tmc/apple-pion` adds `github.com/tmc/apple-pion/nwtransport` and `github.com/tmc/apple-pion/icepolicy`; this demo imports `apple-pion` through one remaining local replace. | Give `apple-pion` a remote/release and remove the remaining local replace. |
| Repo hygiene | PASS | The untracked `awdl-webrtc-apple-demo` binary was removed; `git status --short` is clean after commits. | None. |

## Verification Commands

```sh
go test ./...
go vet ./...
bash -n scripts/remote-matrix.sh
bash -n scripts/remote-diagnostics.sh
bash -n scripts/release-preflight.sh
scripts/release-preflight.sh
go run . -profile lan -backend network -mode gather -timeout 8s
go run . -profile thunderbolt -backend network -mode gather -timeout 5s
go run . -profile awdl -backend network -mdns disabled -raw-candidates -mode gather -timeout 10s
go run . -profile lan -backend network -mode udp-perf -count 2 -warmup 0 -trials 2 -perf-json -timeout 15s
go run . -profile lan -backend network -mode udp-perf -count 4 -warmup 0 -window 2 -perf-json -timeout 10s
go run . -profile awdl -backend network -mode udp-perf-send -peer '[fe80::peer%awdl0]:12345' -count 20 -warmup 0 -perf-json -require-path-interface awdl0 -forbid-loopback-path -timeout 30s
go run . -profile lan -iface lo0 -backend go -mode udp-perf -count 2 -warmup 0 -trials 1 -streams 2 -window 1 -packet-timeout 100ms -perf-json -timeout 5s
go run . -profile lan -iface lo0 -backend go -mode udp-latency -count 3 -warmup 0 -streams 2 -size 64 -packet-timeout 100ms -perf-json -timeout 5s
# callback smoke: run udp-callback-listen on lo0, then udp-callback-request to the printed address
go run . -profile thunderbolt -backend go -mode udp-perf -duration 20ms -warmup 0 -window 1 -packet-timeout 20ms -perf-json -timeout 5s
go run . -profile lan -backend go -mode udp-perf-send -peer 10.0.199.147:9 -count 2 -warmup 0 -packet-timeout 100ms -perf-json -timeout 5s
go run . -profile lan -backend network -pion-net -mdns disabled -raw-candidates -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 35s
go run . -profile thunderbolt -backend network -pion-net -mdns disabled -raw-candidates -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 40s
go run . -profile awdl -backend network -pion-net -mdns disabled -raw-candidates -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 45s
SSH_TARGET=tmc2@10.0.18.249 scripts/remote-matrix.sh
SSH_TARGET=tmc2@10.0.18.249 OUTPUT=/tmp/awdl-webrtc-diagnostics.txt scripts/remote-diagnostics.sh
```

## Stop Condition

This pass productizes the proof enough to expose reusable package boundaries,
better measurement/candidate-policy surfaces, a Pion `transport.Net` adapter,
and current two-host direction checks. Native AWDL `SetNet` WebRTC now passes
with explicit raw candidate publication. The remaining productization boundary
is `apple-pion` release/remote setup, remote reruns, longer performance
samples, and UDP asymmetry cleanup.
