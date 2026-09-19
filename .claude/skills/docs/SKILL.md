---
name: docs
description: Improves and simplifies this repo's documentation without losing a fact — finds contradictions between surfaces, duplicated facts, sentences no longer true of the binary, pages nobody links, and prose a reader cannot act on; proposes the smaller true version through the second-family door. Use "simplify the docs", "is the README still true", "docs review", "too long", "where should this go".
argument-hint: "[path or surface]"
allowed-tools: Read, Grep, Glob, Bash, Edit, Write, Agent
---

Target: $ARGUMENTS (empty = the published set: `README.md`, `docs/README.md`, `FEATURES.md`,
`ROADMAP.md`, `docs/*.md`, `site/llms.txt`, `PRIVACY.md`).

## Where a fact belongs (one fact, one place)

| Kind of fact | Home | Everything else |
|---|---|---|
| what the binary can do, counts, flags | `docs/reference.json` via `make docs`; `data-claim` spans | links, never restates |
| what exists and since when | `FEATURES.md` | — |
| what shipped per release | `CHANGELOG.md` | — |
| what might be built | `BACKLOG.md` (ids) under `ROADMAP.md` (direction) | — |
| a figure that was wrong | `docs/corrections.md` | the changelog line links it |
| how to extend | `docs/extending.md` + one page per surface | — |
| contributor rules | `CONTRIBUTING.md` | `AGENTS.md` compresses, never contradicts |
| maintainer operations | `RELEASING.md`, `docs/site.md`, `docs/operations.md` | — |

## The pass

1. **Mechanical first**: `make docs && make test` (published-or-excused, links, claims, ADR
   index, flags), then `surface-auditor` for the judgement half — the stale sentence, the
   closed caveat still printed, the roadmap wording about something that shipped.
2. **Duplicates and contradictions**: for each fact stated twice, name the home and cut the
   copy to a link. A sentence true in one file and false in another is a defect, not style.
3. **Readability**: sentences a reader can act on, one idea each; the narrative behind a rule
   stays (it is why a reader trusts a figure) but lives once, where the rule lives.
4. **The second family reads it**: through `/content-model --engine both`, per batch of pages,
   with a schema of `keep / cut / rewrite` per paragraph id and the reason. A cut counts only
   where both agree; rewrites are proposals the maintainer reads, quoted next to the original.

## What never changes here

Generated files; `CHANGELOG.md` history; ADR bodies (amend with a status note); the refusals;
any sentence a `data-claim` span or a test checks, unless the test changes with it.

End with a table: file → sentence → problem → proposed text, and the list of cuts both engines
agreed on. Apply only what the task asked to apply; the rest is a proposal.
