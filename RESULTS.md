# Results

This is the at-a-glance record of the demo answers and observed outputs.

Scope terms:

- `local`: same-host socket check through the selected interface.
- `remote`: two Apple hosts; this is required for real AWDL or Thunderbolt link speed.
- `policy`: Apple Network.framework configuration was created and read back in `check` mode.
- `network backend`: the demo's `-backend network` path.
- `pion-net`: `-backend network -pion-net`, which passes Network.framework UDP listeners to Pion through `transport.Net`.

## Result Matrix

| Question / Area | Status | Answer | Evidence | Caveat / Next |
| --- | --- | --- | --- | --- |
| Repo location | PASS | The demo is in `~/go/src/github.com/tmc/awdl-webrtc-apple-demo`. | Productization includes reusable packet and Pion transport packages plus docs/audit tables. | Uses a local replace for sibling `tmc/apple-pion` until that module has a release or remote. |
| Build | PASS | The module builds and the focused tests pass. | `go test ./...`; `go vet ./...`; `git diff --check`. | Darwin-only demo. |
| WebRTC support | PASS | Yes, via Pion WebRTC with constrained ICE. | Modes: `check`, `gather`, `pair`, `answer-stdio`, `offer-ssh`. | `offer-ssh` is explicit demo signaling, not a general signaling service. |
| Pion changes | PASS | No Pion fork or patch was needed. | Uses `SettingEngine` filters, mDNS mode, `SetICEUDPMux`, `SetNet`, explicit signaling, and `apple-pion/icepolicy`. | AWDL raw-candidate publication is now explicit candidate signaling; fresh two-host rerun is pending while SSH is unreachable. |
| Pluggable backend shape | PASS | The CLI has `-backend go\|network`; WebRTC can use either an ICE UDP mux or `-pion-net`. | `go test ./...`; `-backend network` gather, UDP perf, remote WebRTC, and `-pion-net` LAN/Thunderbolt/AWDL remote WebRTC pass. | Release placement and remote candidate reruns remain. |
| Reusable packet package | PASS | The Network.framework PacketConn lives at `github.com/tmc/apple/x/network/nwpacket`, and the demo uses released `github.com/tmc/apple v0.6.7`. | `tmc/apple` release `v0.6.7` resolves from the module cache; `go test ./x/network/nwpacket` and `go vet ./x/network/nwpacket` pass locally. | No local `tmc/apple` replace remains. |
| Pion `transport.Net` package | PASS | `github.com/tmc/apple-pion/nwtransport` implements the Pion `transport.Net` UDP surface needed by ICE, and the demo imports it. | `tmc/apple-pion` commits `d2698d5`, `167087c`, `3ea6faf`, and `7d4aced` add package docs, tests, an example, UDP dials, concrete UDP listens, configured wildcard UDP listens, the current `x/network/nwpacket` import path, and test-covered fallback policy; LAN, Thunderbolt, and AWDL `-pion-net -mdns disabled -raw-candidates offer-ssh` opened datachannels and exchanged `ping`/`pong`. | DNS, TCP, unconstrained wildcard UDP, TURN/STUN helper traffic outside the selected UDP surface, and unsupported UDP cases intentionally fall back to Pion's standard network. |
| Explicit ICE policy | PARTIAL | AWDL host-candidate handling is factored into `github.com/tmc/apple-pion/icepolicy`, and raw candidates are signaled separately from unmodified SDP. | `apple-pion/icepolicy` has package docs, tests, and examples for synthetic host-candidate publication and `ICECandidateInit` extraction; AWDL raw-candidate gather passes. | Fresh two-host AWDL/Thunderbolt/LAN rerun is pending while `tmc2@10.0.18.249` is unreachable. |
| Apple public policy | PASS | Public Network.framework parameters work for all profiles. | AWDL: `include_peer_to_peer=true`, Wi-Fi. Thunderbolt: wired. LAN: Wi-Fi. | The clear UDP backend applies these through `NWParametersCreate` plus `NWUDPCreateOptions`. |
| Apple private policy | PASS | Private knobs are reachable without importing the broken local generated private package. | Raw Objective-C probes report `required_interface`, `use_awdl`, `use_p2p`, `allow_socket_access`, `reuse_local_address`, `prohibit_fallback`. | Exact `NWInterface.cInterface` is enabled for AWDL only; Thunderbolt uses the link-local bridge address. |
| Network.framework clear UDP | PASS | The backend must use `NWParametersCreate` plus `NWUDPCreateOptions`; `NWParametersCreateSecureUDP(nil, nil)` attempted DTLS and failed. | Trace showed `NWErrorDomainTLS` before the clear UDP change; clear UDP then reached `NWConnectionStateReady`. | This is documented in code, not a Pion change. |
| Network.framework Pion gather | PASS | ICE gathering works over both the Network.framework PacketConn mux and the Pion `transport.Net` adapter. | LAN `-pion-net` gathered 10 mDNS candidates; AWDL `-pion-net -mdns disabled -raw-candidates` gathered 2 raw `fe80::...` candidates from `awdl0`. | AWDL mDNS gathers but did not open a remote datachannel here. |
| Network.framework readiness retry | PASS | The demo defaults to `nwpacket.Config.ConnectTimeout=2s` and `ConnectRetries=2` for Network.framework UDP and Pion `transport.Net` PacketConns, with CLI and matrix overrides. | `tmc/apple v0.6.7` adds outbound peer eviction/recreate after readiness timeout; demo flags `-nw-connect-timeout` and `-nw-connect-retries`; matrix envs `NW_CONNECT_TIMEOUT` and `NW_CONNECT_RETRIES`; `go test ./x/network/nwpacket`; demo `go test ./...`. | Two-host rerun is still needed to see whether this clears the old local-to-remote AWDL/Thunderbolt readiness asymmetry. |
| Network.framework remote WebRTC | PASS | Remote Pion datachannel exchange works over Network.framework PacketConn. | LAN, Thunderbolt, and AWDL `offer-ssh` each opened a datachannel and exchanged `ping`/`pong`. | AWDL requires `-mdns disabled -raw-candidates`. |
| Network.framework Pion-native remote WebRTC | PASS | Remote WebRTC through `SettingEngine.SetNet` works for LAN, Thunderbolt, and AWDL. | LAN, Thunderbolt, and AWDL `-pion-net -mdns disabled -raw-candidates offer-ssh` opened datachannels and exchanged `ping`/`pong`. | AWDL still depends on explicit raw candidate publication. |
| Network.framework remote UDP | PARTIAL | Multi-datagram echo works in the verified directions, but direct UDP remains asymmetric. | LAN local-to-remote passed; Thunderbolt and AWDL remote-to-local passed; Thunderbolt and AWDL local-to-remote listeners received zero datagrams in the latest clean run. Callback probe modes now distinguish request arrival from reverse callback delivery. | WebRTC succeeds despite this raw UDP listener asymmetry; callback probes still need two-host validation. |
| AWDL discovery | PASS | AWDL ICE gathering works on `awdl0`. | `gathered 2 host candidate(s) from awdl0-bound UDP mux`. | Candidates are mDNS host candidates. |
| AWDL WebRTC datachannel | PASS | Remote datachannel exchange works over Network.framework on `awdl0`. | Historical `go run . -profile awdl -backend network -mdns disabled -raw-candidates -mode offer-ssh ...` opened and exchanged payload with `tmc2@10.0.18.249`; current code preserves raw gather with explicit candidate signaling. | Fresh remote rerun of the new wire format is pending. |
| AWDL UDP | PASS | Direct UDP over AWDL works with interface-bound sockets. | Echo over `[fe80::...%awdl0]` with `server_bound_if=awdl0(16) client_bound_if=awdl0(16)`. | Cold AWDL can time out; `gather` activates the path. |
| AWDL remote UDP perf | PASS | Remote-to-local AWDL UDP completed. | Latest Network.framework sample: 10 datagrams, `2.61 Mbits/sec`, avg RTT `7.360ms`. | Local-to-remote Network.framework listener direction timed out before the first write became ready. |
| AWDL Network.framework UDP | PARTIAL | Network.framework can send raw UDP perf traffic over AWDL in the remote-to-local direction. | Remote-to-local: 10 datagrams, zero loss, `2.61 Mbits/sec`; local-to-remote: 0/10 datagrams, readiness timeout. | Smoke-level runs only; direction asymmetry remains. |
| Thunderbolt discovery | PASS | Thunderbolt ICE gathering works through the first usable Thunderbolt interface. | Historical bridge sample: `gathered 2 host candidate(s) from bridge0-bound UDP mux`; current no-bridge-address sample falls back to `en1` and gathers 2 Network.framework candidates. | Override with `-iface` when you need an exact member or bridge interface. |
| Thunderbolt WebRTC datachannel | PASS | Local and remote Thunderbolt-constrained datachannels work. | Local `pair`; remote `offer-ssh` with `tmc2@10.0.18.249` over `bridge0`. | Remote proof uses explicit SSH signaling. |
| Thunderbolt remote UDP perf | PASS | Remote-to-local Thunderbolt UDP completed. | Latest Network.framework sample: 20 datagrams, `11.25 Mbits/sec`, avg RTT `1.706ms`. | Local-to-remote UDP listener saw zero datagrams in both Go and Network.framework backend checks. |
| Thunderbolt Network.framework UDP | PARTIAL | Network.framework can send raw UDP perf traffic over Thunderbolt Bridge in the remote-to-local direction. | Remote-to-local: 20 datagrams, zero loss, `11.25 Mbits/sec`; local-to-remote: 0/20 datagrams, readiness timeout. | Required interface type is omitted; the link-local bridge address selects `bridge0`. |
| LAN remote UDP perf | PARTIAL | LAN UDP is reachable in both directions, but sustained runs are unstable here. | Remote-to-local 20 datagrams at `755.17 Kbits/sec`; Network.framework local-to-remote 10 datagrams at `1.12 Mbits/sec`, zero loss. | Longer LAN runs still need repeated trials. |
| LAN Network.framework UDP | PASS | Network.framework can send raw UDP perf traffic over LAN in both directions. | Remote-to-local 20 datagrams, `753.72 Kbits/sec`; local-to-remote 10 datagrams, `1.12 Mbits/sec`, zero loss. | LAN remains slower/less stable than Thunderbolt in this environment. |
| UDP perf output | PASS | The demo prints iperf-like UDP summaries with loss columns, repeated trials, aggregate trial summaries, optional JSON records, sender-side timeout loss accounting, fixed-duration trials, concurrent streams, explicit latency-only output, observed Network.framework paths, and a bounded in-flight window. | Columns: interval, transfer, bitrate, datagrams, lost, loss, omit, RTT min/avg/p50/p95/max; `-trials` repeats runs and prints summaries; `-duration` runs each trial for a fixed time and records `duration_ns`; `-streams` opens multiple client PacketConns and records `streams`; `udp-latency`/`udp-latency-send` emit `udp_latency` JSON records; `-perf-json` includes `paths` for Network.framework peer connections when available; `-require-path-interface` and `-forbid-loopback-path` fail closed; `-window` enables pipelined echo requests; `-packet-timeout` bounds each echo wait. | Same-host Network.framework sends can now reach readiness and report path JSON, but echo delivery is still weak on this host; use the two-host matrix for real link evidence. Full-duplex mode is still future work. |
| AWDL local perf | PASS | Local AWDL sample produced `221.70 Mbits/sec`. | 1000 datagrams, 1200-byte payload, 5 warmup packets omitted. | Same-host sample after AWDL activation. |
| Thunderbolt local perf | PASS | Local Thunderbolt sample produced `283.48 Mbits/sec`. | 1000 datagrams, 1200-byte payload, 5 warmup packets omitted. | Same-host sample, not peer-to-peer cable speed. |
| Two-host UDP proof | READY | Listener/sender and callback probe modes are implemented. | `udp-listen`, `udp-send`, `udp-callback-listen`, `udp-callback-request`, `udp-perf-listen`, `udp-perf-send`. | Use printed scoped peer address on the sender. |

