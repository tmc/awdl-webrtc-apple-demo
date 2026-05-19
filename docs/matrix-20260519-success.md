# Remote Matrix Success, 2026-05-19

Command:

```sh
WEBRTC_TRACE=1 \
REMOTE_READY_TIMEOUT=30 \
REMOTE_STEP_READY_TIMEOUT=30 \
CANDIDATE_POLICY=auto \
REQUIRE_PATHS=1 \
SSH_TARGET=tmc2@10.0.18.249 \
REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin \
OUTPUT=/tmp/awdl-webrtc-matrix-signalonly-followup.txt \
SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-signalonly-followup.md \
scripts/remote-matrix-bundle.sh
```

Result: `matrix passed profiles=lan thunderbolt awdl`.

The run first failed while stale demo listener processes from an interrupted
older matrix still held the remote binary path. After killing only those demo
processes on both hosts, the same command completed. No code changed between
the failed guard and this successful run.

## At A Glance

| Link | WebRTC `-pion-net` | UDP perf remote-to-local | UDP perf local-to-remote | Bidirectional UDP | Latency summaries | Path evidence |
| --- | --- | --- | --- | --- | --- | --- |
| LAN `en0` | PASS, datachannel opened over `en0` | 60/60, 0.00% loss, 6.98 Mbps, 10.517ms avg RTT | 57/60, 5.00% loss, 979.39 Kbps, 9.373ms avg RTT | PASS both senders, 0.00% loss | 11.385ms and 9.295ms avg RTT | `en0/NWInterfaceTypeWifi` |
| Thunderbolt `bridge0` | PASS, datachannel opened over `bridge0` | 60/60, 0.00% loss, 29.35 Mbps, 2.376ms avg RTT | 60/60, 0.00% loss, 39.22 Mbps, 1.044ms avg RTT | PASS both senders, 0.00% loss | 821.232us and 651.397us avg RTT | `bridge0/NWInterfaceTypeWired` |
| AWDL `awdl0` | PASS, datachannel opened over `awdl0` | 60/60, 0.00% loss, 8.71 Mbps, 8.358ms avg RTT | 60/60, 0.00% loss, 8.12 Mbps, 8.422ms avg RTT | PASS both senders, 0.00% loss | 6.271ms and 5.968ms avg RTT | `awdl0/NWInterfaceTypeWifi` |

## Full Generated Table

