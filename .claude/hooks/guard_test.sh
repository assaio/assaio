#!/usr/bin/env bash
# Cases for guard.sh. Run: bash .claude/hooks/guard_test.sh
# Not a CI gate -- this is harness config, not product. It exists because a guard that
# silently stopped matching looks exactly like a guard that was never needed.
cd "$(dirname "$0")/../.." || exit 1
fails=0

t() {
	want=$1
	got=ALLOW
	[ -n "$(printf '{"tool_name":"Bash","tool_input":{"command":%s}}' "$(jq -Rn --arg c "$2" '$c')" | bash .claude/hooks/guard.sh)" ] && got=DENY
	if [ "$got" = "$want" ]; then
		printf 'ok   %-5s %s\n' "$got" "$2"
	else
		printf 'FAIL want=%s got=%s  %s\n' "$want" "$got" "$2"
		fails=$((fails + 1))
	fi
}

t DENY 'assaio-agent clear --all --yes'
t DENY './bin/assaio-agent clear --older-than 90d --yes'
t DENY 'go run ./cmd/assaio-agent clear --all --yes'
t DENY 'assaio clear --tool codex --yes'
t DENY 'git tag -d v0.21.0'
t DENY 'git tag --force v0.21.0 HEAD'
t DENY 'git push --force origin main'
t DENY 'git push -f origin refs/tags/v0.21.0'
t DENY 'git push origin :refs/tags/v0.21.0'
t DENY 'git push --delete origin v0.21.0'
t DENY 'git commit -m "feat: x

Co-Authored-By: Claude <noreply@anthropic.com>"'
t DENY 'git commit -m "fix: y

🤖 Generated with [Claude Code](https://claude.com/claude-code)"'
t DENY 'git commit --no-verify -m "chore: x"'
t DENY 'git commit -n -m "chore: x"'

t ALLOW 'make test lint'
t ALLOW 'go test ./internal/cli/ -run TestClear'
t ALLOW 'git push --force-with-lease'
t ALLOW 'git push origin feat/codex-steps'
t ALLOW 'git commit -s -m "feat(agent): read the codex sequence"'
t ALLOW 'git tag --list "v*"'
t ALLOW 'git tag -a v0.22.0 -m "release v0.22.0"'
t ALLOW 'grep -rn clear internal/cli/'
t ALLOW 'go run ./cmd/assaio-agent report --since 7d'
t ALLOW 'make release VERSION=v0.22.0 CONFIRM=yes'
t ALLOW 'XDG_DATA_HOME=$(mktemp -d) go run ./cmd/assaio-agent clear --all --yes'
t ALLOW 'XDG_DATA_HOME=/tmp/assaio-ab go run ./cmd/assaio-agent clear --all --yes'
t ALLOW 'assaio-agent report --since 30d | grep -c clear'

[ "$fails" -eq 0 ] || printf '\n%d case(s) failed\n' "$fails"
exit $((fails > 0))
