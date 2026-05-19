# Two-Host Link-Health UI Proof

This proof validates the SwiftUI link monitor and the headless discovery path
that feeds it. The UI should show all possible local and peer paths, prefer
Thunderbolt when it completes, and fall back to AWDL when Thunderbolt is removed
or stops replying.

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

## Current State

The remote Mac was unreachable on 2026-05-19, so the two-live-Mac UI proof is
still blocked on peer availability.
