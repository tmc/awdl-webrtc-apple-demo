# WebRTC Link Demo

This temporary module shows WebRTC ICE candidate discovery over Apple-specific
local links using `github.com/tmc/apple` plus Pion WebRTC.

See [RESULTS.md](RESULTS.md) for the current answer/output table and
[docs/completion-audit.md](docs/completion-audit.md) for the strict
completion checklist.

The reusable Network.framework surfaces are:

- `github.com/tmc/apple/x/network/nwpacket`: a Network.framework
  `net.PacketConn`, consumed from the sibling `../apple` checkout through
  `go.work` during active development.
- `github.com/tmc/apple-pion/nwtransport`: a small Pion `transport.Net`
  adapter, consumed from the sibling `../apple-pion` checkout through
  `go.work`. It
  routes concrete UDP listeners, configured wildcard UDP listeners, and UDP
  dials through Network.framework, while leaving DNS, TCP, unconstrained
  wildcard UDP, and TURN/STUN helper traffic outside that selected UDP surface
  on Pion's standard network fallback.
- `github.com/tmc/apple-pion/icepolicy`: explicit link-local host-candidate
  publication helpers used by the demo's `-candidate-policy` signaling path.

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
go run . -profile awdl -backend network -mdns disabled -mode gather -timeout 10s

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
go run . -mode ui -backend network -ui-interval 3s -ui-count 20 -ui-window 4
```

The UI advertises a Bonjour service, starts local UDP echo listeners for
Thunderbolt, AWDL, and LAN when those paths are available, and samples the peer
in that order. A Thunderbolt path with no replies is marked unavailable for
that sample and the monitor immediately tries AWDL, then LAN. The checked-in
`go.work` file intentionally resolves the local sibling Apple bindings.
See [docs/ui-two-host.md](docs/ui-two-host.md) for the two-live-Mac validation
steps and the local fallback-test coverage.

For a terminal-friendly view of the same Bonjour/TXT discovery data, use
`discover` mode:

```sh
go run . -mode discover -backend network -timeout 10s -ui-interval 1s
```

It prints JSON records with the local Thunderbolt, AWDL, and LAN listener
addresses plus the newest discovered peer metadata, including advertised
version, commit, and supported modes when available. This mode is useful for
checking whether both Macs can see each other before running the UI or the
manual WebRTC signaling flow.

To wait for one peer record and exit, use:

```sh
go run . -mode discover-wait -backend network -timeout 30s
go run . -mode discover-wait -backend network -discover-peer peer-service-name -timeout 30s
```

`-discover-peer` matches the peer id, host name, or Bonjour service name. An
empty value accepts the newest discovered peer. Discovery modes write setup
diagnostics to stderr and JSON records to stdout.

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
seam when explicit signaling uses `-mdns disabled -candidate-policy auto`.
AWDL mDNS candidate exchange still does not open a remote datachannel in this
environment.

The `gather` mode binds Pion's ICE UDP mux to the selected interface address,
enables mDNS host candidates, and verifies candidates are either from the
selected IP set or mDNS publication. mDNS is intentional: Pion filters raw IPv6
link-local host candidates for privacy.

For explicit two-process signaling, `-candidate-policy auto` is the default. It
enables the reusable `apple-pion/icepolicy` host-candidate policy when mDNS is
disabled on AWDL or another IPv6 link-local interface. The policy uses a
synthetic non-link-local host candidate inside Pion and publishes the selected
interface IP as explicit `ICECandidateInit` records alongside the unmodified
SDP. This keeps the demo no-fork while making AWDL link-local ICE testable
without SDP rewriting. Use `-candidate-policy mdns` to suppress explicit
candidate publication, `-candidate-policy raw` to force it, or the legacy
`-raw-candidates` alias for older scripts.

For WebRTC diagnostics, set `-webrtc-trace` or `AWDL_DEMO_WEBRTC_TRACE=1`.
The trace prints Pion signaling, ICE gathering, ICE connection, peer
connection, datachannel transitions, and the wire-signaling candidate split
between SDP candidates and explicit `ICECandidateInit` records. Timeout
snapshots include Pion local-candidate, remote-candidate, and candidate-pair
stats so failed runs show which ICE pairs were checked and whether requests or
responses moved.

The `pair` mode creates two local PeerConnections and exchanges a datachannel
payload over the constrained interface. On this host, Thunderbolt Bridge pairing
passes. AWDL same-host pairing is not useful because AWDL traffic is peer-link
traffic; use `offer-stdio` or `offer-ssh` against another Apple host.

The `offer-stdio` and `answer-stdio` modes use the same WebRTC wire signal as
`offer-ssh`, but leave the control plane to the operator. This is useful when
SSH to the peer is unavailable or when testing another discovery/signaling
transport. Start the answer side on one Mac:

```sh
go run . -profile awdl -backend network -pion-net -mdns disabled \
  -candidate-policy auto -mode answer-stdio -timeout 90s
