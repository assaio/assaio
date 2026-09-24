#!/usr/bin/env bash
# Cases for guard.sh. Run: bash .claude/hooks/guard_test.sh
# Not a product test -- this is harness config. It exists because a guard that silently
# stopped matching looks exactly like a guard that was never needed.
cd "$(dirname "$0")/../.." || exit 1
fails=0
# An empty project dir: no case may depend on the runner's branch or open work file.
CLAUDE_PROJECT_DIR=$(mktemp -d) || exit 1
export CLAUDE_PROJECT_DIR
trap 'rm -rf "$CLAUDE_PROJECT_DIR"' EXIT

decision() {
	out=$(printf '{"tool_name":"%s","tool_input":%s}' "$1" "$2" | bash .claude/hooks/guard.sh)
	[ -n "$out" ] && printf '%s' "$out" | jq -r '.hookSpecificOutput.permissionDecision' | tr '[:lower:]' '[:upper:]' || echo ALLOW
}

t() {
	want=$1
	got=$(decision Bash "$(jq -cn --arg c "$2" '{command:$c}')")
	if [ "$got" = "$want" ]; then
		printf 'ok   %-5s %s\n' "$got" "$2"
	else
		printf 'FAIL want=%s got=%s  %s\n' "$want" "$got" "$2"
		fails=$((fails + 1))
	fi
}

tw() {
	want=$1
	got=$(decision Write "$(jq -cn --arg c "$2" '{file_path:"x.md",content:$c}')")
	if [ "$got" = "$want" ]; then
		printf 'ok   %-5s write: %s\n' "$got" "$(printf '%s' "$2" | head -c 60 | tr '\n' ' ')"
	else
		printf 'FAIL want=%s got=%s  write: %s\n' "$want" "$got" "$2"
		fails=$((fails + 1))
	fi
}

# --- the real store
t DENY 'assaio-agent clear --all --yes'
t DENY './bin/assaio-agent clear --older-than 90d --yes'
t DENY 'go run ./cmd/assaio-agent clear --all --yes'
t DENY 'assaio clear --tool codex --yes'
t DENY 'make build && ./bin/assaio-agent clear --all --yes'
t DENY 'env XDG_CONFIG_HOME=/tmp/x assaio-agent clear --all --yes'
t ALLOW 'XDG_DATA_HOME=$(mktemp -d) go run ./cmd/assaio-agent clear --all --yes'
t ALLOW 'XDG_DATA_HOME=/tmp/assaio-ab go run ./cmd/assaio-agent clear --all --yes'
t ALLOW 'assaio-agent report --since 30d | grep -c clear'
t ALLOW 'cat > docs/x.md <<EOF
Run the deletion subcommand: assaio-agent clear --all --yes
EOF'
t ALLOW 'echo "assaio-agent clear --all wipes the store"'
t ALLOW 'grep -rn clear internal/cli/'
t ALLOW 'go run ./cmd/assaio-agent report --since 7d'

# --- tags and main
t DENY 'git tag -d v0.21.0'
t DENY 'git tag --force v0.21.0 HEAD'
t DENY 'git push --force origin main'
t DENY 'git push -f origin refs/tags/v0.21.0'
t DENY 'git push origin :refs/tags/v0.21.0'
t DENY 'git push --delete origin v0.21.0'
t ASK 'git push --force origin feat/x'
t ALLOW 'git push --force-with-lease'
t ALLOW 'git push origin feat/codex-steps'
t ALLOW 'git tag --list "v*"'
t ALLOW 'git tag -a v0.22.0 -m "release v0.22.0"'
t ALLOW 'make release VERSION=v0.22.0 CONFIRM=yes'

# --- staging and commits
t DENY 'git add .'
t DENY 'git add -A'
t DENY 'git add --all'
t DENY 'git add -u'
t DENY 'cd /x && git add . && git commit -s -m "feat: y"'
t DENY 'git commit -a -m "feat: x"'
t DENY 'git commit -am "feat: x"'
t DENY 'git commit --all -s -m "feat: x"'
t DENY 'git add .env'
t DENY 'git add config/.env.production'
t ALLOW 'git add .env.example'
t ALLOW 'git add README.md .claude/settings.json'
t ALLOW 'git add internal/cli/report.go'
t ALLOW 'git commit -s -m "feat(agent): read the codex sequence"'
t ALLOW 'git commit -s -m "chore: x" -- README.md'
t DENY 'git commit --no-verify -m "chore: x"'
t DENY 'git commit -n -m "chore: x"'
t DENY 'git commit -m "feat: x

Co-Authored-By: Claude <noreply@anthropic.com>"'
t DENY 'git commit -m "fix: y

🤖 Generated with [Claude Code](https://claude.com/claude-code)"'

# --- secrets on screen
t DENY 'cat .env'
t DENY 'cat ~/.codex/auth.json'
t DENY 'head -5 ~/.claude/.credentials.json'
t DENY 'sed -n 1,3p ~/.netrc'
t DENY 'grep TOKEN .env.local'
t ALLOW 'cat .env.example'
t ALLOW 'ls -la .env'
t ALLOW 'test -f ~/.codex/auth.json && echo present'
t ALLOW 'grep -rn ASSAIO_SERVER_TOKEN internal/'

# --- model family
t DENY 'claude --model claude-fable-5-1 -p "x"'
t DENY 'codex exec -m fable-x'
t DENY 'python3 run.py --judge claude-mythos-5'
t DENY 'echo "model: fable" > .claude/agents/x.md'
t ALLOW 'grep -rn fable .claude/'
t ALLOW 'echo "never use the Fable family in scripts"'
tw DENY 'model: fable
effort: high'
tw DENY '{"model": "claude-fable-5-1"}'
tw DENY 'claude --model claude-mythos-5 -p x'
tw ALLOW 'Fable/Mythos are never written into a script (see CLAUDE.md).'
tw ALLOW 'model: opus'

# --- paid sessions
t ASK 'claude -p "summarise" < notes.txt'
t ASK 'for f in a b; do claude --print "$f"; done'
t ASK 'codex exec --sandbox read-only "review this"'
t ASK 'agy --print="hello" --model gemini-3.1-pro-high'
t ALLOW 'python3 .claude/scripts/text_model.py --engine both --prompt "x" --out o.txt'
t ALLOW 'python3 .claude/scripts/text_model.py --detect'
t ALLOW 'claude --version'

# --- push to main with an open work file
wip=$(mktemp -d)
mkdir -p "$wip/docs/work" && touch "$wip/docs/work/2026-09-18-thing.md" "$wip/docs/work/TEMPLATE.md"
CLAUDE_PROJECT_DIR=$wip t DENY 'git push origin main'
CLAUDE_PROJECT_DIR=$wip t DENY 'git push origin HEAD:main'
CLAUDE_PROJECT_DIR=$wip t DENY 'gh pr merge 42 --squash'
CLAUDE_PROJECT_DIR=$wip t ALLOW 'git push origin feat/x'
rm -f "$wip/docs/work/2026-09-18-thing.md"
CLAUDE_PROJECT_DIR=$wip t ALLOW 'git push origin main'
CLAUDE_PROJECT_DIR=$wip t ALLOW 'gh pr merge 42 --squash'
rm -rf "$wip"

# --- ordinary work
t ALLOW 'make test lint'
t ALLOW 'go test ./internal/cli/ -run TestClear'
t ALLOW 'bash .claude/checks/gate.sh'

[ "$fails" -eq 0 ] || printf '\n%d case(s) failed\n' "$fails"
exit $((fails > 0))
