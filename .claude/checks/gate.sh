#!/usr/bin/env bash
# The local gate, quiet: one line per step with the tool's own verdict, the log tail only
# when a step is red, full logs under $XDG_CACHE_HOME/assaio-gate/. It mirrors ci.yml plus
# the harness checks; `full` adds fuzz and vuln unconditionally, otherwise fuzz runs only when
# a parser, the reconciler or the plugin boundary changed. Nothing here mutates the tree:
# formatting is checked with --diff, never applied.
#
# Usage: gate.sh [quick|full]
set -uo pipefail
cd "$(dirname "$0")/../.." || exit 1
mode=${1:-quick}
logdir=${XDG_CACHE_HOME:-$HOME/.cache}/assaio-gate
mkdir -p "$logdir"
red=0

step() {
	name=$1
	shift
	log=$logdir/$name.log
	if "$@" >"$log" 2>&1; then
		printf 'ok    %-9s %s\n' "$name" "$*"
	else
		red=1
		printf 'FAIL  %-9s %s\n' "$name" "$*"
		tail -n 25 "$log" | sed 's/^/      /'
		printf '      full log: %s\n' "$log"
	fi
}

fmt_check() {
	out=$(golangci-lint fmt --diff 2>&1) || return 1
	[ -z "$out" ] || { printf '%s\n' "$out"; return 1; }
}

changed() {
	{ git diff --name-only main...HEAD 2>/dev/null; git status --porcelain | awk '{print $2}'; } | sort -u
}

step fmt fmt_check
step lint make lint
step test make test
step build env CGO_ENABLED=0 go build ./...
step docs sh -c 'make docs >/dev/null && git status --porcelain docs/reference.json site/reference.html site/docs | { ! grep .; }'
step harness sh -c 'python3 .claude/checks/setup.py && python3 .claude/checks/ops_catalogue.py && bash .claude/hooks/guard_test.sh >/dev/null'

if [ "$mode" = full ] || changed | grep -qE '^internal/(parser|reconcile|plugin)/'; then
	step fuzz make fuzz
fi
if [ "$mode" = full ]; then
	step vuln make vuln
fi

[ "$red" -eq 0 ] && printf 'gate: green (%s)\n' "$mode" || printf 'gate: RED (%s)\n' "$mode"
exit $red