## Current Live State

| Check | Result |
| --- | --- |
| Local Thunderbolt default | With `bridge0` addressless, `go run . -profile thunderbolt -backend network -mode check -timeout 3s` selected `en1` at `172.31.253.1`. |
| Local Thunderbolt gather | `go run . -profile thunderbolt -backend network -mode gather -timeout 5s` gathered 2 mDNS candidates from an `en1` Network.framework UDP mux. |
| Remote SSH | `ssh -o ConnectTimeout=5 -o BatchMode=yes tmc2@10.0.18.249 ...` timed out, so fresh two-host Thunderbolt/AWDL/LAN reruns are gated on remote reachability. |
| Remote diagnostics | `RUN_REMOTE=0 PROFILES=lan OUTPUT=/tmp/awdl-webrtc-diagnostics-smoke.txt scripts/remote-diagnostics.sh` | Local smoke captured hostname, OS, `ifconfig`, IPv4/IPv6 routes, route-to-peer, `scutil --nwi`, hardware ports, and UDP sockets. With `RUN_REMOTE=1`, the same script collects the remote side over SSH before the productization matrix. |

## Host Matrix

| Host | Access | LAN `en0` | Thunderbolt `bridge0` | AWDL `awdl0` |
| --- | --- | --- | --- | --- |
| Local | shell | `10.0.199.147` | historical `bridge0` `169.254.61.91`; current fallback `en1` `172.31.253.1` | `fe80::cd4:b4ff:fe63:bc03%awdl0` |
| Remote | `ssh tmc2@10.0.18.249` | `10.0.18.249` | `169.254.88.35` | `fe80::9477:6dff:fe11:6a55%awdl0` |

