# Results

This is the at-a-glance record of the demo answers and observed outputs.

Scope terms:

- `local`: same-host socket check through the selected interface.
- `remote`: two Apple hosts; this is required for real AWDL or Thunderbolt link speed.
- `policy`: Apple Network.framework configuration was created and read back in `check` mode.
- `network backend`: the demo's `-backend network` path.
- `pion-net`: `-backend network -pion-net`, which passes Network.framework UDP listeners to Pion through `transport.Net`.

## Current Remote Matrix

The current two-host run is summarized in
[docs/matrix-workspace-20260519.md](docs/matrix-workspace-20260519.md).

| Area | Current result |
| --- | --- |
| Dependency mode | Checked-in `go.work` uses `../apple` and `../apple-pion`; no new tags were published for this run. |
| LAN | Pion-native WebRTC PASS; Network.framework UDP perf/latency/callback probes pass over `en0/NWInterfaceTypeWifi`. |
| Thunderbolt | Pion-native WebRTC PASS; Network.framework UDP perf/latency/callback probes pass over `bridge0/NWInterfaceTypeWired`; throughput summaries reached 219.61 Mbps remote-to-local and 304.94 Mbps local-to-remote. |
| AWDL | Pion-native WebRTC PASS; Network.framework UDP perf/latency/callback probes pass both directions over `awdl0/NWInterfaceTypeWifi`; throughput summaries reached 28.93 Mbps remote-to-local and 26.72 Mbps local-to-remote. |
| Matrix artifacts | Raw transcript `/tmp/awdl-webrtc-matrix-workspace2-20260519.txt`; generated table `/tmp/awdl-webrtc-matrix-workspace2-20260519.md`; checked-in table [docs/matrix-workspace-20260519.md](docs/matrix-workspace-20260519.md). |

## Result Matrix

