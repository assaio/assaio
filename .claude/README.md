# `.claude/` — the harness config for this repo

Checked in, because the rules here are the repo's, not one machine's. `settings.local.json`
stays untracked for personal overrides. `AGENTS.md` says *what* the rules are; this directory
is the part a harness can execute, plus the procedures that repeat here. Each file exists
because something failed without it, or because the same sequence was retyped enough times.

## What loads when

| Layer | Loads | Holds |
|---|---|---|
| `AGENTS.md` (via `CLAUDE.md`) | every session and subagent | what this is, hard rules, honesty, protected contracts, layout — ≤150 lines, checked |
| `rules/*.md` with `paths:` | when a matching file is read | the traps of one area, compressed, pointing at the long form |
| `skills/*/SKILL.md` | on invocation (descriptions always) | procedures: the SDLC and this repo's recurring operations |
| `agents/*.md` | when delegated to (descriptions always) | one job each, with the model and effort it deserves |
| `docs/work/<date>-<slug>.md` | never automatically | one task's spec, plan, journal and handoff |

## Skills

| Skill | Does |
|---|---|
| `/work` | entry: still true? unclaimed? size S/M/L → path → who does what on which model |
| `/plan` | the one work file, from `skills/plan/template.md`; alternatives and a challenger for L |
| `/build` | tracks executed here or by parallel `coder`s on disjoint files; checkpoint per track |
| `/gate` | `checks/gate.sh [quick\|full]`: one line per step, verdicts verbatim, tails only when red |
| `/review` | the four reviewers in fresh contexts, then review → fix → gate → re-review, ≤3 rounds |
| `/ship` | signed commit of explicit paths, the lifecycle files, PR, the deploy check, retro line |
| `/auto` | the chain unattended; most-reversible choice recorded; stops at hard lines; exits clean |
| `/research` | a dated, sourced note for the roadmap with concrete BACKLOG/ROADMAP proposals |
| `/add-source` | new parser or validator with the same-commit surfaces this repo has missed |
| `/debug`, `/refactor`, `/docs` | root cause with this repo's suspects; behaviour-preserving splits; docs simplification |
| `/ui-check`, `/smoke` | the page in a real browser with screenshots; the first-run path on a throwaway store |
| `/content-model` | when and how text goes through `scripts/text_model.py` |
| `/release` | `RELEASING.md` end to end via `release-captain`, stopping at the tag command |

## Agents

| Agent | Model | Holds |
|---|---|---|
| `scout` | haiku | a lookup, ≤15 lines with file:line; `omitClaudeMd` |
| `runner` | haiku | one noisy command → the verdict; never diagnoses |
| `coder` | opus / high | one track of a work file, its files only, never commits |
| `go-reviewer` | opus / high | the norms `.golangci.yml` omits on purpose, and the defects this repo shipped |
| `honesty-auditor` | opus / high | layer, provenance, confidence, denominators, refusals — the v0.12 class of defect |
| `surface-auditor` | opus / high | whether the published *prose* still describes this binary |
| `store-steward` | opus / high | migration immutability; size in bytes on a real corpus |
| `corpus-prover` | sonnet | proof on real logs that the figure that should move, moved — and nothing else |
| `text-broker` | sonnet | one batch through the GPT/Gemini door; never writes the text itself |
| `browser` | sonnet | the dashboard or site in Playwright, screenshots as evidence |
| `release-captain` | opus / high | `RELEASING.md` end to end, stopping before the tag |

Every agent names its model: `inherit` is refused by `checks/setup.py`, because the session
may run a model that is never a working model here.

## Hooks

`hooks/guard.sh` runs on every Bash, Write and Edit. It denies: the deletion subcommand
without a redirected `XDG_DATA_HOME` (the real store is 170 MB of history the sources have
deleted); rewriting or deleting a tag or force-pushing `main`; an AI-authorship trailer;
`--no-verify`; `git add .`/`-A`/`commit -a` (another session's work may be in the tree);
staging or printing a secret file; assigning a Fable/Mythos model id; a push to `main` or a PR
merge while `docs/work/` holds an open file (a merge publishes the site). It asks before
`claude -p`, a direct `codex`/`agy` call, and a force-push. `hooks/guard_test.sh` is its table,
allow, ask and deny alike. `hooks/session_start.sh` prints three lines of state.

## Checks

`checks/gate.sh` is the one gate every skill calls. `checks/setup.py` and
`checks/ops_catalogue.py` keep the harness and `docs/operations.md` honest;
`checks/checks_test.sh` proves both can fail. None of them is a CI job: they hold harness
config, not product, and a harness check in CI would make a contributor without Claude Code
fix a file they never use.

## The content door

`scripts/text_model.py` is the only way prose reaches GPT (`codex exec`, read-only sandbox,
empty working directory) or Gemini (`agy --sandbox`, `gemini-*` only). One call per batch,
mechanical validation, exit 2 when no engine is present — the caller stops rather than writing
the text itself.

## settings.json

Permissions allow the read-only and verify commands, ask for every commit, push, tag, merge
and release, and deny reading secret files. `enabledPlugins` switches off the user-level
plugins that add nothing here (Atlassian, Sentry, Railway, Cloudflare, chrome-devtools, GitHub)
so their skill and tool listings stay out of every session's context; Playwright stays for
`browser`. `attribution` is empty because this project never credits an assistant.