## Command Matrix

| Goal | Command | Observed Result |
| --- | --- | --- |
| Build gate | `go test ./...`; `go vet ./...`; `git diff --check` | Demo tests pass; vet and whitespace checks pass. |
| Productized packages | In `tmc/apple`: `go test ./x/network/nwpacket`; in `tmc/apple-pion`: `go test ./...` | Promoted `nwpacket`, `nwtransport`, and `icepolicy` tests pass. |
| Network backend build pin | `go list -m -f '{{.Path}} {{.Version}} {{.Replace.Path}}' github.com/tmc/apple github.com/tmc/apple-pion` | `github.com/tmc/apple v0.6.7`; `github.com/tmc/apple-pion v0.0.0 ../apple-pion`. |
| Release preflight | `scripts/release-preflight.sh` | Local gates and package availability pass; preflight currently fails because the demo and `apple-pion` have no remotes and the demo still uses a local `replace` for unreleased `apple-pion`. |
| Remote productization matrix | `SSH_TARGET=tmc2@10.0.18.249 scripts/remote-matrix.sh` | Builds and copies the demo, then runs LAN/Thunderbolt/AWDL `-pion-net` WebRTC plus Network.framework UDP perf in both directions. `scripts/remote-diagnostics.sh` captures route/interface state before the run; `NW_CONNECT_TIMEOUT` and `NW_CONNECT_RETRIES` tune Network.framework readiness retry for every local and remote demo command. Currently blocked because SSH to `tmc2@10.0.18.249` times out. |
| Network LAN gather | `go run . -profile lan -backend network -mode gather -timeout 8s` | Two mDNS host candidates from an `en0` Network.framework UDP mux. |
| Pion-native LAN gather | `AWDL_DEMO_NETWORK_TRACE=1 go run . -profile lan -backend network -pion-net -mode gather -timeout 12s` | Ten mDNS host candidates; trace showed Network.framework listeners for each selected `en0` address. |
| Pion-native LAN pair | `go run . -profile lan -backend network -pion-net -mode pair -timeout 15s` | Same-host datachannel opened and exchanged payload over `en0` through Pion `transport.Net`. |
| Current Thunderbolt fallback | `go run . -profile thunderbolt -backend network -mode gather -timeout 5s` | With `bridge0` addressless, default selection fell back to `en1` and gathered two mDNS host candidates. |
| Network Thunderbolt gather | `go run . -profile thunderbolt -backend network -mode gather -timeout 8s` | Two mDNS host candidates from a `bridge0` Network.framework UDP mux. |
| Network AWDL gather | `go run . -profile awdl -backend network -mode gather -timeout 12s` | Two mDNS host candidates from an `awdl0` Network.framework UDP mux. |
| Pion-native AWDL gather | `AWDL_DEMO_NETWORK_TRACE=1 go run . -profile awdl -backend network -pion-net -mode gather -timeout 15s` | Two mDNS host candidates from `awdl0`; trace showed a Network.framework listener on `[fe80::...%awdl0]`. |
| Network AWDL raw-candidate gather | `go run . -profile awdl -backend network -mdns disabled -raw-candidates -mode gather -timeout 10s` | Two raw `fe80::...` host candidates from an `awdl0` Network.framework UDP mux. |
| Pion-native AWDL raw-candidate gather | `AWDL_DEMO_NETWORK_TRACE=1 go run . -profile awdl -backend network -pion-net -mdns disabled -raw-candidates -mode gather -timeout 15s` | Two raw `fe80::...` candidates from `awdl0`; trace showed a Network.framework listener on `[fe80::...%awdl0]`. |
| Network LAN remote WebRTC | `go run . -profile lan -backend network -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -raw-candidates -timeout 30s` | Datachannel opened and exchanged `ping`/`pong` with the remote answer process. |
| Pion-native LAN remote WebRTC | `go run . -profile lan -backend network -pion-net -mdns disabled -raw-candidates -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 35s` | Datachannel opened and exchanged `ping`/`pong` through `SettingEngine.SetNet`. |
| Network Thunderbolt remote WebRTC | `go run . -profile thunderbolt -backend network -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -raw-candidates -timeout 35s` | Datachannel opened and exchanged `ping`/`pong` over `bridge0`-constrained ICE. |
| Pion-native Thunderbolt remote WebRTC | `go run . -profile thunderbolt -backend network -pion-net -mdns disabled -raw-candidates -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 40s` | Datachannel opened and exchanged `ping`/`pong` over `bridge0` through `SettingEngine.SetNet`. |
| Network AWDL remote WebRTC | `go run . -profile awdl -backend network -mdns disabled -raw-candidates -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 45s` | Datachannel opened and exchanged `ping`/`pong` over `awdl0`-constrained ICE. |
| Pion-native AWDL remote WebRTC | `go run . -profile awdl -backend network -pion-net -mdns disabled -raw-candidates -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 45s` | Datachannel opened and exchanged `ping`/`pong` over `awdl0` through `SettingEngine.SetNet`. |
| Pion-native AWDL mDNS boundary | `go run . -profile awdl -backend network -pion-net -mode offer-ssh -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 35s` | Both sides gathered `awdl0` mDNS candidates in earlier runs, but the datachannel timed out waiting to open. |
| Network LAN remote perf listener | Local: `/tmp/awdl-webrtc-apple-demo-bin -profile lan -backend network -mode udp-perf-listen -count 20 -warmup 0 -timeout 30s` | Listener received 20 datagrams. |
| Network LAN remote perf sender | Remote: `ssh tmc2@10.0.18.249 '/tmp/awdl-webrtc-apple-demo-bin -profile lan -backend network -mode udp-perf-send -peer 10.0.199.147:51491 -count 20 -warmup 0 -size 1200 -timeout 30s'` | `46.88 KBytes`, `753.72 Kbits/sec`, avg RTT `25.473ms`. |
| Network Thunderbolt remote perf listener | Local: `/tmp/awdl-webrtc-apple-demo-bin -profile thunderbolt -backend network -mode udp-perf-listen -count 20 -warmup 0 -timeout 30s` | Listener received 20 datagrams. |
| Network Thunderbolt remote perf sender | Remote: `ssh tmc2@10.0.18.249 '/tmp/awdl-webrtc-apple-demo-bin -profile thunderbolt -backend network -mode udp-perf-send -peer 169.254.61.91:54003 -count 20 -warmup 0 -size 1200 -timeout 30s'` | `46.88 KBytes`, `11.25 Mbits/sec`, avg RTT `1.706ms`. |
| Network AWDL remote perf listener | Local: `/tmp/awdl-webrtc-apple-demo-bin -profile awdl -backend network -mode udp-perf-listen -count 10 -warmup 0 -timeout 35s` | Listener received 10 datagrams. |
| Network AWDL remote perf sender | Remote: `ssh tmc2@10.0.18.249 '/tmp/awdl-webrtc-apple-demo-bin -profile awdl -backend network -mode udp-perf-send -peer "[fe80::cd4:b4ff:fe63:bc03%awdl0]:54790" -count 10 -warmup 0 -size 1200 -timeout 35s'` | `23.44 KBytes`, `2.61 Mbits/sec`, avg RTT `7.360ms`. |
| Network LAN reverse perf | Remote listener plus local sender to `10.0.18.249:61616` | 10 datagrams, zero loss, `1.12 Mbits/sec`, avg RTT `17.076ms`. |
| Network Thunderbolt reverse perf | Remote listener plus local sender to `169.254.88.35:57837` | Failed: remote listener received 0/20 datagrams; sender timed out waiting for Network.framework connection readiness. |
| Network AWDL reverse perf | Remote listener plus local sender to `[fe80::9477:6dff:fe11:6a55%awdl0]:64413` | Failed: remote listener received 0/10 datagrams; sender timed out waiting for Network.framework connection readiness. |
| Network repeated perf smoke | `go run . -profile lan -backend network -mode udp-perf -count 2 -warmup 0 -trials 2 -perf-json -timeout 15s` | Trial summaries print with `Lost` and `Loss` columns; repeated runs also print `udp perf summary trials=2`; `-perf-json` emits per-trial records and an `udp_perf_summary` record. A no-listener sender smoke counted 2/2 lost datagrams instead of aborting. |
| Network pipelined perf smoke | `AWDL_DEMO_NETWORK_TRACE=1 go run . -profile lan -backend network -mode udp-perf -count 4 -warmup 0 -window 2 -perf-json -timeout 12s` | With `tmc/apple v0.6.7`, the sender retried readiness once, reached `NWConnectionStateReady`, and emitted JSON with `"window":2` plus `paths:[{interfaces:[{name:"en0"}]}]`; this same-host run still saw 4/4 echo replies lost. |
| Duration perf smoke | `go run . -profile thunderbolt -backend go -mode udp-perf -duration 20ms -warmup 0 -window 1 -packet-timeout 20ms -perf-json -timeout 5s` | Duration mode printed the normal summary and a JSON record with `"duration_ns":20000000`; the current same-host Thunderbolt sample saw 1/1 datagram lost on `en1`, so this is CLI/record-shape evidence rather than link throughput evidence. |
| Multi-stream perf smoke | `go run . -profile lan -iface lo0 -backend go -mode udp-perf -count 2 -warmup 0 -trials 1 -streams 2 -window 1 -packet-timeout 100ms -perf-json -timeout 5s` | Loopback functional smoke completed 4 datagrams with zero loss and JSON including `"streams":2`; this is CLI/aggregation evidence only, not link evidence. |
| Latency smoke | `go run . -profile lan -iface lo0 -backend go -mode udp-latency -count 3 -warmup 0 -streams 2 -size 64 -packet-timeout 100ms -perf-json -timeout 5s` | Loopback functional smoke completed 6 datagrams with zero loss and JSON kind `"udp_latency"` including `"streams":2`; this is CLI/record-shape evidence only, not link evidence. |
| Path policy smoke | `AWDL_DEMO_NETWORK_TRACE=1 go run . -profile lan -backend network -mode udp-perf -count 2 -warmup 0 -window 2 -require-path-interface en0 -forbid-loopback-path -perf-json -timeout 12s` | The sender reached `NWConnectionStateReady`, reported an `en0` path, and passed the path policy; the same-host server still read no echo packets and exited with `read i/o timeout`, so two-host sender rerun is still required. |
| Callback probe smoke | Local listener plus request on `-profile lan -iface lo0 -backend go` | The request side received `callback:smoke`; the listener printed the callback address and sent the response. This is functional CLI evidence only, not link evidence. |
| AWDL policy check | `go run . -profile awdl -mode check` | Prints AWDL profile, public Wi-Fi peer-to-peer policy, and private `use_awdl=true use_p2p=true`. |
| AWDL WebRTC gather | `go run . -profile awdl -mode gather -timeout 10s` | Two mDNS host candidates from an `awdl0` UDP mux. |
| AWDL UDP echo | `go run . -profile awdl -mode udp -timeout 10s` | Echoed `ping` over scoped IPv6 on `awdl0`. |
| AWDL local perf | `go run . -profile awdl -mode udp-perf -count 1000 -size 1200 -warmup 5 -timeout 30s` | `2.29 MBytes`, `221.70 Mbits/sec`, average RTT `86.499us`. |
| Thunderbolt policy check | `go run . -profile thunderbolt -mode check` | Prints wired policy and the selected Thunderbolt interface. Current default is `en1` while `bridge0` has no address. |
| Thunderbolt WebRTC gather | `go run . -profile thunderbolt -mode gather -timeout 10s` | Two mDNS host candidates from the selected Thunderbolt interface. Historical bridge evidence used `bridge0`; current fallback evidence uses `en1`. |
| Thunderbolt WebRTC pair | `go run . -profile thunderbolt -mode pair -timeout 12s` | Datachannel opened and exchanged payload over `bridge0`-constrained ICE. |
| Thunderbolt local perf | `go run . -profile thunderbolt -mode udp-perf -count 1000 -size 1200 -warmup 5 -timeout 30s` | `2.29 MBytes`, `283.48 Mbits/sec`, average RTT `67.606us`. |
| LAN remote perf listener | Local: `/tmp/awdl-webrtc-apple-demo-bin -profile lan -mode udp-perf-listen -count 20 -warmup 2 -timeout 15s` | Listener received 22 datagrams including warmup. |
| LAN remote perf sender | Remote: `ssh tmc2@10.0.18.249 '/tmp/awdl-webrtc-apple-demo-bin -profile lan -mode udp-perf-send -peer 10.0.199.147:52981 -count 20 -size 1200 -warmup 2 -timeout 15s'` | `46.88 KBytes`, `755.17 Kbits/sec`, average RTT `25.424ms`. |
| Thunderbolt remote perf listener | Local: `/tmp/awdl-webrtc-apple-demo-bin -profile thunderbolt -mode udp-perf-listen -count 1000 -warmup 5 -timeout 20s` | Listener received 1005 datagrams including warmup. |
| Thunderbolt remote perf sender | Remote: `ssh tmc2@10.0.18.249 '/tmp/awdl-webrtc-apple-demo-bin -profile thunderbolt -mode udp-perf-send -peer 169.254.7.165:54641 -count 1000 -size 1200 -warmup 5 -timeout 20s'` | `2.29 MBytes`, `46.48 Mbits/sec`, average RTT `412.745us`. |
| AWDL remote perf listener | Local: `/tmp/awdl-webrtc-apple-demo-bin -profile awdl -mode udp-perf-listen -count 100 -warmup 5 -timeout 30s` | Listener received 105 datagrams including warmup. |
| AWDL remote perf sender | Remote: `ssh tmc2@10.0.18.249 "/tmp/awdl-webrtc-apple-demo-bin -profile awdl -mode udp-perf-send -peer '[fe80::bcb7:e5ff:fe2d:13b4%awdl0]:62324' -count 100 -size 1200 -warmup 5 -timeout 30s"` | `234.38 KBytes`, `2.11 Mbits/sec`, average RTT `9.091ms`. |
| Two-host AWDL echo listener | `go run . -profile awdl -mode udp-listen -timeout 60s` | Prints a scoped listener address for the peer. |
| Two-host AWDL echo sender | `go run . -profile awdl -mode udp-send -peer '[fe80::peer%awdl0]:12345' -timeout 10s` | Sends one payload and waits for echo. |
| Two-host AWDL perf listener | `go run . -profile awdl -mode udp-perf-listen -timeout 60s` | Echoes perf datagrams and prints ingress summary when done. |
| Two-host AWDL perf sender | `go run . -profile awdl -mode udp-perf-send -peer '[fe80::peer%awdl0]:12345' -count 1000 -size 1200 -warmup 5 -trials 3 -window 4 -perf-json -timeout 20s` | Prints iperf-like transfer, bitrate, datagrams, loss, omit, and RTT summary for each trial plus aggregate output and JSON records including the in-flight window. |