| Question / Area | Status | Answer | Evidence | Caveat / Next |
| --- | --- | --- | --- | --- |
| Repo location | PASS | The demo is in `~/go/src/github.com/tmc/awdl-webrtc-apple-demo`. | Productization includes reusable packet and Pion transport packages plus docs/audit tables. | No local `tmc/apple` or `tmc/apple-pion` replace remains. |
| Build | PASS | The module builds and the focused tests pass. | `go test ./...`; `go vet ./...`; `git diff --check`. | Darwin-only demo. |
| WebRTC support | PASS | Yes, via Pion WebRTC with constrained ICE. | Modes: `check`, `gather`, `pair`, `answer-stdio`, `offer-stdio`, `answer-bonjour`, `offer-bonjour`, `offer-ssh`. | `offer-stdio` makes the wire signal independent of SSH; Bonjour signaling is implemented but still needs a two-host proof. |
| Pion changes | PASS | No Pion fork or patch was needed. | Uses `SettingEngine` filters, mDNS mode, `SetICEUDPMux`, `SetNet`, explicit signaling, and `apple-pion/icepolicy`. | AWDL link-local candidate publication is explicit candidate signaling through `-candidate-policy auto` or `raw`; the current matrix proves the `-pion-net` datachannel path on AWDL too. |
| Pluggable backend shape | PASS | The CLI has `-backend go\|network`; WebRTC can use either an ICE UDP mux or `-pion-net`. | `go test ./...`; `-backend network` gather, UDP perf, remote WebRTC, and `-pion-net` LAN/Thunderbolt/AWDL remote WebRTC pass in the current matrix. | A broader all-Network.framework DNS/TCP/TURN/STUN backend remains outside this demo slice. |
| Reusable packet package | PASS | The Network.framework PacketConn lives at `github.com/tmc/apple/x/network/nwpacket`, and this demo consumes the sibling checkout through `go.work` for active development. | `go test ./x/network/nwpacket` and `go vet ./x/network/nwpacket` pass in `../apple`; the workspace matrix proves the local PacketConn changes across LAN, Thunderbolt, and AWDL. | Keep release-tag checks separate from workspace proof runs. |
| Pion `transport.Net` package | PASS | `github.com/tmc/apple-pion/nwtransport` implements the Pion `transport.Net` UDP surface needed by ICE, and the demo consumes the sibling checkout through `go.work`. | Current LAN, Thunderbolt, and AWDL `-pion-net -mdns disabled -candidate-policy auto offer-ssh` opened datachannels and exchanged `ping`/`pong`; `../apple-pion` tests assert native numeric UDP handling, named UDP fallback, and `CreateListenConfig.ListenPacket` native wildcard behavior. | DNS, TCP, unconstrained wildcard UDP, named UDP endpoints, TURN/STUN helper traffic outside the selected UDP surface, and unsupported UDP cases intentionally fall back to Pion's standard network. |
| Explicit ICE policy | PASS | AWDL host-candidate handling is factored into `github.com/tmc/apple-pion/icepolicy`, raw candidates are signaled separately from unmodified SDP, and the demo exposes `-candidate-policy auto\|mdns\|raw`. | `apple-pion/icepolicy` has package docs, tests, and examples for synthetic host-candidate publication and `ICECandidateInit` extraction; `TestNewCandidatePolicyConfig`; AWDL `-pion-net -mdns disabled` gather prints `candidate_policy=auto raw_candidates=true`; current matrix used auto policy and opened the AWDL `-pion-net` datachannel. | mDNS candidate exchange remains a separate boundary; use disabled mDNS plus `auto` for link-local AWDL proof. |
| Apple public policy | PASS | Public Network.framework parameters work for all profiles. | AWDL: `include_peer_to_peer=true`, Wi-Fi. Thunderbolt: wired. LAN: Wi-Fi. | The clear UDP backend applies these through `NWParametersCreate` plus `NWUDPCreateOptions`. |
| Apple private policy | PASS | Private knobs are reachable without importing the broken local generated private package. | Raw Objective-C probes report `required_interface`, `use_awdl`, `use_p2p`, `allow_socket_access`, `reuse_local_address`, `prohibit_fallback`. | Exact `NWInterface.cInterface` is enabled for AWDL only; Thunderbolt uses the link-local bridge address. |
| Network.framework clear UDP | PASS | The backend must use `NWParametersCreate` plus `NWUDPCreateOptions`; `NWParametersCreateSecureUDP(nil, nil)` attempted DTLS and failed. | Trace showed `NWErrorDomainTLS` before the clear UDP change; clear UDP then reached `NWConnectionStateReady`. | This is documented in code, not a Pion change. |
| Network.framework Pion gather | PASS | ICE gathering works over both the Network.framework PacketConn mux and the Pion `transport.Net` adapter. | LAN `-pion-net` gathered 10 mDNS candidates; AWDL `-pion-net -mdns disabled` now auto-enables the link-local policy and gathered 2 raw `fe80::...` candidates from `awdl0`. | AWDL mDNS gathers remain less useful than explicit link-local candidates for this two-host proof. |
| Network.framework readiness retry | PASS | The demo defaults to `nwpacket.Config.ConnectTimeout=2s` and `ConnectRetries=2` for Network.framework UDP and Pion `transport.Net` PacketConns, with CLI and matrix overrides. | The local `../apple` PacketConn retries readiness, avoids idle path lookups, binds outbound peers to the listener only for IPv6 link-local/AWDL, and leaves IPv4 LAN/Thunderbolt outbound peers ephemeral; current two-host UDP and `-pion-net` WebRTC probes pass on LAN, Thunderbolt, and AWDL. | Longer stress sweeps should tune timeout/retry values under load. |
| Network.framework remote WebRTC | PASS | Remote Pion datachannel exchange works over Network.framework PacketConn. | LAN, Thunderbolt, and AWDL UDP-mux `offer-ssh` runs have opened datachannels and exchanged `ping`/`pong`. | Pion-native `SetNet` is tracked separately below. |
| Network.framework Pion-native remote WebRTC | PASS | Remote WebRTC through `SettingEngine.SetNet` passes for LAN, Thunderbolt, and AWDL in the current matrix. | LAN, Thunderbolt, and AWDL `-pion-net -mdns disabled -candidate-policy auto offer-ssh` opened datachannels and exchanged `ping`/`pong`. | The proof is selected-link UDP; broader DNS/TCP/TURN/STUN ownership is outside this demo backend. |
| Network.framework remote UDP | PASS | Multi-datagram echo, latency, and callback probes work across LAN, Thunderbolt, and AWDL in both directions. | Current workspace matrix records Network.framework path evidence for `en0`, `bridge0`, and `awdl0`; callback probes passed both directions on all three profiles. | Longer sweeps are still useful before making stable throughput claims. |
| AWDL discovery | PASS | AWDL ICE gathering works on `awdl0`. | `gathered 2 host candidate(s) from awdl0-bound UDP mux`. | Candidates are mDNS host candidates. |
| AWDL WebRTC datachannel | PASS | Remote datachannel exchange works over both the Network.framework UDP-mux path and the Pion `transport.Net` path on `awdl0`. | Historical UDP-mux `offer-ssh` opened and exchanged payload; current matrix `-pion-net -mdns disabled -candidate-policy auto offer-ssh` opened and exchanged `ping`/`pong` over `awdl0`. | Keep using explicit link-local candidate policy for this proof. |
| AWDL UDP | PASS | Direct UDP over AWDL works with interface-bound sockets. | Echo over `[fe80::...%awdl0]` with `server_bound_if=awdl0(16) client_bound_if=awdl0(16)`. | Cold AWDL can time out; `gather` activates the path. |
| AWDL remote UDP perf | PASS | Two-host AWDL UDP completed in both sequential and bidirectional directions. | Current workspace matrix: remote-to-local 22653/22653 datagrams, `28.93 Mbits/sec`; local-to-remote 20894/20908 datagrams, `26.72 Mbits/sec`; both report path `awdl0/NWInterfaceTypeWifi`. | Longer duration sweeps remain useful for stable throughput claims. |
| AWDL Network.framework UDP | PASS | Network.framework sends raw UDP perf, latency, and callback traffic over AWDL in both directions. | Current workspace matrix: latency summaries average `7.042ms` and `7.289ms`; callback probes returned `callback:callback` both directions. | Short-run RTT has occasional outliers. |
| Thunderbolt discovery | PASS | Thunderbolt ICE gathering works through the first usable Thunderbolt interface. | Historical bridge sample: `gathered 2 host candidate(s) from bridge0-bound UDP mux`; current no-bridge-address sample falls back to `en1` and gathers 2 Network.framework candidates. | Override with `-iface` when you need an exact member or bridge interface. |
| Thunderbolt WebRTC datachannel | PASS | Local and remote Thunderbolt-constrained datachannels work. | Local `pair`; remote `offer-ssh` with `tmc2@10.0.18.249` over `bridge0`. | Remote proof uses explicit SSH signaling. |
| Thunderbolt remote UDP perf | PASS | Two-host Thunderbolt UDP completed in both sequential and bidirectional directions. | Current workspace matrix: remote-to-local 171664/171664 datagrams, `219.61 Mbits/sec`; local-to-remote 238290/238290 datagrams, `304.94 Mbits/sec`; both report path `bridge0/NWInterfaceTypeWired`. | Required interface type is omitted; the link-local bridge address selects `bridge0`. |
| Thunderbolt Network.framework UDP | PASS | Network.framework sends raw UDP perf, latency, and callback traffic over Thunderbolt Bridge in both directions. | Current workspace matrix: latency summaries average `171.870us` and `164.972us`; callback probes returned `callback:callback` both directions. | Short-run matrix numbers include occasional high RTT outliers. |
| LAN remote UDP perf | PASS | LAN UDP is reachable in both directions under the current repeated-duration matrix. | Current workspace matrix: remote-to-local 11290/11300 datagrams at `14.12 Mbits/sec`; local-to-remote 10049/10049 datagrams at `12.72 Mbits/sec`. | LAN remains slower and more variable than Thunderbolt in this environment. |
| LAN Network.framework UDP | PASS | Network.framework sends raw UDP perf, latency, and callback traffic over LAN in both directions. | Current workspace matrix: remote-to-local 11290/11300 datagrams, `14.12 Mbits/sec`; local-to-remote 10049/10049 datagrams, `12.72 Mbits/sec`; callback probes returned `callback:callback` both directions. | LAN remains slower and more variable than Thunderbolt in this environment. |
| UDP perf output | PASS | The demo prints iperf-like UDP summaries with loss columns, repeated trials, aggregate trial summaries, optional JSON records, sender-side timeout loss accounting, fixed-duration trials, concurrent streams, explicit latency-only output, observed Network.framework paths, a bounded in-flight window, listener idle stop for duration sweeps, and a bidirectional matrix probe. | Columns: interval, transfer, bitrate, datagrams, lost, loss, omit, RTT min/avg/p50/p95/max; `-trials` repeats runs and prints summaries; `-duration` runs each trial for a fixed time and records `duration_ns`; `-streams` opens multiple client PacketConns and records `streams`; `udp-latency`/`udp-latency-send` emit `udp_latency` JSON records; `-perf-json` includes `paths` for Network.framework peer connections when available; `-require-path-interface` and `-forbid-loopback-path` fail closed; `-window` enables pipelined echo requests; `-packet-timeout` bounds each echo wait; `-listen-idle-timeout` lets listeners with unknown expected counts stop after traffic goes idle; `remote-matrix.sh` runs both perf directions concurrently after the sequential direction checks. | Same-host Network.framework sends can now reach readiness and report path JSON, but echo delivery is still weak on this host; use the two-host matrix for real link evidence. |
| AWDL local perf | PASS | Local AWDL sample produced `221.70 Mbits/sec`. | 1000 datagrams, 1200-byte payload, 5 warmup packets omitted. | Same-host sample after AWDL activation. |
| Thunderbolt local perf | PASS | Local Thunderbolt sample produced `283.48 Mbits/sec`. | 1000 datagrams, 1200-byte payload, 5 warmup packets omitted. | Same-host sample, not peer-to-peer cable speed. |
| Two-host UDP proof | READY | Listener/sender and callback probe modes are implemented. | `udp-listen`, `udp-send`, `udp-callback-listen`, `udp-callback-request`, `udp-perf-listen`, `udp-perf-send`. | Use printed scoped peer address on the sender. |
| Non-SSH WebRTC signaling | PARTIAL | Manual stdio signaling is implemented, and Network.framework Bonjour signaling now advertises/browses a peer service for the same wire format. | `offer-stdio`/`answer-stdio`; `offer-bonjour`/`answer-bonjour`; parser tests pass; Bonjour `-signal-only` smoke advertised `_awdl-webrtc-signal._tcp`, resolved the service, fell back to `NSNetService` host/port resolution, and exchanged `OFFER`/`ANSWER` lines; `scripts/remote-bonjour.sh` now runs two-host `signal` and `full` Bonjour phases for LAN, Thunderbolt, and AWDL. | Same-host `offer-bonjour` without `-signal-only` still times out at ICE/datachannel open with zero candidate-pair responses; the two-host harness is ready, but the peer was unreachable on 2026-05-19. |
| Headless discovery | PASS | The Bonjour/TXT discovery path used by the SwiftUI monitor is also available as JSON output and can feed the remote harnesses. | `go run . -mode discover -backend network -timeout 1s -ui-interval 500ms` emitted `link_health_discovery` records with ready Thunderbolt, AWDL, and LAN listener addresses; `discover-wait` found a simultaneous local `discover` publisher and emitted one peer record with `version`, `commit`, and `modes` metadata; `scripts/remote-diagnostics.sh` and `scripts/remote-matrix.sh` accept `DISCOVERY_FILE` or `USE_DISCOVERY=1` to consume peer addrs. | The local success path proves the control-plane shape, but two-Mac discovery still needs live peer verification. |
| UI fallback selection | PASS | The link-health sampler tries Thunderbolt, then AWDL, then LAN. | `TestLinkHealthSamplePreferredFallsBackToAWDL` and `TestLinkHealthSamplePreferredSkipsUnavailableThunderbolt` pass; local headless discovery found Thunderbolt, AWDL, and LAN rows with commit `4c0c0851f9017fcb0ee25d22321457fd0fea5539`. | The SwiftUI two-live-Mac proof still needs a reachable peer and visible UI observation. |
| Long soak wrapper | READY | Longer repeated sweeps have a dedicated wrapper. | `scripts/remote-soak.sh` sets 30s duration, 5 trials, 2 streams, path requirements, WebRTC retries, and timestamped transcript/summary outputs; a short readiness exercise wrote `/tmp/awdl-webrtc-dry-run-test.{txt,md}` and stopped at remote reachability. | Full long-soak numbers still need the peer to come back online. |

