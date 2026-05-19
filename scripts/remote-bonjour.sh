#!/usr/bin/env bash
set -euo pipefail

ssh_target=${SSH_TARGET:-tmc2@10.0.18.249}
ssh_host=${SSH_HOST:-${ssh_target##*@}}
ssh_host=${ssh_host#[}
ssh_host=${ssh_host%]}
connect_timeout=${CONNECT_TIMEOUT:-5}
remote_ready_timeout=${REMOTE_READY_TIMEOUT:-0}
remote_ready_interval=${REMOTE_READY_INTERVAL:-5}
service_ready_timeout=${SERVICE_READY_TIMEOUT:-20}
profiles=${PROFILES:-lan thunderbolt awdl}
phases=${PHASES:-signal full}
local_bin=${LOCAL_BIN:-/tmp/awdl-webrtc-apple-demo-bonjour-bin}
remote_bin=${REMOTE_BIN:-/tmp/awdl-webrtc-apple-demo-bonjour-bin}
timeout=${TIMEOUT:-90s}
candidate_policy=${CANDIDATE_POLICY:-auto}
signal_prefix=${SIGNAL_PREFIX:-awdl-webrtc}
webrtc_trace=${WEBRTC_TRACE:-1}
cleanup_stale_processes=${CLEAN_STALE_PROCESSES:-1}
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

webrtc_trace_enabled() {
	case "$webrtc_trace" in
	1 | true | TRUE | yes | YES | on | ON)
		return 0
		;;
	esac
	return 1
}

case "$remote_ready_timeout" in
'' | *[!0-9]*)
	printf 'REMOTE_READY_TIMEOUT must be an integer number of seconds, got %q\n' "$remote_ready_timeout" >&2
	exit 2
	;;
esac
case "$remote_ready_interval" in
'' | *[!0-9]*)
	printf 'REMOTE_READY_INTERVAL must be an integer number of seconds, got %q\n' "$remote_ready_interval" >&2
	exit 2
	;;
esac
case "$service_ready_timeout" in
'' | *[!0-9]*)
	printf 'SERVICE_READY_TIMEOUT must be an integer number of seconds, got %q\n' "$service_ready_timeout" >&2
	exit 2
	;;
esac

bin_process_pattern() {
	printf '^%s([[:space:]]|$)' "$1"
}

cleanup_local_bin_processes() {
	case "$cleanup_stale_processes" in
	0 | false | FALSE | no | NO | off | OFF)
		return
		;;
	esac
	if command -v pkill >/dev/null 2>&1; then
		pkill -f "$(bin_process_pattern "$local_bin")" 2>/dev/null || true
	fi
}

cleanup_remote_bin_processes() {
	case "$cleanup_stale_processes" in
	0 | false | FALSE | no | NO | off | OFF)
		return
		;;
	esac
	ssh -o "ConnectTimeout=$connect_timeout" -o BatchMode=yes "$ssh_target" \
		"if command -v pkill >/dev/null 2>&1; then pkill -f $(sq "$(bin_process_pattern "$remote_bin")") 2>/dev/null || true; fi" \
		>/dev/null 2>&1 || true
}

cleanup_processes() {
	cleanup_local_bin_processes
	cleanup_remote_bin_processes
}

cleanup_on_exit() {
	local rc=$?
	cleanup_processes
	exit "$rc"
}

ssh_run() {
	local cmd=$1
	run ssh -o "ConnectTimeout=$connect_timeout" -o BatchMode=yes "$ssh_target" "$cmd"
}

wait_for_remote_ready() {
	if ((remote_ready_timeout == 0)); then
		ssh_run "true"
		return
	fi
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
	run_diag scutil --nwi
}

wait_for_service() {
	local log=$1
	local name=$2
	local label=$3
	local deadline=$((SECONDS + service_ready_timeout))
	while :; do
		if grep -q "^signal service=${name} " "$log"; then
			return 0
		fi
		if grep -q 'link-webrtc-demo:' "$log"; then
			printf 'remote %s failed before service was ready\n' "$label" >&2
			cat "$log" >&2
			return 1
		fi
		if ((SECONDS >= deadline)); then
			printf 'timed out waiting for remote %s service %s\n' "$label" "$name" >&2
			cat "$log" >&2
			return 1
		fi
		sleep 0.1
	done
}