## Performance Snapshot

Local same-host samples:

| Link | Scope | Transfer | Bitrate | Datagrams | Omit | RTT min | RTT avg | RTT p50 | RTT p95 | RTT max |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| AWDL `awdl0` | local | 2.29 MBytes | 221.70 Mbits/sec | 1000 | 5 | 40.583us | 86.499us | 68.583us | 155.292us | 4.948ms |
| Thunderbolt `bridge0` | local | 2.29 MBytes | 283.48 Mbits/sec | 1000 | 5 | 31.709us | 67.606us | 59.125us | 138.208us | 676.958us |

Remote-to-local samples using `ssh tmc2@10.0.18.249`:

| Link | Scope | Transfer | Bitrate | Datagrams | Omit | RTT min | RTT avg | RTT p50 | RTT p95 | RTT max |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| LAN `en0` | remote | 46.88 KBytes | 755.17 Kbits/sec | 20 | 2 | 7.262ms | 25.424ms | 10.973ms | 51.320ms | 200.418ms |
| Thunderbolt `bridge0` | remote | 2.29 MBytes | 46.48 Mbits/sec | 1000 | 5 | 197.625us | 412.745us | 315.250us | 816.917us | 11.782ms |
| AWDL `awdl0` | remote | 234.38 KBytes | 2.11 Mbits/sec | 100 | 5 | 4.368ms | 9.091ms | 6.940ms | 17.168ms | 58.309ms |

