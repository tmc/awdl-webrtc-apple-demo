#!/usr/bin/env bash
set -euo pipefail

ssh_target=${SSH_TARGET:-tmc2@10.0.18.249}
ssh_host=${SSH_HOST:-${ssh_target##*@}}
ssh_host=${ssh_host#[}
ssh_host=${ssh_host%]}
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
nw_connect_timeout=${NW_CONNECT_TIMEOUT:-2s}
nw_connect_retries=${NW_CONNECT_RETRIES:-2}
candidate_policy=${CANDIDATE_POLICY:-auto}
webrtc_trace=${WEBRTC_TRACE:-0}
listen_idle_timeout=${LISTEN_IDLE_TIMEOUT:-2s}
require_paths=${REQUIRE_PATHS:-0}
lan_path_interface=${LAN_PATH_INTERFACE:-en0}
awdl_path_interface=${AWDL_PATH_INTERFACE:-awdl0}
thunderbolt_path_interface=${THUNDERBOLT_PATH_INTERFACE:-}
output=${OUTPUT:-}

if [[ -n $output ]]; then
	mkdir -p "$(dirname "$output")"
	exec > >(tee "$output") 2>&1
fi

webrtc_trace_args=()
case "$webrtc_trace" in
1 | true | TRUE | yes | YES | on | ON)
	webrtc_trace_args=(-webrtc-trace)
	;;
esac

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

addr_host() {
	local addr=$1
	if [[ $addr == \[*\]:* ]]; then
		addr=${addr#\[}
		printf '%s\n' "${addr%\]:*}"
		return
	fi
	printf '%s\n' "${addr%:*}"
}

addr_port() {
	local addr=$1
	if [[ $addr == \[*\]:* ]]; then
		printf '%s\n' "${addr##*\]:}"
		return
	fi
	printf '%s\n' "${addr##*:}"
}

diagnose_local_udp_endpoint() {
	local label=$1
	local peer=$2
	local host port
	host=$(addr_host "$peer")
	port=$(addr_port "$peer")
	printf '## %s local UDP diagnostics\n' "$label"
	printf 'peer=%s host=%s port=%s\n' "$peer" "$host" "$port"
	if [[ $host == *:* ]]; then
		run_diag route -n get -inet6 "$host"
	else
		run_diag route -n get "$host"
	fi
	if command -v lsof >/dev/null 2>&1; then
		run_diag lsof -nP -iUDP:"$port"
	fi
	run_diag sh -c "netstat -anv -p udp | grep -E '(^|[.:])${port}([[:space:]]|$)'"
}

diagnose_remote_udp_endpoint() {
	local label=$1
	local peer=$2
	local host port route_cmd
	host=$(addr_host "$peer")
	port=$(addr_port "$peer")
	if [[ $host == *:* ]]; then
		route_cmd="route -n get -inet6 $(sq "$host")"
	else
		route_cmd="route -n get $(sq "$host")"
	fi
	remote "printf '## %s remote UDP diagnostics\n' $(sq "$label"); printf 'peer=%s host=%s port=%s\n' $(sq "$peer") $(sq "$host") $(sq "$port"); date -u; $route_cmd || true; if command -v lsof >/dev/null 2>&1; then lsof -nP -iUDP:$(sq "$port") || true; fi; netstat -anv -p udp | grep -E '(^|[.:])$(sq "$port")([[:space:]]|$)' || true"
}

diagnose_local_udp_path() {
	local label=$1
	local peer=$2
	local host
	host=$(addr_host "$peer")
	printf '## %s local UDP route\n' "$label"
	printf 'peer=%s host=%s\n' "$peer" "$host"
	if [[ $host == *:* ]]; then
		run_diag route -n get -inet6 "$host"
	else
		run_diag route -n get "$host"
	fi
}

diagnose_remote_udp_path() {
	local label=$1
	local peer=$2
	local host route_cmd
	host=$(addr_host "$peer")
	if [[ $host == *:* ]]; then
		route_cmd="route -n get -inet6 $(sq "$host")"
	else
		route_cmd="route -n get $(sq "$host")"
	fi
	remote "printf '## %s remote UDP route\n' $(sq "$label"); printf 'peer=%s host=%s\n' $(sq "$peer") $(sq "$host"); date -u; $route_cmd || true"
}

diagnose_local_reachability() {
	printf '## local reachability diagnostics\n'
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
	run_diag scutil --nwi
	run_diag ifconfig -l
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

remote_network_args() {
	printf ' -nw-connect-timeout %s -nw-connect-retries %s' "$(sq "$nw_connect_timeout")" "$(sq "$nw_connect_retries")"
}

append_local_network_args() {
	local_args+=(-nw-connect-timeout "$nw_connect_timeout" -nw-connect-retries "$nw_connect_retries")
}

path_interface_for_profile() {
	case "$1" in
	lan)
		printf '%s\n' "$lan_path_interface"
		;;
	awdl)
		printf '%s\n' "$awdl_path_interface"
		;;
	thunderbolt)
		printf '%s\n' "$thunderbolt_path_interface"
		;;
	*)
		printf '\n'
		;;
	esac
}

