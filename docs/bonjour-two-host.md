# Two-Host Bonjour WebRTC Proof

This is the non-SSH signaling proof path. SSH is used only to install and start
the peer process; the WebRTC `OFFER` and `ANSWER` are exchanged through the
Network.framework Bonjour/TCP signal service.

## Command

```sh
REMOTE_READY_TIMEOUT=30 \
REMOTE_STEP_READY_TIMEOUT=30 \
PROFILES="lan thunderbolt awdl" \
PHASES="signal full" \
SSH_TARGET=tmc2@10.0.18.249 \
OUTPUT=/tmp/awdl-webrtc-bonjour-20260519.txt \
scripts/remote-bonjour.sh
```

`PHASES=signal` runs only the `-signal-only` control-plane exchange. `PHASES=full`
runs the WebRTC datachannel exchange after signaling. The default runs both.

The harness builds the local `go.work` workspace, copies the binary to the peer,
starts `answer-bonjour` remotely, waits until the Bonjour service is advertised,
and then runs local `offer-bonjour`. It captures the remote answer log for each
profile and phase, enables WebRTC trace by default, and cleans stale local and
remote demo processes whose command line starts with the configured binary path.

## Current State

The two-host Bonjour proof passed on 2026-05-19 for LAN, Thunderbolt, and AWDL
in both `signal` and `full` phases. The successful run is summarized in
[bonjour-live-20260519.md](bonjour-live-20260519.md) and saved at:

```text
/tmp/awdl-webrtc-bonjour-live2-20260519.txt
```

Result:

| Profile | Signal-only | Full datachannel | Interface |
| --- | --- | --- | --- |
| LAN | PASS | PASS | `en0` |
| Thunderbolt | PASS | PASS | `bridge0` |
| AWDL | PASS | PASS | `awdl0` |

Final summary:

```text
bonjour passed profiles=lan thunderbolt awdl phases=signal full
```

Earlier unreachable/refused SSH-control runs remain historical control-plane
diagnostics, not the current state.

## Passing Criteria

| Profile | Signal-only | Full datachannel | Required evidence |
| --- | --- | --- | --- |
| LAN | PASS | PASS | `bonjour signal exchanged...` and `webrtc datachannel opened...` |
| Thunderbolt | PASS | PASS | Same, with Thunderbolt-constrained profile output |
| AWDL | PASS | PASS | Same, with explicit link-local candidates and `awdl0` path evidence |

Do not treat same-host Bonjour success as completion. The completion boundary is
two live Macs.
