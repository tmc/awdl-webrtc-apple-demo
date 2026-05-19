# Long Soak Sweeps

Use `scripts/remote-soak.sh` for longer repeated LAN, Thunderbolt, and AWDL
performance runs. It wraps `remote-matrix-bundle.sh` with product-oriented
defaults while keeping the checked-in `go.work` workspace and no-new-tags
workflow.

## Command

```sh
SOAK_LABEL=workspace-soak \
SSH_TARGET=tmc2@10.0.18.249 \
scripts/remote-soak.sh
```

Default settings:

| Setting | Default |
| --- | --- |
| `DURATION` | `30s` |
| `TRIALS` | `5` |
| `WINDOW` | `8` |
| `STREAMS` | `2` |
| `TIMEOUT` | `180s` |
| `LISTEN_IDLE_TIMEOUT` | `5s` |
| `REQUIRE_PATHS` | `1` |
| `WEBRTC_ATTEMPTS` | `3` |
| `REMOTE_READY_TIMEOUT` | `60` |
| `REMOTE_STEP_READY_TIMEOUT` | `30` |

The wrapper writes a raw transcript and Markdown summary using a timestamped
name under `/tmp` unless `OUTPUT` and `SUMMARY_OUTPUT` are set explicitly.
It passes through matrix options such as `USE_DISCOVERY=1`, `DISCOVERY_PEER`,
and `DISCOVERY_FILE` when you want discovery-fed peer addresses in the soak
transcript.

## Passing Criteria

| Requirement | Evidence |
| --- | --- |
| LAN, Thunderbolt, and AWDL covered | Summary contains all three profiles. |
| Repeated throughput | Each profile has `udp_perf_summary` rows for remote-to-local, local-to-remote, and bidirectional senders. |
| Latency | Each profile has `udp_latency_summary` rows in both directions. |
| Loss accounting | Summary rows include datagrams, expected datagrams, lost packets, and loss percentage. |
| Path evidence | Network.framework path rows include `en0`, `bridge0`, or `awdl0` as appropriate when `REQUIRE_PATHS=1`. |
| WebRTC regression | Each profile's Pion `transport.Net` WebRTC step opens a datachannel before UDP sweeps are trusted. |

## Current State

The wrapper was syntax-checked and exercised with a short remote-readiness
timeout on 2026-05-19. Because `tmc2@10.0.18.249` was unreachable, it stopped at
the reachability gate and wrote:

```text
/tmp/awdl-webrtc-dry-run-test.txt
/tmp/awdl-webrtc-dry-run-test.md
```

No long-soak performance claim is made until the peer is reachable and the full
default command passes.
