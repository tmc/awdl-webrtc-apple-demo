#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_dir=$(cd -- "$script_dir/.." && pwd)

ssh_target=${SSH_TARGET:-tmc2@10.0.18.249}
ssh_host=${SSH_HOST:-${ssh_target##*@}}
ssh_host=${ssh_host#[}
ssh_host=${ssh_host%]}
connect_timeout=${CONNECT_TIMEOUT:-5}
remote_ready_timeout=${REMOTE_READY_TIMEOUT:-30}
remote_ready_interval=${REMOTE_READY_INTERVAL:-5}
remote_publish_timeout=${REMOTE_PUBLISH_TIMEOUT:-2h}
remote_publish_interval=${REMOTE_PUBLISH_INTERVAL:-2s}
remote_ready_log_timeout=${REMOTE_READY_LOG_TIMEOUT:-20}
local_bin=${LOCAL_BIN:-/tmp/awdl-webrtc-ui-harness-bin}
remote_bin=${REMOTE_BIN:-/Volumes/Shared/awdl-webrtc-ui-harness-bin}
backend=${BACKEND:-network}
ui_interval=${UI_INTERVAL:-2s}
ui_count=${UI_COUNT:-20}
ui_window=${UI_WINDOW:-4}
discover_timeout=${DISCOVER_TIMEOUT:-30s}
discover_interval=${DISCOVER_INTERVAL:-1s}
discover_peer=${DISCOVER_PEER:-}
preflight_discovery=${PREFLIGHT_DISCOVERY:-1}
cleanup_stale_processes=${CLEAN_STALE_PROCESSES:-1}
remote_log=${REMOTE_LOG:-}
output=${OUTPUT:-}

if [[ -n $output ]]; then
	mkdir -p "$(dirname "$output")"
	exec > >(tee "$output") 2>&1
fi

sq() {
	local s=${1//\'/\'\\\'\'}
	printf "'%s'" "$s"
}

run() {
	printf '+'
	printf ' %q' "$@"
	printf '\n'
	"$@"
}

run_diag() {
	printf '+'
	printf ' %q' "$@"
	printf '\n'
	"$@" 2>&1 || printf 'command failed exit=%d\n' "$?"
}

truthy() {
	case "${1:-}" in
	1 | true | TRUE | yes | YES | on | ON)
		return 0
		;;
	esac
	return 1
}

validate_uint() {
	local name=$1
	local value=$2
	case "$value" in
	'' | *[!0-9]*)
		printf '%s must be an integer number of seconds, got %q\n' "$name" "$value" >&2
		exit 2
		;;
	esac
}

validate_uint CONNECT_TIMEOUT "$connect_timeout"
validate_uint REMOTE_READY_TIMEOUT "$remote_ready_timeout"
validate_uint REMOTE_READY_INTERVAL "$remote_ready_interval"
validate_uint REMOTE_READY_LOG_TIMEOUT "$remote_ready_log_timeout"

bin_process_pattern() {
	printf '^%s([[:space:]]|$)' "$1"
}

cleanup_local_bin_processes() {
	truthy "$cleanup_stale_processes" || return
	if command -v pkill >/dev/null 2>&1; then
		pkill -f "$(bin_process_pattern "$local_bin")" 2>/dev/null || true
	fi
}

cleanup_remote_bin_processes() {
	truthy "$cleanup_stale_processes" || return
	ssh -o "ConnectTimeout=$connect_timeout" -o BatchMode=yes "$ssh_target" \
		"if command -v pkill >/dev/null 2>&1; then pkill -f $(sq "$(bin_process_pattern "$remote_bin")") 2>/dev/null || true; fi" \
		>/dev/null 2>&1 || true
}

remote_ssh_pid=
remote_log_tmp=

cleanup() {
	local rc=$?
	trap - EXIT INT TERM
	if [[ -n $remote_ssh_pid ]]; then
		kill "$remote_ssh_pid" 2>/dev/null || true
		wait "$remote_ssh_pid" 2>/dev/null || true
	fi
	cleanup_local_bin_processes
	cleanup_remote_bin_processes
	if [[ -n $remote_log_tmp && -f $remote_log_tmp ]]; then
		printf '## remote discovery log\n'
		cat "$remote_log_tmp"
		if [[ -z $remote_log ]]; then
			rm -f "$remote_log_tmp"
		fi
	fi
	exit "$rc"
}

ssh_run() {
	local cmd=$1
	run ssh -o "ConnectTimeout=$connect_timeout" -o BatchMode=yes "$ssh_target" "$cmd"
}

wait_for_remote_ready() {
	local deadline=$((SECONDS + remote_ready_timeout))
	local attempt=0
	local rc=1
	while :; do
		attempt=$((attempt + 1))
		printf 'remote readiness attempt=%d remaining=%ds\n' "$attempt" "$((deadline - SECONDS))"
		if ssh_run "true"; then
			return 0
		else
			rc=$?
		fi
		if ((SECONDS >= deadline)); then
			return "$rc"
		fi
		local sleep_for=$remote_ready_interval
		local remaining=$((deadline - SECONDS))
		if ((sleep_for > remaining)); then
			sleep_for=$remaining
		fi
		if ((sleep_for <= 0)); then
			return "$rc"
		fi
		sleep "$sleep_for"
	done
}

diagnose_local_reachability() {
	printf '## local reachability diagnostics\n'
	printf 'ssh_target=%s\n' "$ssh_target"
	printf 'ssh_host=%s\n' "$ssh_host"
	run_diag date -u
	if [[ $ssh_host == *:* ]]; then
		run_diag route -n get -inet6 "$ssh_host"
		run_diag ping6 -c 1 "$ssh_host"
	else
		run_diag route -n get "$ssh_host"
		run_diag ping -c 1 -W 1000 "$ssh_host"
	fi
	if command -v nc >/dev/null 2>&1; then
		run_diag nc -vz -G "$connect_timeout" "$ssh_host" 22
	fi
}

wait_for_remote_discovery_log() {
	local deadline=$((SECONDS + remote_ready_log_timeout))
	while :; do
		if grep -q '"kind":"link_health_discovery"' "$remote_log_tmp"; then
			return 0
		fi
		if grep -q 'link-webrtc-demo:' "$remote_log_tmp"; then
			printf 'remote discovery failed before first record\n' >&2
			cat "$remote_log_tmp" >&2
			return 1
		fi
		if ((SECONDS >= deadline)); then
			printf 'timed out waiting for remote discovery record\n' >&2
			cat "$remote_log_tmp" >&2
			return 1
		fi
		sleep 0.1
	done
}

printf '## ui harness config\n'
printf 'ssh_target=%s\n' "$ssh_target"
printf 'backend=%s\n' "$backend"
printf 'local_bin=%s\n' "$local_bin"
printf 'remote_bin=%s\n' "$remote_bin"
printf 'remote_publish_timeout=%s\n' "$remote_publish_timeout"
printf 'remote_publish_interval=%s\n' "$remote_publish_interval"
printf 'ui_interval=%s\n' "$ui_interval"
printf 'ui_count=%s\n' "$ui_count"
printf 'ui_window=%s\n' "$ui_window"
printf 'preflight_discovery=%s\n' "$preflight_discovery"
printf 'output=%s\n' "$output"

diagnose_local_reachability

printf '## remote reachability\n'
wait_for_remote_ready

trap cleanup EXIT INT TERM
cleanup_local_bin_processes
cleanup_remote_bin_processes

printf '## build local binary\n'
(
	cd "$repo_dir"
	run go build -o "$local_bin" .
)

printf '## install remote binary\n'
run scp -o "ConnectTimeout=$connect_timeout" -o BatchMode=yes "$local_bin" "$ssh_target:$remote_bin"
ssh_run "chmod +x $(sq "$remote_bin") && $(sq "$remote_bin") -mode check -profile lan -backend $(sq "$backend") -timeout 3s >/dev/null"

if [[ -n $remote_log ]]; then
	mkdir -p "$(dirname "$remote_log")"
	remote_log_tmp=$remote_log
	: >"$remote_log_tmp"
else
	remote_log_tmp=$(mktemp)
fi

printf '## start remote discovery publisher\n'
ssh -o "ConnectTimeout=$connect_timeout" -o BatchMode=yes "$ssh_target" \
	"$(sq "$remote_bin") -mode discover -backend $(sq "$backend") -timeout $(sq "$remote_publish_timeout") -ui-interval $(sq "$remote_publish_interval")" \
	>"$remote_log_tmp" 2>&1 &
remote_ssh_pid=$!
printf 'remote_ssh_pid=%s\n' "$remote_ssh_pid"
wait_for_remote_discovery_log

if truthy "$preflight_discovery"; then
	printf '## local discovery preflight\n'
	local discover_args=(
		"$local_bin"
		-mode discover-wait
		-backend "$backend"
		-timeout "$discover_timeout"
		-ui-interval "$discover_interval"
	)
	if [[ -n $discover_peer ]]; then
		discover_args+=(-discover-peer "$discover_peer")
	fi
	run "${discover_args[@]}"
fi

cat <<EOF
## physical UI validation
1. Wait until the local SwiftUI window shows Thunderbolt as the active path.
2. Physically remove the Thunderbolt cable or adapter.
3. Pass condition: the visible active path changes to AWDL, bandwidth rows
   continue updating, and neither process is restarted.
4. Close the SwiftUI window or press Ctrl-C here when the observation is done.
EOF

printf '## start local SwiftUI monitor\n'
run "$local_bin" -mode ui -backend "$backend" -ui-interval "$ui_interval" -ui-count "$ui_count" -ui-window "$ui_window"
