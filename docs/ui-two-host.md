# Two-Host Link-Health UI Proof

This proof validates the SwiftUI link monitor and the headless discovery path
that feeds it. The UI should show all possible local and peer paths, prefer
Thunderbolt when it completes, and fall back to AWDL when Thunderbolt is removed
or stops replying.

## One-Command Harness

Use the harness to build the current workspace binary, install it on the remote
Mac, start the remote headless discovery publisher, run a local discovery
preflight, and then launch the local SwiftUI monitor:

```sh
SSH_TARGET=tmc2@10.0.18.249 OUTPUT=/tmp/awdl-webrtc-ui-harness.txt \
  scripts/run-ui-harness.sh
```

Useful knobs:

| Variable | Default | Purpose |
| --- | --- | --- |
| `SSH_TARGET` | `tmc2@10.0.18.249` | Remote Mac used for the headless discovery publisher. |
| `REMOTE_BIN` | `/Volumes/Shared/awdl-webrtc-ui-harness-bin` | Remote install path for the built binary. |
| `REMOTE_PUBLISH_TIMEOUT` | `2h` | Keeps the remote publisher alive long enough for manual testing. |
| `UI_INTERVAL` | `2s` | SwiftUI link-health sample interval. |
| `UI_COUNT` | `20` | Datagrams per UI sample. |
| `UI_WINDOW` | `4` | In-flight datagram window per UI sample. |
| `PREFLIGHT_DISCOVERY` | `1` | Runs `discover-wait` locally before opening the UI. |
| `DISCOVER_PEER` | empty | Optional peer id, host name, or service name for preflight. When empty, the harness targets the current remote publisher's service name. |
| `LAUNCH_UI` | `1` | Set to `0` to smoke-test remote publishing and local discovery without opening SwiftUI. |
| `OUTPUT` | empty | Captures the harness transcript with `tee`. |
| `REMOTE_LOG` | temp file | Optional persistent path for the remote discovery log. |

The harness prints the physical validation steps before opening the UI. Closing
the SwiftUI window or pressing Ctrl-C tears down the remote publisher and prints
the remote discovery log.

For a non-visual smoke test of the harness plumbing:

```sh
LAUNCH_UI=0 OUTPUT=/tmp/awdl-webrtc-ui-harness-smoke.txt \
  scripts/run-ui-harness.sh
```

## Manual UI Run

Run this on both Macs:

```sh
go run . -mode ui -backend network -ui-interval 3s -ui-count 20 -ui-window 4
```

Passing evidence:

| Step | Required observation |
| --- | --- |
| Initial discovery | Both Macs show a peer and LAN/Thunderbolt/AWDL path rows when available. |
| Thunderbolt present | Active path is Thunderbolt after a successful sample. |
| Thunderbolt removed | Thunderbolt row becomes unavailable or stops completing samples; active path changes to AWDL. |
| AWDL fallback | Bandwidth-over-time rows continue with AWDL samples and no process restart. |

## Headless Companion Check

Use this when collecting terminal evidence alongside the UI:

```sh
go run . -mode discover -backend network -timeout 10s -ui-interval 1s
go run . -mode discover-wait -backend network -timeout 30s
```

Local smoke on 2026-05-19 passed: `discover-wait` found a local publisher with
Thunderbolt, AWDL, and LAN listener addresses plus TXT metadata including
`version`, `commit`, and `modes`.

Live two-host discovery on 2026-05-19 also passed through the remote matrix:
[discovery-matrix-20260519.md](discovery-matrix-20260519.md) records discovered
LAN, Thunderbolt, and AWDL peer addresses and WebRTC datachannel success on all
three profiles. That proves the headless discovery/control path used by the UI,
but it does not replace visual observation of the SwiftUI window.

## Automated Coverage

`TestLinkHealthSamplePreferredFallsBackToAWDL` and
`TestLinkHealthSamplePreferredSkipsUnavailableThunderbolt` cover the fallback
decision directly:

- Thunderbolt is sampled first.
- A Thunderbolt `no replies` result is remembered.
- AWDL is sampled next and becomes the successful active path.
- If Thunderbolt has no local address, it is skipped and AWDL is sampled.

These tests cover the selection rule. They do not replace the two-live-Mac UI
proof, because they do not render the SwiftUI window or exercise a real cable
removal.

## Live UI Active-Path Check - 2026-05-19

After fixing the collapsed text rows in `ui.go`, a local SwiftUI UI run was
paired with a remote headless discover publisher on `tmc2@10.0.18.249`.

Remote publisher:

```sh
/Volumes/Shared/awdl-webrtc-ui-proof-bin \
  -mode discover -backend network -timeout 20m -ui-interval 2s
```

Local UI:

```sh
go run . -mode ui -backend network -ui-interval 2s -ui-count 20 -ui-window 4
```

Observed visible state:

| UI area | Observation |
| --- | --- |
| Active path | `thunderbolt`, about 18-25 Mbit/s, peer `MacBook-m4small.local`. |
| Possible paths | Thunderbolt `bridge0`, AWDL `awdl0`, and LAN `en0` all had local and peer endpoints from Bonjour/TXT discovery. |
| Bandwidth history | Rows updated every two seconds with Thunderbolt samples, 0.0% loss, and millisecond RTTs. |
| Status | `sampling thunderbolt`. |

Rendering note: the bad layout came from using `Frame(width, 0)` on text cells.
`tmc/swiftui` maps `Frame` directly to SwiftUI's fixed
`.frame(width:height:)`, so height `0` collapses the row. The UI now uses
explicit row heights, shortened endpoint text, and middle truncation for long
addresses.

## Harness Smoke - 2026-05-19

The non-visual harness smoke passed from clean commit
`d7da303b7fc0a4a1ae655cbc8b4833c745528097`:

```sh
LAUNCH_UI=0 \
OUTPUT=/tmp/awdl-webrtc-ui-harness-smoke-clean.txt \
REMOTE_LOG=/tmp/awdl-webrtc-ui-harness-remote-clean.log \
  scripts/run-ui-harness.sh
```

Result:

| Check | Evidence |
| --- | --- |
| Remote publisher | `remote_service_name=MacBook-m4small-local-18016`. |
| Local preflight | `discover-wait` targeted `MacBook-m4small-local-18016` and returned `status:"peer found"`. |
| Version metadata | Remote TXT metadata reported `commit:"d7da303b7fc0a4a1ae655cbc8b4833c745528097"` and `vcs_modified:"false"`. |
| Thunderbolt | Local `bridge0` `169.254.61.91:59340`; remote `169.254.82.92:52174`; state `ready`. |
| AWDL | Local `awdl0` `[fe80::c814:f7ff:fe87:2c83%awdl0]:61523`; remote `[fe80::4c41:acff:fec5:96f1%awdl0]:55995`; state `ready`. |
| LAN | Local `en0` `10.0.199.147:50025`; remote `10.0.18.249:58068`; state `ready`. |

This proves the harness can build, install, publish, and preflight the exact
remote peer it just started. It intentionally skipped the SwiftUI window, so it
does not prove physical Thunderbolt-removal fallback.

## Current State

Peer availability is no longer the blocker: Bonjour signaling, live headless
discovery, and selected-link soak passed on two Macs. A local visible UI
active-path check also passed while fed by a remote headless publisher. The
new harness makes the final proof repeatable and its non-visual smoke passes,
but it does not replace the human observation. The remaining UI proof is a
two-live-Mac visual run plus physical Thunderbolt removal, confirming that the
visible active path moves from Thunderbolt to AWDL without restarting either
process.
