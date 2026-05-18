# Productization Audit

This checklist maps the productization request to concrete artifacts and current
evidence.

| Requirement | Status | Evidence | Remaining work |
| --- | --- | --- | --- |
| Full Pion-native backend closer to `transport.Net` | PARTIAL | `nwtransport` implements Pion `transport.Net` for UDP host listeners through `nwpacket`; `-pion-net` remote WebRTC passes on LAN and Thunderbolt. | Finish AWDL native `SetNet` candidate handling; decide whether TCP, connected UDP, and TURN/STUN should stay fallback-backed or become Network.framework-native. |
| Durable AWDL/link-local candidate handling | PARTIAL | `internal/icepolicy` centralizes the AWDL raw-host-candidate policy; UDP-mux AWDL raw-candidate gather and remote WebRTC pass. | Native `SetNet` raw AWDL is blocked because Pion tries to rewrite `fe80::...%awdl0` as a plain IP; fix or upstream scoped link-local address rewrite support to eliminate SDP mutation. |
| Direction/asymmetry cleanup | PASS | Local-to-remote Network.framework UDP listener tests now pass for LAN, Thunderbolt, and AWDL against `tmc2@10.0.18.249`, all with zero loss. | Longer sustained runs are still useful, but the earlier zero-datagram asymmetry was not reproduced with the current binary. |
| Performance hardening | PARTIAL | `udp-perf` and `udp-perf-send` support `-trials`; output includes `Lost` and `Loss`; local and two-host Network.framework smoke tests pass. | Run longer repeated remote trials and add deeper batching/backpressure tuning if needed. |
| Reusable package | PASS | `github.com/tmc/awdl-webrtc-apple-demo/nwpacket` and `github.com/tmc/awdl-webrtc-apple-demo/nwtransport` contain package docs and tests. | Decide whether these graduate to `github.com/tmc/apple`, a standalone module, or stay as demo packages. |
| Repo hygiene | PASS | The untracked `awdl-webrtc-apple-demo` binary was removed; `git status --short` is clean after commits. | None. |

## Verification Commands

```sh
go test ./...
go run . -profile lan -backend network -mode gather -timeout 8s
go run . -profile awdl -backend network -mdns disabled -raw-candidates -mode gather -timeout 10s
go run . -profile lan -backend network -mode udp-perf -count 2 -warmup 0 -trials 2 -timeout 15s
go run . -profile lan -backend network -pion-net -mdns disabled -raw-candidates -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 35s
go run . -profile thunderbolt -backend network -pion-net -mdns disabled -raw-candidates -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 40s
```

## Stop Condition

This pass productizes the proof enough to expose reusable package boundaries,
better measurement/candidate-policy surfaces, a Pion `transport.Net` adapter,
and current two-host direction checks. It does not complete native AWDL
`SetNet` WebRTC because scoped link-local candidate publication still needs a
Pion-side fix or a cleaner publication hook.
