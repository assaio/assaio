#!/bin/sh
# Enforces Conventional Commit subjects and DCO sign-off. See CONTRIBUTING.md.
set -eu

msg_file="$1"
subject="$(head -n 1 "$msg_file")"

subject_re='^(feat|fix|docs|refactor|test|chore|perf|ci)(\([a-z0-9/-]+\))?!?: .+'
if ! echo "$subject" | grep -qE "$subject_re"; then
	echo "commit-msg: subject does not match Conventional Commits format" >&2
	echo "  got:      $subject" >&2
	echo "  expected: <type>(<scope>): <summary>, e.g. feat(agent): parse Codex rollout logs" >&2
	echo "  types:    feat fix docs refactor test chore perf ci" >&2
	exit 1
fi

if ! grep -qE '^Signed-off-by: .+ <.+>$' "$msg_file"; then
	echo "commit-msg: missing DCO sign-off" >&2
	echo "  fix: commit with 'git commit -s' (or add --signoff)" >&2
	exit 1
fi