## Current Live State

| Check | Result |
| --- | --- |
| Local Thunderbolt default | With `bridge0` addressless, `go run . -profile thunderbolt -backend network -mode check -timeout 3s` selected `en1` at `172.31.253.1`. |
| Local Thunderbolt gather | `go run . -profile thunderbolt -backend network -mode gather -timeout 5s` gathered 2 mDNS candidates from an `en1` Network.framework UDP mux. |
| Local AWDL auto candidate gather | `go run . -profile awdl -backend network -pion-net -mdns disabled -mode gather -timeout 15s` gathered 2 raw `fe80::...` candidates from `awdl0` and printed `candidate_policy=auto raw_candidates=true`. |
| Current Pion backend boundary | `../apple-pion` commits `2cca84d` and `15e13e4` are pushed. Native Network.framework ownership is restricted to numeric UDP endpoints and configured wildcard listeners; named UDP endpoints go through the fallback `transport.Net` without DNS resolution in the native branch. |
| Remote SSH | LAST PASS for the workspace matrix run: `ssh -o BatchMode=yes -o ConnectTimeout=5 tmc2@10.0.18.249 'echo ok'` returned `ok` before the matrix, and no SSH command failed during the successful run. CURRENT BLOCKED on 2026-05-19: a fresh check reached the host by ping, but TCP/22 returned `Connection refused` and SSH exited 255. Earlier SSH-control-plane failures are preserved in [docs/matrix-try-again-20260519.md](docs/matrix-try-again-20260519.md). |
| Remote matrix success | `WEBRTC_TRACE=0 WEBRTC_ATTEMPTS=3 WEBRTC_RETRY_DELAY=3 REMOTE_READY_TIMEOUT=30 REMOTE_STEP_READY_TIMEOUT=30 CANDIDATE_POLICY=auto REQUIRE_PATHS=1 DURATION=5s TRIALS=3 WINDOW=8 STREAMS=2 TIMEOUT=90s LISTEN_IDLE_TIMEOUT=3s SSH_TARGET=tmc2@10.0.18.249 REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin-workspace OUTPUT=/tmp/awdl-webrtc-matrix-workspace2-20260519.txt SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-workspace2-20260519.md scripts/remote-matrix-bundle.sh` wrote [docs/matrix-workspace-20260519.md](docs/matrix-workspace-20260519.md). LAN, Thunderbolt, and AWDL Pion-native WebRTC passed, and UDP perf, latency, and callback probes passed with path evidence. |
| Stdio signaling smoke | Local two-process `offer-stdio` and `answer-stdio` runs on `en0` exchanged `OFFER` and `ANSWER` lines. With raw candidates, both sides installed host candidates but the candidate pair failed with zero responses; with mDNS candidates, no remote candidates resolved before timeout. This validates the wire exchange shape but not a datachannel. |
| Bonjour signaling smoke | `go run . -mode answer-bonjour -signal-name codex-signalonly1 -signal-only -profile lan -backend network -pion-net -mdns disabled -candidate-policy raw -timeout 20s` plus matching `offer-bonjour -signal-only` exchanged `OFFER`/`ANSWER` via Bonjour signaling and exited 0 on both sides. | This validates the Bonjour control-plane shape locally but not same-host WebRTC datachannel success. A non-`-signal-only` run still reaches ICE checking and times out with same-host candidates. |
| Two-host Bonjour harness | `REMOTE_READY_TIMEOUT=30 PROFILES="lan thunderbolt awdl" PHASES="signal full" SSH_TARGET=tmc2@10.0.18.249 OUTPUT=/tmp/awdl-webrtc-bonjour.txt scripts/remote-bonjour.sh` | Builds the local `go.work` workspace, installs it on the peer, starts remote `answer-bonjour`, waits for the service line, runs local `offer-bonjour`, captures the remote answer log, and enables WebRTC trace by default. | Current blocked evidence is `/tmp/awdl-webrtc-bonjour-unreachable-20260519.txt`: route to `10.0.18.249` was down, ping lost 100%, and SSH readiness failed with exit 255. |
| Headless discovery smoke | `go run . -mode discover -backend network -timeout 1s -ui-interval 500ms` emitted JSON records listing local `en1`, `awdl0`, and `en0` listener addresses, with status `waiting for peer`. A parallel `discover-wait` run found that local publisher and emitted one `peer found` JSON record with TXT metadata, including `version`, `commit`, and `modes`. `discover-wait -discover-peer definitely-missing-peer` timed out with no stdout and a clear stderr error. |
| Matrix reachability diagnostics | The current matrix captured route-to-peer on `en0`, ping success, TCP/22 success, `scutil --nwi`, local interface inventory, and then continued into all LAN/Thunderbolt/AWDL probes. Discovery parsing was locally exercised with `/tmp/awdl-discovery-fake.json`; the matrix parsed LAN, Thunderbolt, and AWDL peer addrs before stopping at the current SSH reachability blocker. |
| Matrix summary table | `go run ./cmd/matrix-summary /tmp/awdl-webrtc-matrix-workspace2-20260519.txt` | Converts matrix `udp_perf`, `udp_perf_summary`, `udp_perf_listen`, `udp_latency`, and `udp_latency_summary` JSON lines plus de-duplicated `FAIL:` lines into a compact Markdown table with section, datagrams, loss, bitrate, RTT, path, and failure rows. |
| Remote diagnostics | `RUN_REMOTE=1 PROFILES=lan OUTPUT=/tmp/awdl-webrtc-diagnostics-v011-lan.txt scripts/remote-diagnostics.sh`; `RUN_REMOTE=0 DISCOVERY_FILE=/tmp/awdl-discovery-fake.json PROFILES="lan thunderbolt awdl" scripts/remote-diagnostics.sh` | Local and remote smoke captured hostname, OS, `ifconfig`, IPv4/IPv6 routes, route-to-peer, `scutil --nwi`, hardware ports, and UDP sockets. The remote side now runs under Bash so interface and route lists split correctly. The discovery-file smoke populated LAN, Thunderbolt, and AWDL routes from a saved `link_health_discovery` record. |