remote_path_args() {
	if [[ $require_paths != 1 ]]; then
		return
	fi
	local iface
	iface=$(path_interface_for_profile "$1")
	if [[ -n $iface ]]; then
		printf ' -require-path-interface %s' "$(sq "$iface")"
	fi
	printf ' -forbid-loopback-path'
}

append_local_path_args() {
	if [[ $require_paths != 1 ]]; then
		return
	fi
	local profile=$1
	local iface
	iface=$(path_interface_for_profile "$profile")
	if [[ -n $iface ]]; then
		local_args+=(-require-path-interface "$iface")
	fi
	local_args+=(-forbid-loopback-path)
}

remote_send() {
	local profile=$1
	local peer=$2
	local duration_arg=
	if [[ -n $duration ]]; then
		duration_arg=" -duration $(sq "$duration")"
	fi
	remote "$(sq "$remote_bin") -profile $(sq "$profile") -backend network$(remote_network_args) -mode udp-perf-send -peer $(sq "$peer") -count $(sq "$count")${duration_arg} -warmup $(sq "$warmup") -size $(sq "$size") -trials $(sq "$trials") -window $(sq "$window") -streams $(sq "$streams") -perf-json$(remote_path_args "$profile") -timeout $(sq "$timeout")"
}

remote_latency_send() {
	local profile=$1
	local peer=$2
	local duration_arg=
	if [[ -n $duration ]]; then
		duration_arg=" -duration $(sq "$duration")"
	fi
	remote "$(sq "$remote_bin") -profile $(sq "$profile") -backend network$(remote_network_args) -mode udp-latency-send -peer $(sq "$peer") -count $(sq "$count")${duration_arg} -warmup $(sq "$warmup") -size $(sq "$size") -trials $(sq "$trials") -streams $(sq "$streams") -perf-json$(remote_path_args "$profile") -timeout $(sq "$timeout")"
}

local_send() {
	local profile=$1
	local peer=$2
	local local_args=()
	if [[ -n $duration ]]; then
		local_args=(-duration "$duration")
	fi
	append_local_network_args
	append_local_path_args "$profile"
	run "$local_bin" -profile "$profile" -backend network -mode udp-perf-send -peer "$peer" -count "$count" "${local_args[@]}" -warmup "$warmup" -size "$size" -trials "$trials" -window "$window" -streams "$streams" -perf-json -timeout "$timeout"
}

local_latency_send() {
	local profile=$1
	local peer=$2
	local local_args=()
	if [[ -n $duration ]]; then
		local_args=(-duration "$duration")
	fi
	append_local_network_args
	append_local_path_args "$profile"
	run "$local_bin" -profile "$profile" -backend network -mode udp-latency-send -peer "$peer" -count "$count" "${local_args[@]}" -warmup "$warmup" -size "$size" -trials "$trials" -streams "$streams" -perf-json -timeout "$timeout"
}

run_local_listener_then_remote_sender() {
	local profile=$1
	local args=()
	if [[ -n $duration ]]; then
		args=(-duration "$duration")
	fi
	args+=(-nw-connect-timeout "$nw_connect_timeout" -nw-connect-retries "$nw_connect_retries")
	args+=(-listen-idle-timeout "$listen_idle_timeout")
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
	diagnose_local_udp_endpoint "$profile local perf listener" "$peer"
	diagnose_remote_udp_path "$profile remote perf sender" "$peer"
	local rc=0
	remote_send "$profile" "$peer" || rc=$?
	wait "$listener_pid" || rc=$?
	cat "$log"
	rm -f "$log"
	return "$rc"
}

