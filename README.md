# WebRTC Link Demo

This temporary module shows WebRTC ICE candidate discovery over Apple-specific
local links using `github.com/tmc/apple` plus Pion WebRTC.

See [RESULTS.md](RESULTS.md) for the current answer/output table and
[docs/completion-audit.md](docs/completion-audit.md) for the strict
completion checklist.

The reusable Network.framework surfaces are:

- `github.com/tmc/apple/x/network/nwpacket`: a Network.framework
  `net.PacketConn`, consumed from released `github.com/tmc/apple v0.6.7`.
- `github.com/tmc/apple-pion/nwtransport`: a small Pion `transport.Net`
  adapter, consumed from released `github.com/tmc/apple-pion v0.1.0`. It
  routes concrete UDP listeners, configured wildcard UDP listeners, and UDP
  dials through Network.framework, while leaving DNS, TCP, unconstrained
  wildcard UDP, and TURN/STUN helper traffic outside that selected UDP surface
  on Pion's standard network fallback.
- `github.com/tmc/apple-pion/icepolicy`: explicit link-local host-candidate
  publication helpers used by the demo's `-raw-candidates` signaling path.

It has three profiles:

- `awdl`: uses `awdl0`, sets public Network.framework UDP parameters to include
  peer-to-peer Wi-Fi, and uses private `NWParameters` knobs for `useAWDL`,
  `useP2P`, exact required interface, socket access, local address reuse, and
  fallback prohibition.
- `thunderbolt`: uses the Thunderbolt Bridge, normally `bridge0`. If `bridge0`
  exists but has no usable address, the default selector falls back to the first
  usable Thunderbolt member interface among `en1`, `en2`, and `en3`. The Go
  backend binds ordinary UDP sockets to the selected interface address; the
  Network.framework backend lets the peer address select the path.
- `lan`: uses `en0`. The Go backend binds ordinary UDP sockets to the LAN
  address; the Network.framework backend uses Wi-Fi UDP policy and the LAN peer
  address selects `en0`.

Run the local checks:

```sh
go run . -profile awdl -backend go -mode check
go run . -profile awdl -backend go -mode gather -timeout 10s
go run . -profile awdl -backend go -mode udp -timeout 10s
go run . -profile awdl -backend go -mode udp-perf -count 1000 -size 1200 -warmup 5 -timeout 20s

go run . -profile lan -backend network -mode gather -timeout 10s
go run . -profile lan -backend network -pion-net -mode gather -timeout 10s
go run . -profile lan -backend network -pion-net -mode pair -timeout 15s
go run . -profile thunderbolt -backend network -mode gather -timeout 10s
go run . -profile awdl -backend network -mode gather -timeout 10s
go run . -profile awdl -backend network -pion-net -mode gather -timeout 10s
go run . -profile awdl -backend network -mdns disabled -raw-candidates -mode gather -timeout 10s

go run . -profile thunderbolt -backend go -mode check
go run . -profile thunderbolt -backend go -mode gather -timeout 10s
go run . -profile thunderbolt -backend go -mode pair -timeout 12s
go run . -profile thunderbolt -backend go -mode udp-perf -count 1000 -size 1200 -warmup 5 -timeout 20s
```

Use `-iface` to override the default interface:

```sh
go run . -profile thunderbolt -iface en1 -backend network -mode gather -timeout 10s
```

To open the SwiftUI link monitor, run the same command on both Macs:

```sh
GOWORK=off go run . -mode ui -backend network -ui-interval 3s -ui-count 20 -ui-window 4
```

The UI advertises a Bonjour service, starts local UDP echo listeners for
Thunderbolt, AWDL, and LAN when those paths are available, and samples the peer
in that order. A Thunderbolt path with no replies is marked unavailable for
that sample and the monitor immediately tries AWDL, then LAN. The monitor is
intended to keep using the published `github.com/tmc/apple v0.6.7` bindings;
`GOWORK=off` avoids accidentally resolving a sibling checkout.

The `-backend` flag selects `go` or `network`.

- `go` uses ordinary Darwin UDP sockets. It is the stable throughput and
  same-host WebRTC datachannel path.
- `network` uses Network.framework as a `net.PacketConn`. It creates clear UDP
  parameters, sets `includePeerToPeer`, uses required interface type where it
  works, and for AWDL uses the private `NWInterface.cInterface` object to force
  the path to `awdl0`. Network.framework PacketConns use a 2s outbound
  readiness timeout and 2 retries so transient waiting states recreate the
  outbound peer connection instead of staying stuck until the full write
  deadline.

