#!/usr/bin/env bash
# PreToolUse guard for Bash, Write and Edit. It denies what is irreversible in this repo and
# invisible in a diff (the maintainer's real store, a published tag, an authorship trailer CI
# rejects, a blanket `git add` over another session's work, a secret on screen, a Fable/Mythos
# model id written into config) and asks before the keystrokes that spend money or publish.
# Everything else is left to the normal permission flow. An internal error exits 0: a guard
# that breaks must fail open, or nothing can be run to fix it.
set -uo pipefail

payload=$(cat)
if command -v jq >/dev/null 2>&1; then
	tool=$(printf '%s' "$payload" | jq -r '.tool_name // ""')
	cmd=$(printf '%s' "$payload" | jq -r '.tool_input.command // ""')
	written=$(printf '%s' "$payload" | jq -r '(.tool_input.content // "") + "\n" + (.tool_input.new_string // "")')
else
	tool=Bash
	cmd=$payload
	written=""
fi

decide() {
	jq -nc --arg d "$1" --arg r "$2" '{hookSpecificOutput:{hookEventName:"PreToolUse",permissionDecision:$d,permissionDecisionReason:$r}}' 2>/dev/null ||
		printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"%s","permissionDecisionReason":"%s"}}' "$1" "$2"
	exit 0
}
deny() { decide deny "$1"; }
ask() { decide ask "$1"; }

# A model id is refused only where it is *assigned*: a flag value, a config key, a frontmatter
# field. Prose naming the family (this file, the rules) has to stay writable.
fable_assignment='(--model|--judge|--candidates|--subagent-model)[[:space:]=]+["'"'"']?(claude-)?(fable|mythos)|(^|[^A-Za-z_])model["'"'"']?[[:space:]]*[:=][[:space:]]*["'"'"']?(claude-)?(fable|mythos)|^(codex|claude|agy)[[:space:]].*[[:space:]]-m[[:space:]=]+["'"'"']?(claude-)?(fable|mythos)'

if [ "$tool" = "Write" ] || [ "$tool" = "Edit" ] || [ "$tool" = "MultiEdit" ]; then
	if printf '%s' "$written" | grep -qiE "$fable_assignment"; then
		deny "Fable/Mythos is never the working model in a script, config, agent or flag (~/.claude/CLAUDE.md). Use haiku, sonnet or opus; the Anthropic default in scripts is claude-opus-5."
	fi
	exit 0
fi
[ "$tool" = "Bash" ] || exit 0

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

strip_prefix() {
	printf '%s' "$1" | sed -E 's/^[[:space:]]*//; s/^((env|sudo|nohup|time|do|then|else|exec)[[:space:]]+|[A-Za-z_][A-Za-z0-9_]*=("[^"]*"|\$\([^)]*\)|[^[:space:]]*)[[:space:]]+)*//'
}

# Files whose contents must never land in a transcript. `.env.example` and friends are docs.
secret_path='(^|[[:space:]/=])\.env(\.[A-Za-z0-9_-]+)?([[:space:]]|$)|\.codex/auth\.json|\.claude/\.credentials\.json|\.netrc|id_(rsa|ed25519|ecdsa)|\.(pem|p12|pfx|keychain)([[:space:]]|$)'
secret_doc='\.env\.(example|sample|template|dist)'