```

Start the offer side on the other Mac:

```sh
go run . -profile awdl -backend network -pion-net -mdns disabled \
  -candidate-policy auto -mode offer-stdio -timeout 90s
```

Move the printed `OFFER ...` line to the answer process, then move the printed
`ANSWER ...` line back to the offer process. The datachannel exchange then runs
over the selected LAN, Thunderbolt, or AWDL ICE path, independent of how the two
signal lines were transported.

The experimental `answer-bonjour` and `offer-bonjour` modes use
Network.framework Bonjour advertise/browse plus Network.framework TCP for the
same `OFFER`/`ANSWER` wire signal:

```sh
go run . -profile awdl -backend network -pion-net -mdns disabled \
  -candidate-policy auto -mode answer-bonjour -signal-name awdl-webrtc-peer-a \
  -timeout 90s

go run . -profile awdl -backend network -pion-net -mdns disabled \
  -candidate-policy auto -mode offer-bonjour -signal-peer awdl-webrtc-peer-a \
  -timeout 90s
```

The answer side advertises `_awdl-webrtc-signal._tcp` with the same version,
commit, and supported-mode TXT metadata used by the link monitor. The current
local same-host smoke confirms Bonjour browse/resolve and exchanges the signal
lines by falling back from the Bonjour endpoint to `NSNetService` host/port
resolution plus Network.framework TCP. Same-host ICE still does not open the
datachannel, so use two Macs for the real link proof. Add `-signal-only` on
both sides to exit successfully after the WebRTC `OFFER`/`ANSWER` exchange
when you only want to verify the Bonjour control plane.

For a repeatable two-host Bonjour proof, use the harness:

```sh
REMOTE_READY_TIMEOUT=30 \
PROFILES="lan thunderbolt awdl" \
PHASES="signal full" \
SSH_TARGET=tmc2@10.0.18.249 \
OUTPUT=/tmp/awdl-webrtc-bonjour.txt \
scripts/remote-bonjour.sh
```

The harness uses SSH only to copy and launch the peer binary. The WebRTC signal
itself is exchanged through Bonjour/TCP. See
[docs/bonjour-two-host.md](docs/bonjour-two-host.md) for passing criteria and
the current blocked reachability evidence.

The `offer-ssh` mode runs the local side, starts `answer-stdio` on a peer over
SSH, exchanges SDP plus optional explicit ICE candidates over stdin/stdout, and
waits for a WebRTC datachannel `ping`/`pong`:

```sh
go run . -profile thunderbolt -backend network -mdns disabled -mode offer-ssh \
  -ssh tmc2@10.0.18.249 -remote-bin /tmp/awdl-webrtc-apple-demo-bin \
  -timeout 35s

go run . -profile awdl -backend network -mdns disabled \
  -mode offer-ssh -ssh tmc2@10.0.18.249 \
  -remote-bin /tmp/awdl-webrtc-apple-demo-bin -timeout 45s

go run . -profile lan -backend network -pion-net -mdns disabled \
  -mode offer-ssh -ssh tmc2@10.0.18.249 \
  -remote-bin /Volumes/Shared/awdl-webrtc-apple-demo-bin -timeout 35s

go run . -profile thunderbolt -backend network -pion-net -mdns disabled \
  -mode offer-ssh -ssh tmc2@10.0.18.249 \
  -remote-bin /Volumes/Shared/awdl-webrtc-apple-demo-bin -timeout 40s

go run . -profile awdl -backend network -pion-net -mdns disabled \
  -mode offer-ssh -ssh tmc2@10.0.18.249 \
  -remote-bin /Volumes/Shared/awdl-webrtc-apple-demo-bin -timeout 45s
