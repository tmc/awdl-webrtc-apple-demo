# Productization Audit

This checklist maps the productization request to concrete artifacts and current
evidence.

| Requirement | Status | Evidence | Remaining work |
| --- | --- | --- | --- |
| Full Pion-native backend closer to `transport.Net` | PARTIAL | `tmc/apple-pion` commit `d2698d5` adds `github.com/tmc/apple-pion/nwtransport`; `-pion-net -mdns disabled -raw-candidates` remote WebRTC passes on LAN, Thunderbolt, and AWDL. | Decide whether TCP, wildcard UDP, and TURN/STUN stay fallback-backed or become Network.framework-native. |
| Durable AWDL/link-local candidate handling | PARTIAL | `internal/icepolicy` centralizes the AWDL raw-host-candidate policy; UDP-mux and native `SetNet` AWDL raw-candidate remote WebRTC pass. A direct no-SDP-rewrite `SetNAT1To1IPs(fe80/fe80)` experiment gathered zero AWDL candidates. | The policy still mutates SDP; a cleaner Pion publication hook would remove that demo-side rewrite. |
| Direction/asymmetry cleanup | PARTIAL | Remote-to-local Network.framework UDP listener tests pass on Thunderbolt and AWDL; LAN local-to-remote passes. Local route inspection showed `169.254.88.35` would route over `en0`, so the Go UDP backend now binds IPv4 link-local sockets to the selected interface. | Re-run two-host Thunderbolt after `bridge0` regains a usable address; current clean local-to-remote Network.framework remote-listener tests for Thunderbolt and AWDL still need deeper tracing. |
| Performance hardening | PARTIAL | `udp-perf` and `udp-perf-send` support `-trials`; output includes `Lost` and `Loss`; `-perf-json` emits machine-readable result records; local and two-host Network.framework smoke tests pass. | Run longer repeated remote trials and add deeper batching/backpressure tuning if needed. |
| Reusable package | PASS | `tmc/apple` commit `ec3a7fea` adds `github.com/tmc/apple/network/nwpacket`; `tmc/apple-pion` commit `d2698d5` adds `github.com/tmc/apple-pion/nwtransport`; this demo imports both through local replaces. | Cut module releases and remove the local replaces. |
| Repo hygiene | PASS | The untracked `awdl-webrtc-apple-demo` binary was removed; `git status --short` is clean after commits. | None. |

## Verification Commands

```sh
go test ./...
go vet ./...
go run . -profile lan -backend network -mode gather -timeout 8s
go run . -profile awdl -backend network -mdns disabled -raw-candidates -mode gather -timeout 10s
go run . -profile lan -backend network -mode udp-perf -count 2 -warmup 0 -trials 2 -perf-json -timeout 15s
go run . -profile lan -backend network -pion-net -mdns disabled -raw-candidates -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 35s
go run . -profile thunderbolt -backend network -pion-net -mdns disabled -raw-candidates -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 40s
go run . -profile awdl -backend network -pion-net -mdns disabled -raw-candidates -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 45s
```

## Stop Condition

This pass productizes the proof enough to expose reusable package boundaries,
better measurement/candidate-policy surfaces, a Pion `transport.Net` adapter,
and current two-host direction checks. Native AWDL `SetNet` WebRTC now passes
with explicit raw candidate publication. The remaining productization boundary
is packaging plus broader transport policy, not the AWDL proof itself.