Network.framework remote-to-local samples:

| Link | Backend | Path Evidence | Transfer | Bitrate | Datagrams | RTT min | RTT avg | RTT p50 | RTT p95 | RTT max |
| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| LAN `en0` | `network` | `en0/NWInterfaceTypeWifi` | 46.88 KBytes | 753.72 Kbits/sec | 20 | 9.754ms | 25.473ms | 15.749ms | 29.719ms | 214.478ms |
| Thunderbolt `bridge0` | `network` | `bridge0/NWInterfaceTypeWired` | 46.88 KBytes | 11.25 Mbits/sec | 20 | 256.250us | 1.706ms | 289.583us | 528.958us | 28.351ms |
| AWDL `awdl0` | `network` | `awdl0/NWInterfaceTypeWifi` | 23.44 KBytes | 2.61 Mbits/sec | 10 | 4.159ms | 7.360ms | 5.277ms | 21.892ms | 21.892ms |

Network.framework local-to-remote samples:

| Link | Backend | Peer | Status | Transfer | Bitrate | Datagrams | Lost | Loss | RTT min | RTT avg | RTT p50 | RTT p95 | RTT max |
| --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| LAN `en0` | `network` | `10.0.18.249:61616` | PASS | 23.44 KBytes | 1.12 Mbits/sec | 10 | 0 | 0.00% | 5.599ms | 17.076ms | 6.319ms | 111.913ms | 111.913ms |
| Thunderbolt `bridge0` | `network` | `169.254.88.35:57837` | FAIL | 0.00 Bytes | 0 bits/sec | 0 | 20 | 100.00% | - | - | - | - | - |
| AWDL `awdl0` | `network` | `[fe80::9477:6dff:fe11:6a55%awdl0]:64413` | FAIL | 0.00 Bytes | 0 bits/sec | 0 | 10 | 100.00% | - | - | - | - | - |

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

