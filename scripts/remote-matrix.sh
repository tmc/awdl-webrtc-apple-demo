#!/usr/bin/env bash
set -euo pipefail

ssh_target=${SSH_TARGET:-tmc2@10.0.18.249}
local_bin=${LOCAL_BIN:-/tmp/awdl-webrtc-apple-demo-bin}
remote_bin=${REMOTE_BIN:-/tmp/awdl-webrtc-apple-demo-bin}
profiles=${PROFILES:-lan thunderbolt awdl}
count=${COUNT:-20}
duration=${DURATION:-}
warmup=${WARMUP:-0}
size=${SIZE:-1200}
trials=${TRIALS:-3}
window=${WINDOW:-4}
streams=${STREAMS:-1}
timeout=${TIMEOUT:-45s}
message=${MESSAGE:-callback}
connect_timeout=${CONNECT_TIMEOUT:-5}

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

wait_for_peer() {
	local log=$1
	local label=$2
	local peer
	for _ in $(seq 1 100); do
		peer=$(sed -n 's/^udp perf listen=\([^ ]*\).*/\1/p' "$log" | tail -1)
		if [[ -n $peer ]]; then
			printf '%s\n' "$peer"
			return 0
		fi
		if grep -q 'link-webrtc-demo:' "$log"; then
			cat "$log"
			return 1
		fi
		sleep 0.1
	done
	printf 'timed out waiting for %s listener address\n' "$label" >&2
	cat "$log" >&2
	return 1
}

wait_for_callback_peer() {
	local log=$1
	local label=$2
	local peer
	for _ in $(seq 1 100); do
		peer=$(sed -n 's/^udp callback listen=\([^ ]*\).*/\1/p' "$log" | tail -1)
		if [[ -n $peer ]]; then
			printf '%s\n' "$peer"
			return 0
		fi
		if grep -q 'link-webrtc-demo:' "$log"; then
			cat "$log"
			return 1
		fi
		sleep 0.1
	done
	printf 'timed out waiting for %s callback listener address\n' "$label" >&2
	cat "$log" >&2
	return 1
}

remote() {
	local cmd=$1
	run ssh -o "ConnectTimeout=$connect_timeout" -o BatchMode=yes "$ssh_target" "$cmd"
}

remote_send() {
	local profile=$1
	local peer=$2
	local duration_arg=
	if [[ -n $duration ]]; then
		duration_arg=" -duration $(sq "$duration")"
	fi
	remote "$(sq "$remote_bin") -profile $(sq "$profile") -backend network -mode udp-perf-send -peer $(sq "$peer") -count $(sq "$count")${duration_arg} -warmup $(sq "$warmup") -size $(sq "$size") -trials $(sq "$trials") -window $(sq "$window") -streams $(sq "$streams") -perf-json -timeout $(sq "$timeout")"
}

local_send() {
	local profile=$1
	local peer=$2
	local args=()
	if [[ -n $duration ]]; then
		args=(-duration "$duration")
	fi
	run "$local_bin" -profile "$profile" -backend network -mode udp-perf-send -peer "$peer" -count "$count" "${args[@]}" -warmup "$warmup" -size "$size" -trials "$trials" -window "$window" -streams "$streams" -perf-json -timeout "$timeout"
}

run_local_listener_then_remote_sender() {
	local profile=$1
	local args=()
	if [[ -n $duration ]]; then
		args=(-duration "$duration")
	fi
	local log
	log=$(mktemp)
	printf '## %s remote-to-local UDP perf\n' "$profile"
	"$local_bin" -profile "$profile" -backend network -mode udp-perf-listen -count "$count" "${args[@]}" -warmup "$warmup" -trials "$trials" -streams "$streams" -perf-json -timeout "$timeout" >"$log" 2>&1 &
	local listener_pid=$!
	local peer
	if ! peer=$(wait_for_peer "$log" "local $profile"); then
		kill "$listener_pid" 2>/dev/null || true
		wait "$listener_pid" 2>/dev/null || true
		rm -f "$log"
		return 1
	fi
	local rc=0
	remote_send "$profile" "$peer" || rc=$?
	wait "$listener_pid" || rc=$?
	cat "$log"
	rm -f "$log"
	return "$rc"
}

