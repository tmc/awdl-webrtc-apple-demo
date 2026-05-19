# Remote Matrix v0.1.1

Run date: 2026-05-19 UTC

Command:

```sh
CANDIDATE_POLICY=auto \
REQUIRE_PATHS=1 \
SSH_TARGET=tmc2@10.0.18.249 \
REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin \
OUTPUT=/tmp/awdl-webrtc-matrix-v011.txt \
SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-v011.md \
scripts/remote-matrix-bundle.sh
```

Module pins:

| Module | Version |
| --- | --- |
| `github.com/tmc/apple` | `v0.6.7` |
| `github.com/tmc/apple-pion` | `v0.1.1` |

## Summary

| Link | Probe | Direction | Status | Datagrams | Loss | Bitrate | RTT avg | RTT p95 | Path |
| --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | --- |
| LAN `en0` | Pion `transport.Net` WebRTC | bidirectional | PASS | - | - | - | - | - | `en0` |
| LAN `en0` | UDP perf | remote-to-local | PASS | 60/60 | 0.00% | 6.78 Mbps | 10.287ms | 21.666ms | `en0/NWInterfaceTypeWifi` |
| LAN `en0` | UDP perf | local-to-remote | PARTIAL | 57/60 | 5.00% | 978.47 Kbps | 9.563ms | 22.670ms | `en0/NWInterfaceTypeWifi` |
| LAN `en0` | UDP perf | bidirectional local sender | PASS | 60/60 | 0.00% | 6.59 Mbps | 10.502ms | 18.168ms | `en0/NWInterfaceTypeWifi` |
| LAN `en0` | UDP perf | bidirectional remote sender | PASS | 60/60 | 0.00% | 3.65 Mbps | 20.535ms | 159.697ms | `en0/NWInterfaceTypeWifi` |
| LAN `en0` | UDP latency | remote-to-local | PASS | 60/60 | 0.00% | - | 8.977ms | 8.718ms | `en0/NWInterfaceTypeWifi` |
| LAN `en0` | UDP latency | local-to-remote | PASS | 60/60 | 0.00% | - | 25.931ms | 74.928ms | `en0/NWInterfaceTypeWifi` |
| LAN `en0` | UDP callback | both directions | PASS | - | - | - | - | - | `en0` |
| Thunderbolt `bridge0` | Pion `transport.Net` WebRTC | bidirectional | PASS | - | - | - | - | - | `bridge0` |
| Thunderbolt `bridge0` | UDP perf | remote-to-local | PASS | 60/60 | 0.00% | 32.05 Mbps | 2.231ms | 30.210ms | `bridge0/NWInterfaceTypeWired` |
| Thunderbolt `bridge0` | UDP perf | local-to-remote | PASS | 60/60 | 0.00% | 35.52 Mbps | 1.337ms | 9.400ms | `bridge0/NWInterfaceTypeWired` |
| Thunderbolt `bridge0` | UDP perf | bidirectional local sender | PASS | 60/60 | 0.00% | 29.54 Mbps | 1.325ms | 8.318ms | `bridge0/NWInterfaceTypeWired` |
| Thunderbolt `bridge0` | UDP perf | bidirectional remote sender | PASS | 60/60 | 0.00% | 31.11 Mbps | 2.294ms | 30.956ms | `bridge0/NWInterfaceTypeWired` |
| Thunderbolt `bridge0` | UDP latency | remote-to-local | PASS | 60/60 | 0.00% | - | 756.951us | 345.334us | `bridge0/NWInterfaceTypeWired` |
| Thunderbolt `bridge0` | UDP latency | local-to-remote | PASS | 60/60 | 0.00% | - | 655.616us | 464.958us | `bridge0/NWInterfaceTypeWired` |
| Thunderbolt `bridge0` | UDP callback | both directions | PASS | - | - | - | - | - | `bridge0` |
| AWDL `awdl0` | Pion `transport.Net` WebRTC | bidirectional | FAIL | - | - | - | - | - | timeout waiting for data channel |
| AWDL `awdl0` | UDP perf | remote-to-local | PASS | 60/60 | 0.00% | 8.15 Mbps | 9.016ms | 26.493ms | `awdl0/NWInterfaceTypeWifi` |
| AWDL `awdl0` | UDP perf | local-to-remote | PASS | 60/60 | 0.00% | 7.89 Mbps | 8.605ms | 21.189ms | `awdl0/NWInterfaceTypeWifi` |
| AWDL `awdl0` | UDP perf | bidirectional local sender | PARTIAL | 57/60 | 5.00% | 988.84 Kbps | 7.935ms | 12.456ms | `awdl0/NWInterfaceTypeWifi` |
| AWDL `awdl0` | UDP perf | bidirectional remote sender | PASS | 60/60 | 0.00% | 8.35 Mbps | 8.594ms | 23.464ms | `awdl0/NWInterfaceTypeWifi` |
| AWDL `awdl0` | UDP latency | remote-to-local | PASS | 60/60 | 0.00% | - | 5.851ms | 10.079ms | `awdl0/NWInterfaceTypeWifi` |
| AWDL `awdl0` | UDP latency | local-to-remote | PASS | 60/60 | 0.00% | - | 6.019ms | 8.445ms | `awdl0/NWInterfaceTypeWifi` |
| AWDL `awdl0` | UDP callback | both directions | PASS | - | - | - | - | - | `awdl0` |

## Interpretation

The `v0.1.1` Pion transport fix resolved the LAN candidate-address problem:
LAN Pion-native WebRTC now opens a datachannel with `candidate_policy=auto`.
Thunderbolt Pion-native WebRTC still passes.

AWDL raw UDP is not the current blocker. Perf, latency, path-policy, and callback
probes all traverse `awdl0` in both directions. The remaining failure is the
AWDL Pion-native WebRTC datachannel open path, which timed out in the matrix even
though a traced retry had opened earlier. Treat AWDL `-pion-net` WebRTC as
intermittent until the ICE/DTLS/SCTP path is instrumented more deeply.

Raw artifacts from this run:

| Artifact | Path |
| --- | --- |
| Raw transcript | `/tmp/awdl-webrtc-matrix-v011.txt` |
| Generated full JSON-derived table | `/tmp/awdl-webrtc-matrix-v011.md` |
| LAN diagnostics smoke after script fix | `/tmp/awdl-webrtc-diagnostics-v011-lan.txt` |