For WebRTC modes, `-pion-net` changes the Pion integration from
`SetICEUDPMux` to `SettingEngine.SetNet` using `nwtransport`. LAN,
Thunderbolt, and AWDL remote datachannels pass through this native Pion network
seam when explicit signaling uses `-mdns disabled -raw-candidates`. AWDL mDNS
candidate exchange still does not open a remote datachannel in this
environment.

The `gather` mode binds Pion's ICE UDP mux to the selected interface address,
enables mDNS host candidates, and verifies candidates are either from the
selected IP set or mDNS publication. mDNS is intentional: Pion filters raw IPv6
link-local host candidates for privacy.

For explicit two-process signaling, `-mdns disabled -raw-candidates` enables
the reusable `apple-pion/icepolicy` host-candidate policy. It uses a synthetic
non-link-local host candidate inside Pion and publishes the selected interface
IP as explicit `ICECandidateInit` records alongside the unmodified SDP.
This keeps the demo no-fork while making AWDL link-local ICE testable without
SDP rewriting.

The `pair` mode creates two local PeerConnections and exchanges a datachannel
payload over the constrained interface. On this host, Thunderbolt Bridge pairing
passes. AWDL same-host pairing is not useful because AWDL traffic is peer-link
traffic; use `offer-ssh` against another Apple host.

The `offer-ssh` mode runs the local side, starts `answer-stdio` on a peer over
SSH, exchanges SDP plus optional explicit ICE candidates over stdin/stdout, and
waits for a WebRTC datachannel `ping`/`pong`:

```sh
go run . -profile thunderbolt -backend network -mode offer-ssh \
  -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin \
  -raw-candidates -timeout 35s

go run . -profile awdl -backend network -mdns disabled -raw-candidates \
  -mode offer-ssh -ssh tmc2@10.0.18.249 \
  -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 45s

go run . -profile lan -backend network -pion-net -mdns disabled \
  -raw-candidates -mode offer-ssh -ssh tmc2@10.0.18.249 \
  -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 35s

go run . -profile thunderbolt -backend network -pion-net -mdns disabled \
  -raw-candidates -mode offer-ssh -ssh tmc2@10.0.18.249 \
  -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 40s

go run . -profile awdl -backend network -pion-net -mdns disabled \
  -raw-candidates -mode offer-ssh -ssh tmc2@10.0.18.249 \
  -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 45s
```

To rerun the two-host productization matrix after the peer is reachable, use:

```sh
SSH_TARGET=tmc2@10.0.18.249 \
  OUTPUT=/tmp/awdl-webrtc-matrix.txt \
  scripts/remote-matrix.sh
```

Before the matrix, capture the interface and route state from both Macs:

```sh
SSH_TARGET=tmc2@10.0.18.249 \
  THUNDERBOLT_PEER=169.254.88.35 \
  AWDL_PEER='fe80::9477:6dff:fe11:6a55%awdl0' \
  OUTPUT=/tmp/awdl-webrtc-diagnostics.txt \
  scripts/remote-diagnostics.sh
```

The script records local reachability diagnostics for `SSH_TARGET`, checks SSH,
then builds a temporary local binary, copies it to `REMOTE_BIN` on the peer, and
runs LAN, Thunderbolt, and AWDL `-pion-net` WebRTC plus Network.framework UDP
perf, latency, and callback probes in both directions. After setup, per-profile
probes continue after failures and end with a matrix summary so one asymmetric
UDP direction does not hide later evidence.
Override `PROFILES`, `COUNT`, `DURATION`, `WARMUP`, `TRIALS`, `WINDOW`,
`STREAMS`, `TIMEOUT`, `LOCAL_BIN`, `REMOTE_BIN`, `OUTPUT`,
`NW_CONNECT_TIMEOUT`, or `NW_CONNECT_RETRIES` to narrow, lengthen, save, or tune
a run. When `DURATION` is set, sender trials run for that duration instead of a
fixed datagram count and listeners run until their timeout. Set
`REQUIRE_PATHS=1` to add `-require-path-interface` and `-forbid-loopback-path`
to sender runs; LAN defaults to `en0`, AWDL defaults to `awdl0`, and
Thunderbolt uses `THUNDERBOLT_PATH_INTERFACE` when set. Set `SSH_HOST` when the
address to probe differs from the SSH target host string.