## Host Matrix

| Host | Access | LAN `en0` | Thunderbolt `bridge0` | AWDL `awdl0` |
| --- | --- | --- | --- | --- |
| Local | shell | `10.0.199.147` | historical `bridge0` `169.254.61.91`; current fallback `en1` `172.31.253.1` | `fe80::cd4:b4ff:fe63:bc03%awdl0` |
| Remote | `tmc2@10.0.18.249` | `10.0.18.249` | `169.254.88.35` | `fe80::5c89:22ff:fe01:380d%awdl0` |

## Command Matrix

| Goal | Command | Observed Result |
| --- | --- | --- |
| Build gate | `go test ./...`; `go vet ./...`; `git diff --check` | Demo tests pass; vet and whitespace checks pass. |
| Productized packages | In `tmc/apple`: `go test ./x/network/nwpacket`; in `tmc/apple-pion`: `go test ./...` | Promoted `nwpacket`, `nwtransport`, and `icepolicy` tests pass. |
| Workspace backend source | `cat go.work`; `go env GOWORK`; local sibling tests | `go.work` uses `.`, `../apple`, and `../apple-pion`; no new tags were published for the current proof. |
| Published release preflight | `GITHUB_RESOLVE_IP=140.82.116.3 scripts/release-preflight.sh` | This is now a separate release gate. The current proof intentionally uses `go.work` and local sibling modules instead of publishing new tags. |
| Remote productization matrix | `WEBRTC_TRACE=0 WEBRTC_ATTEMPTS=3 WEBRTC_RETRY_DELAY=3 REMOTE_READY_TIMEOUT=30 REMOTE_STEP_READY_TIMEOUT=30 CANDIDATE_POLICY=auto REQUIRE_PATHS=1 DURATION=5s TRIALS=3 WINDOW=8 STREAMS=2 TIMEOUT=90s LISTEN_IDLE_TIMEOUT=3s SSH_TARGET=tmc2@10.0.18.249 REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin-workspace OUTPUT=/tmp/awdl-webrtc-matrix-workspace2-20260519.txt SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-workspace2-20260519.md scripts/remote-matrix-bundle.sh` | Runs LAN/Thunderbolt/AWDL `-pion-net -candidate-policy auto` WebRTC plus Network.framework UDP perf in both directions, simultaneous bidirectional UDP perf, latency, and callback probes. Current result: all three WebRTC legs pass; UDP probes pass with path evidence. |
| Earlier remote productization retry | `WEBRTC_TRACE=1 REMOTE_READY_TIMEOUT=30 CANDIDATE_POLICY=auto REQUIRE_PATHS=1 SSH_TARGET=tmc2@10.0.18.249 REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin OUTPUT=/tmp/awdl-webrtc-matrix-try-again.txt SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-try-again.md scripts/remote-matrix-bundle.sh` | Preserved in [docs/matrix-try-again-20260519.md](docs/matrix-try-again-20260519.md) as an SSH-control-plane failure, not a link measurement. |
| Manual WebRTC signaling | Offer side: `go run . -profile awdl -backend network -pion-net -mdns disabled -candidate-policy auto -mode offer-stdio -timeout 90s`; answer side: same flags with `-mode answer-stdio` | IMPLEMENTED: uses the same `OFFER`/`ANSWER` wire format and explicit candidate policy as `offer-ssh`, but does not require SSH. Needs a two-host run for link proof. |
| Bonjour WebRTC signaling | Answer side: `go run . -profile awdl -backend network -pion-net -mdns disabled -candidate-policy auto -mode answer-bonjour -signal-name awdl-webrtc-peer-a -timeout 90s`; offer side: same flags with `-mode offer-bonjour -signal-peer awdl-webrtc-peer-a`; add `-signal-only` on both sides for control-plane-only proof. | IMPLEMENTED BUT UNPROVEN ON TWO HOSTS: advertises `_awdl-webrtc-signal._tcp` and uses Network.framework browse plus `NSNetService` host/port fallback for the same `OFFER`/`ANSWER` wire format. Local same-host `-signal-only` exchanged the lines and exited 0; same-host ICE still timed out without `-signal-only`. |
| Headless link discovery | `go run . -mode discover -backend network -timeout 10s -ui-interval 1s` | Emits JSON records with local Thunderbolt, AWDL, and LAN listener addresses plus newest peer metadata from Network.framework Bonjour browse results, including advertised version, commit, and supported modes when available. |
| Wait for discovered peer | `go run . -mode discover-wait -backend network -timeout 30s`; optional `-discover-peer peer-id-or-name` | Emits one JSON record and exits when a matching peer is discovered; setup diagnostics go to stderr so stdout remains JSON. |
| Remote matrix summary | `scripts/remote-matrix-bundle.sh` or `go run ./cmd/matrix-summary /tmp/awdl-webrtc-matrix-workspace2-20260519.txt > /tmp/awdl-webrtc-matrix-workspace2-20260519.md` | Produces the clean Markdown result table after the matrix has emitted JSON records or `FAIL:` lines. The bundle writes the summary even when the matrix fails, then returns the matrix exit code. |
| Network LAN gather | `go run . -profile lan -backend network -mode gather -timeout 8s` | Two mDNS host candidates from an `en0` Network.framework UDP mux. |
| Pion-native LAN gather | `AWDL_DEMO_NETWORK_TRACE=1 go run . -profile lan -backend network -pion-net -mode gather -timeout 12s` | Ten mDNS host candidates; trace showed Network.framework listeners for each selected `en0` address. |
| Pion-native LAN pair | `go run . -profile lan -backend network -pion-net -mode pair -timeout 15s`; `go run . -profile lan -backend network -pion-net -mdns disabled -candidate-policy raw -mode pair -timeout 15s` | Current same-host checks timed out before datachannel open. With mDNS, no remote candidates resolved; with raw candidates, both sides installed `10.0.199.147` host candidates but candidate-pair checks saw requests and zero responses. The recorded two-host LAN `offer-ssh` matrix remains the stronger LAN Pion-native evidence. |
| Current Thunderbolt fallback | `go run . -profile thunderbolt -backend network -mode gather -timeout 5s` | With `bridge0` addressless, default selection fell back to `en1` and gathered two mDNS host candidates. |
| Network Thunderbolt gather | `go run . -profile thunderbolt -backend network -mode gather -timeout 8s` | Two mDNS host candidates from a `bridge0` Network.framework UDP mux. |
| Network AWDL gather | `go run . -profile awdl -backend network -mode gather -timeout 12s` | Two mDNS host candidates from an `awdl0` Network.framework UDP mux. |
| Pion-native AWDL gather | `AWDL_DEMO_NETWORK_TRACE=1 go run . -profile awdl -backend network -pion-net -mode gather -timeout 15s` | Two mDNS host candidates from `awdl0`; trace showed a Network.framework listener on `[fe80::...%awdl0]`. |
| Network AWDL auto-candidate gather | `go run . -profile awdl -backend network -mdns disabled -mode gather -timeout 12s` | Two raw `fe80::...` host candidates from an `awdl0` Network.framework UDP mux; output included `candidate_policy=auto raw_candidates=true`. |
| Pion-native AWDL auto-candidate gather | `go run . -profile awdl -backend network -pion-net -mdns disabled -mode gather -timeout 15s` | Two raw `fe80::...` candidates from `awdl0`; output included `candidate_policy=auto raw_candidates=true`. |
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
| Network LAN reverse perf | Remote listener plus local sender to `10.0.18.249` | Current workspace matrix: 10049/10049 datagrams, zero loss, `12.72 Mbits/sec`, avg RTT `24.057ms`. |
| Network Thunderbolt reverse perf | Remote listener plus local sender to `169.254.88.35` | Current workspace matrix: 238290/238290 datagrams, zero loss, `304.94 Mbits/sec`, avg RTT `883.484us`. |
| Network AWDL reverse perf | Remote listener plus local sender to `[fe80::5c89:22ff:fe01:380d%awdl0]` | Current workspace matrix: 20894/20908 datagrams, 0.07% loss, `26.72 Mbits/sec`, avg RTT `10.730ms`. |
| Network repeated perf smoke | `go run . -profile lan -backend network -mode udp-perf -count 2 -warmup 0 -trials 2 -perf-json -timeout 15s` | Trial summaries print with `Lost` and `Loss` columns; repeated runs also print `udp perf summary trials=2`; `-perf-json` emits per-trial records and an `udp_perf_summary` record. A no-listener sender smoke counted 2/2 lost datagrams instead of aborting. |
| Network pipelined perf smoke | `AWDL_DEMO_NETWORK_TRACE=1 go run . -profile lan -backend network -mode udp-perf -count 4 -warmup 0 -window 2 -perf-json -timeout 12s` | The workspace PacketConn sender retried readiness once, reached `NWConnectionStateReady`, and emitted JSON with `"window":2` plus `paths:[{interfaces:[{name:"en0"}]}]`; this same-host run still saw 4/4 echo replies lost. |
| Duration perf smoke | `go run . -profile thunderbolt -backend go -mode udp-perf -duration 20ms -warmup 0 -window 1 -packet-timeout 20ms -perf-json -timeout 5s` | Duration mode printed the normal summary and a JSON record with `"duration_ns":20000000`; the current same-host Thunderbolt sample saw 1/1 datagram lost on `en1`, so this is CLI/record-shape evidence rather than link throughput evidence. |
| Duration listener idle smoke | Loopback `udp-perf-listen -duration 10ms -listen-idle-timeout 50ms` plus `udp-perf-send -duration 10ms -window 2` | Listener exited after the sender stopped and reported 289 datagrams instead of waiting for the full outer timeout. |
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
| LAN `en0` | `network` | `en0/NWInterfaceTypeWifi` | 25.84 MiB | 14.12 Mbits/sec | 11290 | 4.293ms | 20.468ms | 12.520ms | 62.640ms | 261.704ms |
| Thunderbolt `bridge0` | `network` | `bridge0/NWInterfaceTypeWired` | 392.91 MiB | 219.61 Mbits/sec | 171664 | 224.417us | 1.363ms | 913.375us | 2.614ms | 231.695ms |
| AWDL `awdl0` | `network` | `awdl0/NWInterfaceTypeWifi` | 51.85 MiB | 28.93 Mbits/sec | 22653 | 1.370ms | 10.466ms | 10.346ms | 14.987ms | 29.059ms |

