# Matrix Retry, 2026-05-19

This failed SSH-control-plane run is preserved for diagnostics. It was
superseded by [matrix-20260519-success.md](matrix-20260519-success.md), which
passed LAN, Thunderbolt, and AWDL after stale demo listener processes were
cleaned up.

Command:

```sh
WEBRTC_TRACE=1 REMOTE_READY_TIMEOUT=30 CANDIDATE_POLICY=auto REQUIRE_PATHS=1 \
  SSH_TARGET=tmc2@10.0.18.249 \
  REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin \
  OUTPUT=/tmp/awdl-webrtc-matrix-try-again.txt \
  SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-try-again.md \
  scripts/remote-matrix-bundle.sh
```

The run reached `10.0.18.249` by ICMP, waited through initial SSH timeouts,
copied the binary after SSH recovered briefly, then lost SSH during the matrix.
Local Network.framework listeners still bound successfully for LAN, Thunderbolt,
and AWDL. Remote sender/listener processes mostly never started, so throughput
rows with zero datagrams are control-plane failures, not link measurements.

| Section | Kind | Trial | Datagrams | Lost | Loss | Transfer | Bitrate | Elapsed | RTT avg | RTT p95 | Path |
| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| lan Pion transport.Net WebRTC | failure | - | - | - | - | - | - | - | - | - | lan Pion transport.Net WebRTC exit=1 |
| lan local perf listener local UDP diagnostics | udp_perf_listen | - | 0/60 | 60 | 100.00% | 0.00 B | 0.00 bps | 45.001s | - | - | - |
| lan local perf listener local UDP diagnostics | failure | - | - | - | - | - | - | - | - | - | lan remote-to-local UDP perf exit=255 |
| lan local-to-remote UDP perf | failure | - | - | - | - | - | - | - | - | - | lan local-to-remote UDP perf exit=1 |
| lan bidirectional UDP perf | failure | - | - | - | - | - | - | - | - | - | lan bidirectional UDP perf exit=1 |
| lan local latency listener local UDP diagnostics | udp_perf_listen | - | 0/60 | 60 | 100.00% | 0.00 B | 0.00 bps | 45.001s | - | - | - |
| lan local latency listener local UDP diagnostics | failure | - | - | - | - | - | - | - | - | - | lan remote-to-local UDP latency exit=255 |
| lan local-to-remote UDP latency | failure | - | - | - | - | - | - | - | - | - | lan local-to-remote UDP latency exit=1 |
| lan local callback listener local UDP diagnostics | failure | - | - | - | - | - | - | - | - | - | lan callback remote-to-local request exit=1 |
| lan callback local-to-remote request | failure | - | - | - | - | - | - | - | - | - | lan callback local-to-remote request exit=1 |
| thunderbolt Pion transport.Net WebRTC | failure | - | - | - | - | - | - | - | - | - | thunderbolt Pion transport.Net WebRTC exit=1 |
| thunderbolt local perf listener local UDP diagnostics | udp_perf_listen | - | 0/60 | 60 | 100.00% | 0.00 B | 0.00 bps | 45.000s | - | - | - |
| thunderbolt local perf listener local UDP diagnostics | failure | - | - | - | - | - | - | - | - | - | thunderbolt remote-to-local UDP perf exit=255 |
| thunderbolt local-to-remote UDP perf | failure | - | - | - | - | - | - | - | - | - | thunderbolt local-to-remote UDP perf exit=1 |
| thunderbolt bidirectional UDP perf | failure | - | - | - | - | - | - | - | - | - | thunderbolt bidirectional UDP perf exit=1 |
| thunderbolt local latency listener local UDP diagnostics | udp_perf_listen | - | 0/60 | 60 | 100.00% | 0.00 B | 0.00 bps | 45.001s | - | - | - |
| thunderbolt local latency listener local UDP diagnostics | failure | - | - | - | - | - | - | - | - | - | thunderbolt remote-to-local UDP latency exit=255 |
| thunderbolt local-to-remote UDP latency | failure | - | - | - | - | - | - | - | - | - | thunderbolt local-to-remote UDP latency exit=1 |
| thunderbolt local callback listener local UDP diagnostics | failure | - | - | - | - | - | - | - | - | - | thunderbolt callback remote-to-local request exit=1 |
| thunderbolt callback local-to-remote request | failure | - | - | - | - | - | - | - | - | - | thunderbolt callback local-to-remote request exit=1 |
| awdl Pion transport.Net WebRTC | failure | - | - | - | - | - | - | - | - | - | awdl Pion transport.Net WebRTC exit=1 |
| awdl local perf listener local UDP diagnostics | udp_perf_listen | - | 0/60 | 60 | 100.00% | 0.00 B | 0.00 bps | 45.001s | - | - | - |
| awdl local perf listener local UDP diagnostics | failure | - | - | - | - | - | - | - | - | - | awdl remote-to-local UDP perf exit=255 |
| awdl local-to-remote UDP perf | failure | - | - | - | - | - | - | - | - | - | awdl local-to-remote UDP perf exit=1 |
| awdl bidirectional UDP perf | failure | - | - | - | - | - | - | - | - | - | awdl bidirectional UDP perf exit=1 |
| awdl local latency listener local UDP diagnostics | udp_perf_listen | - | 0/60 | 60 | 100.00% | 0.00 B | 0.00 bps | 45.001s | - | - | - |
| awdl local latency listener local UDP diagnostics | failure | - | - | - | - | - | - | - | - | - | awdl remote-to-local UDP latency exit=255 |
| awdl local-to-remote UDP latency | failure | - | - | - | - | - | - | - | - | - | awdl local-to-remote UDP latency exit=1 |
| awdl local callback listener local UDP diagnostics | failure | - | - | - | - | - | - | - | - | - | awdl callback remote-to-local request exit=1 |
| awdl callback local-to-remote request | failure | - | - | - | - | - | - | - | - | - | awdl callback local-to-remote request exit=1 |