| Section | Kind | Trial | Datagrams | Lost | Loss | Transfer | Bitrate | Elapsed | RTT avg | RTT p95 | Path |
| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| lan remote perf sender remote UDP route | udp_perf | 1/3 | 20 | 0 | 0.00% | 46.88 KiB | 6.08 Mbps | 63.107ms | 12.320ms | 25.578ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan remote perf sender remote UDP route | udp_perf | 2/3 | 20 | 0 | 0.00% | 46.88 KiB | 9.04 Mbps | 42.472ms | 7.637ms | 9.684ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan remote perf sender remote UDP route | udp_perf | 3/3 | 20 | 0 | 0.00% | 46.88 KiB | 6.45 Mbps | 59.580ms | 11.594ms | 21.180ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan remote perf sender remote UDP route | udp_perf_summary | summary/3 | 60 | 0 | 0.00% | 140.62 KiB | 6.98 Mbps | 165.158ms | 10.517ms | 25.540ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan remote perf sender remote UDP route | udp_perf_listen | - | 60/60 | 0 | 0.00% | 70.31 KiB | 341.46 Kbps | 1.687s | - | - | - |
| lan local perf sender local UDP route | udp_perf | 1/3 | 17 | 3 | 15.00% | 39.84 KiB | 322.79 Kbps | 1.011s | 7.190ms | 19.075ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan local perf sender local UDP route | udp_perf | 2/3 | 20 | 0 | 0.00% | 46.88 KiB | 6.63 Mbps | 57.935ms | 11.164ms | 23.173ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan local perf sender local UDP route | udp_perf | 3/3 | 20 | 0 | 0.00% | 46.88 KiB | 7.95 Mbps | 48.324ms | 9.437ms | 10.930ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan local perf sender local UDP route | udp_perf_summary | summary/3 | 57 | 3 | 5.00% | 133.59 KiB | 979.39 Kbps | 1.117s | 9.373ms | 23.122ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan local perf sender local UDP route | udp_perf_listen | - | 57/60 | 3 | 5.00% | 66.80 KiB | 12.16 Kbps | 45.002s | - | - | - |
| lan bidirectional remote sender remote UDP route | udp_perf | 1/3 | 20 | 0 | 0.00% | 46.88 KiB | 6.23 Mbps | 61.674ms | 10.049ms | 19.467ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan bidirectional remote sender remote UDP route | udp_perf | 2/3 | 20 | 0 | 0.00% | 46.88 KiB | 7.52 Mbps | 51.039ms | 9.265ms | 12.598ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan bidirectional remote sender remote UDP route | udp_perf | 3/3 | 20 | 0 | 0.00% | 46.88 KiB | 8.33 Mbps | 46.089ms | 8.777ms | 11.604ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan bidirectional remote sender remote UDP route | udp_perf_summary | summary/3 | 60 | 0 | 0.00% | 140.62 KiB | 7.25 Mbps | 158.801ms | 9.364ms | 17.102ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan bidirectional remote sender remote UDP route | udp_perf | 1/3 | 20 | 0 | 0.00% | 46.88 KiB | 6.54 Mbps | 58.754ms | 10.910ms | 22.060ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan bidirectional remote sender remote UDP route | udp_perf | 2/3 | 20 | 0 | 0.00% | 46.88 KiB | 9.02 Mbps | 42.578ms | 8.111ms | 9.652ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan bidirectional remote sender remote UDP route | udp_perf | 3/3 | 20 | 0 | 0.00% | 46.88 KiB | 7.62 Mbps | 50.373ms | 9.218ms | 10.430ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan bidirectional remote sender remote UDP route | udp_perf_summary | summary/3 | 60 | 0 | 0.00% | 140.62 KiB | 7.59 Mbps | 151.705ms | 9.413ms | 21.803ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan bidirectional remote sender remote UDP route | udp_perf_listen | - | 60/60 | 0 | 0.00% | 70.31 KiB | 258.93 Kbps | 2.225s | - | - | - |
| lan bidirectional remote sender remote UDP route | udp_perf_listen | - | 60/60 | 0 | 0.00% | 70.31 KiB | 393.01 Kbps | 1.466s | - | - | - |
| lan remote latency sender remote UDP route | udp_latency | 1/3 | 20 | 0 | 0.00% | 0.00 B | 0.00 bps | 272.634ms | 13.629ms | 9.967ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan remote latency sender remote UDP route | udp_latency | 2/3 | 20 | 0 | 0.00% | 0.00 B | 0.00 bps | 133.548ms | 6.675ms | 8.116ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan remote latency sender remote UDP route | udp_latency | 3/3 | 20 | 0 | 0.00% | 0.00 B | 0.00 bps | 277.062ms | 13.850ms | 9.942ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan remote latency sender remote UDP route | udp_latency_summary | summary/3 | 60 | 0 | 0.00% | 0.00 B | 0.00 bps | 683.244ms | 11.385ms | 9.942ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan remote latency sender remote UDP route | udp_perf_listen | - | 60/60 | 0 | 0.00% | 70.31 KiB | 320.64 Kbps | 1.796s | - | - | - |
| lan local latency sender local UDP route | udp_latency | 1/3 | 20 | 0 | 0.00% | 0.00 B | 0.00 bps | 303.738ms | 15.186ms | 28.883ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan local latency sender local UDP route | udp_latency | 2/3 | 20 | 0 | 0.00% | 0.00 B | 0.00 bps | 131.258ms | 6.562ms | 8.491ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan local latency sender local UDP route | udp_latency | 3/3 | 20 | 0 | 0.00% | 0.00 B | 0.00 bps | 122.783ms | 6.138ms | 7.679ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan local latency sender local UDP route | udp_latency_summary | summary/3 | 60 | 0 | 0.00% | 0.00 B | 0.00 bps | 557.780ms | 9.295ms | 9.206ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan local latency sender local UDP route | udp_perf_listen | - | 60/60 | 0 | 0.00% | 70.31 KiB | 481.62 Kbps | 1.196s | - | - | - |
| thunderbolt remote perf sender remote UDP route | udp_perf | 1/3 | 20 | 0 | 0.00% | 46.88 KiB | 11.07 Mbps | 34.689ms | 6.614ms | 32.065ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt remote perf sender remote UDP route | udp_perf | 2/3 | 20 | 0 | 0.00% | 46.88 KiB | 197.82 Mbps | 1.941ms | 256.587us | 303.375us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt remote perf sender remote UDP route | udp_perf | 3/3 | 20 | 0 | 0.00% | 46.88 KiB | 146.39 Mbps | 2.623ms | 256.368us | 400.209us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt remote perf sender remote UDP route | udp_perf_summary | summary/3 | 60 | 0 | 0.00% | 140.62 KiB | 29.35 Mbps | 39.253ms | 2.376ms | 32.040ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt remote perf sender remote UDP route | udp_perf_listen | - | 60/60 | 0 | 0.00% | 70.31 KiB | 426.50 Kbps | 1.351s | - | - | - |
| thunderbolt local perf sender local UDP route | udp_perf | 1/3 | 20 | 0 | 0.00% | 46.88 KiB | 16.86 Mbps | 22.773ms | 2.077ms | 7.972ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt local perf sender local UDP route | udp_perf | 2/3 | 20 | 0 | 0.00% | 46.88 KiB | 111.14 Mbps | 3.455ms | 502.543us | 622.875us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt local perf sender local UDP route | udp_perf | 3/3 | 20 | 0 | 0.00% | 46.88 KiB | 122.05 Mbps | 3.146ms | 552.768us | 606.500us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt local perf sender local UDP route | udp_perf_summary | summary/3 | 60 | 0 | 0.00% | 140.62 KiB | 39.22 Mbps | 29.375ms | 1.044ms | 7.558ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt local perf sender local UDP route | udp_perf_listen | - | 60/60 | 0 | 0.00% | 70.31 KiB | 843.01 Kbps | 683.267ms | - | - | - |
| thunderbolt bidirectional remote sender remote UDP route | udp_perf | 1/3 | 20 | 0 | 0.00% | 46.88 KiB | 19.66 Mbps | 19.535ms | 1.754ms | 7.142ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt bidirectional remote sender remote UDP route | udp_perf | 2/3 | 20 | 0 | 0.00% | 46.88 KiB | 149.51 Mbps | 2.568ms | 404.954us | 882.709us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt bidirectional remote sender remote UDP route | udp_perf | 3/3 | 20 | 0 | 0.00% | 46.88 KiB | 204.42 Mbps | 1.879ms | 299.648us | 327.500us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt bidirectional remote sender remote UDP route | udp_perf_summary | summary/3 | 60 | 0 | 0.00% | 140.62 KiB | 48.04 Mbps | 23.982ms | 819.645us | 7.125ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt bidirectional remote sender remote UDP route | udp_perf | 1/3 | 20 | 0 | 0.00% | 46.88 KiB | 12.37 Mbps | 31.046ms | 5.898ms | 28.243ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt bidirectional remote sender remote UDP route | udp_perf | 2/3 | 20 | 0 | 0.00% | 46.88 KiB | 207.45 Mbps | 1.851ms | 250.227us | 314.083us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt bidirectional remote sender remote UDP route | udp_perf | 3/3 | 20 | 0 | 0.00% | 46.88 KiB | 149.65 Mbps | 2.566ms | 289.502us | 524.709us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt bidirectional remote sender remote UDP route | udp_perf_summary | summary/3 | 60 | 0 | 0.00% | 140.62 KiB | 32.48 Mbps | 35.463ms | 2.146ms | 28.232ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt bidirectional remote sender remote UDP route | udp_perf_listen | - | 60/60 | 0 | 0.00% | 70.31 KiB | 282.65 Kbps | 2.038s | - | - | - |
| thunderbolt bidirectional remote sender remote UDP route | udp_perf_listen | - | 60/60 | 0 | 0.00% | 70.31 KiB | 452.61 Kbps | 1.273s | - | - | - |
| thunderbolt remote latency sender remote UDP route | udp_latency | 1/3 | 20 | 0 | 0.00% | 0.00 B | 0.00 bps | 39.072ms | 1.953ms | 578.917us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt remote latency sender remote UDP route | udp_latency | 2/3 | 20 | 0 | 0.00% | 0.00 B | 0.00 bps | 4.932ms | 246.000us | 264.250us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt remote latency sender remote UDP route | udp_latency | 3/3 | 20 | 0 | 0.00% | 0.00 B | 0.00 bps | 5.312ms | 264.937us | 262.791us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt remote latency sender remote UDP route | udp_latency_summary | summary/3 | 60 | 0 | 0.00% | 0.00 B | 0.00 bps | 49.316ms | 821.232us | 362.709us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt remote latency sender remote UDP route | udp_perf_listen | - | 60/60 | 0 | 0.00% | 70.31 KiB | 435.39 Kbps | 1.323s | - | - | - |
| thunderbolt local latency sender local UDP route | udp_latency | 1/3 | 20 | 0 | 0.00% | 0.00 B | 0.00 bps | 27.141ms | 1.356ms | 907.917us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt local latency sender local UDP route | udp_latency | 2/3 | 20 | 0 | 0.00% | 0.00 B | 0.00 bps | 6.298ms | 314.312us | 414.375us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt local latency sender local UDP route | udp_latency | 3/3 | 20 | 0 | 0.00% | 0.00 B | 0.00 bps | 5.679ms | 283.485us | 298.250us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt local latency sender local UDP route | udp_latency_summary | summary/3 | 60 | 0 | 0.00% | 0.00 B | 0.00 bps | 39.118ms | 651.397us | 559.125us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt local latency sender local UDP route | udp_perf_listen | - | 60/60 | 0 | 0.00% | 70.31 KiB | 832.85 Kbps | 691.603ms | - | - | - |
| awdl remote perf sender remote UDP route | udp_perf | 1/3 | 20 | 0 | 0.00% | 46.88 KiB | 7.05 Mbps | 54.468ms | 10.405ms | 26.442ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl remote perf sender remote UDP route | udp_perf | 2/3 | 20 | 0 | 0.00% | 46.88 KiB | 9.47 Mbps | 40.557ms | 7.732ms | 11.260ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl remote perf sender remote UDP route | udp_perf | 3/3 | 20 | 0 | 0.00% | 46.88 KiB | 10.32 Mbps | 37.225ms | 6.935ms | 8.675ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl remote perf sender remote UDP route | udp_perf_summary | summary/3 | 60 | 0 | 0.00% | 140.62 KiB | 8.71 Mbps | 132.250ms | 8.358ms | 26.404ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl remote perf sender remote UDP route | udp_perf_listen | - | 60/60 | 0 | 0.00% | 70.31 KiB | 366.71 Kbps | 1.571s | - | - | - |
| awdl local perf sender local UDP route | udp_perf | 1/3 | 20 | 0 | 0.00% | 46.88 KiB | 6.24 Mbps | 61.580ms | 10.205ms | 24.253ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl local perf sender local UDP route | udp_perf | 2/3 | 20 | 0 | 0.00% | 46.88 KiB | 9.90 Mbps | 38.773ms | 7.410ms | 8.788ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl local perf sender local UDP route | udp_perf | 3/3 | 20 | 0 | 0.00% | 46.88 KiB | 9.26 Mbps | 41.469ms | 7.650ms | 12.730ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl local perf sender local UDP route | udp_perf_summary | summary/3 | 60 | 0 | 0.00% | 140.62 KiB | 8.12 Mbps | 141.822ms | 8.422ms | 24.196ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl local perf sender local UDP route | udp_perf_listen | - | 60/60 | 0 | 0.00% | 70.31 KiB | 780.12 Kbps | 738.351ms | - | - | - |
| awdl bidirectional remote sender remote UDP route | udp_perf | 1/3 | 20 | 0 | 0.00% | 46.88 KiB | 5.71 Mbps | 67.301ms | 10.987ms | 19.848ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl bidirectional remote sender remote UDP route | udp_perf | 2/3 | 20 | 0 | 0.00% | 46.88 KiB | 7.68 Mbps | 49.968ms | 9.857ms | 14.334ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl bidirectional remote sender remote UDP route | udp_perf | 3/3 | 20 | 0 | 0.00% | 46.88 KiB | 9.07 Mbps | 42.342ms | 8.385ms | 9.128ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl bidirectional remote sender remote UDP route | udp_perf_summary | summary/3 | 60 | 0 | 0.00% | 140.62 KiB | 7.22 Mbps | 159.611ms | 9.743ms | 19.599ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl bidirectional remote sender remote UDP route | udp_perf | 1/3 | 20 | 0 | 0.00% | 46.88 KiB | 6.63 Mbps | 57.890ms | 11.142ms | 24.024ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl bidirectional remote sender remote UDP route | udp_perf | 2/3 | 20 | 0 | 0.00% | 46.88 KiB | 9.25 Mbps | 41.506ms | 7.720ms | 11.883ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl bidirectional remote sender remote UDP route | udp_perf | 3/3 | 20 | 0 | 0.00% | 46.88 KiB | 9.24 Mbps | 41.577ms | 7.911ms | 9.305ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl bidirectional remote sender remote UDP route | udp_perf_summary | summary/3 | 60 | 0 | 0.00% | 140.62 KiB | 8.17 Mbps | 140.972ms | 8.925ms | 23.948ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl bidirectional remote sender remote UDP route | udp_perf_listen | - | 60/60 | 0 | 0.00% | 70.31 KiB | 250.66 Kbps | 2.298s | - | - | - |
| awdl bidirectional remote sender remote UDP route | udp_perf_listen | - | 60/60 | 0 | 0.00% | 70.31 KiB | 406.05 Kbps | 1.419s | - | - | - |
| awdl remote latency sender remote UDP route | udp_latency | 1/3 | 20 | 0 | 0.00% | 0.00 B | 0.00 bps | 129.417ms | 6.470ms | 9.979ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl remote latency sender remote UDP route | udp_latency | 2/3 | 20 | 0 | 0.00% | 0.00 B | 0.00 bps | 131.801ms | 6.587ms | 11.360ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl remote latency sender remote UDP route | udp_latency | 3/3 | 20 | 0 | 0.00% | 0.00 B | 0.00 bps | 115.185ms | 5.757ms | 6.968ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl remote latency sender remote UDP route | udp_latency_summary | summary/3 | 60 | 0 | 0.00% | 0.00 B | 0.00 bps | 376.402ms | 6.271ms | 11.360ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl remote latency sender remote UDP route | udp_perf_listen | - | 60/60 | 0 | 0.00% | 70.31 KiB | 353.44 Kbps | 1.630s | - | - | - |
| awdl local latency sender local UDP route | udp_latency | 1/3 | 20 | 0 | 0.00% | 0.00 B | 0.00 bps | 137.877ms | 6.893ms | 10.367ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl local latency sender local UDP route | udp_latency | 2/3 | 20 | 0 | 0.00% | 0.00 B | 0.00 bps | 107.542ms | 5.376ms | 5.649ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl local latency sender local UDP route | udp_latency | 3/3 | 20 | 0 | 0.00% | 0.00 B | 0.00 bps | 112.697ms | 5.634ms | 6.539ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl local latency sender local UDP route | udp_latency_summary | summary/3 | 60 | 0 | 0.00% | 0.00 B | 0.00 bps | 358.116ms | 5.968ms | 9.290ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl local latency sender local UDP route | udp_perf_listen | - | 60/60 | 0 | 0.00% | 70.31 KiB | 565.48 Kbps | 1.019s | - | - | - |