Network.framework local-to-remote samples:

| Link | Backend | Peer | Status | Transfer | Bitrate | Datagrams | Lost | Loss | RTT min | RTT avg | RTT p50 | RTT p95 | RTT max |
| --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| LAN `en0` | `network` | `10.0.18.249` | PASS | 23.00 MiB | 12.72 Mbits/sec | 10049 | 0 | 0.00% | 4.436ms | 24.057ms | 12.873ms | 109.376ms | 438.347ms |
| Thunderbolt `bridge0` | `network` | `169.254.88.35` | PASS | 545.40 MiB | 304.94 Mbits/sec | 238290 | 0 | 0.00% | 65.875us | 883.484us | 694.458us | 1.598ms | 108.285ms |
| AWDL `awdl0` | `network` | `[fe80::5c89:22ff:fe01:380d%awdl0]` | PASS | 47.82 MiB | 26.72 Mbits/sec | 20894 | 14 | 0.07% | 5.916us | 10.730ms | 10.466ms | 14.795ms | 184.952ms |

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
offer pion net local_ip=fe80::c814:f7ff:fe87:2c83 network=udp6 backend=network bound_if=awdl0(16) mdns=disabled candidate_policy=auto raw_candidates=true
remote: answer pion net local_ip=fe80::5c89:22ff:fe01:380d network=udp6 backend=network bound_if=awdl0(16) mdns=disabled candidate_policy=auto raw_candidates=true
webrtc datachannel opened and exchanged payload with tmc2@10.0.18.249 over awdl0-constrained ICE
remote: webrtc answer received "ping" and sent pong over awdl0-constrained ICE
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
| UDP direction is no longer the main blocker | The current workspace matrix passes sequential Network.framework UDP perf, latency, callback, and bidirectional probes in both directions for LAN, Thunderbolt, and AWDL. |
| LAN sustained UDP is variable here | The current workspace matrix completed repeated 5s LAN sweeps in both directions, but LAN remains slower and more variable than Thunderbolt in this environment. |
| AWDL can be demand-activated | A cold UDP run can time out; running `gather` first activates the AWDL path on this host. |
| Network.framework backend is still a demo backend | It proves Pion ICE gathering, raw UDP perf, UDP-mux WebRTC, and LAN/Thunderbolt/AWDL `transport.Net` WebRTC, but only for the selected UDP surface. |
| Secure UDP is not plain UDP | `NWParametersCreateSecureUDP(nil, nil)` attempted DTLS and failed; the backend uses `NWParametersCreate` plus `NWUDPCreateOptions`. |
| AWDL needs exact private interface selection | Without the private `NWInterface.cInterface` requirement, Network.framework selected `en0`; with it, the path was `awdl0/NWInterfaceTypeWifi`. |
| Thunderbolt required-interface policy was too strict | `required_interface_type=wired` stayed in Waiting/Preparing; omitting the required type and dialing the bridge link-local address selected `bridge0/NWInterfaceTypeWired`. |
| AWDL WebRTC needs explicit candidate handling | Pion mDNS did not resolve into an open AWDL datachannel in earlier runs. The current proof uses `-mdns disabled -candidate-policy auto`, publishes link-local candidates explicitly, and opens the `SetNet` AWDL datachannel. |
| All-Network.framework backend is still broader | `github.com/tmc/apple-pion/nwtransport` intentionally proves a hybrid Pion `transport.Net`: selected-link UDP is native, while DNS, TCP, TURN/STUN helper traffic outside that UDP surface, and unsupported cases use the fallback. |
