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
| `DISCOVERY_PUBLISH_TIMEOUT` | `2h` |
| `DISCOVERY_PUBLISH_INTERVAL` | `5s` |

The wrapper writes a raw transcript and Markdown summary using a timestamped
name under `/tmp` unless `OUTPUT` and `SUMMARY_OUTPUT` are set explicitly.
It passes through matrix options such as `USE_DISCOVERY=1`, `DISCOVERY_PEER`,
and `DISCOVERY_FILE` when you want discovery-fed peer addresses in the soak
transcript. Runtime discovery keeps the remote publisher alive until matrix
cleanup by default, so long discovery-fed probes do not expire advertised
listener ports mid-run.

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

The long selected-link soak passed for LAN, Thunderbolt, and AWDL on
2026-05-19. The run is summarized in
[soak-live-20260519.md](soak-live-20260519.md) and wrote:

```text
/tmp/awdl-webrtc-soak-live-20260519.txt
/tmp/awdl-webrtc-soak-live-20260519.md
```

The same run exposed one discovery-fed probe failure: the old 60s remote
discovery publisher timeout expired advertised listener ports during the long
probe. `scripts/remote-matrix.sh` now defaults `DISCOVERY_PUBLISH_TIMEOUT=2h`,
and a LAN-only 70s discovery-fed smoke passed with:

```text
/tmp/awdl-webrtc-discovery-lifetime-smoke-20260519.txt
/tmp/awdl-webrtc-discovery-lifetime-smoke-20260519.md
```
