# Live Bonjour WebRTC Proof - 2026-05-19

This run proves the non-SSH WebRTC control plane on two Macs. SSH was used to
install and start the remote process only; WebRTC `OFFER` and `ANSWER` payloads
were exchanged over `_awdl-webrtc-signal._tcp`.

Command:

```sh
REMOTE_READY_TIMEOUT=30 \
REMOTE_STEP_READY_TIMEOUT=30 \
PROFILES="lan thunderbolt awdl" \
PHASES="signal full" \
SSH_TARGET=tmc2@10.0.18.249 \
OUTPUT=/tmp/awdl-webrtc-bonjour-live2-20260519.txt \
scripts/remote-bonjour.sh
```

Artifact:

```text
/tmp/awdl-webrtc-bonjour-live2-20260519.txt
```

## Result

| Profile | Interface | Signal-only | Full datachannel | Evidence |
| --- | --- | --- | --- | --- |
| LAN | `en0` | PASS | PASS | `bonjour signal exchanged... over en0`; `webrtc datachannel opened... over en0-constrained ICE`; remote answer sent pong. |
| Thunderbolt | `bridge0` | PASS | PASS | `bonjour signal exchanged... over bridge0`; `webrtc datachannel opened... over bridge0-constrained ICE`; remote answer sent pong. |
| AWDL | `awdl0` | PASS | PASS | `bonjour signal exchanged... over awdl0`; `webrtc datachannel opened... over awdl0-constrained ICE`; remote answer sent pong. |

Final summary:

```text
bonjour passed profiles=lan thunderbolt awdl phases=signal full
```

AWDL used explicit link-local ICE candidates on `awdl0`: local
`fe80::c814:f7ff:fe87:2c83`, remote `fe80::4c41:acff:fec5:96f1`.
