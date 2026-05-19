# Completion Audit

This audit maps the productization goal to concrete artifacts and the current
evidence. It is intentionally strict: local gates and historical remote success
do not complete requirements that need a fresh two-host run.

## Current Gate State

| Gate | Current result |
| --- | --- |
| Release preflight | PASS: `scripts/release-preflight.sh` passes; use `GITHUB_RESOLVE_IP=140.82.116.3` when local DNS cannot resolve GitHub. |
| Published modules | PASS: the demo resolves `github.com/tmc/apple v0.6.7` and `github.com/tmc/apple-pion v0.1.3` from the module cache with no local replace. |
| Published repos | PASS: demo and `apple-pion` HEADs are published at `origin/main`; `apple-pion` is tagged `v0.1.3`. |
| Remote reachability | BLOCKED: the most recent `ssh -o ConnectTimeout=3 -o BatchMode=yes tmc2@10.0.18.249 true` check timed out on port 22. `REMOTE_READY_TIMEOUT` can now wait for SSH before matrix setup, but the two-host trace run still needs the peer reachable. |
| Remote matrix | PARTIAL: `CANDIDATE_POLICY=auto REQUIRE_PATHS=1 SSH_TARGET=tmc2@10.0.18.249 REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin OUTPUT=/tmp/awdl-webrtc-matrix-v011.txt SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-v011.md scripts/remote-matrix-bundle.sh` completed all probes and failed only `awdl Pion transport.Net WebRTC`. LAN and Thunderbolt Pion-native WebRTC passed; LAN/Thunderbolt/AWDL raw Network.framework UDP perf, latency, and callback probes passed with path evidence. |

## Requirement Checklist

| Requirement | Artifact / evidence | Status | Remaining proof |
| --- | --- | --- | --- |
| Full Pion-native backend | `github.com/tmc/apple-pion/nwtransport` implements Pion `transport.Net`; demo uses `-pion-net`; `apple-pion v0.1.3` filters selected interfaces to the configured address and uses Pion address rewrite rules for link-local host publication. | PARTIAL | LAN and Thunderbolt Pion-native WebRTC pass in the current matrix. AWDL Pion-native WebRTC timed out and needs deeper instrumentation. Broader TCP/TURN/STUN Network.framework ownership is explicitly out of this demo slice. |
| Durable AWDL/link-local candidates | `github.com/tmc/apple-pion/icepolicy`; demo keeps SDP unmodified and publishes explicit `ICECandidateInit` records; `-candidate-policy auto\|mdns\|raw`; tests cover raw candidate extraction and auto-policy selection; local AWDL `-pion-net -mdns disabled` gather prints `candidate_policy=auto raw_candidates=true`; the current matrix used auto policy. | PARTIAL | Candidate publication is durable; AWDL Pion-native datachannel open is still intermittent after candidate exchange. |
| Direction/asymmetry cleanup | `udp-callback-listen`, `udp-callback-request`, path policy checks, `nwpacket` readiness retries, `remote-diagnostics.sh`, matrix-local reachability diagnostics, per-listener UDP socket/route diagnostics, and a simultaneous bidirectional UDP matrix step are implemented. | PASS | Current LAN/Thunderbolt/AWDL sequential raw UDP perf, latency, and callback probes pass both directions with path evidence. Remaining work is performance tuning, not basic reachability. |
| Performance hardening | `udp-perf` supports `-duration`, `-trials`, `-window`, `-streams`, `-packet-timeout`, `-listen-idle-timeout`, loss columns, JSON, latency mode, bidirectional matrix probing, Network.framework path records, and `cmd/matrix-summary` Markdown summaries with failure and `webrtc_trace` candidate-pair rows. | PARTIAL | Current short matrix is in [matrix-v0.1.1.md](matrix-v0.1.1.md). Run longer `DURATION`, `TRIALS`, `WINDOW`, and `STREAMS` sweeps after the AWDL WebRTC issue is instrumented. |
| Reusable package split | `github.com/tmc/apple/x/network/nwpacket v0.6.7`; `github.com/tmc/apple-pion v0.1.3`; demo has no local replaces. | PASS | None. |
| Repo hygiene | Demo and `apple-pion` worktrees are clean; owned `tmc/apple/x/network/nwpacket` is clean; release preflight passes. | PASS | Preserve unrelated dirty generated/private files in `tmc/apple`; they are outside this slice. |

## Required Remote Commands

Use these for follow-up runs:

```sh
SSH_TARGET=tmc2@10.0.18.249 \
  THUNDERBOLT_PEER=169.254.88.35 \
  AWDL_PEER='fe80::9477:6dff:fe11:6a55%awdl0' \
  OUTPUT=/tmp/awdl-webrtc-diagnostics.txt \
  scripts/remote-diagnostics.sh

CANDIDATE_POLICY=auto \
REQUIRE_PATHS=1 \
SSH_TARGET=tmc2@10.0.18.249 \
REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin \
OUTPUT=/tmp/awdl-webrtc-matrix.txt \
SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-summary.md \
scripts/remote-matrix-bundle.sh

DURATION=10s TRIALS=5 WINDOW=8 STREAMS=2 CANDIDATE_POLICY=auto REQUIRE_PATHS=1 \
SSH_TARGET=tmc2@10.0.18.249 \
REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin \
OUTPUT=/tmp/awdl-webrtc-matrix-long.txt \
SUMMARY_OUTPUT=/tmp/awdl-webrtc-matrix-long-summary.md \
scripts/remote-matrix-bundle.sh
```

The remaining incomplete item is AWDL Pion-native WebRTC reliability. Raw AWDL
UDP now passes both directions with Network.framework path evidence. The demo
now has opt-in `-webrtc-trace` instrumentation for Pion signaling, ICE
gathering, ICE connection, peer connection, datachannel transitions, and the
wire-signaling split between SDP candidates and explicit `ICECandidateInit`
records. Timeout snapshots also include local-candidate, remote-candidate, and
candidate-pair stats from Pion `GetStats`. The next remote run should use that
trace while isolating whether the AWDL timeout is in candidate installation,
connectivity checks, DTLS/SCTP, or Network.framework reads.