run_bonjour_pair() {
	local profile=$1
	local phase=$2
	local name="${signal_prefix}-${profile}-${phase}-$$"
	local label="$profile bonjour $phase"

	local remote_log
	remote_log=$(mktemp)
	local remote_trace_arg=
	if webrtc_trace_enabled; then
		remote_trace_arg=" -webrtc-trace"
	fi
	local remote_signal_arg=
	if [[ $phase == signal ]]; then
		remote_signal_arg=" -signal-only"
	fi

	printf '## %s\n' "$label"
	printf 'signal_name=%s\n' "$name"
	ssh -o "ConnectTimeout=$connect_timeout" -o BatchMode=yes "$ssh_target" \
		"$(sq "$remote_bin") -profile $(sq "$profile") -backend network -pion-net -mdns disabled -candidate-policy $(sq "$candidate_policy") -mode answer-bonjour -signal-name $(sq "$name") -timeout $(sq "$timeout")${remote_trace_arg}${remote_signal_arg}" \
		>"$remote_log" 2>&1 &
	local remote_pid=$!

	local rc=0
	if wait_for_service "$remote_log" "$name" "$label"; then
		local local_args=(
			"$local_bin"
			-profile "$profile"
			-backend network
			-pion-net
			-mdns disabled
			-candidate-policy "$candidate_policy"
			-mode offer-bonjour
			-signal-peer "$name"
			-timeout "$timeout"
		)
		if webrtc_trace_enabled; then
			local_args+=(-webrtc-trace)
		fi
		if [[ $phase == signal ]]; then
			local_args+=(-signal-only)
		fi
		run "${local_args[@]}" || rc=$?
	else
		rc=1
	fi

	if ((rc != 0)); then
		kill "$remote_pid" 2>/dev/null || true
	fi
	if ! wait "$remote_pid"; then
		rc=1
	fi
	printf '## %s remote answer log\n' "$label"
	cat "$remote_log"
	rm -f "$remote_log"
	return "$rc"
}

bonjour_failures=0
bonjour_failed=()

record_step() {
	local label=$1
	shift
	if "$@"; then
		return
	fi
	local rc=$?
	printf 'FAIL: %s exit=%d\n' "$label" "$rc" >&2
	bonjour_failed+=("$label exit=$rc")
	bonjour_failures=$((bonjour_failures + 1))
}

printf '## bonjour config\n'
printf 'ssh_target=%s\n' "$ssh_target"
printf 'ssh_host=%s\n' "$ssh_host"
printf 'profiles=%s\n' "$profiles"
printf 'phases=%s\n' "$phases"
printf 'timeout=%s\n' "$timeout"
printf 'candidate_policy=%s\n' "$candidate_policy"
printf 'webrtc_trace=%s\n' "$webrtc_trace"
printf 'signal_prefix=%s\n' "$signal_prefix"
printf 'local_bin=%s\n' "$local_bin"
printf 'remote_bin=%s\n' "$remote_bin"
printf 'output=%s\n' "$output"

diagnose_local_reachability

printf '## remote reachability\n'
if wait_for_remote_ready; then
	:
else
	rc=$?
	printf 'FAIL: remote reachability exit=%d\n' "$rc" >&2
	exit "$rc"
fi

trap cleanup_on_exit EXIT
cleanup_processes

printf '## build local binary\n'
run go build -o "$local_bin" .

printf '## install remote binary\n'
run scp -o "ConnectTimeout=$connect_timeout" -o BatchMode=yes "$local_bin" "$ssh_target:$remote_bin"
ssh_run "chmod +x $(sq "$remote_bin") && $(sq "$remote_bin") -mode check -profile lan -backend network -timeout 3s >/dev/null"

for profile in $profiles; do
	for phase in $phases; do
		case "$phase" in
		signal | full)
			record_step "$profile bonjour $phase" run_bonjour_pair "$profile" "$phase"
			;;
		*)
			printf 'unknown PHASES value %q; want signal or full\n' "$phase" >&2
			exit 2
			;;
		esac
	done
done

printf '## bonjour summary\n'
if ((bonjour_failures != 0)); then
	printf 'bonjour failed steps=%d\n' "$bonjour_failures" >&2
	for failure in "${bonjour_failed[@]}"; do
		printf 'FAIL: %s\n' "$failure" >&2
	done
	exit 1
fi
printf 'bonjour passed profiles=%s phases=%s\n' "$profiles" "$phases"
