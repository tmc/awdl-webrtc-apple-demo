# Productization Audit

This checklist maps the productization request to concrete artifacts and current
evidence.

| Requirement | Status | Evidence | Remaining work |
| --- | --- | --- | --- |
| Full Pion-native backend closer to `transport.Net` | PARTIAL | `tmc/apple-pion` commits `d2698d5`, `167087c`, and `3ea6faf` add `github.com/tmc/apple-pion/nwtransport`; concrete UDP listeners, configured wildcard UDP listeners, and UDP dials route through Network.framework using `github.com/tmc/apple/x/network/nwpacket`; `-pion-net -mdns disabled -raw-candidates` remote WebRTC passes on LAN, Thunderbolt, and AWDL. | Decide whether TCP, unconstrained wildcard UDP, and TURN/STUN stay fallback-backed or become Network.framework-native. |
| Durable AWDL/link-local candidate handling | PARTIAL | `internal/icepolicy` centralizes the AWDL raw-host-candidate policy. Commit `d1ef7ad` removes raw-candidate SDP mutation from signaling by stripping candidates from SDP and publishing explicit `ICECandidateInit` records with the selected link-local IP. AWDL raw-candidate gather passes. A direct no-SDP-rewrite `SetNAT1To1IPs(fe80/fe80)` experiment gathered zero AWDL candidates. | Re-run two-host raw-candidate WebRTC with `tmc2@10.0.18.249` once SSH is reachable again. |
| Direction/asymmetry cleanup | PARTIAL | Remote-to-local Network.framework UDP listener tests pass on Thunderbolt and AWDL; LAN local-to-remote passes. Local route inspection showed `169.254.88.35` would route over `en0`, so the Go UDP backend now binds IPv4 link-local sockets to the selected interface. The Thunderbolt default selector now falls back from an addressless `bridge0` to a usable member interface, and local Network.framework gather passed on `en1`. | Re-run two-host Thunderbolt/AWDL/LAN when `tmc2@10.0.18.249` is reachable again; current clean local-to-remote Network.framework remote-listener tests for Thunderbolt and AWDL still need deeper tracing. |
| Performance hardening | PARTIAL | `udp-perf` and `udp-perf-send` support `-trials` and bounded in-flight `-window`; output includes `Lost` and `Loss`; `-perf-json` emits machine-readable result records including `window`; `-packet-timeout` counts echo timeouts as lost datagrams; local and two-host Network.framework smoke tests pass. | Run longer repeated remote trials and tune the window/backpressure behavior with the second host once reachable. |
| Reusable package | PASS | `tmc/apple` commit `eb350571` adds `github.com/tmc/apple/x/network/nwpacket`, and `tmc/apple-pion` commits `d2698d5`, `167087c`, and `3ea6faf` add `github.com/tmc/apple-pion/nwtransport`; this demo imports both through local replaces. | Cut module releases and remove the local replaces. |
| Repo hygiene | PASS | The untracked `awdl-webrtc-apple-demo` binary was removed; `git status --short` is clean after commits. | None. |

## Verification Commands

```sh
go test ./...
go vet ./...
go run . -profile lan -backend network -mode gather -timeout 8s
go run . -profile thunderbolt -backend network -mode gather -timeout 5s
go run . -profile awdl -backend network -mdns disabled -raw-candidates -mode gather -timeout 10s
go run . -profile lan -backend network -mode udp-perf -count 2 -warmup 0 -trials 2 -perf-json -timeout 15s
go run . -profile lan -backend network -mode udp-perf -count 4 -warmup 0 -window 2 -perf-json -timeout 10s
go run . -profile lan -backend go -mode udp-perf-send -peer 10.0.199.147:9 -count 2 -warmup 0 -packet-timeout 100ms -perf-json -timeout 5s
go run . -profile lan -backend network -pion-net -mdns disabled -raw-candidates -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 35s
go run . -profile thunderbolt -backend network -pion-net -mdns disabled -raw-candidates -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 40s
go run . -profile awdl -backend network -pion-net -mdns disabled -raw-candidates -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 45s
```

## Stop Condition

This pass productizes the proof enough to expose reusable package boundaries,
better measurement/candidate-policy surfaces, a Pion `transport.Net` adapter,
and current two-host direction checks. Native AWDL `SetNet` WebRTC now passes
with explicit raw candidate publication. The remaining productization boundary
is package release, broader transport policy, remote reruns, and UDP asymmetry
cleanup.