Thunderbolt remote-to-local perf:

```text
profile=thunderbolt interface=bridge0 index=15 flags=up|broadcast|multicast|running ips=169.254.88.35,fe80::1016:9010:5380:528b
udp perf local=169.254.88.35:53519 peer=169.254.7.165:54641 network=udp4 bound_if=none
[ ID] Interval           Transfer     Bitrate         Datagrams  Omit  RTT min/avg/p50/p95/max
[  5] 0.00-0.41    sec  2.29 MBytes  46.48 Mbits/sec       1000     5  197.625us/412.745us/315.250us/816.917us/11.782ms
```

AWDL remote-to-local perf:

```text
profile=awdl interface=awdl0 index=16 flags=up|broadcast|multicast|running ips=fe80::bccd:80ff:fe58:eafe
udp perf local=[fe80::bccd:80ff:fe58:eafe%awdl0]:52775 peer=[fe80::bcb7:e5ff:fe2d:13b4%awdl0]:62324 network=udp6 bound_if=awdl0(16)
[ ID] Interval           Transfer     Bitrate         Datagrams  Omit  RTT min/avg/p50/p95/max
[  5] 0.00-0.91    sec  234.38 KBytes  2.11 Mbits/sec        100     5  4.368ms/9.091ms/6.940ms/17.168ms/58.309ms
```

