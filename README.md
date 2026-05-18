# WebRTC Link Demo

This temporary module shows WebRTC ICE candidate discovery over Apple-specific
local links using `github.com/tmc/apple` plus Pion WebRTC.

See [RESULTS.md](RESULTS.md) for the current answer/output table.

The reusable Network.framework surfaces are:

- `github.com/tmc/apple/x/network/nwpacket`: a Network.framework
  `net.PacketConn`, consumed from released `github.com/tmc/apple v0.6.4`.
- `github.com/tmc/apple-pion/nwtransport`: a small Pion `transport.Net`
  adapter, consumed from the sibling `tmc/apple-pion` checkout through the
  temporary `replace github.com/tmc/apple-pion => ../apple-pion`. It routes
  concrete UDP listeners, configured wildcard UDP listeners, and UDP dials
  through Network.framework, while leaving DNS, TCP, unconstrained wildcard
  UDP, and TURN/STUN helper traffic outside that selected UDP surface on
  Pion's standard network fallback.

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

The `-backend` flag selects `go` or `network`.

- `go` uses ordinary Darwin UDP sockets. It is the stable throughput and
  same-host WebRTC datachannel path.
- `network` uses Network.framework as a `net.PacketConn`. It creates clear UDP
  parameters, sets `includePeerToPeer`, uses required interface type where it
  works, and for AWDL uses the private `NWInterface.cInterface` object to force
  the path to `awdl0`.

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
the demo's AWDL host-candidate policy. It uses a synthetic non-link-local host
candidate inside Pion, strips candidates from the SDP, and publishes the
selected interface IP as explicit `ICECandidateInit` records. This keeps the
demo no-fork while making AWDL link-local ICE testable without SDP rewriting.

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
SSH_TARGET=tmc2@10.0.18.249 scripts/remote-matrix.sh
```

The script builds a temporary local binary, copies it to `REMOTE_BIN` on the
peer, then runs LAN, Thunderbolt, and AWDL `-pion-net` WebRTC plus
Network.framework UDP perf in both directions. Override `PROFILES`, `COUNT`,
`WARMUP`, `TRIALS`, `WINDOW`, `TIMEOUT`, `LOCAL_BIN`, or `REMOTE_BIN` to narrow
or lengthen a run.

Before cutting releases or removing local module replaces, run:

```sh
scripts/release-preflight.sh
```

It verifies the local gates and package availability, then fails until the demo
and `apple-pion` have remotes and the remaining local `apple-pion` replace can
be removed.

The `udp` mode opens two ordinary Go UDP sockets on the selected interface,
sets Darwin `IP_BOUND_IF` or `IPV6_BOUND_IF` for AWDL or scoped IPv6 sockets,
and sends one echo datagram. On this host, AWDL UDP passes with IPv6 link-local
addresses on `awdl0` after the AWDL gather path has activated the interface.
If a cold `udp` run times out, run `gather` first or use the two-host form.

The `udp-perf` mode runs a small iperf-like request/echo benchmark over the
same sockets and prints transfer, bitrate, datagram count, loss, and RTT
summary columns. The `-warmup` packets are omitted from the summary, and
`-trials` repeats the same run on one connection and prints an aggregate
summary when more than one trial runs. `-window` sets the maximum number of
in-flight echo requests for bounded pipelining; the default `1` preserves the
serial request/echo behavior. Use `-perf-json` to also print JSON result
records per trial plus an aggregate summary or listener summary. Echo timeouts
are counted as lost datagrams after `-packet-timeout`, while write and
corrupt-reply errors still fail the run. This is a smoke benchmark, not a
replacement for `iperf3`.

For a two-host AWDL UDP proof, run the listener on one Mac:

```sh
go run . -profile awdl -backend go -mode udp-listen -timeout 60s
go run . -profile awdl -backend go -mode udp-perf-listen -timeout 60s
```

Then send to the printed scoped address from another Mac:

```sh
go run . -profile awdl -backend go -mode udp-send -peer '[fe80::peer%awdl0]:12345' -timeout 10s
go run . -profile awdl -backend go -mode udp-perf-send -peer '[fe80::peer%awdl0]:12345' -count 1000 -size 1200 -warmup 5 -trials 3 -window 4 -perf-json -timeout 20s
```

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
