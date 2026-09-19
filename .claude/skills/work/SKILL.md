---
name: work
description: Entry point for any task in this repo — "let's do X", "pick up B123", "fix", "add", "change". Checks the ask is still true and unclaimed, sizes it S/M/L, picks the lightest safe path (direct edit, /plan, /research) and says who does what on which model. Use before touching code; skip only for a one-line answer.
argument-hint: "[task or B-id]"
allowed-tools: Read, Grep, Glob, Bash, Agent
---

Task: $ARGUMENTS

## 1. Is it still true, and does anyone hold it?

- If it names a `B` id: the line exists in `BACKLOG.md` (a deleted line means it shipped —
  check `CHANGELOG.md` and `FEATURES.md` and stop cleanly).
- `git log --oneline -15 -- <paths it touches>` and `gh pr list --search "<keywords>"`: work that
  already landed or is open in a PR is continued, not duplicated. A GitHub issue is claimed
  before work starts (`gh issue edit <n> --add-assignee @me`) and released if the work parks.
- `docs/work/`: an open work file on this topic means resume it from its **Handoff** section.
- The premise: read the code the ask describes. If the described behaviour is not what the code
  does, say so before planning anything.

## 2. Size it

| Size | Shape | Path |
|---|---|---|
| S | one file, no schema/protocol/surface change, verification obvious | do it here, then `/gate` and `/ship` |
| M | 2–5 files, a test change, maybe a doc | `/plan` with one track per file group, then `/build` |
| L | new capability, new source, new metric, migration, protocol, a number a reader sees | `/plan` with alternatives and a challenger; `/research` first if the roadmap does not already justify it |

Anything that changes a stored field or a migration is L regardless of line count
(`.claude/rules/store.md`). Anything that changes a figure a reader sees is at least M and ends
with `corpus-prover`.

## 3. Who does what

Write the table before starting; delegate only what the table says.

| Work | Executor |
|---|---|
| finding where things live, reading a log | `scout` (haiku) — never for what one Grep answers |
| a long or noisy command | `runner` (haiku) |
| one track of a work file, disjoint files | `coder` (opus) — parallel only on disjoint file sets |
| review | `/review` (the four reviewers, fresh context) |
| text a reader sees, and verdicts about it | `text-broker` via `/content-model` |
| proof on the real corpus | `corpus-prover` |
| the dashboard or site in a browser | `browser` via `/ui-check` |
| design, cross-cutting code, integration | this session |

Rules of delegation: every agent gets exact files, one question and the shape of the answer;
an agent's report is evidence to check, not a fact; agents never spawn agents; parallel only
on disjoint files; when in doubt, a worktree (`EnterWorktree`) rather than a shared tree.

## 4. Say it, then go

One paragraph: what is true now, the size, the path, the table. Then start the path — do
not wait for a confirmation the ask already gave.
