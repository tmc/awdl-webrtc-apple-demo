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
| Fail closed on the actual Network.framework path | Grove `cmd/grove-perftest/perftest.go` exposes `-nw-require-interface` and `-nw-forbid-loopback`; `transport/nw/path.go` reports `NWPath` status, interface names, types, and indexes. WeightShare `awdl_data_darwin.go` reports `AWDLPathInfo` and lets `awdl-speed -require-interface awdl0` fail if the transfer did not use AWDL. | Export path reporting from `github.com/tmc/apple/x/network/nwpacket`, then add demo flags such as `-require-path-interface awdl0` and `-forbid-loopback`. Make `scripts/remote-matrix.sh` require `awdl0` for AWDL, `en0` for LAN, and the selected Thunderbolt interface for Thunderbolt. |
| Add a Bonjour-discovered control path | Grove's Network.framework transport can advertise one service and dial a peer service by name; WeightShare uses `NWEndpointCreateBonjourService` for AWDL data transfers. | Add an optional Network.framework/Bonjour signaling mode for offer/answer exchange. SSH would stay useful for orchestration, but the WebRTC proof would no longer depend on SSH as the only control plane. |
| Add callback-style reverse probes | WeightShare `MeasureAWDLSpeed` starts a temporary callback listener, sends the callback service name to the peer, and has the peer connect back to deliver data. | Use the added `udp-callback-listen` and `udp-callback-request` modes to make the peer send one datagram back to a temporary callback address. This should isolate listener/firewall/path issues behind the LAN/Thunderbolt/AWDL UDP asymmetry once the second Mac is reachable. |
| Broaden performance modes | Grove perftest has message size, warmup, iterations, duration mode, multi-stream/full-duplex mode, ping-pong latency, CPU/mem profiles, and JSON including path data. WeightShare transfers large chunks and prints path names with throughput. | Keep the current iperf-like UDP output, use the added `-duration` mode for longer samples, then add `-streams` and an explicit latency-only ping-pong mode. Include Network.framework path info in JSON once `nwpacket` exposes it. |
| Handle transient Network.framework waiting states | Grove's `dialEndpoint` retries until a deadline and treats `NWConnectionStateWaiting` with a grace timer instead of immediately failing. | Consider adding a short waiting-state grace/retry loop to `nwpacket` outbound UDP connections. This is a plausible improvement for Thunderbolt/AWDL local-to-remote readiness timeouts. |

## Discovery Ideas

Grove's `discovery` package uses Network.framework Bonjour browse/advertise
with TXT metadata, cluster scoping, deterministic peer IDs, and deterministic
rank assignment. WeightShare concurrently tries AWDL/NWBrowser discovery,
`dns-sd` Bonjour, and UDP multicast, then merges peer records.

Useful demo changes:

- Add `-mode discover` that publishes the selected profile, interface, address,
  git commit, and supported modes as TXT metadata.
- Add `-mode discover-wait` that waits for a named peer and prints a machine-
  readable peer record.
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

This demo already prints requested profile/interface and can trace paths with
`AWDL_DEMO_NETWORK_TRACE=1`, but the path is not a structured result. The next
clean API is an exported `nwpacket.PathInfo` plus helper methods:

```go
type PathInfo struct {
	Status       string
	UsesWifi     bool
	UsesLoopback bool
	UsesWired    bool
	Interfaces   []PathInterface
}

func (p PathInfo) UsesInterface(name string) bool
func (p PathInfo) InterfaceNames() []string
```

The demo can then include `path` in `udp_perf` JSON records and fail the matrix
when the observed path is inconsistent with the requested profile.

## What Not To Copy Directly

- Grove's Swift sidecar is useful history, but this demo should keep proving the
  pure Go + `tmc/apple` path.
- WeightShare's `dns-sd` parsing is pragmatic for app discovery, but the demo
  should prefer Network.framework browser APIs for structured TXT records and
  AWDL peer-to-peer parameters.
- Grove's full distributed-training transport is broader than this demo needs;
  the reusable piece is the path validation and performance-harness shape.

## Suggested Order

1. Export path reporting from `nwpacket` and require it in the remote matrix.
2. Add `-streams` to UDP perf so longer trials can exercise multiple flows
   after the fixed-duration path is validated remotely.
3. Validate callback-style reverse probes on the second Mac to isolate the
   listener asymmetry.
4. Add Bonjour discovery/signaling as a second control plane after SSH-based
   validation is green again.
