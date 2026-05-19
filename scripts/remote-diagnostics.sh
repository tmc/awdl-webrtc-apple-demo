#!/usr/bin/env bash
set -euo pipefail

ssh_target=${SSH_TARGET:-tmc2@10.0.18.249}
connect_timeout=${CONNECT_TIMEOUT:-5}
run_remote=${RUN_REMOTE:-1}
profiles=${PROFILES:-lan thunderbolt awdl}
lan_iface=${LAN_IFACE:-en0}
awdl_iface=${AWDL_IFACE:-awdl0}
thunderbolt_ifaces=${THUNDERBOLT_INTERFACES:-bridge0 en1 en2 en3}
lan_peer=${LAN_PEER:-${ssh_target##*@}}
thunderbolt_peer=${THUNDERBOLT_PEER:-}
awdl_peer=${AWDL_PEER:-}
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

run_cmd() {
	printf '+'
	printf ' %q' "$@"
	printf '\n'
	"$@" 2>&1 || printf 'command failed exit=%d\n' "$?"
}

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
printf 'interfaces=%s\n' "$interfaces"
printf 'routes=%s\n' "$routes"

diagnose_body local "$interfaces" "$routes"

if [[ $run_remote == 1 ]]; then
	remote_script="$(declare -f run_cmd diagnose_body); diagnose_body remote $(sq "$interfaces") $(sq "$routes")"
	run_cmd ssh -o "ConnectTimeout=$connect_timeout" -o BatchMode=yes "$ssh_target" "bash -lc $(sq "$remote_script")"
fi
