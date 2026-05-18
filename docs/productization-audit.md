# Productization Audit

This checklist maps the productization request to concrete artifacts and current
evidence.

| Requirement | Status | Evidence | Remaining work |
| --- | --- | --- | --- |
| Full Pion-native backend closer to `transport.Net` | PARTIAL | `nwpacket` exposes a Network.framework-backed `net.PacketConn`; `main` still wires it through Pion `SetICEUDPMux`. | Design and implement a Pion network/transport adapter rather than only a UDP mux PacketConn. |
| Durable AWDL/link-local candidate handling | PARTIAL | `internal/icepolicy` centralizes the AWDL raw-host-candidate policy; tests cover SDP host-candidate publishing; AWDL raw-candidate gather passes. | Eliminate SDP mutation by adding or upstreaming a Pion-native candidate publication mechanism that can safely publish scoped link-local addresses. |
| Direction/asymmetry cleanup | NOT DONE | Remote-to-local LAN/Thunderbolt/AWDL sender runs are recorded in `RESULTS.md`. | Diagnose local-to-remote listener failures against `tmc2@10.0.18.249` with firewall/socket/path probes. |
| Performance hardening | PARTIAL | `udp-perf` and `udp-perf-send` support `-trials`; output includes `Lost` and `Loss`; local repeated Network.framework smoke passes. | Run longer repeated remote trials and add deeper batching/backpressure tuning if needed. |
| Reusable package | PASS | `github.com/tmc/awdl-webrtc-apple-demo/nwpacket` contains package docs, an example, and unit tests. | Decide whether this graduates to `github.com/tmc/apple`, a standalone module, or stays as a demo package. |
| Repo hygiene | PASS | The untracked `awdl-webrtc-apple-demo` binary was removed; `git status --short` is clean after commits. | None. |

## Verification Commands

```sh
go test ./...
go run . -profile lan -backend network -mode gather -timeout 8s
go run . -profile awdl -backend network -mdns disabled -raw-candidates -mode gather -timeout 10s
go run . -profile lan -backend network -mode udp-perf -count 2 -warmup 0 -trials 2 -timeout 15s
```

## Stop Condition

This pass productizes the proof enough to expose reusable package boundaries and
better measurement/candidate-policy surfaces. It does not complete the full
Pion-native backend or the remote listener asymmetry investigation.
