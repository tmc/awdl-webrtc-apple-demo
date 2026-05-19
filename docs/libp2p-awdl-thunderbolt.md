# libp2p AWDL and Thunderbolt Investigation

Date: 2026-05-19

Scope:

- `github.com/libp2p/go-libp2p` v0.48.0
- `github.com/multiformats/go-multiaddr` v0.16.1
- `github.com/tmc/apple` v0.6.9 as the published Apple binding surface

## Short Answer

go-libp2p does not have first-class AWDL or Thunderbolt support today. It does
have the right extension points to add it without forking libp2p.

Thunderbolt is mostly an address selection and discovery problem. Thunderbolt
Bridge appears as a normal IP interface, usually `bridge0`, with link-local IPv4
addresses. A peer can advertise and dial ordinary libp2p multiaddrs such as:

```text
/ip4/169.254.61.91/tcp/4001
/ip4/169.254.61.91/udp/4001/quic-v1
```

AWDL has usable address representation in multiaddr, but reliable operation
needs Darwin-specific discovery and path activation. AWDL peers use IPv6
link-local addresses scoped to `awdl0`, represented as:

```text
/ip6zone/awdl0/ip6/fe80::c814:f7ff:fe87:2c83/udp/4001/quic-v1
/ip6zone/awdl0/ip6/fe80::c814:f7ff:fe87:2c83/tcp/4001
```

The missing piece is not a new multiaddr protocol. The missing piece is an
Apple-aware discovery and transport policy that can activate peer-to-peer AWDL,
prefer Thunderbolt when present, fall back to AWDL when it disappears, and fail
closed when the selected path is not the requested interface.

## Current libp2p Fit

| Link | Current libp2p representation | Works with stock go-libp2p? | Missing for product-quality support |
| --- | --- | --- | --- |
| Thunderbolt Bridge | `/ip4/169.254.x.y/tcp/...` or `/ip4/169.254.x.y/udp/.../quic-v1` | Likely yes, once explicit peer addresses are discovered and advertised. | Discovery, address ranking, and fail-closed proof that the path is `bridge0`. |
| AWDL | `/ip6zone/awdl0/ip6/fe80::.../tcp/...` or `/udp/.../quic-v1` | Addressing yes. Socket success depends on AWDL being active and selected by macOS. | Network.framework peer-to-peer browse/advertise, path policy, scoped listener/dialer, fallback, and path evidence. |
| LAN | Normal `/ip4`, `/ip6`, `/dns*` libp2p addresses | Yes. | Keep as fallback, but avoid advertising it ahead of Thunderbolt or AWDL for local high-bandwidth peers. |

go-libp2p already exposes the major control points:

- `libp2p.ListenAddrs` sets explicit listen multiaddrs.
- `libp2p.AddrsFactory` can filter or order advertised addresses.
- `libp2p.Transport` accepts custom transports.
- `transport.Transport` is small: `Dial`, `CanDial`, `Listen`,
  `Protocols`, and `Proxy`.
- The TCP transport has `WithDialerForAddr`, which can replace the dial side.
- The QUIC reuse manager has `OverrideListenUDP`, which can provide a custom
  `net.PacketConn` for QUIC sockets.
- Built-in mDNS advertises interface listen addresses, but it is ordinary
  mDNS. It does not expose Apple Network.framework peer-to-peer policy or an
  AWDL-only path guarantee.

## Recommended Shape

### 1. External package first

Build a Darwin-only package outside upstream go-libp2p:

```text
github.com/tmc/apple-libp2p/nwdiscovery
github.com/tmc/apple-libp2p/nwtransport
```

Keep the first version application-owned. That lets us prove the behavior on two
Macs without waiting for upstream API review and without putting Apple-specific
policy into go-libp2p core.

### 2. Discovery

Use Network.framework Bonjour browse/advertise through `github.com/tmc/apple`.
Publish TXT metadata with:

- peer ID
- service instance ID
- supported paths: `thunderbolt`, `awdl`, `lan`
- selected interface names
- listen multiaddrs
- version/build metadata

The discovery package should emit `peer.AddrInfo` values. The application can
insert them into the peerstore and call `host.Connect`.

This mirrors the working demo discovery path instead of relying only on
go-libp2p mDNS. mDNS is still useful for normal LAN behavior, but it is not the
right primitive for AWDL activation and path proof.

### 3. Address factory

Use `ListenAddrs` and `AddrsFactory` to make address publication explicit:

```go
h, err := libp2p.New(
	libp2p.ListenAddrs(tbQUIC, awdlQUIC, lanQUIC),
	libp2p.AddrsFactory(func(addrs []ma.Multiaddr) []ma.Multiaddr {
		return preferAppleLinks(addrs)
	}),
)
```