Network.framework AWDL remote WebRTC:

```text
offer udp mux listen=[fe80::bcb7:e5ff:fe2d:13b4%awdl0]:65149 network=udp6 backend=network bound_if=awdl0(16) mdns=disabled raw_candidates=true
remote: answer udp mux listen=[fe80::bccd:80ff:fe58:eafe%awdl0]:62723 network=udp6 backend=network bound_if=awdl0(16) mdns=disabled raw_candidates=true
webrtc datachannel opened and exchanged payload with tmc2@10.0.18.249 over awdl0-constrained ICE
remote: webrtc answer received "ping" and sent pong over awdl0-constrained ICE
```

Network.framework Pion-native Thunderbolt remote WebRTC:

```text
offer pion net local_ip=169.254.61.91 network=udp4 backend=network bound_if=bridge0(15) mdns=disabled raw_candidates=true
remote: answer pion net local_ip=169.254.88.35 network=udp4 backend=network bound_if=bridge0(15) mdns=disabled raw_candidates=true
webrtc datachannel opened and exchanged payload with tmc2@10.0.18.249 over bridge0-constrained ICE
remote: webrtc answer received "ping" and sent pong over bridge0-constrained ICE
```

Network.framework Pion-native AWDL remote WebRTC:

```text
offer pion net local_ip=fe80::cd4:b4ff:fe63:bc03 network=udp6 backend=network bound_if=awdl0(16) mdns=disabled raw_candidates=true
remote: answer pion net local_ip=fe80::9477:6dff:fe11:6a55 network=udp6 backend=network bound_if=awdl0(16) mdns=disabled raw_candidates=true
remote: webrtc answer received "ping" and sent pong over awdl0-constrained ICE
webrtc datachannel opened and exchanged payload with tmc2@10.0.18.249 over awdl0-constrained ICE
```

