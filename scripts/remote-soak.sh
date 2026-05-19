#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_dir=$(cd -- "$script_dir/.." && pwd)

stamp=${SOAK_STAMP:-$(date -u +%Y%m%dT%H%M%SZ)}
label=${SOAK_LABEL:-workspace-soak}
safe_label=${label//[^A-Za-z0-9_.-]/-}
output=${OUTPUT:-/tmp/awdl-webrtc-${safe_label}-${stamp}.txt}
summary_output=${SUMMARY_OUTPUT:-/tmp/awdl-webrtc-${safe_label}-${stamp}.md}

export OUTPUT="$output"
export SUMMARY_OUTPUT="$summary_output"
export REMOTE_BIN="${REMOTE_BIN:-/Volumes/Shared/awdl-webrtc-apple-demo-bin-${safe_label}}"
export REMOTE_READY_TIMEOUT="${REMOTE_READY_TIMEOUT:-60}"
export REMOTE_READY_INTERVAL="${REMOTE_READY_INTERVAL:-5}"
export REMOTE_STEP_READY_TIMEOUT="${REMOTE_STEP_READY_TIMEOUT:-30}"
export CANDIDATE_POLICY="${CANDIDATE_POLICY:-auto}"
export REQUIRE_PATHS="${REQUIRE_PATHS:-1}"
export WEBRTC_ATTEMPTS="${WEBRTC_ATTEMPTS:-3}"
export WEBRTC_RETRY_DELAY="${WEBRTC_RETRY_DELAY:-5}"
export DURATION="${DURATION:-30s}"
export TRIALS="${TRIALS:-5}"
export WINDOW="${WINDOW:-8}"
export STREAMS="${STREAMS:-2}"
export TIMEOUT="${TIMEOUT:-180s}"
export LISTEN_IDLE_TIMEOUT="${LISTEN_IDLE_TIMEOUT:-5s}"

printf '## soak config\n'
printf 'label=%s\n' "$label"
printf 'stamp=%s\n' "$stamp"
printf 'output=%s\n' "$OUTPUT"
printf 'summary_output=%s\n' "$SUMMARY_OUTPUT"
printf 'remote_bin=%s\n' "$REMOTE_BIN"
printf 'duration=%s\n' "$DURATION"
printf 'trials=%s\n' "$TRIALS"
printf 'window=%s\n' "$WINDOW"
printf 'streams=%s\n' "$STREAMS"
printf 'timeout=%s\n' "$TIMEOUT"
printf 'require_paths=%s\n' "$REQUIRE_PATHS"

(
	cd "$repo_dir"
	"$script_dir/remote-matrix-bundle.sh"
)