run_remote_listener_then_local_sender() {
	local profile=$1
	local duration_arg=
	if [[ -n $duration ]]; then
		duration_arg=" -duration $(sq "$duration")"
	fi
	local log
	log=$(mktemp)
	printf '## %s local-to-remote UDP perf\n' "$profile"
	ssh -o "ConnectTimeout=$connect_timeout" -o BatchMode=yes "$ssh_target" "$(sq "$remote_bin") -profile $(sq "$profile") -backend network -mode udp-perf-listen -count $(sq "$count")${duration_arg} -warmup $(sq "$warmup") -trials $(sq "$trials") -streams $(sq "$streams") -perf-json -timeout $(sq "$timeout")" >"$log" 2>&1 &
	local listener_pid=$!
	local peer
	if ! peer=$(wait_for_peer "$log" "remote $profile"); then
		kill "$listener_pid" 2>/dev/null || true
		wait "$listener_pid" 2>/dev/null || true
		rm -f "$log"
		return 1
	fi
	local rc=0
	local_send "$profile" "$peer" || rc=$?
	wait "$listener_pid" || rc=$?
	cat "$log"
	rm -f "$log"
	return "$rc"
}

run_local_callback_then_remote_request() {
	local profile=$1
	local log
	log=$(mktemp)
	printf '## %s callback remote-to-local request\n' "$profile"
	"$local_bin" -profile "$profile" -backend network -mode udp-callback-listen -timeout "$timeout" >"$log" 2>&1 &
	local listener_pid=$!
	local peer
	if ! peer=$(wait_for_callback_peer "$log" "local $profile"); then
		kill "$listener_pid" 2>/dev/null || true
		wait "$listener_pid" 2>/dev/null || true
		rm -f "$log"
		return 1
	fi
	local rc=0
	remote "$(sq "$remote_bin") -profile $(sq "$profile") -backend network -mode udp-callback-request -peer $(sq "$peer") -message $(sq "$message") -timeout $(sq "$timeout")" || rc=$?
	wait "$listener_pid" || rc=$?
	cat "$log"
	rm -f "$log"
	return "$rc"
}

run_remote_callback_then_local_request() {
	local profile=$1
	local log
	log=$(mktemp)
	printf '## %s callback local-to-remote request\n' "$profile"
	ssh -o "ConnectTimeout=$connect_timeout" -o BatchMode=yes "$ssh_target" "$(sq "$remote_bin") -profile $(sq "$profile") -backend network -mode udp-callback-listen -timeout $(sq "$timeout")" >"$log" 2>&1 &
	local listener_pid=$!
	local peer
	if ! peer=$(wait_for_callback_peer "$log" "remote $profile"); then
		kill "$listener_pid" 2>/dev/null || true
		wait "$listener_pid" 2>/dev/null || true
		rm -f "$log"
		return 1
	fi
	local rc=0
	run "$local_bin" -profile "$profile" -backend network -mode udp-callback-request -peer "$peer" -message "$message" -timeout "$timeout" || rc=$?
	wait "$listener_pid" || rc=$?
	cat "$log"
	rm -f "$log"
	return "$rc"
}

printf '## build local binary\n'
run go build -o "$local_bin" .

printf '## install remote binary\n'
run scp -o "ConnectTimeout=$connect_timeout" -o BatchMode=yes "$local_bin" "$ssh_target:$remote_bin"
remote "chmod +x $(sq "$remote_bin") && $(sq "$remote_bin") -mode check -profile lan -backend network -timeout 3s >/dev/null"

for profile in $profiles; do
	printf '## %s Pion transport.Net WebRTC\n' "$profile"
	run "$local_bin" -profile "$profile" -backend network -pion-net -mdns disabled -raw-candidates -mode offer-ssh -ssh "$ssh_target" -remote-bin "$remote_bin" -timeout "$timeout"

	run_local_listener_then_remote_sender "$profile"
	run_remote_listener_then_local_sender "$profile"
	run_local_callback_then_remote_request "$profile"
	run_remote_callback_then_local_request "$profile"
done