run_local_listener_then_remote_latency() {
	local profile=$1
	local args=()
	if [[ -n $duration ]]; then
		args=(-duration "$duration")
	fi
	args+=(-nw-connect-timeout "$nw_connect_timeout" -nw-connect-retries "$nw_connect_retries")
	args+=(-listen-idle-timeout "$listen_idle_timeout")
	local log
	log=$(mktemp)
	printf '## %s remote-to-local UDP latency\n' "$profile"
	"$local_bin" -profile "$profile" -backend network -mode udp-perf-listen -count "$count" "${args[@]}" -warmup "$warmup" -trials "$trials" -streams "$streams" -perf-json -timeout "$timeout" >"$log" 2>&1 &
	local listener_pid=$!
	local peer
	if ! peer=$(wait_for_peer "$log" "local latency $profile"); then
		kill "$listener_pid" 2>/dev/null || true
		wait "$listener_pid" 2>/dev/null || true
		rm -f "$log"
		return 1
	fi
	diagnose_local_udp_endpoint "$profile local latency listener" "$peer"
	diagnose_remote_udp_path "$profile remote latency sender" "$peer"
	local rc=0
	remote_latency_send "$profile" "$peer" || rc=$?
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
	ssh -o "ConnectTimeout=$connect_timeout" -o BatchMode=yes "$ssh_target" "$(sq "$remote_bin") -profile $(sq "$profile") -backend network$(remote_network_args) -mode udp-perf-listen -count $(sq "$count")${duration_arg} -warmup $(sq "$warmup") -trials $(sq "$trials") -streams $(sq "$streams") -listen-idle-timeout $(sq "$listen_idle_timeout") -perf-json -timeout $(sq "$timeout")" >"$log" 2>&1 &
	local listener_pid=$!
	local peer
	if ! peer=$(wait_for_peer "$log" "remote $profile"); then
		kill "$listener_pid" 2>/dev/null || true
		wait "$listener_pid" 2>/dev/null || true
		rm -f "$log"
		return 1
	fi
	diagnose_remote_udp_endpoint "$profile remote perf listener" "$peer"
	diagnose_local_udp_path "$profile local perf sender" "$peer"
	local rc=0
	local_send "$profile" "$peer" || rc=$?
	wait "$listener_pid" || rc=$?
	cat "$log"
	rm -f "$log"
	return "$rc"
}

run_remote_listener_then_local_latency() {
	local profile=$1
	local duration_arg=
	if [[ -n $duration ]]; then
		duration_arg=" -duration $(sq "$duration")"
	fi
	local log
	log=$(mktemp)
	printf '## %s local-to-remote UDP latency\n' "$profile"
	ssh -o "ConnectTimeout=$connect_timeout" -o BatchMode=yes "$ssh_target" "$(sq "$remote_bin") -profile $(sq "$profile") -backend network$(remote_network_args) -mode udp-perf-listen -count $(sq "$count")${duration_arg} -warmup $(sq "$warmup") -trials $(sq "$trials") -streams $(sq "$streams") -listen-idle-timeout $(sq "$listen_idle_timeout") -perf-json -timeout $(sq "$timeout")" >"$log" 2>&1 &
	local listener_pid=$!
	local peer
	if ! peer=$(wait_for_peer "$log" "remote latency $profile"); then
		kill "$listener_pid" 2>/dev/null || true
		wait "$listener_pid" 2>/dev/null || true
		rm -f "$log"
		return 1
	fi
	diagnose_remote_udp_endpoint "$profile remote latency listener" "$peer"
	diagnose_local_udp_path "$profile local latency sender" "$peer"
	local rc=0
	local_latency_send "$profile" "$peer" || rc=$?
	wait "$listener_pid" || rc=$?
	cat "$log"
	rm -f "$log"
	return "$rc"
}

