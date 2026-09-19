---
name: research
description: Research for the roadmap — a market or evidence question, a competitor's surface, a milestone's exit criterion, whether a backlog pool is still justified — producing a dated, sourced note under docs/work/ and concrete proposals (ROADMAP evidence rows, BACKLOG items with ids, gates). Use "research", "what does the evidence say", "should we build", "is this still worth it", "roadmap".
argument-hint: "<question or milestone>"
allowed-tools: Read, Grep, Glob, Bash, Write, WebSearch, WebFetch, Agent
---

Question: $ARGUMENTS

`ROADMAP.md` is a direction with exit criteria and a sourced evidence table; `BACKLOG.md` is the
pool behind it, with pools already born from research ("Pool — from the 2026-08 needs
research"). This skill produces the next such input, never a rewrite of either file.

## 1. Frame it

State the decision the answer would change (keep / expand / revise / stop which workstream),
the milestone it belongs to, and what would count as evidence against your own expectation.

## 2. Gather, with sources

- **External**: vendor docs, published studies, competitor surfaces — each with URL, date, who
  published it and their interest in the result. Vendor research is directional, never causal
  (the roadmap's own rule).
- **Internal**: what the code already does (`scout`), what the corpus already answers
  (`corpus-prover` or `signals coverage` on a throwaway store), what `BACKLOG.md` already holds
  on the topic (ids, not paraphrase), what `docs/corrections.md` says went wrong nearby.
- **Users**: issues and PRs (`gh issue list --search`), `SUPPORT.md` routes, pilot feedback the
  maintainer names.

## 3. Write the note

`docs/work/<date>-research-<slug>.md`: question → decision → evidence table (claim, source,
date, weight) → what it changes for the product → proposals. Proposals are concrete: a new
ROADMAP evidence row with its footnote, a BACKLOG item in the pool's shape (next free `B` id,
effort S/M/L, scope solo/team/both, the honesty caveat), a gate to add or drop. Unknowns are
listed as unknowns; "no evidence either way" is a finding.

## 4. Challenge it

Before handing it over, have a fresh context attack it: a `go-reviewer`-style reviewer for the
engineering claims, and — for a note that will shape what the product says about itself — a
second model family through `/content-model --engine both` asking for the strongest case
against each proposal. Record the disagreements in the note.

End with the note's path and the proposals in five lines. Editing `ROADMAP.md` or `BACKLOG.md`
is the maintainer's call (or `/ship`'s, once accepted).
