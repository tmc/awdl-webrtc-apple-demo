# Cross-Repo Ideas

This note records ideas from:

- `/Users/tmc/go/src/github.com/swarnim-j/grove`
- `/Users/tmc/go/src/github.com/tmc/weightshare`

Both comparison worktrees were dirty during inspection, so this pass was
read-only and treats their current contents as local implementation evidence,
not released API.

## Highest-Value Changes

| Idea | Source pattern | Change for this demo |
| --- | --- | --- |
| Fail closed on the actual Network.framework path | Grove `cmd/grove-perftest/perftest.go` exposes `-nw-require-interface` and `-nw-forbid-loopback`; `transport/nw/path.go` reports `NWPath` status, interface names, types, and indexes. WeightShare `awdl_data_darwin.go` reports `AWDLPathInfo` and lets `awdl-speed -require-interface awdl0` fail if the transfer did not use AWDL. | `github.com/tmc/apple/x/network/nwpacket` now exposes `PathReporter`, and the demo has `-require-path-interface`, `-forbid-loopback-path`, and opt-in `REQUIRE_PATHS=1` remote-matrix enforcement. |
| Add a Bonjour-discovered control path | Grove's Network.framework transport can advertise one service and dial a peer service by name; WeightShare uses `NWEndpointCreateBonjourService` for AWDL data transfers. | Add an optional Network.framework/Bonjour signaling mode for offer/answer exchange. SSH would stay useful for orchestration, but the WebRTC proof would no longer depend on SSH as the only control plane. |
| Add callback-style reverse probes | WeightShare `MeasureAWDLSpeed` starts a temporary callback listener, sends the callback service name to the peer, and has the peer connect back to deliver data. | Use the added `udp-callback-listen` and `udp-callback-request` modes to make the peer send one datagram back to a temporary callback address. This should isolate listener/firewall/path issues behind the LAN/Thunderbolt/AWDL UDP asymmetry once the second Mac is reachable. |
| Broaden performance modes | Grove perftest has message size, warmup, iterations, duration mode, multi-stream/full-duplex mode, ping-pong latency, CPU/mem profiles, and JSON including path data. WeightShare transfers large chunks and prints path names with throughput. | Keep the current iperf-like UDP output, use the added `-duration`, `-streams`, `udp-latency`, and path JSON modes for longer samples. |
| Handle transient Network.framework waiting states | Grove's `dialEndpoint` retries until a deadline and treats `NWConnectionStateWaiting` with a grace timer instead of immediately failing. | `tmc/apple v0.6.7` adds `nwpacket.Config.ConnectTimeout` and `ConnectRetries`; this demo uses a 2s readiness timeout with 2 outbound connection retries for both raw UDP and Pion `transport.Net` PacketConns, with `-nw-connect-timeout`, `-nw-connect-retries`, `NW_CONNECT_TIMEOUT`, and `NW_CONNECT_RETRIES` for remote sweeps. |
| Capture the route state before perf | Grove and WeightShare debugging both depended on knowing the actual interface and route selected by macOS, not only the requested profile. | `scripts/remote-diagnostics.sh` captures local and remote `ifconfig`, IPv4/IPv6 route tables, route-to-peer output, `scutil --nwi`, hardware ports, and UDP sockets before rerunning the matrix. |

## Discovery Ideas

Grove's `discovery` package uses Network.framework Bonjour browse/advertise
with TXT metadata, cluster scoping, deterministic peer IDs, and deterministic
rank assignment. WeightShare concurrently tries AWDL/NWBrowser discovery,
`dns-sd` Bonjour, and UDP multicast, then merges peer records.

Useful demo changes:

- `-mode discover` now publishes and browses the same Network.framework
  Bonjour/TXT metadata as the SwiftUI monitor and prints JSON records with
  local Thunderbolt, AWDL, and LAN listener addresses plus the newest peer.
  TXT metadata includes version, commit, and supported modes when available.
- `-mode discover-wait` waits for the newest peer or a peer matched by id, name,
  or Bonjour service name, prints one JSON record on stdout, and exits.
- Teach `remote-matrix.sh` to optionally use discovery output instead of hard-
  coded addresses for AWDL and Thunderbolt.
- Keep multicast out of the core WebRTC path; it is useful as a diagnostic
  fallback, not as proof of AWDL candidate handling.

## Path Evidence Shape

The two comparison repos converge on the same evidence shape:

```text
status: satisfied
uses_wifi: true
uses_loopback: false
uses_wired: false
interfaces: [{name: awdl0, type: NWInterfaceTypeWifi, index: ...}]
```

This demo now prints requested profile/interface and can trace paths with
`AWDL_DEMO_NETWORK_TRACE=1`. `nwpacket.PathReporter` exposes the current
`NWPath` for established peers:

```go
type Path struct {
	Status     network.NWPathStatus
	Interfaces []PathInterface
}

func (p Path) UsesInterface(name string) bool
func (p Path) InterfaceNames() []string
```

The demo includes `paths` in `udp_perf` and `udp_latency` JSON records when a
Network.framework peer path is available, and `-require-path-interface` /
`-forbid-loopback-path` fail a sender run when the observed path is
inconsistent with the requested profile.

## What Not To Copy Directly

- Grove's Swift sidecar is useful history, but this demo should keep proving the
  pure Go + `tmc/apple` path.
- WeightShare's `dns-sd` parsing is pragmatic for app discovery, but the demo
  should prefer Network.framework browser APIs for structured TXT records and
  AWDL peer-to-peer parameters.
- Grove's full distributed-training transport is broader than this demo needs;
  the reusable piece is the path validation and performance-harness shape.

## Suggested Order

1. Run `scripts/remote-diagnostics.sh` once the second Mac is reachable.
2. Run `REQUIRE_PATHS=1` remote matrix.
3. Validate whether `tmc/apple v0.6.7` readiness retries clear the old reverse
   AWDL/Thunderbolt timeout.
4. Validate fixed-duration, multi-stream, and latency-only runs remotely.
5. Validate callback-style reverse probes on the second Mac to isolate the
   listener asymmetry.
6. Add Bonjour discovery/signaling as a second control plane after SSH-based
   validation is green again.
