# Workspace Remote Matrix, 2026-05-19

Command:

```sh
WEBRTC_TRACE=0 \
WEBRTC_ATTEMPTS=3 \
WEBRTC_RETRY_DELAY=3 \
REMOTE_READY_TIMEOUT=30 \
REMOTE_STEP_READY_TIMEOUT=30 \
CANDIDATE_POLICY=auto \
REQUIRE_PATHS=1 \
DURATION=5s \
TRIALS=3 \
WINDOW=8 \
STREAMS=2 \
TIMEOUT=90s \
LISTEN_IDLE_TIMEOUT=3s \
SSH_TARGET=tmc2@10.0.18.249 \
REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin-workspace \
OUTPUT=/tmp/awdl-webrtc-matrix-workspace2-20260519.txt \
SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-workspace2-20260519.md \
scripts/remote-matrix-bundle.sh
```

Result: `matrix passed profiles=lan thunderbolt awdl`.

This run used the checked-in `go.work` file so the demo binary was built from
the sibling `../apple` and `../apple-pion` checkouts. No new tags were published
for this run.

## At A Glance

| Link | WebRTC `-pion-net` | UDP perf remote-to-local | UDP perf local-to-remote | Bidirectional UDP | Latency summaries | Path evidence |
| --- | --- | --- | --- | --- | --- | --- |
| LAN `en0` | PASS, datachannel opened over `en0` | 11290/11300, 0.09% loss, 14.12 Mbps, 20.468ms avg RTT | 10049/10049, 0.00% loss, 12.72 Mbps, 24.057ms avg RTT | PASS both senders, <=0.02% loss | 10.439ms and 10.942ms avg RTT | `en0/NWInterfaceTypeWifi` |
| Thunderbolt `bridge0` | PASS, datachannel opened over `bridge0` | 171664/171664, 0.00% loss, 219.61 Mbps, 1.363ms avg RTT | 238290/238290, 0.00% loss, 304.94 Mbps, 883.484us avg RTT | PASS both senders, 0.00% loss | 171.870us and 164.972us avg RTT | `bridge0/NWInterfaceTypeWired` |
| AWDL `awdl0` | PASS, datachannel opened over `awdl0` | 22653/22653, 0.00% loss, 28.93 Mbps, 10.466ms avg RTT | 20894/20908, 0.07% loss, 26.72 Mbps, 10.730ms avg RTT | PASS both senders, <=0.03% loss | 7.042ms and 7.289ms avg RTT | `awdl0/NWInterfaceTypeWifi` |

## Full Generated Table