Before cutting releases or removing local module replaces, run:

```sh
scripts/release-preflight.sh
```

It verifies the local gates, package availability, published module resolution,
published HEADs, and absence of local replaces. If local DNS cannot resolve
GitHub, set `GITHUB_RESOLVE_IP` to a current `github.com` address for the Git
remote checks:

```sh
GITHUB_RESOLVE_IP=140.82.116.3 scripts/release-preflight.sh
```

The `udp` mode opens two ordinary Go UDP sockets on the selected interface,
sets Darwin `IP_BOUND_IF` or `IPV6_BOUND_IF` for AWDL or scoped IPv6 sockets,
and sends one echo datagram. On this host, AWDL UDP passes with IPv6 link-local
addresses on `awdl0` after the AWDL gather path has activated the interface.
If a cold `udp` run times out, run `gather` first or use the two-host form.

The `udp-perf` mode runs a small iperf-like request/echo benchmark over the
same sockets and prints transfer, bitrate, datagram count, loss, and RTT
summary columns. The `-warmup` packets are omitted from the summary, and
`-trials` repeats the same run on one connection and prints an aggregate
summary when more than one trial runs. `-duration` makes each trial run for a
fixed time instead of a fixed `-count`; the JSON record includes both the
requested `duration_ns` and the actual elapsed time. `-window` sets the maximum
number of in-flight echo requests for bounded pipelining; the default `1`
preserves the serial request/echo behavior. `-streams` opens multiple client
PacketConns and aggregates the concurrent stream results into one trial record.
`udp-latency` and `udp-latency-send` use the same echo path but print only
datagram, loss, and RTT columns; `udp-latency-send` talks to a normal
`udp-perf-listen` listener. Use `-perf-json` to also print JSON result records
per trial plus an aggregate summary or listener summary. Network.framework
connections include observed path records in JSON when available. Use
`-require-path-interface awdl0` or `-forbid-loopback-path` to fail a perf or
latency run when the observed path is inconsistent with the requested link. Use
`-nw-connect-timeout` and `-nw-connect-retries` to tune outbound
Network.framework readiness retry during asymmetric link tests.
Echo timeouts are counted as lost datagrams after `-packet-timeout`, while
write and corrupt-reply errors still fail the run. This is a smoke benchmark,
not a replacement for `iperf3`.

For a two-host AWDL UDP proof, run the listener on one Mac:

```sh
go run . -profile awdl -backend go -mode udp-listen -timeout 60s
go run . -profile awdl -backend go -mode udp-perf-listen -timeout 60s
go run . -profile awdl -backend go -mode udp-callback-listen -timeout 60s
```

Then send to the printed scoped address from another Mac:

```sh
go run . -profile awdl -backend go -mode udp-send -peer '[fe80::peer%awdl0]:12345' -timeout 10s
go run . -profile awdl -backend go -mode udp-callback-request -peer '[fe80::peer%awdl0]:12345' -message ping -timeout 10s
go run . -profile awdl -backend go -mode udp-latency-send -peer '[fe80::peer%awdl0]:12345' -count 100 -size 64 -warmup 5 -streams 2 -perf-json -timeout 20s
go run . -profile awdl -backend go -mode udp-perf-send -peer '[fe80::peer%awdl0]:12345' -duration 10s -size 1200 -warmup 5 -trials 3 -window 4 -streams 2 -perf-json -timeout 40s
```

The callback probe sends a small JSON request containing a temporary callback
address. The listener writes one datagram back to that callback address. This is
meant to isolate whether the request packet reached the listener and whether
the reverse callback packet can get back.

For Network.framework diagnostics, set:

```sh
AWDL_DEMO_NETWORK_TRACE=1 go run . -profile awdl -backend network -mode udp-perf-send -peer '[fe80::peer%awdl0]:12345' -count 20 -warmup 0 -size 1200
```

The Network.framework backend proves Pion ICE gathering, raw UDP echo/perf, and
remote WebRTC datachannel exchange over LAN, Thunderbolt, and AWDL. The
`nwtransport` path demonstrates a Pion-native `transport.Net` backend for LAN,
Thunderbolt, and AWDL WebRTC. AWDL still needs explicit SSH signaling plus raw
candidate publication; the mDNS-only AWDL `SetNet` path gathers candidates but
does not open the remote datachannel here.
