# Completion Audit

This audit maps the productization goal to concrete artifacts and the current
evidence. It is intentionally strict: local gates and historical remote success
do not complete requirements that need a fresh two-host run.

## Current Gate State

| Gate | Current result |
| --- | --- |
| Release preflight | PASS: `scripts/release-preflight.sh` passes; use `GITHUB_RESOLVE_IP=140.82.116.3` when local DNS cannot resolve GitHub. |
| Published modules | PASS: the demo resolves `github.com/tmc/apple v0.6.7` and `github.com/tmc/apple-pion v0.1.0` from the module cache with no local replace. |
| Published repos | PASS: demo, `apple-pion`, and `apple` HEADs are published at `origin/main`; `apple-pion` is tagged `v0.1.0`. |
| Remote reachability | BLOCKED: `ssh -o ConnectTimeout=5 -o BatchMode=yes tmc2@10.0.18.249 true` times out. |
| Remote matrix | BLOCKED: `scripts/remote-matrix.sh` now records local route, ping, TCP/22, `scutil --nwi`, interface-list diagnostics, and `candidate_policy=auto` before checking SSH reachability; it emits `FAIL: remote reachability exit=255` and exits before building/copying while `tmc2` is unreachable. `scripts/remote-matrix-bundle.sh` writes the raw transcript and Markdown summary even for that setup failure, then returns the matrix exit code. Once setup succeeds, per-profile probes capture listener-side route, `lsof`, `netstat`, and sender-side route context, continue after failures, and end with a matrix summary. |

## Requirement Checklist

| Requirement | Artifact / evidence | Status | Remaining proof |
| --- | --- | --- | --- |
| Full Pion-native backend | `github.com/tmc/apple-pion/nwtransport` implements Pion `transport.Net`; demo uses `-pion-net`; `RESULTS.md` records LAN, Thunderbolt, and AWDL historical datachannel success. | PASS | None for the packaged UDP transport backend. Broader TCP/TURN/STUN Network.framework ownership is explicitly out of this demo slice. |
| Durable AWDL/link-local candidates | `github.com/tmc/apple-pion/icepolicy`; demo keeps SDP unmodified and publishes explicit `ICECandidateInit` records; `-candidate-policy auto\|mdns\|raw`; tests cover raw candidate extraction and auto-policy selection; local AWDL `-pion-net -mdns disabled` gather prints `candidate_policy=auto raw_candidates=true`. | PARTIAL | Re-run `-pion-net -mdns disabled -candidate-policy auto offer-ssh` on LAN, Thunderbolt, and AWDL once `tmc2` is reachable. |
| Direction/asymmetry cleanup | `udp-callback-listen`, `udp-callback-request`, path policy checks, `nwpacket` readiness retries, `remote-diagnostics.sh`, matrix-local reachability diagnostics, per-listener UDP socket/route diagnostics, and a simultaneous bidirectional UDP matrix step are implemented. | PARTIAL | Run `SSH_TARGET=tmc2@10.0.18.249 scripts/remote-diagnostics.sh`, then `REQUIRE_PATHS=1 SSH_TARGET=tmc2@10.0.18.249 scripts/remote-matrix.sh`; inspect callback, local-to-remote UDP, bidirectional UDP, listener `lsof`/`netstat`, and sender route sections. |
| Performance hardening | `udp-perf` supports `-duration`, `-trials`, `-window`, `-streams`, `-packet-timeout`, `-listen-idle-timeout`, loss columns, JSON, latency mode, bidirectional matrix probing, Network.framework path records, and `cmd/matrix-summary` Markdown summaries with failure rows. | PARTIAL | Run longer two-host matrix samples with `DURATION`, `TRIALS`, `WINDOW`, and `STREAMS` after remote reachability returns, then summarize the transcript with `go run ./cmd/matrix-summary`. |
| Reusable package split | `github.com/tmc/apple/x/network/nwpacket v0.6.7`; `github.com/tmc/apple-pion v0.1.0`; demo has no local replaces. | PASS | None. |
| Repo hygiene | Demo and `apple-pion` worktrees are clean; owned `tmc/apple/x/network/nwpacket` is clean; release preflight passes. | PASS | Preserve unrelated dirty generated/private files in `tmc/apple`; they are outside this slice. |

## Required Remote Commands

Run these when `tmc2@10.0.18.249` is reachable:

```sh
SSH_TARGET=tmc2@10.0.18.249 \
  THUNDERBOLT_PEER=169.254.88.35 \
  AWDL_PEER='fe80::9477:6dff:fe11:6a55%awdl0' \
  OUTPUT=/tmp/awdl-webrtc-diagnostics.txt \
  scripts/remote-diagnostics.sh

CANDIDATE_POLICY=auto \
REQUIRE_PATHS=1 \
SSH_TARGET=tmc2@10.0.18.249 \
OUTPUT=/tmp/awdl-webrtc-matrix.txt \
SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-summary.md \
scripts/remote-matrix-bundle.sh

DURATION=10s TRIALS=5 WINDOW=8 STREAMS=2 CANDIDATE_POLICY=auto REQUIRE_PATHS=1 \
SSH_TARGET=tmc2@10.0.18.249 \
OUTPUT=/tmp/awdl-webrtc-matrix-long.txt \
SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-long-summary.md \
scripts/remote-matrix-bundle.sh
```

The goal is not complete until the fresh remote matrix proves the auto-candidate
WebRTC path and produces longer two-host UDP performance evidence, or until the
asymmetry failures have been explained and documented with diagnostics.
