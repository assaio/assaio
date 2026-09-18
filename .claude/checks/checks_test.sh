#!/usr/bin/env bash
# Negative controls for the two setup checks: each must fail on a fixture that breaks the rule
# it holds, and pass on the real tree. A check that cannot fail is decoration.
# Run: bash .claude/checks/checks_test.sh
cd "$(dirname "$0")/../.." || exit 1
fails=0
expect() {
	want=$1
	label=$2
	shift 2
	if "$@" >/dev/null 2>&1; then got=0; else got=1; fi
	if [ "$got" = "$want" ]; then
		printf 'ok   exit=%s %s\n' "$got" "$label"
	else
		printf 'FAIL want exit=%s got=%s %s\n' "$want" "$got" "$label"
		fails=$((fails + 1))
	fi
}

fixture() {
	dir=$(mktemp -d)
	git -C "$dir" init -q
	mkdir -p "$dir/.claude/agents" "$dir/.claude/skills/one" "$dir/.claude/rules" "$dir/.github/workflows" "$dir/docs" "$dir/internal/x"
	cp .claude/settings.json "$dir/.claude/settings.json"
	mkdir -p "$dir/.claude/hooks" "$dir/.claude/checks" "$dir/.claude/scripts"
	cp .claude/hooks/*.sh "$dir/.claude/hooks/"
	cp .claude/checks/*.py .claude/checks/*.sh "$dir/.claude/checks/"
	cp .claude/scripts/*.py "$dir/.claude/scripts/"
	printf '%s\n' 'name: a' > "$dir/CLAUDE.md"
	printf '%s\n' '# agents' > "$dir/AGENTS.md"
	printf '%s\n' 'package x' > "$dir/internal/x/x.go"
	printf 'build:\n\tgo build ./...\n' > "$dir/Makefile"
	printf 'name: ci\n' > "$dir/.github/workflows/ci.yml"
	cat > "$dir/.claude/agents/scout.md" <<'EOF'
---
name: scout
description: Finds where something lives and reports file:line, nothing else. Read-only, cheap, used for lookups before any edit.
tools: Read, Grep, Glob
model: haiku
---
body
EOF
	cat > "$dir/.claude/skills/one/SKILL.md" <<'EOF'
---
name: one
description: A skill fixture with a description long enough to pass the length rule that the check enforces on every skill.
---
body
EOF
	cat > "$dir/.claude/rules/x.md" <<'EOF'
---
paths:
  - "internal/x/**"
---
rule
EOF
	git -C "$dir" add -A >/dev/null 2>&1
	printf '%s' "$dir"
}

ops_ok() {
	cat > "$1/docs/operations.md" <<'EOF'
| Entry point | What it does |
|---|---|
| `make build` | builds |
| `.github/workflows/ci.yml` | ci |
EOF
	for f in "$1"/.claude/hooks/*.sh "$1"/.claude/checks/*.py "$1"/.claude/checks/*.sh "$1"/.claude/scripts/*.py; do
		printf '| `%s` | harness |\n' "${f#"$1"/}" >> "$1/docs/operations.md"
	done
}

echo "--- setup.py"
expect 0 "real tree" python3 .claude/checks/setup.py
d=$(fixture); ops_ok "$d"
expect 0 "fixture passes" python3 .claude/checks/setup.py --root "$d"
printf '%s\n' '---' 'name: scout' 'description: short' 'tools: Read' 'model: haiku' '---' > "$d/.claude/agents/scout.md"
expect 1 "description too short" python3 .claude/checks/setup.py --root "$d"
d=$(fixture); ops_ok "$d"
printf '%s\n' '---' 'name: scout' 'description: A description long enough to pass the length rule, with a colon: inside it that breaks YAML parsing.' 'tools: Read' 'model: haiku' '---' > "$d/.claude/agents/scout.md"
expect 1 "unquoted ': '" python3 .claude/checks/setup.py --root "$d"
d=$(fixture); ops_ok "$d"
sed -i '' 's/model: haiku/model: inherit/' "$d/.claude/agents/scout.md"
expect 1 "model must be explicit" python3 .claude/checks/setup.py --root "$d"
d=$(fixture); ops_ok "$d"
printf '%s\n' '---' 'paths:' '  - "internal/nothing/**"' '---' 'rule' > "$d/.claude/rules/x.md"
expect 1 "glob hits nothing" python3 .claude/checks/setup.py --root "$d"
d=$(fixture); ops_ok "$d"
printf '%s\n' '---' 'description: unscoped' '---' 'rule' > "$d/.claude/rules/x.md"
expect 1 "rule without paths" python3 .claude/checks/setup.py --root "$d"
d=$(fixture); ops_ok "$d"
printf 'see `internal/missing/file.go`\n' >> "$d/AGENTS.md"
expect 1 "cited path missing" python3 .claude/checks/setup.py --root "$d"
d=$(fixture); ops_ok "$d"
seq 1 160 | sed 's/^/line /' > "$d/AGENTS.md"
expect 1 "AGENTS.md over budget" python3 .claude/checks/setup.py --root "$d"
d=$(fixture); ops_ok "$d"
printf '%s\n' '---' 'name: x' 'description: A description long enough to pass the length rule that the check enforces on every file.' 'tools: Read' 'model: fable' '---' > "$d/.claude/agents/x.md"
expect 1 "fable assigned" python3 .claude/checks/setup.py --root "$d"
d=$(fixture); ops_ok "$d"
printf '%s\n' '---' 'name: one' 'description: A skill fixture with a description long enough to pass the length rule that the check enforces.' 'argument-hint: [a | b [c]]' '---' > "$d/.claude/skills/one/SKILL.md"
expect 1 "unquoted [ in value" python3 .claude/checks/setup.py --root "$d"

echo "--- ops_catalogue.py"
expect 0 "real tree" python3 .claude/checks/ops_catalogue.py
d=$(fixture); ops_ok "$d"
expect 0 "fixture passes" python3 .claude/checks/ops_catalogue.py --root "$d"
printf 'extra:\n\techo x\n' >> "$d/Makefile"
expect 1 "target without a row" python3 .claude/checks/ops_catalogue.py --root "$d"
d=$(fixture); ops_ok "$d"
printf '| `make gone` | nothing |\n' >> "$d/docs/operations.md"
expect 1 "row without a target" python3 .claude/checks/ops_catalogue.py --root "$d"
d=$(fixture); ops_ok "$d"
rm "$d/docs/operations.md"
expect 1 "catalogue missing" python3 .claude/checks/ops_catalogue.py --root "$d"

[ "$fails" -eq 0 ] || printf '\n%d case(s) failed\n' "$fails"
exit $((fails > 0))