run_bidirectional_perf() {
	local profile=$1
	local duration_arg=
	local local_args=()
	if [[ -n $duration ]]; then
		duration_arg=" -duration $(sq "$duration")"
		local_args=(-duration "$duration")
	fi
	local_args+=(-nw-connect-timeout "$nw_connect_timeout" -nw-connect-retries "$nw_connect_retries")
	local_args+=(-listen-idle-timeout "$listen_idle_timeout")
	local local_log remote_log local_send_log remote_send_log
	local_log=$(mktemp)
	remote_log=$(mktemp)
	local_send_log=$(mktemp)
	remote_send_log=$(mktemp)
	printf '## %s bidirectional UDP perf\n' "$profile"
	"$local_bin" -profile "$profile" -backend network -mode udp-perf-listen -count "$count" "${local_args[@]}" -warmup "$warmup" -trials "$trials" -streams "$streams" -perf-json -timeout "$timeout" >"$local_log" 2>&1 &
	local local_listener_pid=$!
	ssh -o "ConnectTimeout=$connect_timeout" -o BatchMode=yes "$ssh_target" "$(sq "$remote_bin") -profile $(sq "$profile") -backend network$(remote_network_args) -mode udp-perf-listen -count $(sq "$count")${duration_arg} -warmup $(sq "$warmup") -trials $(sq "$trials") -streams $(sq "$streams") -listen-idle-timeout $(sq "$listen_idle_timeout") -perf-json -timeout $(sq "$timeout")" >"$remote_log" 2>&1 &
	local remote_listener_pid=$!
	local local_peer remote_peer
	if ! local_peer=$(wait_for_peer "$local_log" "local bidirectional $profile"); then
		kill "$local_listener_pid" "$remote_listener_pid" 2>/dev/null || true
		wait "$local_listener_pid" "$remote_listener_pid" 2>/dev/null || true
		cat "$local_log" "$remote_log" "$local_send_log" "$remote_send_log"
		rm -f "$local_log" "$remote_log" "$local_send_log" "$remote_send_log"
		return 1
	fi
	if ! remote_peer=$(wait_for_peer "$remote_log" "remote bidirectional $profile"); then
		kill "$local_listener_pid" "$remote_listener_pid" 2>/dev/null || true
		wait "$local_listener_pid" "$remote_listener_pid" 2>/dev/null || true
		cat "$local_log" "$remote_log" "$local_send_log" "$remote_send_log"
		rm -f "$local_log" "$remote_log" "$local_send_log" "$remote_send_log"
		return 1
	fi
	diagnose_local_udp_endpoint "$profile bidirectional local listener" "$local_peer"
	diagnose_remote_udp_endpoint "$profile bidirectional remote listener" "$remote_peer"
	diagnose_local_udp_path "$profile bidirectional local sender" "$remote_peer"
	diagnose_remote_udp_path "$profile bidirectional remote sender" "$local_peer"
	local_send "$profile" "$remote_peer" >"$local_send_log" 2>&1 &
	local local_sender_pid=$!
	remote_send "$profile" "$local_peer" >"$remote_send_log" 2>&1 &
	local remote_sender_pid=$!
	local rc=0
	wait "$local_sender_pid" || rc=$?
	wait "$remote_sender_pid" || rc=$?
	wait "$local_listener_pid" || rc=$?
	wait "$remote_listener_pid" || rc=$?
	cat "$local_send_log" "$remote_send_log" "$local_log" "$remote_log"
	rm -f "$local_log" "$remote_log" "$local_send_log" "$remote_send_log"
	return "$rc"
}

run_local_callback_then_remote_request() {
	local profile=$1
	local log
	log=$(mktemp)
	printf '## %s callback remote-to-local request\n' "$profile"
	"$local_bin" -profile "$profile" -backend network -nw-connect-timeout "$nw_connect_timeout" -nw-connect-retries "$nw_connect_retries" -mode udp-callback-listen -timeout "$timeout" >"$log" 2>&1 &
	local listener_pid=$!
	local peer
	if ! peer=$(wait_for_callback_peer "$log" "local $profile"); then
		kill "$listener_pid" 2>/dev/null || true
		wait "$listener_pid" 2>/dev/null || true
		rm -f "$log"
		return 1
	fi
	diagnose_local_udp_endpoint "$profile local callback listener" "$peer"
	diagnose_remote_udp_path "$profile remote callback request" "$peer"
	local rc=0
	remote "$(sq "$remote_bin") -profile $(sq "$profile") -backend network$(remote_network_args) -mode udp-callback-request -peer $(sq "$peer") -message $(sq "$message") -timeout $(sq "$timeout")" || rc=$?
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
	ssh -o "ConnectTimeout=$connect_timeout" -o BatchMode=yes "$ssh_target" "$(sq "$remote_bin") -profile $(sq "$profile") -backend network$(remote_network_args) -mode udp-callback-listen -timeout $(sq "$timeout")" >"$log" 2>&1 &
	local listener_pid=$!
	local peer
	if ! peer=$(wait_for_callback_peer "$log" "remote $profile"); then
		kill "$listener_pid" 2>/dev/null || true
		wait "$listener_pid" 2>/dev/null || true
		rm -f "$log"
		return 1
	fi
	diagnose_remote_udp_endpoint "$profile remote callback listener" "$peer"
	diagnose_local_udp_path "$profile local callback request" "$peer"
	local rc=0
	run "$local_bin" -profile "$profile" -backend network -nw-connect-timeout "$nw_connect_timeout" -nw-connect-retries "$nw_connect_retries" -mode udp-callback-request -peer "$peer" -message "$message" -timeout "$timeout" || rc=$?
	wait "$listener_pid" || rc=$?
	cat "$log"
	rm -f "$log"
	return "$rc"
}

