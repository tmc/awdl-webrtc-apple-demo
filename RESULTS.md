# Results

This is the at-a-glance record of the demo answers and observed outputs.

Scope terms:

- `local`: same-host socket check through the selected interface.
- `remote`: two Apple hosts; this is required for real AWDL or Thunderbolt link speed.
- `policy`: Apple Network.framework configuration was created and read back, but Pion still uses Go UDP sockets.

## Result Matrix

| Question / Area | Status | Answer | Evidence | Caveat / Next |
| --- | --- | --- | --- | --- |
| Repo location | PASS | The demo is in `~/go/src/github.com/tmc/awdl-webrtc-apple-demo`. | Commits `fceea2e` and `0a68c9e`. | Uses a local `github.com/tmc/apple` replace. |
| Build | PASS | The module builds. | `go test ./...` -> `[no test files]`. | Darwin-only demo. |
| WebRTC support | PASS | Yes, via Pion WebRTC with constrained ICE. | Modes: `check`, `gather`, `pair`. | Not a Network.framework-native WebRTC transport. |
| Pion changes | PASS | No Pion fork or patch was needed. | Uses `SettingEngine` filters, mDNS mode, and `SetICEUDPMux`. | Native Network.framework ICE would require a deeper backend. |
| Pluggable backend shape | PROTOTYPE | The socket policy is outside Pion and supplied through a UDP mux. | `newLinkUDPMux` creates the interface-constrained `net.UDPConn`. | This is a seam, not a full backend registry. |
| Apple public policy | PASS | Public Network.framework parameters work for both profiles. | AWDL: `include_peer_to_peer=true`, Wi-Fi. Thunderbolt: wired. | Policy sits next to the Go UDP transport. |
| Apple private policy | PASS | Private `network` knobs expose the required controls. | `required_interface`, `use_awdl`, `use_p2p`, `allow_socket_access`, `reuse_local_address`, `prohibit_fallback`. | Depends on local `tmc/apple/private/network`. |
| AWDL discovery | PASS | AWDL ICE gathering works on `awdl0`. | `gathered 2 host candidate(s) from awdl0-bound UDP mux`. | Candidates are mDNS host candidates. |
| AWDL WebRTC datachannel | REMOTE NEEDED | Discovery works; full datachannel proof needs a second Apple peer. | Local gather passes. | Add remote signaling for an end-to-end AWDL WebRTC proof. |
| AWDL UDP | PASS | Direct UDP over AWDL works with interface-bound sockets. | Echo over `[fe80::...%awdl0]` with `server_bound_if=awdl0(16) client_bound_if=awdl0(16)`. | Cold AWDL can time out; `gather` activates the path. |
| Thunderbolt discovery | PASS | Thunderbolt Bridge ICE gathering works. | `gathered 2 host candidate(s) from bridge0-bound UDP mux`. | Default is `bridge0`; override with `-iface`. |
| Thunderbolt WebRTC datachannel | PASS | Local Thunderbolt-constrained datachannel works. | `webrtc datachannel opened and exchanged payload over bridge0-constrained ICE`. | Same-host proof, not cable throughput. |
| UDP perf output | PASS | The demo prints iperf-like UDP summaries. | Columns: interval, transfer, bitrate, datagrams, omit, RTT min/avg/p50/p95/max. | Smoke benchmark, not an `iperf3` replacement. |
| AWDL local perf | PASS | Local AWDL sample produced `221.70 Mbits/sec`. | 1000 datagrams, 1200-byte payload, 5 warmup packets omitted. | Same-host sample after AWDL activation. |
| Thunderbolt local perf | PASS | Local Thunderbolt sample produced `283.48 Mbits/sec`. | 1000 datagrams, 1200-byte payload, 5 warmup packets omitted. | Same-host sample, not peer-to-peer cable speed. |
| Two-host UDP proof | READY | Listener/sender modes are implemented. | `udp-listen`, `udp-send`, `udp-perf-listen`, `udp-perf-send`. | Use printed scoped peer address on the sender. |

## Command Matrix

