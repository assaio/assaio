#!/usr/bin/env bash
# PreToolUse(Bash) guard. Denies only what is irreversible in this repo and invisible in a
# diff: the maintainer's real store, a published tag, and an authorship trailer CI rejects.
# Everything else is left to the normal permission flow.
set -uo pipefail

payload=$(cat)
if command -v jq >/dev/null 2>&1; then
	cmd=$(printf '%s' "$payload" | jq -r '.tool_input.command // ""')
else
	cmd=$payload
fi

deny() {
	jq -nc --arg r "$1" '{hookSpecificOutput:{hookEventName:"PreToolUse",permissionDecision:"deny",permissionDecisionReason:$r}}' 2>/dev/null ||
		printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"%s"}}' "$1"
	exit 0
}

# A command is what a segment *starts* with, not what its text happens to contain. Heredoc
# bodies are data -- this repo's own docs and changelog name the deletion subcommand -- so
# they are cut before matching, and each remaining segment is judged by its leading word
# past any env/assignment prefix. Without that, writing about the subcommand was refused as
# running it.
segments() {
	printf '%s\n' "$1" | awk '
		inbody { line = $0; sub(/^[[:space:]]+/, "", line); if (line == delim) inbody = 0; next }
		{
			if (match($0, "<<-?[[:space:]]*[\047\042]?[A-Za-z_][A-Za-z0-9_]*")) {
				delim = substr($0, RSTART, RLENGTH)
				sub(/^<<-?[[:space:]]*/, "", delim)
				gsub("[\047\042]", "", delim)
				inbody = 1
			}
			print
		}
	' | tr '|;&' '\n\n\n'
}

# The deletion subcommand has no --db: it opens whatever DataDir() resolves to, which is the
# maintainer's real store unless XDG_DATA_HOME was redirected to a throwaway first. That
# redirect is the only accepted way to run it, so it is required in the same segment rather
# than assumed to be in effect from somewhere else.
while IFS= read -r seg; do
	bare=$(printf '%s' "$seg" | sed -E 's/^[[:space:]]*//; s/^((env|sudo|nohup|time)[[:space:]]+|[A-Za-z_][A-Za-z0-9_]*=("[^"]*"|\$\([^)]*\)|[^[:space:]]*)[[:space:]]+)*//')
	printf '%s' "$bare" | grep -qE '^(([^[:space:]]*/)?(assaio-agent|assaio)|go[[:space:]]+run[[:space:]]+[^[:space:]]*assaio-agent)[[:space:]]' || continue
	printf '%s' "$bare" | grep -qE '[[:space:]]clear([[:space:]]|$)' || continue
	printf '%s' "$seg" | grep -qE 'XDG_DATA_HOME=("|\$\(mktemp|/tmp/|/private/tmp/|/var/folders/)' && continue
	deny "This subcommand has no --db and would open the real store (~/.local/share/assaio/assaio.db, 170 MB of history older than the sources keep). Point it at a throwaway in the same command: XDG_DATA_HOME=\$(mktemp -d) assaio-agent ..."
done <<EOF
$(segments "$cmd")
EOF

# Published tags are immutable by repository ruleset; a bad release is fixed by the next one.
if printf '%s' "$cmd" | grep -qE 'git[[:space:]]+tag[[:space:]]+(-d|--delete|-f|--force)([[:space:]]|$)'; then
	deny "Published tags are immutable (RELEASING.md, enforced by the immutable-release-tags ruleset). Fix a bad release with the next patch release."
fi
if printf '%s' "$cmd" | grep -qE 'git[[:space:]]+push[^|;&]*(--delete[[:space:]]|:refs/tags/)'; then
	deny "Deleting a published ref is refused here. Fix a bad release with the next patch release."
fi
if printf '%s' "$cmd" | grep -qE 'git[[:space:]]+push[^|;&]*(-f|--force)[^|;&]*([[:space:]]main([[:space:]]|$)|refs/tags|[[:space:]]v[0-9])'; then
	deny "main is protected and tags are immutable. Force-push only a feature branch, with --force-with-lease."
fi

# The commit-msg hook is opt-in (make hooks); the dco CI job is not, and rejects these.
if printf '%s' "$cmd" | grep -qiE 'co-authored-by:[^"'"'"']*(claude|anthropic|copilot|codex|gemini|cursor|gpt)|generated with[[:space:]]*\[?(claude|anthropic)|🤖 generated with'; then
	deny "This repo never credits an AI assistant as an author (CONTRIBUTING.md rule 4; the dco CI job rejects the trailer). The human who signs off is the author."
fi
if printf '%s' "$cmd" | grep -qE 'git[[:space:]]+commit[^|;&]*([[:space:]]--no-verify|[[:space:]]-n([[:space:]]|$))'; then
	deny "--no-verify skips the Conventional-Commits and Signed-off-by check. Fix the message instead."
fi

exit 0
