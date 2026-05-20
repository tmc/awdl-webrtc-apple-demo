# Workspace Gate 2026-05-20

This note records the local gate state after the UI harness smoke docs were
committed. It separates the current demo gate from the checked-in workspace
gate, because the workspace includes sibling checkouts that can be dirty for
unrelated work.

## Summary

| Gate | Result | Evidence |
| --- | --- | --- |
| Published-module demo gate | PASS | `GOWORK=off go test ./...` and `GOWORK=off go vet ./...` passed with `github.com/tmc/apple v0.6.7` and `github.com/tmc/apple-pion v0.1.3`. |
| UI harness syntax and published-module smoke | PASS | `test -x scripts/run-ui-harness.sh && bash -n scripts/run-ui-harness.sh` passed. `LAUNCH_UI=0 BUILD_GOWORK=off ... scripts/run-ui-harness.sh` passed from clean commit `1288332`, using `env GOWORK=off go build` and discovering LAN, Thunderbolt, and AWDL paths on the remote Mac. |
| Checked-in `go.work` gate | BLOCKED | `go test ./...` fails before demo tests finish because local sibling `../apple` is dirty. |
| `../apple-pion` sibling | CLEAN | `git -C ../apple-pion status --short --branch` reports `## main...origin/main`. |
| `../apple` sibling | DIRTY | `git -C ../apple status --porcelain | wc -l` reports 382 entries, including generated `foundation` and `kernel` files. |

## Commands

```sh
go env GOWORK
# /Volumes/tmc/go/src/github.com/tmc/awdl-webrtc-apple-demo/go.work

cat go.work
# go 1.25.0
# use (
#     .
#     ../apple
#     ../apple-pion
# )

GOWORK=off go list -m github.com/tmc/apple github.com/tmc/apple-pion
# github.com/tmc/apple v0.6.7
# github.com/tmc/apple-pion v0.1.3

GOWORK=off go test ./...
# ok   github.com/tmc/awdl-webrtc-apple-demo (cached)
# ok   github.com/tmc/awdl-webrtc-apple-demo/cmd/matrix-summary (cached)

GOWORK=off go vet ./...
# pass

go test ./...
# FAIL github.com/tmc/awdl-webrtc-apple-demo [build failed]
# ok   github.com/tmc/awdl-webrtc-apple-demo/cmd/matrix-summary (cached)
# FAIL
# github.com/tmc/apple/foundation
# ../apple/foundation/nsuuid.gen.go:131:33: undefined: kernel.Uuid_t
```

## Interpretation

The 2026-05-19 two-host matrix remains valid historical evidence that the
workspace-built binary passed LAN, Thunderbolt, and AWDL probes. It should not
be cited as a current clean local workspace gate while `../apple` is in this
state.

Do not regenerate, revert, or otherwise repair `../apple` as part of this demo
without an explicit separate instruction. Use the published-module gate above
for current demo verification, use `BUILD_GOWORK=off` for an explicit
published-module UI harness run, and rerun the checked-in `go.work` gate only
after the sibling checkout is clean or intentionally repaired.

The product proof still remaining is unchanged: run the SwiftUI link monitor on
two live Macs, observe Thunderbolt as the active path, physically remove the
Thunderbolt link, and observe visible fallback to AWDL without restarting either
process.
