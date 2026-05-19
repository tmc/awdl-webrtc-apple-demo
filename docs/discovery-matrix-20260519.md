# Discovery-Fed Matrix - 2026-05-19

This run proves that live Bonjour/TXT discovery can feed the remote matrix with
current LAN, Thunderbolt, and AWDL peer addresses.

Command:

```sh
USE_DISCOVERY=1 \
CANDIDATE_POLICY=auto \
REQUIRE_PATHS=1 \
WEBRTC_TRACE=0 \
WEBRTC_ATTEMPTS=3 \
WEBRTC_RETRY_DELAY=3 \
REMOTE_READY_TIMEOUT=30 \
REMOTE_STEP_READY_TIMEOUT=30 \
COUNT=20 \
TRIALS=2 \
WINDOW=4 \
STREAMS=1 \
TIMEOUT=90s \
LISTEN_IDLE_TIMEOUT=3s \
SSH_TARGET=tmc2@10.0.18.249 \
REMOTE_BIN=/Volumes/Shared/awdl-webrtc-apple-demo-bin-discovery \
OUTPUT=/tmp/awdl-webrtc-discovery-matrix-20260519.txt \
SUMMARY_OUTPUT=/tmp/awdl-webrtc-discovery-matrix-20260519.md \
scripts/remote-matrix-bundle.sh
```

Artifacts:

```text
/tmp/awdl-webrtc-discovery-matrix-20260519.txt
/tmp/awdl-webrtc-discovery-matrix-20260519.md
```

## Result

| Area | Result |
| --- | --- |
| Matrix exit | PASS: `matrix_exit=0`. |
| Discovered peer | `MacBook-m4small.local` / `MacBook-m4small-local-2063`. |
| Discovered LAN | `10.0.18.249:65517`; local path `en0`. |
| Discovered Thunderbolt | `169.254.82.92:57000`; local path `bridge0`. |
| Discovered AWDL | `[fe80::4c41:acff:fec5:96f1%awdl0]:54771`; local path `awdl0`. |
| WebRTC | LAN, Thunderbolt, and AWDL Pion `transport.Net` datachannels opened. |

Discovery-sourced UDP probes:

| Link | Datagrams | Loss | Bitrate | Path |
| --- | ---: | ---: | ---: | --- |
| LAN | 38/40 | 5.00% | 581.63 Kbps | `en0` |
| Thunderbolt | 40/40 | 0.00% | 27.65 Mbps | `bridge0` |
| AWDL | 40/40 | 0.00% | 4.95 Mbps | `awdl0` |

The run used remote metadata commit
`a0941a9705e87640f0152db0913574e5e8734b87`, version
`v0.0.0-20260519094719-a0941a9705e8`, with `vcs_modified=false`.