| Section | Kind | Trial | Datagrams | Lost | Loss | Transfer | Bitrate | Elapsed | RTT avg | RTT p95 | Path |
| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| lan remote perf sender remote UDP route | udp_perf | 1/3 | 4202 | 8 | 0.19% | 9.62 MiB | 15.56 Mbps | 5.184s | 17.076ms | 21.330ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan remote perf sender remote UDP route | udp_perf | 2/3 | 4466 | 2 | 0.04% | 10.22 MiB | 17.12 Mbps | 5.009s | 17.322ms | 22.120ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan remote perf sender remote UDP route | udp_perf | 3/3 | 2622 | 0 | 0.00% | 6.00 MiB | 9.77 Mbps | 5.154s | 31.264ms | 159.200ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan remote perf sender remote UDP route | udp_perf_summary | summary/3 | 11290 | 10 | 0.09% | 25.84 MiB | 14.12 Mbps | 15.347s | 20.468ms | 62.640ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan remote perf sender remote UDP route | udp_perf_listen | - | 11298 | 0 | 0.00% | 12.93 MiB | 5.51 Mbps | 19.701s | - | - | - |
| lan local perf sender local UDP route | udp_perf | 1/3 | 1575 | 0 | 0.00% | 3.60 MiB | 6.04 Mbps | 5.007s | 50.615ms | 229.646ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan local perf sender local UDP route | udp_perf | 2/3 | 3737 | 0 | 0.00% | 8.55 MiB | 13.94 Mbps | 5.146s | 21.973ms | 53.232ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan local perf sender local UDP route | udp_perf | 3/3 | 4737 | 0 | 0.00% | 10.84 MiB | 18.14 Mbps | 5.013s | 16.870ms | 19.658ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan local perf sender local UDP route | udp_perf_summary | summary/3 | 10049 | 0 | 0.00% | 23.00 MiB | 12.72 Mbps | 15.166s | 24.057ms | 109.376ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan local perf sender local UDP route | udp_perf_listen | - | 10049 | 0 | 0.00% | 11.50 MiB | 4.71 Mbps | 20.463s | - | - | - |
| lan bidirectional remote sender remote UDP route | udp_perf | 1/3 | 3620 | 0 | 0.00% | 8.29 MiB | 13.88 Mbps | 5.006s | 21.994ms | 30.537ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan bidirectional remote sender remote UDP route | udp_perf | 2/3 | 4047 | 0 | 0.00% | 9.26 MiB | 15.15 Mbps | 5.129s | 20.146ms | 26.657ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan bidirectional remote sender remote UDP route | udp_perf | 3/3 | 4418 | 0 | 0.00% | 10.11 MiB | 16.95 Mbps | 5.005s | 18.052ms | 18.990ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan bidirectional remote sender remote UDP route | udp_perf_summary | summary/3 | 12085 | 0 | 0.00% | 27.66 MiB | 15.33 Mbps | 15.140s | 19.934ms | 24.847ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan bidirectional remote sender remote UDP route | udp_perf | 1/3 | 3596 | 3 | 0.08% | 8.23 MiB | 13.76 Mbps | 5.017s | 21.307ms | 34.056ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan bidirectional remote sender remote UDP route | udp_perf | 2/3 | 4020 | 0 | 0.00% | 9.20 MiB | 15.13 Mbps | 5.102s | 20.127ms | 56.054ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan bidirectional remote sender remote UDP route | udp_perf | 3/3 | 4472 | 0 | 0.00% | 10.24 MiB | 17.14 Mbps | 5.010s | 17.769ms | 19.261ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan bidirectional remote sender remote UDP route | udp_perf_summary | summary/3 | 12088 | 3 | 0.02% | 27.67 MiB | 15.34 Mbps | 15.129s | 19.606ms | 24.143ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan bidirectional remote sender remote UDP route | udp_perf_listen | - | 12091 | 0 | 0.00% | 13.84 MiB | 5.79 Mbps | 20.033s | - | - | - |
| lan bidirectional remote sender remote UDP route | udp_perf_listen | - | 12085 | 0 | 0.00% | 13.83 MiB | 6.01 Mbps | 19.308s | - | - | - |
| lan remote latency sender remote UDP route | udp_latency | 1/3 | 984 | 0 | 0.00% | 0.00 B | 0.00 bps | 5.144s | 10.454ms | 11.657ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan remote latency sender remote UDP route | udp_latency | 2/3 | 856 | 1 | 0.12% | 0.00 B | 0.00 bps | 5.004s | 10.520ms | 10.790ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan remote latency sender remote UDP route | udp_latency | 3/3 | 966 | 0 | 0.00% | 0.00 B | 0.00 bps | 5.001s | 10.353ms | 10.259ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan remote latency sender remote UDP route | udp_latency_summary | summary/3 | 2806 | 1 | 0.04% | 0.00 B | 0.00 bps | 15.149s | 10.439ms | 10.856ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan remote latency sender remote UDP route | udp_perf_listen | - | 2807 | 0 | 0.00% | 3.21 MiB | 1.37 Mbps | 19.603s | - | - | - |
| lan local latency sender local UDP route | udp_latency | 1/3 | 934 | 0 | 0.00% | 0.00 B | 0.00 bps | 5.106s | 10.931ms | 11.944ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan local latency sender local UDP route | udp_latency | 2/3 | 914 | 0 | 0.00% | 0.00 B | 0.00 bps | 5.004s | 10.948ms | 12.190ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan local latency sender local UDP route | udp_latency | 3/3 | 914 | 0 | 0.00% | 0.00 B | 0.00 bps | 5.005s | 10.948ms | 11.973ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan local latency sender local UDP route | udp_latency_summary | summary/3 | 2762 | 0 | 0.00% | 0.00 B | 0.00 bps | 15.114s | 10.942ms | 12.038ms | NWPathStatusSatisfied en0:NWInterfaceTypeWifi |
| lan local latency sender local UDP route | udp_perf_listen | - | 2762 | 0 | 0.00% | 3.16 MiB | 1.38 Mbps | 19.145s | - | - | - |
| thunderbolt remote perf sender remote UDP route | udp_perf | 1/3 | 49043 | 0 | 0.00% | 112.25 MiB | 188.26 Mbps | 5.002s | 1.596ms | 2.867ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt remote perf sender remote UDP route | udp_perf | 2/3 | 78124 | 0 | 0.00% | 178.81 MiB | 299.66 Mbps | 5.006s | 990.343us | 1.358ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt remote perf sender remote UDP route | udp_perf | 3/3 | 44497 | 0 | 0.00% | 101.85 MiB | 170.84 Mbps | 5.001s | 1.761ms | 3.995ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt remote perf sender remote UDP route | udp_perf_summary | summary/3 | 171664 | 0 | 0.00% | 392.91 MiB | 219.61 Mbps | 15.008s | 1.363ms | 2.614ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt remote perf sender remote UDP route | udp_perf_listen | - | 171664 | 0 | 0.00% | 196.45 MiB | 84.81 Mbps | 19.432s | - | - | - |
| thunderbolt local perf sender local UDP route | udp_perf | 1/3 | 85875 | 0 | 0.00% | 196.55 MiB | 329.56 Mbps | 5.003s | 814.905us | 1.334ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt local perf sender local UDP route | udp_perf | 2/3 | 81902 | 0 | 0.00% | 187.46 MiB | 314.48 Mbps | 5.000s | 858.309us | 1.617ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt local perf sender local UDP route | udp_perf | 3/3 | 70513 | 0 | 0.00% | 161.39 MiB | 270.75 Mbps | 5.000s | 996.243us | 1.977ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt local perf sender local UDP route | udp_perf_summary | summary/3 | 238290 | 0 | 0.00% | 545.40 MiB | 304.94 Mbps | 15.004s | 883.484us | 1.598ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt local perf sender local UDP route | udp_perf_listen | - | 238290 | 0 | 0.00% | 272.70 MiB | 121.09 Mbps | 18.891s | - | - | - |
| thunderbolt bidirectional remote sender remote UDP route | udp_perf | 1/3 | 94885 | 0 | 0.00% | 217.17 MiB | 363.88 Mbps | 5.007s | 746.264us | 1.190ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt bidirectional remote sender remote UDP route | udp_perf | 2/3 | 41249 | 0 | 0.00% | 94.41 MiB | 158.39 Mbps | 5.000s | 1.703ms | 4.051ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt bidirectional remote sender remote UDP route | udp_perf | 3/3 | 90223 | 0 | 0.00% | 206.50 MiB | 346.40 Mbps | 5.001s | 782.624us | 1.138ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt bidirectional remote sender remote UDP route | udp_perf_summary | summary/3 | 226357 | 0 | 0.00% | 518.09 MiB | 289.59 Mbps | 15.008s | 935.075us | 1.617ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt bidirectional remote sender remote UDP route | udp_perf | 1/3 | 49292 | 0 | 0.00% | 112.82 MiB | 189.22 Mbps | 5.002s | 1.584ms | 2.383ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt bidirectional remote sender remote UDP route | udp_perf | 2/3 | 22924 | 0 | 0.00% | 52.47 MiB | 88.00 Mbps | 5.001s | 3.436ms | 8.568ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt bidirectional remote sender remote UDP route | udp_perf | 3/3 | 53763 | 0 | 0.00% | 123.05 MiB | 206.42 Mbps | 5.001s | 1.449ms | 2.118ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt bidirectional remote sender remote UDP route | udp_perf_summary | summary/3 | 125979 | 0 | 0.00% | 288.34 MiB | 161.21 Mbps | 15.004s | 1.863ms | 3.455ms | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt bidirectional remote sender remote UDP route | udp_perf_listen | - | 125979 | 0 | 0.00% | 144.17 MiB | 60.12 Mbps | 20.118s | - | - | - |
| thunderbolt bidirectional remote sender remote UDP route | udp_perf_listen | - | 226357 | 0 | 0.00% | 259.05 MiB | 112.50 Mbps | 19.315s | - | - | - |
| thunderbolt remote latency sender remote UDP route | udp_latency | 1/3 | 50573 | 0 | 0.00% | 0.00 B | 0.00 bps | 5.000s | 197.654us | 318.542us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt remote latency sender remote UDP route | udp_latency | 2/3 | 58234 | 0 | 0.00% | 0.00 B | 0.00 bps | 5.000s | 171.631us | 253.042us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt remote latency sender remote UDP route | udp_latency | 3/3 | 65657 | 0 | 0.00% | 0.00 B | 0.00 bps | 5.000s | 152.222us | 204.583us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt remote latency sender remote UDP route | udp_latency_summary | summary/3 | 174464 | 0 | 0.00% | 0.00 B | 0.00 bps | 15.001s | 171.870us | 261.750us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt remote latency sender remote UDP route | udp_perf_listen | - | 174464 | 0 | 0.00% | 199.66 MiB | 87.02 Mbps | 19.247s | - | - | - |
| thunderbolt local latency sender local UDP route | udp_latency | 1/3 | 60370 | 0 | 0.00% | 0.00 B | 0.00 bps | 5.000s | 165.529us | 240.375us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt local latency sender local UDP route | udp_latency | 2/3 | 57315 | 0 | 0.00% | 0.00 B | 0.00 bps | 5.000s | 174.353us | 268.334us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt local latency sender local UDP route | udp_latency | 3/3 | 64038 | 0 | 0.00% | 0.00 B | 0.00 bps | 5.000s | 156.052us | 213.916us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt local latency sender local UDP route | udp_latency_summary | summary/3 | 181723 | 0 | 0.00% | 0.00 B | 0.00 bps | 15.001s | 164.972us | 241.125us | NWPathStatusSatisfied bridge0:NWInterfaceTypeWired |
| thunderbolt local latency sender local UDP route | udp_perf_listen | - | 181723 | 0 | 0.00% | 207.97 MiB | 92.50 Mbps | 18.861s | - | - | - |
| awdl remote perf sender remote UDP route | udp_perf | 1/3 | 7591 | 0 | 0.00% | 17.37 MiB | 29.07 Mbps | 5.015s | 10.421ms | 15.031ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl remote perf sender remote UDP route | udp_perf | 2/3 | 7500 | 0 | 0.00% | 17.17 MiB | 28.74 Mbps | 5.011s | 10.536ms | 15.347ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl remote perf sender remote UDP route | udp_perf | 3/3 | 7562 | 0 | 0.00% | 17.31 MiB | 28.99 Mbps | 5.008s | 10.442ms | 14.502ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl remote perf sender remote UDP route | udp_perf_summary | summary/3 | 22653 | 0 | 0.00% | 51.85 MiB | 28.93 Mbps | 15.033s | 10.466ms | 14.987ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl remote perf sender remote UDP route | udp_perf_listen | - | 22653 | 0 | 0.00% | 25.92 MiB | 11.28 Mbps | 19.282s | - | - | - |
| awdl local perf sender local UDP route | udp_perf | 1/3 | 5830 | 14 | 0.24% | 13.34 MiB | 22.36 Mbps | 5.007s | 11.124ms | 15.399ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl local perf sender local UDP route | udp_perf | 2/3 | 7535 | 0 | 0.00% | 17.25 MiB | 28.92 Mbps | 5.002s | 10.564ms | 14.491ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl local perf sender local UDP route | udp_perf | 3/3 | 7529 | 0 | 0.00% | 17.23 MiB | 28.88 Mbps | 5.005s | 10.590ms | 14.661ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl local perf sender local UDP route | udp_perf_summary | summary/3 | 20894 | 14 | 0.07% | 47.82 MiB | 26.72 Mbps | 15.014s | 10.730ms | 14.795ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl local perf sender local UDP route | udp_perf_listen | - | 20894 | 0 | 0.00% | 23.91 MiB | 10.74 Mbps | 18.683s | - | - | - |
| awdl bidirectional remote sender remote UDP route | udp_perf | 1/3 | 6410 | 6 | 0.09% | 14.67 MiB | 24.57 Mbps | 5.008s | 11.474ms | 16.597ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl bidirectional remote sender remote UDP route | udp_perf | 2/3 | 6823 | 0 | 0.00% | 15.62 MiB | 26.12 Mbps | 5.015s | 11.695ms | 16.554ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl bidirectional remote sender remote UDP route | udp_perf | 3/3 | 6562 | 0 | 0.00% | 15.02 MiB | 25.17 Mbps | 5.006s | 12.127ms | 17.453ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl bidirectional remote sender remote UDP route | udp_perf_summary | summary/3 | 19795 | 6 | 0.03% | 45.31 MiB | 25.29 Mbps | 15.029s | 11.767ms | 16.733ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl bidirectional remote sender remote UDP route | udp_perf | 1/3 | 6863 | 0 | 0.00% | 15.71 MiB | 26.26 Mbps | 5.018s | 11.500ms | 16.569ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl bidirectional remote sender remote UDP route | udp_perf | 2/3 | 6871 | 0 | 0.00% | 15.73 MiB | 26.35 Mbps | 5.006s | 11.480ms | 16.500ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl bidirectional remote sender remote UDP route | udp_perf | 3/3 | 6672 | 0 | 0.00% | 15.27 MiB | 25.59 Mbps | 5.006s | 11.834ms | 16.598ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl bidirectional remote sender remote UDP route | udp_perf_summary | summary/3 | 20406 | 0 | 0.00% | 46.71 MiB | 26.07 Mbps | 15.030s | 11.602ms | 16.562ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl bidirectional remote sender remote UDP route | udp_perf_listen | - | 20406 | 0 | 0.00% | 23.35 MiB | 9.46 Mbps | 20.706s | - | - | - |
| awdl bidirectional remote sender remote UDP route | udp_perf_listen | - | 19795 | 0 | 0.00% | 22.65 MiB | 9.60 Mbps | 19.786s | - | - | - |
| awdl remote latency sender remote UDP route | udp_latency | 1/3 | 1062 | 0 | 0.00% | 0.00 B | 0.00 bps | 5.005s | 9.423ms | 23.591ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl remote latency sender remote UDP route | udp_latency | 2/3 | 1600 | 0 | 0.00% | 0.00 B | 0.00 bps | 5.005s | 6.253ms | 9.231ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl remote latency sender remote UDP route | udp_latency | 3/3 | 1600 | 0 | 0.00% | 0.00 B | 0.00 bps | 5.002s | 6.251ms | 9.275ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl remote latency sender remote UDP route | udp_latency_summary | summary/3 | 4262 | 0 | 0.00% | 0.00 B | 0.00 bps | 15.012s | 7.042ms | 12.544ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl remote latency sender remote UDP route | udp_perf_listen | - | 4262 | 0 | 0.00% | 4.88 MiB | 2.12 Mbps | 19.306s | - | - | - |
| awdl local latency sender local UDP route | udp_latency | 1/3 | 1442 | 0 | 0.00% | 0.00 B | 0.00 bps | 5.016s | 6.947ms | 12.019ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl local latency sender local UDP route | udp_latency | 2/3 | 1086 | 0 | 0.00% | 0.00 B | 0.00 bps | 5.005s | 9.213ms | 25.701ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl local latency sender local UDP route | udp_latency | 3/3 | 1593 | 0 | 0.00% | 0.00 B | 0.00 bps | 5.014s | 6.287ms | 8.785ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl local latency sender local UDP route | udp_latency_summary | summary/3 | 4121 | 0 | 0.00% | 0.00 B | 0.00 bps | 15.034s | 7.289ms | 11.907ms | NWPathStatusSatisfied awdl0:NWInterfaceTypeWifi |
| awdl local latency sender local UDP route | udp_perf_listen | - | 4121 | 0 | 0.00% | 4.72 MiB | 2.10 Mbps | 18.844s | - | - | - |