```

To rerun the two-host productization matrix, use:

```sh
CANDIDATE_POLICY=auto \
REQUIRE_PATHS=1 \
WEBRTC_ATTEMPTS=3 \
WEBRTC_RETRY_DELAY=3 \
DURATION=5s \
TRIALS=3 \
WINDOW=8 \
STREAMS=2 \
TIMEOUT=90s \
LISTEN_IDLE_TIMEOUT=3s \
REMOTE_READY_TIMEOUT=30 \
REMOTE_STEP_READY_TIMEOUT=30 \
SSH_TARGET=tmc2@10.0.18.249 \
REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin-workspace \
OUTPUT=/tmp/awdl-webrtc-matrix.txt \
SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-summary.md \
scripts/remote-matrix-bundle.sh
```

For longer repeated performance sweeps, use:

```sh
SOAK_LABEL=workspace-soak \
SSH_TARGET=tmc2@10.0.18.249 \
scripts/remote-soak.sh
```

`remote-soak.sh` wraps the matrix bundle with longer duration/trial defaults and
writes timestamped raw and Markdown artifacts. See
[docs/soak-sweeps.md](docs/soak-sweeps.md) for defaults and passing criteria.

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
perf, simultaneous bidirectional perf, latency, and callback probes. On a
reachability failure, the transcript includes a normal `FAIL:` line so
`cmd/matrix-summary` can still render the setup blocker in its Markdown table.
After setup, per-profile probes continue after failures and end with a matrix summary
so one asymmetric UDP direction does not hide later evidence. The
UDP probes record listener-side route, `lsof`, and `netstat` output for the
printed listener host/port plus sender-side route checks before sending, so
zero-datagram failures have socket and route context in the transcript. The
`cmd/matrix-summary` command reads the saved transcript and renders the JSON
perf/latency records plus de-duplicated failed matrix steps as a compact
Markdown table. WebRTC timeout trace snapshots are summarized as `webrtc_trace`
rows with candidate addresses and candidate-pair request/response counters. Use
`scripts/remote-matrix-bundle.sh` when you want both files written in one run;
it preserves the matrix exit code after writing the summary.
Override `PROFILES`, `COUNT`, `DURATION`, `WARMUP`, `TRIALS`, `WINDOW`,
`STREAMS`, `TIMEOUT`, `LOCAL_BIN`, `REMOTE_BIN`, `OUTPUT`,
`NW_CONNECT_TIMEOUT`, `NW_CONNECT_RETRIES`, `CANDIDATE_POLICY`, or
`LISTEN_IDLE_TIMEOUT` to narrow, lengthen, save, or tune a run. Set
`WEBRTC_TRACE=1` to add `-webrtc-trace` to each Pion `transport.Net` WebRTC
step. Set `REMOTE_READY_TIMEOUT` to an integer number of seconds to retry SSH
reachability before setup; `REMOTE_READY_INTERVAL` controls the retry sleep.
Set `REMOTE_STEP_READY_TIMEOUT` to an integer number of seconds to retry SSH
reachability before each remote sender or listener command when the peer is
flaky after setup. By default the matrix also kills stale local and remote demo
processes whose command line starts with `LOCAL_BIN` or `REMOTE_BIN`, so an
interrupted previous run cannot keep the temporary binary active while the next
run replaces it. Set `CLEAN_STALE_PROCESSES=0` to disable that cleanup.
`CANDIDATE_POLICY` defaults to `auto` for the `-pion-net -mdns disabled`
WebRTC step. When `DURATION` is set, sender trials run for that duration
instead of a fixed datagram count and listeners stop after
`LISTEN_IDLE_TIMEOUT` once traffic goes idle. Set
`REQUIRE_PATHS=1` to add `-require-path-interface` and `-forbid-loopback-path`
to sender runs; LAN defaults to `en0`, AWDL defaults to `awdl0`, and
Thunderbolt uses `THUNDERBOLT_PATH_INTERFACE` when set. Set `SSH_HOST` when the
address to probe differs from the SSH target host string.

For a published-module release gate, run:

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

This release gate is intentionally separate from the current `go.work` proof
path, which builds against sibling `../apple` and `../apple-pion` checkouts
without publishing new tags.

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
write and corrupt-reply errors still fail the run. For duration-based
two-process tests, set `-listen-idle-timeout` on `udp-perf-listen` so the
listener exits after the sender stops instead of waiting for the outer
`-timeout`. This is a smoke benchmark, not a replacement for `iperf3`.

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
Thunderbolt, and AWDL WebRTC. AWDL still needs explicit signaling plus
link-local candidate publication through `-candidate-policy auto` or `raw`;
the mDNS-only AWDL `SetNet` path gathers candidates but does not open the
remote datachannel here.