`preferAppleLinks` should:

- retain only addresses that are actually present on local interfaces
- order Thunderbolt before AWDL before LAN
- keep AWDL addresses scoped with `/ip6zone/awdl0`
- optionally suppress LAN when a direct Apple link is healthy

Do not put `/p2p/<peer>` in listen addresses. Add the peer ID when sharing
`peer.AddrInfo` values.

### 4. Transport

Start with the smallest transport that proves the path:

1. TCP over Network.framework, if the published `tmc/apple` TCP listener and
   connection surfaces are ready.
2. QUIC over Network.framework UDP once a `net.PacketConn` wrapper can satisfy
   quic-go and go-libp2p's QUIC reuse manager.

For QUIC, the useful existing seam is `quicreuse.OverrideListenUDP`. The
transport can provide a Network.framework-backed `net.PacketConn` that binds
listeners and dials to a requested interface. This is similar to the Pion
`transport.Net` shape already proven in this repo, but adapted to libp2p's QUIC
reuse layer.

The transport must expose path policy:

- require interface name: `bridge0`, `awdl0`, or a configured LAN interface
- enable peer-to-peer Wi-Fi for AWDL
- prohibit silent fallback when a path was required
- report the actual Network.framework path for tests and logs

### 5. Dial ranking and fallback

Use address ordering plus a swarm dial ranker:

```text
Thunderbolt bridge0
AWDL awdl0
LAN
Relay or public addresses
```

The fallback should be observable:

- connect over Thunderbolt when `bridge0` is available
- open streams over the Thunderbolt connection
- when Thunderbolt is removed, new dials or reconnects select AWDL
- if AWDL is unavailable, LAN remains the fallback

Existing streams may need application-level reconnect behavior. libp2p can open
new streams over an existing connection, but if the physical path disappears the
application should expect connection close, reconnect, and stream recreation.

## Upstream Candidates

Keep Apple-specific Network.framework code out of go-libp2p at first. The
pieces that may be upstreamable are smaller:

- tests and docs for `/ip6zone/<iface>/ip6/fe80::...` link-local addresses
- clearer examples for interface-scoped listen addresses
- a public path/address ranking example for local high-bandwidth links
- any missing hooks discovered while wiring QUIC to an injected `net.PacketConn`

If go-libp2p accepts Apple path policy later, it should likely be an optional
transport module, not a change to the core transport interfaces.

## Proof Plan

| Step | Result |
| --- | --- |
| Address smoke | Two nodes exchange `peer.AddrInfo` with Thunderbolt, AWDL, and LAN multiaddrs. |
| Thunderbolt dial | `libp2p ping` and a test stream pass over `/ip4/169.254.../udp/.../quic-v1`; path reporter says `bridge0`. |
| AWDL dial | `libp2p ping` and a test stream pass over `/ip6zone/awdl0/ip6/fe80::.../udp/.../quic-v1`; path reporter says `awdl0`. |
| Priority | With all links present, the selected libp2p connection is Thunderbolt. |
| Fallback | Remove Thunderbolt cable; reconnect selects AWDL without restarting either node. |
| LAN fallback | Disable AWDL or move peers out of range; reconnect selects LAN. |
| Fail closed | A run requiring `awdl0` fails if Network.framework reports any other interface. |

## Conclusion

We do not need to change much about libp2p to demonstrate this. The path is:

1. explicit link-local multiaddrs,
2. Network.framework Bonjour discovery,
3. `AddrsFactory` and dial ranking,
4. Darwin Network.framework transport or QUIC `PacketConn` injection,
5. path evidence and fallback tests.

The highest-value next artifact is a small `apple-libp2p` demo that uses
published `github.com/tmc/apple` bindings to run a libp2p ping/stream test over
Thunderbolt and AWDL with the same result table style as this repo.

## Sources

- go-libp2p v0.48.0 `transport.Transport` API:
  https://pkg.go.dev/github.com/libp2p/go-libp2p@v0.48.0/core/transport#Transport
- go-libp2p v0.48.0 options:
  https://pkg.go.dev/github.com/libp2p/go-libp2p@v0.48.0#ListenAddrs
  and https://pkg.go.dev/github.com/libp2p/go-libp2p@v0.48.0#AddrsFactory
- go-multiaddr v0.16.1 scoped IPv6 conversion:
  https://pkg.go.dev/github.com/multiformats/go-multiaddr@v0.16.1/net#FromIPAndZone
- libp2p transport overview:
  https://docs.libp2p.io/concepts/transports/overview/
- libp2p mDNS overview:
  https://docs.libp2p.io/concepts/discovery-routing/mdns/
