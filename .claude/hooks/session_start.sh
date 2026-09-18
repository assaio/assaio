#!/usr/bin/env bash
# SessionStart hook: three lines of state, so a session starts from facts rather than from
# re-deriving them. Anything that fails prints nothing for that line; the hook never blocks.
set -uo pipefail
root=${CLAUDE_PROJECT_DIR:-.}
cd "$root" 2>/dev/null || exit 0

wip=$(ls docs/work/*.md 2>/dev/null | grep -v TEMPLATE | xargs -n1 basename 2>/dev/null | tr '\n' ' ')
parked=$(ls docs/work/parked/*.md 2>/dev/null | wc -l | tr -d ' ')
branch=$(git symbolic-ref --short HEAD 2>/dev/null || echo detached)
dirty=$(git status --porcelain 2>/dev/null | wc -l | tr -d ' ')
engines=$(python3 .claude/scripts/text_model.py --detect 2>/dev/null || echo '{}')

line1="Work in progress: ${wip:-none} (parked: ${parked:-0})"
line2="Branch ${branch}, uncommitted paths: ${dirty} (other sessions may own them; stage explicit paths only)"
line3="Content engines (GPT/Gemini door): ${engines}"

if command -v jq >/dev/null 2>&1; then
	jq -nc --arg c "$line1"$'\n'"$line2"$'\n'"$line3" '{hookSpecificOutput:{hookEventName:"SessionStart",additionalContext:$c}}'
else
	printf '%s\n%s\n%s\n' "$line1" "$line2" "$line3"
fi
exit 0
