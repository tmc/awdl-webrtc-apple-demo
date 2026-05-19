#!/usr/bin/env bash
set -euo pipefail

ssh_target=${SSH_TARGET:-tmc2@10.0.18.249}
connect_timeout=${CONNECT_TIMEOUT:-5}
run_remote=${RUN_REMOTE:-1}
profiles=${PROFILES:-lan thunderbolt awdl}
lan_iface=${LAN_IFACE:-en0}
awdl_iface=${AWDL_IFACE:-awdl0}
thunderbolt_ifaces=${THUNDERBOLT_INTERFACES:-bridge0 en1 en2 en3}
lan_peer_explicit=${LAN_PEER+x}
thunderbolt_peer_explicit=${THUNDERBOLT_PEER+x}
awdl_peer_explicit=${AWDL_PEER+x}
lan_peer=${LAN_PEER:-${ssh_target##*@}}
thunderbolt_peer=${THUNDERBOLT_PEER:-}
awdl_peer=${AWDL_PEER:-}
use_discovery=${USE_DISCOVERY:-0}
discovery_file=${DISCOVERY_FILE:-}
discovery_peer=${DISCOVERY_PEER:-}
discovery_timeout=${DISCOVERY_TIMEOUT:-20s}
discovery_interval=${DISCOVERY_INTERVAL:-500ms}
discovery_backend=${DISCOVERY_BACKEND:-network}
output=${OUTPUT:-}

if [[ -n $output ]]; then
	mkdir -p "$(dirname "$output")"
	exec > >(tee "$output") 2>&1
fi

sq() {
	local s=${1//\'/\'\\\'\'}
	printf "'%s'" "$s"
}

unique_words() {
	awk '{
		for (i = 1; i <= NF; i++) {
			if (!seen[$i]++) {
				printf "%s%s", sep, $i
				sep = " "
			}
		}
	}'
}

truthy() {
	case "${1:-}" in
	1 | true | TRUE | yes | YES | on | ON)
		return 0
		;;
	esac
	return 1
}

discovery_requested() {
	[[ -n $discovery_file ]] || truthy "$use_discovery"
}

require_jq() {
	if ! command -v jq >/dev/null 2>&1; then
		printf 'jq is required when USE_DISCOVERY=1 or DISCOVERY_FILE is set\n' >&2
		exit 2
	fi
}

read_discovery_json() {
	if [[ -n $discovery_file ]]; then
		cat "$discovery_file"
		return
	fi
	go run . -mode discover-wait -backend "$discovery_backend" -discover-peer "$discovery_peer" -timeout "$discovery_timeout" -ui-interval "$discovery_interval"
}

discovery_addr() {
	local profile=$1
	jq -r -s --arg profile "$profile" '
		map(select(.kind == "link_health_discovery" and ((.peer.addrs[$profile] // "") != ""))) |
		last |
		if . == null then "" else .peer.addrs[$profile] end
	'
}

load_discovery_peers() {
	discovery_requested || return
	require_jq

	local json addr
	json=$(read_discovery_json)
	printf '## discovery peer record\n'
	printf '%s\n' "$json"

	if [[ -z $lan_peer_explicit ]]; then
		addr=$(printf '%s\n' "$json" | discovery_addr lan)
		if [[ -n $addr ]]; then
			lan_peer=$addr
		fi
	fi
	if [[ -z $thunderbolt_peer_explicit ]]; then
		addr=$(printf '%s\n' "$json" | discovery_addr thunderbolt)
		if [[ -n $addr ]]; then
			thunderbolt_peer=$addr
		fi
	fi
	if [[ -z $awdl_peer_explicit ]]; then
		addr=$(printf '%s\n' "$json" | discovery_addr awdl)
		if [[ -n $addr ]]; then
			awdl_peer=$addr
		fi
	fi
}

diagnostic_interfaces() {
	local out=
	for profile in $profiles; do
		case "$profile" in
		lan)
			out+=" $lan_iface"
			;;
		awdl)
			out+=" $awdl_iface"
			;;
		thunderbolt)
			out+=" $thunderbolt_ifaces"
			;;
		esac
	done
	printf '%s\n' "$out" | unique_words
}

diagnostic_routes() {
	printf '%s\n' "$lan_peer $thunderbolt_peer $awdl_peer" | unique_words
}

route_target() {
	local target=$1
	if [[ $target == \[*\]:* ]]; then
		target=${target#\[}
		printf '%s\n' "${target%\]:*}"
		return
	fi
	if [[ $target == *:* && $target != *:*:* ]]; then
		printf '%s\n' "${target%:*}"
		return
	fi
	printf '%s\n' "$target"
}

run_cmd() {
	printf '+'
	printf ' %q' "$@"
	printf '\n'
	"$@" 2>&1 || printf 'command failed exit=%d\n' "$?"
}

load_discovery_peers

diagnose_body() {
	local label=$1
	local ifaces=$2
	local routes=$3

	printf '## %s metadata\n' "$label"
	run_cmd date -u
	run_cmd hostname
	run_cmd sw_vers
	run_cmd uname -a

	printf '## %s interfaces\n' "$label"
	run_cmd ifconfig -l
	for iface in $ifaces; do
		run_cmd ifconfig "$iface"
	done

	printf '## %s routes\n' "$label"
	run_cmd netstat -rn -f inet
	run_cmd netstat -rn -f inet6
	for target in $routes; do
		target=$(route_target "$target")
		if [[ $target == *:* ]]; then
			run_cmd route -n get -inet6 "$target"
		else
			run_cmd route -n get "$target"
		fi
	done

	printf '## %s network state\n' "$label"
	run_cmd scutil --nwi
	run_cmd networksetup -listallhardwareports
	if command -v lsof >/dev/null 2>&1; then
		run_cmd lsof -nP -iUDP
	fi
}

interfaces=$(diagnostic_interfaces)
routes=$(diagnostic_routes)

printf '## diagnostics config\n'
printf 'ssh_target=%s\n' "$ssh_target"
printf 'profiles=%s\n' "$profiles"
printf 'use_discovery=%s\n' "$use_discovery"
printf 'discovery_file=%s\n' "$discovery_file"
printf 'discovery_peer=%s\n' "$discovery_peer"
printf 'discovery_timeout=%s\n' "$discovery_timeout"
printf 'discovery_interval=%s\n' "$discovery_interval"
printf 'interfaces=%s\n' "$interfaces"
printf 'routes=%s\n' "$routes"

diagnose_body local "$interfaces" "$routes"

if [[ $run_remote == 1 ]]; then
	remote_script="$(declare -f route_target run_cmd diagnose_body); diagnose_body remote $(sq "$interfaces") $(sq "$routes")"
	run_cmd ssh -o "ConnectTimeout=$connect_timeout" -o BatchMode=yes "$ssh_target" "bash -lc $(sq "$remote_script")"
fi