Network.framework AWDL remote UDP perf:

```text
udp perf local=[fe80::9477:6dff:fe11:6a55%awdl0]:58624 peer=[fe80::cd4:b4ff:fe63:bc03%awdl0]:54790 network=udp6 backend=network bound_if=awdl0(16)
[ ID] Interval           Transfer     Bitrate         Datagrams  Omit  RTT min/avg/p50/p95/max
[  5] 0.00-0.07    sec  23.44 KBytes  2.61 Mbits/sec         10     0  4.159ms/7.360ms/5.277ms/21.892ms/21.892ms
```

## Boundaries

| Boundary | Meaning |
| --- | --- |
| Same-host perf is not link throughput | The local samples prove the constrained socket path and output format, not real radio/cable speed. |
| UDP direction is asymmetric | Remote-to-local Network.framework UDP passes on Thunderbolt and AWDL; local-to-remote remote listeners received zero datagrams on Thunderbolt and AWDL in the latest clean run. |
| LAN sustained UDP is unstable here | LAN completed 20 datagrams but 50/100/1000-datagram runs timed out after partial progress. |
| AWDL can be demand-activated | A cold UDP run can time out; running `gather` first activates the AWDL path on this host. |
| Network.framework backend is still a demo backend | It proves Pion ICE gathering, raw UDP perf in verified directions, UDP-mux WebRTC, and LAN/Thunderbolt/AWDL `transport.Net` WebRTC. |
| Secure UDP is not plain UDP | `NWParametersCreateSecureUDP(nil, nil)` attempted DTLS and failed; the backend uses `NWParametersCreate` plus `NWUDPCreateOptions`. |
| AWDL needs exact private interface selection | Without the private `NWInterface.cInterface` requirement, Network.framework selected `en0`; with it, the path was `awdl0/NWInterfaceTypeWifi`. |
| Thunderbolt required-interface policy was too strict | `required_interface_type=wired` stayed in Waiting/Preparing; omitting the required type and dialing the bridge link-local address selected `bridge0/NWInterfaceTypeWired`. |
| AWDL WebRTC needs explicit candidate handling | Pion mDNS did not resolve into an open AWDL datachannel here; `-mdns disabled -raw-candidates` uses explicit candidate signaling to open the AWDL `SetNet` datachannel without a Pion fork. |
| All-Network.framework backend is still broader | `github.com/tmc/apple-pion/nwtransport` intentionally proves a hybrid Pion `transport.Net`: selected-link UDP is native, while DNS, TCP, TURN/STUN helper traffic outside that UDP surface, and unsupported cases use the fallback. |
