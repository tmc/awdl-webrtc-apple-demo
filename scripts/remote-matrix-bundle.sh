#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_dir=$(cd -- "$script_dir/.." && pwd)
output=${OUTPUT:-/tmp/awdl-webrtc-matrix.txt}
summary_output=${SUMMARY_OUTPUT:-}

if [[ -z $summary_output ]]; then
	base=${output%.*}
	if [[ $base == "$output" ]]; then
		summary_output="${output}-summary.md"
	else
		summary_output="${base}-summary.md"
	fi
fi

mkdir -p "$(dirname "$output")" "$(dirname "$summary_output")"

rc=0
if OUTPUT="$output" "$script_dir/remote-matrix.sh"; then
	rc=0
else
	rc=$?
fi

(
	cd "$repo_dir"
	go run ./cmd/matrix-summary "$output" >"$summary_output"
)

printf 'matrix_output=%s\n' "$output"
printf 'summary_output=%s\n' "$summary_output"
printf 'matrix_exit=%d\n' "$rc"
exit "$rc"