matrix_failures=0
matrix_failed=()

record_matrix_step() {
	local label=$1
	shift
	if "$@"; then
		return
	else
		local rc=$?
		printf 'FAIL: %s exit=%d\n' "$label" "$rc" >&2
		matrix_failed+=("$label exit=$rc")
		matrix_failures=$((matrix_failures + 1))
	fi
}

printf '## matrix config\n'
printf 'ssh_target=%s\n' "$ssh_target"
printf 'ssh_host=%s\n' "$ssh_host"
printf 'profiles=%s\n' "$profiles"
printf 'count=%s\n' "$count"
printf 'duration=%s\n' "$duration"
printf 'warmup=%s\n' "$warmup"
printf 'size=%s\n' "$size"
printf 'trials=%s\n' "$trials"
printf 'window=%s\n' "$window"
printf 'streams=%s\n' "$streams"
printf 'timeout=%s\n' "$timeout"
printf 'connect_timeout=%s\n' "$connect_timeout"
printf 'nw_connect_timeout=%s\n' "$nw_connect_timeout"
printf 'nw_connect_retries=%s\n' "$nw_connect_retries"
printf 'candidate_policy=%s\n' "$candidate_policy"
printf 'webrtc_trace=%s\n' "$webrtc_trace"
printf 'listen_idle_timeout=%s\n' "$listen_idle_timeout"
printf 'require_paths=%s\n' "$require_paths"
printf 'lan_path_interface=%s\n' "$lan_path_interface"
printf 'awdl_path_interface=%s\n' "$awdl_path_interface"
printf 'thunderbolt_path_interface=%s\n' "$thunderbolt_path_interface"
printf 'local_bin=%s\n' "$local_bin"
printf 'remote_bin=%s\n' "$remote_bin"
printf 'output=%s\n' "$output"

diagnose_local_reachability

printf '## remote reachability\n'
if remote "true"; then
	:
else
	rc=$?
	printf 'FAIL: remote reachability exit=%d\n' "$rc" >&2
	exit "$rc"
fi

printf '## build local binary\n'
run go build -o "$local_bin" .

printf '## install remote binary\n'
run scp -o "ConnectTimeout=$connect_timeout" -o BatchMode=yes "$local_bin" "$ssh_target:$remote_bin"
remote "chmod +x $(sq "$remote_bin") && $(sq "$remote_bin") -mode check -profile lan -backend network$(remote_network_args) -timeout 3s >/dev/null"

for profile in $profiles; do
	printf '## %s Pion transport.Net WebRTC\n' "$profile"
	record_matrix_step "$profile Pion transport.Net WebRTC" run "$local_bin" -profile "$profile" -backend network -nw-connect-timeout "$nw_connect_timeout" -nw-connect-retries "$nw_connect_retries" -pion-net -mdns disabled -candidate-policy "$candidate_policy" -mode offer-ssh -ssh "$ssh_target" -remote-bin "$remote_bin" -timeout "$timeout" "${webrtc_trace_args[@]}"

	record_matrix_step "$profile remote-to-local UDP perf" run_local_listener_then_remote_sender "$profile"
	record_matrix_step "$profile local-to-remote UDP perf" run_remote_listener_then_local_sender "$profile"
	record_matrix_step "$profile bidirectional UDP perf" run_bidirectional_perf "$profile"
	record_matrix_step "$profile remote-to-local UDP latency" run_local_listener_then_remote_latency "$profile"
	record_matrix_step "$profile local-to-remote UDP latency" run_remote_listener_then_local_latency "$profile"
	record_matrix_step "$profile callback remote-to-local request" run_local_callback_then_remote_request "$profile"
	record_matrix_step "$profile callback local-to-remote request" run_remote_callback_then_local_request "$profile"
done

printf '## matrix summary\n'
if ((matrix_failures != 0)); then
	printf 'matrix failed steps=%d\n' "$matrix_failures" >&2
	for failure in "${matrix_failed[@]}"; do
		printf 'FAIL: %s\n' "$failure" >&2
	done
	exit 1
fi
printf 'matrix passed profiles=%s\n' "$profiles"
