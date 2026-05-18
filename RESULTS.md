# Results

This table records the current answers and observed outputs from this demo.
Local perf rows are same-host socket checks unless the command says
`udp-perf-send`/`udp-perf-listen` across two Macs.

| Topic | Answer | Command or output | Caveat |
| --- | --- | --- | --- |
| Repo location | Demo lives in `~/go/src/github.com/tmc/awdl-webrtc-apple-demo`. | Code and module files were committed first; README and results docs are layered separately. | The module uses a local `github.com/tmc/apple` replace. |
| WebRTC support | Yes, using Pion WebRTC with interface-constrained ICE. | Modes: `check`, `gather`, `pair`. | This is not a Network.framework-native Pion transport. |
| Pion changes | No Pion patch or fork was needed. | Uses `SettingEngine.SetInterfaceFilter`, `SetIPFilter`, `SetNetworkTypes`, `SetICEMulticastDNSMode`, and `SetICEUDPMux`. | A deeper backend would be needed only for Network.framework-native ICE sockets. |
| Pluggable backend shape | The demo is structured so the socket policy is outside Pion and supplied through a UDP mux. | `newLinkUDPMux` creates the constrained `net.UDPConn`; Pion receives it through `SetICEUDPMux`. | It is a prototype seam, not a full backend registry yet. |
| Apple public policy | Public Network.framework parameters can be set for both links. | AWDL: `include_peer_to_peer=true required_interface_type=NWInterfaceTypeWifi`; Thunderbolt: `include_peer_to_peer=false required_interface_type=NWInterfaceTypeWired`. | These are policy checks next to the Go UDP transport. |
| Apple private policy | Private `network` bindings expose the knobs we wanted. | Output includes `required_interface=awdl0`, `use_awdl=true`, `use_p2p=true`, `allow_socket_access=true`, `reuse_local_address=true`, `prohibit_fallback=true`, `valid=true`. | Uses local `replace github.com/tmc/apple => /Users/tmc/go/src/github.com/tmc/apple`. |
| AWDL WebRTC discovery | AWDL ICE gather works. | `go run . -profile awdl -mode gather -timeout 10s` produced two mDNS host candidates from an `awdl0` UDP mux. | Full AWDL datachannel still needs a second Apple peer and signaling. |
| Thunderbolt discovery | Thunderbolt Bridge ICE gather works. | `go run . -profile thunderbolt -mode gather -timeout 10s` produced two mDNS host candidates from `bridge0`. | Uses default `bridge0`; pass `-iface` to override. |
| Thunderbolt WebRTC pair | Local Thunderbolt-constrained datachannel works. | `go run . -profile thunderbolt -mode pair -timeout 12s` ended with `webrtc datachannel opened and exchanged payload over bridge0-constrained ICE`. | Same-host proof, not a cable-to-peer throughput test. |
| UDP over AWDL | Yes, direct UDP over AWDL works with interface-bound sockets. | `go run . -profile awdl -mode udp -timeout 10s` echoed `ping` over `[fe80::...%awdl0]` with `server_bound_if=awdl0(16) client_bound_if=awdl0(16)`. | A cold AWDL run can time out; `gather` activates the path, and a real proof should use two Macs. |
| Darwin socket requirement | Binding only to the AWDL address is not enough. | Code sets `IPV6_BOUND_IF` for AWDL/scoped IPv6 and `IP_BOUND_IF` where needed. | This is separate from the Network.framework policy objects. |
| Iperf-like output | Added request/echo UDP perf modes. | `udp-perf`, `udp-perf-listen`, `udp-perf-send` print interval, transfer, bitrate, datagrams, omitted warmup packets, and RTT summary. | It is a smoke benchmark, not a full `iperf3` replacement. |
| AWDL UDP local sample | AWDL local sample shows the output format and constrained socket path. | `[  5] 0.00-0.09 sec  2.29 MBytes  221.70 Mbits/sec  1000  5  40.583us/86.499us/68.583us/155.292us/4.948ms` | Same-host sample after `gather`; measure two Macs for real link speed. |
| Thunderbolt UDP local sample | Thunderbolt local sample is faster on this machine. | `[  5] 0.00-0.07 sec  2.29 MBytes  283.48 Mbits/sec  1000  5  31.709us/67.606us/59.125us/138.208us/676.958us` | Same-host sample, not peer-to-peer cable throughput. |
| Two-host AWDL UDP proof | Listener/sender modes are ready. | Listener: `go run . -profile awdl -mode udp-perf-listen -timeout 60s`; sender: `go run . -profile awdl -mode udp-perf-send -peer '[fe80::peer%awdl0]:12345' -count 1000 -size 1200 -warmup 5 -timeout 20s`. | Use the listener's printed scoped address as the sender peer. |
| Build gate | The module builds. | `go test ./...` passes with `[no test files]`. | The local `github.com/tmc/apple` checkout is dirty and used through `replace`. |

## Sample Perf Output

AWDL:

```text
profile=awdl interface=awdl0 index=16 flags=up|broadcast|multicast|running ips=fe80::bcb7:e5ff:fe2d:13b4
udp perf server=[fe80::bcb7:e5ff:fe2d:13b4%awdl0]:49533 client=[fe80::bcb7:e5ff:fe2d:13b4%awdl0]:53198 network=udp6 server_bound_if=awdl0(16) client_bound_if=awdl0(16)
[ ID] Interval           Transfer     Bitrate         Datagrams  Omit  RTT min/avg/p50/p95/max
[  5] 0.00-0.09    sec  2.29 MBytes  221.70 Mbits/sec       1000     5  40.583us/86.499us/68.583us/155.292us/4.948ms
```

Thunderbolt:

```text
profile=thunderbolt interface=bridge0 index=15 flags=up|broadcast|multicast|running ips=169.254.7.165,fe80::1cd0:ead:d44a:d885
udp perf server=169.254.7.165:56937 client=169.254.7.165:56315 network=udp4 server_bound_if=none client_bound_if=none
[ ID] Interval           Transfer     Bitrate         Datagrams  Omit  RTT min/avg/p50/p95/max
[  5] 0.00-0.07    sec  2.29 MBytes  283.48 Mbits/sec       1000     5  31.709us/67.606us/59.125us/138.208us/676.958us
```
