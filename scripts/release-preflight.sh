#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
demo_dir=${DEMO_DIR:-$(cd -- "$script_dir/.." && pwd)}
apple_dir=${APPLE_DIR:-$(cd -- "$demo_dir/../apple" && pwd)}
apple_pion_dir=${APPLE_PION_DIR:-$(cd -- "$demo_dir/../apple-pion" && pwd)}
github_resolve_ip=${GITHUB_RESOLVE_IP:-}

failures=0

section() {
	printf '## %s\n' "$1"
}

run() {
	printf '+'
	printf ' %q' "$@"
	printf '\n'
	"$@"
}

fail() {
	printf 'FAIL: %s\n' "$*" >&2
	failures=$((failures + 1))
}

warn() {
	printf 'WARN: %s\n' "$*" >&2
}

git_repo() {
	local repo=$1
	shift
	if [[ -n $github_resolve_ip ]]; then
		git -C "$repo" -c "http.curloptResolve=github.com:443:$github_resolve_ip" "$@"
	else
		git -C "$repo" "$@"
	fi
}

require_clean() {
	local repo=$1
	local name=$2
	local status
	status=$(git_repo "$repo" status --porcelain)
	if [[ -n $status ]]; then
		fail "$name worktree is dirty"
		printf '%s\n' "$status"
	fi
}

require_remote() {
	local repo=$1
	local name=$2
	local remote
	remote=$(git_repo "$repo" remote)
	if [[ -z $remote ]]; then
		fail "$name has no configured git remote"
		return
	fi
	printf '%s remotes:\n' "$name"
	git_repo "$repo" remote -v
}

require_head_published() {
	local repo=$1
	local name=$2
	local remote=${3:-origin}
	local branch=${4:-main}
	local head remote_head
	if ! git_repo "$repo" remote get-url "$remote" >/dev/null 2>&1; then
		fail "$name has no $remote remote"
		return
	fi
	head=$(git_repo "$repo" rev-parse HEAD)
	if ! remote_head=$(git_repo "$repo" ls-remote "$remote" "refs/heads/$branch" | awk '{print $1}'); then
		fail "$name cannot read $remote/$branch"
		return
	fi
	if [[ -z $remote_head ]]; then
		fail "$name cannot read $remote/$branch"
		return
	fi
	printf '%s local=%s remote=%s/%s=%s\n' "$name" "$head" "$remote" "$branch" "$remote_head"
	if [[ $head != "$remote_head" ]]; then
		fail "$name HEAD is not published at $remote/$branch"
	fi
}

require_no_replace() {
	local repo=$1
	local name=$2
	if grep -q '^replace ' "$repo/go.mod"; then
		fail "$name go.mod still uses local replace directives"
		grep '^replace ' "$repo/go.mod"
	fi
}

require_published_module() {
	local repo=$1
	local name=$2
	local module=$3
	local modcache version dir replace
	modcache=$(GOWORK=off go env GOMODCACHE)
	version=$(GOWORK=off go -C "$repo" list -m -f '{{.Version}}' "$module")
	dir=$(GOWORK=off go -C "$repo" list -m -f '{{.Dir}}' "$module")
	replace=$(GOWORK=off go -C "$repo" list -m -f '{{if .Replace}}{{.Replace.Path}}{{end}}' "$module")
	printf '%s %s version=%s dir=%s\n' "$name" "$module" "$version" "$dir"
	if [[ -z $version || $version == "<nil>" ]]; then
		fail "$name uses an unpublished $module version"
	fi
	if [[ -n $replace ]]; then
		fail "$name resolves $module through local replace: $replace"
	fi
	case "$dir/" in
	"$modcache"/*) ;;
	*) fail "$name resolves $module outside GOMODCACHE: $dir" ;;
	esac
}

section "local gates"
if [[ -n $github_resolve_ip ]]; then
	printf 'github_resolve_ip=%s\n' "$github_resolve_ip"
fi
run bash -n "$demo_dir/scripts/remote-diagnostics.sh"
run bash -n "$demo_dir/scripts/remote-matrix.sh"
run bash -n "$demo_dir/scripts/remote-matrix-bundle.sh"
run env GOWORK=off go -C "$demo_dir" test ./...
run env GOWORK=off go -C "$demo_dir" vet ./...
run env GOWORK=off go -C "$apple_pion_dir" test ./...
run env GOWORK=off go -C "$apple_pion_dir" vet ./...
run go -C "$apple_dir" test ./x/network/nwpacket
run go -C "$apple_dir" vet ./x/network/nwpacket

section "package availability"
run env GOWORK=off go -C "$apple_pion_dir" list ./...
run env GOWORK=off go -C "$demo_dir" list ./...
require_published_module "$demo_dir" "awdl-webrtc-apple-demo" "github.com/tmc/apple"
require_published_module "$demo_dir" "awdl-webrtc-apple-demo" "github.com/tmc/apple-pion"
require_published_module "$apple_pion_dir" "apple-pion" "github.com/tmc/apple"
run go -C "$apple_dir" list ./x/network/nwpacket

section "worktree state"
require_clean "$demo_dir" "awdl-webrtc-apple-demo"
require_clean "$apple_pion_dir" "apple-pion"
if [[ -n $(git_repo "$apple_dir" status --porcelain -- x/network/nwpacket) ]]; then
	fail "apple x/network/nwpacket has local changes"
fi
if [[ -n $(git_repo "$apple_dir" status --porcelain) ]]; then
	warn "apple worktree has unrelated local changes"
	git_repo "$apple_dir" status --short | sed -n '1,80p'
fi

section "release blockers"
require_remote "$demo_dir" "awdl-webrtc-apple-demo"
require_remote "$apple_pion_dir" "apple-pion"
require_remote "$apple_dir" "apple"
require_head_published "$demo_dir" "awdl-webrtc-apple-demo"
require_head_published "$apple_pion_dir" "apple-pion"
require_head_published "$apple_dir" "apple"
require_no_replace "$demo_dir" "awdl-webrtc-apple-demo"
require_no_replace "$apple_pion_dir" "apple-pion"

if ((failures != 0)); then
	printf 'release preflight failed with %d blocker(s)\n' "$failures" >&2
	exit 1
fi
printf 'release preflight passed\n'