wip_files() {
	ls "${CLAUDE_PROJECT_DIR:-.}"/docs/work/*.md 2>/dev/null | grep -v 'TEMPLATE' || true
}

while IFS= read -r seg; do
	bare=$(strip_prefix "$seg")
	[ -n "$bare" ] || continue

	# The deletion subcommand has no --db: it opens whatever DataDir() resolves to, which is the
	# maintainer's real store unless XDG_DATA_HOME was redirected to a throwaway first. That
	# redirect is the only accepted way to run it, so it is required in the same segment.
	if printf '%s' "$bare" | grep -qE '^(([^[:space:]]*/)?(assaio-agent|assaio)|go[[:space:]]+run[[:space:]]+[^[:space:]]*assaio-agent)[[:space:]]' &&
		printf '%s' "$bare" | grep -qE '[[:space:]]clear([[:space:]]|$)' &&
		! printf '%s' "$seg" | grep -qE 'XDG_DATA_HOME=("|\$\(mktemp|/tmp/|/private/tmp/|/var/folders/)'; then
		deny "This subcommand has no --db and would open the real store (~/.local/share/assaio/assaio.db, 170 MB of history older than the sources keep). Point it at a throwaway in the same command: XDG_DATA_HOME=\$(mktemp -d) assaio-agent ..."
	fi

	# Other sessions may hold uncommitted work in this tree. A commit names its paths.
	if printf '%s' "$bare" | grep -qE '^git[[:space:]]+add([[:space:]]+[^[:space:]]+)*[[:space:]]+(\.|-A|--all|-u|--update)([[:space:]]|$)'; then
		deny "Stage explicit paths only (git add <path> ...). Another session's uncommitted work may be in this tree."
	fi
	if printf '%s' "$bare" | grep -qE '^git[[:space:]]+commit([[:space:]]+[^[:space:]]+)*[[:space:]]+(-a|-am|-as|-asm|-sam|--all)([[:space:]]|$)'; then
		deny "git commit -a stages everything, including another session's work. Stage explicit paths, then commit -s."
	fi
	if printf '%s' "$bare" | grep -qE '^git[[:space:]]+add[[:space:]]' && printf '%s' "$bare" | grep -qE "$secret_path" && ! printf '%s' "$bare" | grep -qE "$secret_doc"; then
		deny "That path looks like a secret. It is never staged; add it to .gitignore instead."
	fi

	# Reading a secret file puts it in the transcript, which is stored and may be shared.
	if printf '%s' "$bare" | grep -qE '^(cat|less|more|head|tail|bat|sed|awk|grep|rg|strings|base64|xxd|hexdump|cut|nl|tee|jq|yq|source|\.|open|code|vim|nano)[[:space:]]' &&
		printf '%s' "$bare" | grep -qE "$secret_path" && ! printf '%s' "$bare" | grep -qE "$secret_doc"; then
		deny "That file holds credentials; printing it puts them in the transcript. Check its presence with ls or test -f, never its contents."
	fi

	if printf '%s' "$bare" | grep -qiE "$fable_assignment"; then
		deny "Fable/Mythos is never the working model in a script, config, agent or flag (~/.claude/CLAUDE.md). Use haiku, sonnet or opus; the Anthropic default in scripts is claude-opus-5."
	fi

	# Each of these is a full agent session billed to the maintainer's subscription -- tens of
	# thousands of tokens of overhead per call before the payload. The count is agreed first.
	if printf '%s' "$bare" | grep -qE '^claude[[:space:]]+.*(-p|--print)([[:space:]]|$)'; then
		ask "claude -p starts a full Claude Code session per call (~78k tokens of cache write each, from the maintainer's subscription). Say how many calls and which model, and wait for a yes."
	fi
	if printf '%s' "$bare" | grep -qE '^(codex[[:space:]]+exec|agy[[:space:]]+.*(-p|--print|--prompt))' && ! printf '%s' "$seg" | grep -q 'text_model.py'; then
		ask "codex/agy calls go through .claude/scripts/text_model.py (one call per batch, mechanical validation). If a direct call is really needed, say how many and why."
	fi

	# Published tags are immutable by repository ruleset; a bad release is fixed by the next one.
	if printf '%s' "$bare" | grep -qE '^git[[:space:]]+tag[[:space:]]+(-d|--delete|-f|--force)([[:space:]]|$)'; then
		deny "Published tags are immutable (RELEASING.md, enforced by the immutable-release-tags ruleset). Fix a bad release with the next patch release."
	fi
	if printf '%s' "$bare" | grep -qE '^git[[:space:]]+push' ; then
		if printf '%s' "$bare" | grep -qE '(--delete[[:space:]]|:refs/tags/)'; then
			deny "Deleting a published ref is refused here. Fix a bad release with the next patch release."
		fi
		if printf '%s' "$bare" | grep -qE '(-f|--force)([[:space:]]|$)' && printf '%s' "$bare" | grep -qE '([[:space:]]main([[:space:]]|$)|refs/tags|[[:space:]]v[0-9])'; then
			deny "main is protected and tags are immutable. Force-push only a feature branch, with --force-with-lease."
		fi
		if printf '%s' "$bare" | grep -qE '(^|[[:space:]])(-f|--force)([[:space:]]|$)'; then
			ask "Force-push rewrites the remote branch. Prefer --force-with-lease; confirm this is a branch nobody else pulled."
		fi
		# A push that lands on main publishes https://assaio.dev/ (Cloudflare builds every push to
		# main, ungated). An open work file means the change is not finished.
		target_main=no
		if printf '%s' "$bare" | grep -qE '([[:space:]]|:)main([[:space:]]|$)'; then
			target_main=yes
		elif ! printf '%s' "$bare" | grep -qE '^git[[:space:]]+push[[:space:]]+[^-][^[:space:]]*[[:space:]]+[^-]'; then
			[ "$(git -C "${CLAUDE_PROJECT_DIR:-.}" symbolic-ref --short HEAD 2>/dev/null)" = "main" ] && target_main=yes
		fi
		if [ "$target_main" = yes ] && [ -n "$(wip_files)" ]; then
			deny "A push to main deploys the site, and docs/work/ still holds an open work file: $(wip_files | xargs -n1 basename | tr '\n' ' '). Finish it with /ship (which deletes the file) or push a branch."
		fi
	fi
	if printf '%s' "$bare" | grep -qE '^gh[[:space:]]+pr[[:space:]]+merge' && [ -n "$(wip_files)" ]; then
		deny "Merging to main deploys the site, and docs/work/ still holds an open work file: $(wip_files | xargs -n1 basename | tr '\n' ' '). Close it with /ship first."
	fi

	# The commit-msg hook is opt-in (make hooks); the dco CI job is not, and rejects these.
	if printf '%s' "$bare" | grep -qE '^git[[:space:]]+commit' && printf '%s' "$bare" | grep -qE '([[:space:]]--no-verify|[[:space:]]-n([[:space:]]|$))'; then
		deny "--no-verify skips the Conventional-Commits and Signed-off-by check. Fix the message instead."
	fi
done <<EOF
$(segments "$cmd")
EOF

if printf '%s' "$cmd" | grep -qiE 'co-authored-by:[^"'"'"']*(claude|anthropic|copilot|codex|gemini|cursor|gpt)|generated with[[:space:]]*\[?(claude|anthropic)|🤖 generated with'; then
	deny "This repo never credits an AI assistant as an author (CONTRIBUTING.md rule 4; the dco CI job rejects the trailer). The human who signs off is the author."
fi

exit 0
