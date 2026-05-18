# WebRTC Link Demo

This temporary module shows WebRTC ICE candidate discovery over Apple-specific
local links using `github.com/tmc/apple` plus Pion WebRTC.

See [RESULTS.md](RESULTS.md) for the current answer/output table.

It has two profiles:

- `awdl`: uses `awdl0`, sets public Network.framework UDP parameters to include
  peer-to-peer Wi-Fi, and uses private `NWParameters` knobs for `useAWDL`,
  `useP2P`, exact required interface, socket access, local address reuse, and
  fallback prohibition.
- `thunderbolt`: uses the Thunderbolt Bridge, normally `bridge0`, sets public
  Network.framework UDP parameters to wired Ethernet, and uses private
  `NWParameters` to require the exact bridge interface.

Run the local checks:

```sh
go run . -profile awdl -mode check
go run . -profile awdl -mode gather -timeout 10s
go run . -profile awdl -mode udp -timeout 10s
go run . -profile awdl -mode udp-perf -count 1000 -size 1200 -warmup 5 -timeout 20s

go run . -profile thunderbolt -mode check
go run . -profile thunderbolt -mode gather -timeout 10s
go run . -profile thunderbolt -mode pair -timeout 12s
go run . -profile thunderbolt -mode udp-perf -count 1000 -size 1200 -warmup 5 -timeout 20s
```

Use `-iface` to override the default interface:

```sh
go run . -profile thunderbolt -iface en1 -mode gather -timeout 10s
```

The `gather` mode binds Pion's ICE UDP mux to the selected interface address,
enables mDNS host candidates, and verifies candidates are either from the
selected IP set or mDNS publication. mDNS is intentional: Pion filters raw IPv6
link-local host candidates for privacy.

The `pair` mode creates two local PeerConnections and exchanges a datachannel
payload over the constrained interface. On this host, Thunderbolt Bridge pairing
passes. AWDL candidate gathering passes, but a real remote AWDL WebRTC proof
still needs another Apple peer and an out-of-band signaling exchange.

The `udp` mode opens two ordinary Go UDP sockets on the selected interface,
sets Darwin `IP_BOUND_IF` or `IPV6_BOUND_IF` for AWDL or scoped IPv6 sockets,
and sends one echo datagram. On this host, AWDL UDP passes with IPv6 link-local
addresses on `awdl0` after the AWDL gather path has activated the interface.
If a cold `udp` run times out, run `gather` first or use the two-host form.

The `udp-perf` mode runs a small iperf-like request/echo benchmark over the
same sockets and prints transfer, bitrate, datagram count, and RTT summary
columns. The `-warmup` packets are omitted from the summary, which is useful
for AWDL activation outliers. This is a smoke benchmark, not a replacement for
`iperf3`.

For a two-host AWDL UDP proof, run the listener on one Mac:

```sh
go run . -profile awdl -mode udp-listen -timeout 60s
go run . -profile awdl -mode udp-perf-listen -timeout 60s
```

Then send to the printed scoped address from another Mac:

```sh
go run . -profile awdl -mode udp-send -peer '[fe80::peer%awdl0]:12345' -timeout 10s
go run . -profile awdl -mode udp-perf-send -peer '[fe80::peer%awdl0]:12345' -count 1000 -size 1200 -warmup 5 -timeout 20s
```

This is not a Network.framework-backed Pion transport. Pion still uses Go UDP
sockets; `github.com/tmc/apple` is used here to prove the Network.framework
policy knobs and private interface constraints that should sit next to the
WebRTC backend. The direct UDP path is constrained separately with Darwin
socket options, because binding only to an AWDL address is not sufficient on
macOS.