| Goal | Command | Observed Result |
| --- | --- | --- |
| Build gate | `go test ./...` | `? github.com/tmc/awdl-webrtc-apple-demo [no test files]` |
| AWDL policy check | `go run . -profile awdl -mode check` | Prints AWDL profile, public Wi-Fi peer-to-peer policy, and private `use_awdl=true use_p2p=true`. |
| AWDL WebRTC gather | `go run . -profile awdl -mode gather -timeout 10s` | Two mDNS host candidates from an `awdl0` UDP mux. |
| AWDL UDP echo | `go run . -profile awdl -mode udp -timeout 10s` | Echoed `ping` over scoped IPv6 on `awdl0`. |
| AWDL local perf | `go run . -profile awdl -mode udp-perf -count 1000 -size 1200 -warmup 5 -timeout 30s` | `2.29 MBytes`, `221.70 Mbits/sec`, average RTT `86.499us`. |
| Thunderbolt policy check | `go run . -profile thunderbolt -mode check` | Prints wired policy and private `required_interface=bridge0`. |
| Thunderbolt WebRTC gather | `go run . -profile thunderbolt -mode gather -timeout 10s` | Two mDNS host candidates from a `bridge0` UDP mux. |
| Thunderbolt WebRTC pair | `go run . -profile thunderbolt -mode pair -timeout 12s` | Datachannel opened and exchanged payload over `bridge0`-constrained ICE. |
| Thunderbolt local perf | `go run . -profile thunderbolt -mode udp-perf -count 1000 -size 1200 -warmup 5 -timeout 30s` | `2.29 MBytes`, `283.48 Mbits/sec`, average RTT `67.606us`. |
| Two-host AWDL echo listener | `go run . -profile awdl -mode udp-listen -timeout 60s` | Prints a scoped listener address for the peer. |
| Two-host AWDL echo sender | `go run . -profile awdl -mode udp-send -peer '[fe80::peer%awdl0]:12345' -timeout 10s` | Sends one payload and waits for echo. |
| Two-host AWDL perf listener | `go run . -profile awdl -mode udp-perf-listen -timeout 60s` | Echoes perf datagrams and prints ingress summary when done. |
| Two-host AWDL perf sender | `go run . -profile awdl -mode udp-perf-send -peer '[fe80::peer%awdl0]:12345' -count 1000 -size 1200 -warmup 5 -timeout 20s` | Prints iperf-like transfer, bitrate, datagrams, omit, and RTT summary. |

## Performance Snapshot

| Link | Scope | Transfer | Bitrate | Datagrams | Omit | RTT min | RTT avg | RTT p50 | RTT p95 | RTT max |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| AWDL `awdl0` | local | 2.29 MBytes | 221.70 Mbits/sec | 1000 | 5 | 40.583us | 86.499us | 68.583us | 155.292us | 4.948ms |
| Thunderbolt `bridge0` | local | 2.29 MBytes | 283.48 Mbits/sec | 1000 | 5 | 31.709us | 67.606us | 59.125us | 138.208us | 676.958us |

## Sample Output

AWDL local perf:

```text
profile=awdl interface=awdl0 index=16 flags=up|broadcast|multicast|running ips=fe80::bcb7:e5ff:fe2d:13b4
udp perf server=[fe80::bcb7:e5ff:fe2d:13b4%awdl0]:49533 client=[fe80::bcb7:e5ff:fe2d:13b4%awdl0]:53198 network=udp6 server_bound_if=awdl0(16) client_bound_if=awdl0(16)
[ ID] Interval           Transfer     Bitrate         Datagrams  Omit  RTT min/avg/p50/p95/max
[  5] 0.00-0.09    sec  2.29 MBytes  221.70 Mbits/sec       1000     5  40.583us/86.499us/68.583us/155.292us/4.948ms
```

Thunderbolt local perf:

```text
profile=thunderbolt interface=bridge0 index=15 flags=up|broadcast|multicast|running ips=169.254.7.165,fe80::1cd0:ead:d44a:d885
udp perf server=169.254.7.165:56937 client=169.254.7.165:56315 network=udp4 server_bound_if=none client_bound_if=none
[ ID] Interval           Transfer     Bitrate         Datagrams  Omit  RTT min/avg/p50/p95/max
[  5] 0.00-0.07    sec  2.29 MBytes  283.48 Mbits/sec       1000     5  31.709us/67.606us/59.125us/138.208us/676.958us
```

## Boundaries

| Boundary | Meaning |
| --- | --- |
| Same-host perf is not link throughput | The local samples prove the constrained socket path and output format, not real radio/cable speed. |
| AWDL can be demand-activated | A cold UDP run can time out; running `gather` first activates the AWDL path on this host. |
| AWDL WebRTC datachannel still needs a peer | ICE gathering works locally, but end-to-end WebRTC over AWDL needs another Apple host and signaling. |
| Network.framework is policy-only here | Pion still uses Go UDP sockets; `github.com/tmc/apple` proves policy/private knobs beside that path. |
