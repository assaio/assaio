---
name: ship
description: Ships finished work the way this repo accepts it — one signed Conventional Commit of explicit paths, the changelog/FEATURES/BACKLOG lifecycle, a PR, the deploy check when the merge would publish the site, the work file closed, one retro line. Use "commit this", "open the PR", "ship it", "merge". Never pushes or merges without a yes.
argument-hint: "[commit|pr|merge]"
allowed-tools: Bash, Read, Edit, Write, Grep, Glob, Agent
---

Stage: $ARGUMENTS (empty = whatever comes next for the current branch).

## Before the commit

- `bash .claude/checks/gate.sh` is green; `/review` ran on this diff.
- Lifecycle for a user-facing change (`.claude/rules/surfaces.md`): `CHANGELOG.md` entry under
  `[Unreleased]` written for a reader with its `B` id; `FEATURES.md` row; `BACKLOG.md` line
  deleted; a corrected figure also gets its `docs/corrections.md` entry. `make docs` output
  committed. `PRIVACY.md` if a parser reads more or less.
- The work file: move its Handoff into the PR description if useful, then delete the file
  (`git rm docs/work/<file>.md`); parked work goes to `docs/work/parked/` with the reason.

## Commit

- Never on `main`: `git switch -c <type>/<slug>` first.
- **Explicit paths only**: `git add <path> <path> …` (the hook denies `.`/`-A`/`-a`); other
  sessions may hold uncommitted work here. `git status --porcelain` afterwards shows only what
  you meant to leave.
- `git commit -s -m "<type>(<scope>): <concrete subject>"` — one logical change, a short body
  only when the why is not obvious, `Signed-off-by` by the human, no AI trailer of any kind.
- One commit per PR: squash your own commits before opening it (`git rebase -i` is
  unavailable here; use `git reset --soft <base>` and recommit).

## Pull request

`gh pr create` with: what changed and why, the linked issue or `B` id, how it was tested (the
gate mode and the corpus proof if a figure moved), documentation impact. CI must be green
(`gh run list --branch <branch>`) before anything else. End the report with one line the next
skill or session can pick up: `PR: #<number> (<url>)`.

## Merge (asks first — merging to `main` publishes https://assaio.dev/)

- The hook denies a merge while `docs/work/` holds an open file.
- `gh pr merge --squash` only after an explicit yes.
- If `site/` changed, verify the deploy:
  `diff <(curl -fsSL https://assaio.dev/) site/index.html && echo "live page matches"`.
  Cloudflare builds every push to `main`, ungated, so this is the only check there is.

## Retro (one minute)

If a missing rule cost a detour this time, add **one line** to the rule or skill that should
have carried it — not a paragraph, not a new file. Say which line you added, or that none was
needed.
