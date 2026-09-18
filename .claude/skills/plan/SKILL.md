---
name: plan
description: Writes the ONE work file for a task — docs/work/<date>-<slug>.md holding spec, plan, journal and handoff together — with tracks a coder can take unread. Use for M and L work, "plan this", "write the spec", "how would we build". For L work it adds alternatives and a challenger review before any code exists.
argument-hint: "<slug> [S|M|L]"
allowed-tools: Read, Grep, Glob, Bash, Write, Edit, Agent
---

Task: $ARGUMENTS

Create `docs/work/<YYYY-MM-DD>-<slug>.md` from `.claude/skills/plan/template.md`. One file per
task; if one exists for this slug, extend it — never a second file. The file is not published
(`internal/docs/guides.go` skips `docs/work/`), and `/ship` deletes it in the shipping commit.

Fill it in this order, reading code for every claim:

1. **Goal and the decision it serves.** For a measurement: which layer (activity / output /
   outcome / impact), what a zero means, what the reader does differently with the figure.
2. **Open questions with a default answer each** — the answer the work proceeds on if nobody
   objects. A question without a default blocks nothing and helps nobody.
3. **Decisions with the alternative considered.** For L work, 2–3 alternatives with their cost,
   and the one chosen with the reason.
4. **Blast radius as a checklist**: every surface the change touches (see
   `.claude/rules/surfaces.md`: changelog, FEATURES row, BACKLOG line, README, site, PRIVACY,
   reference regeneration, ADR, golden files, corpus proof). Each box is a completion criterion.
5. **Tracks**: files, behaviour change, tests, verification command, executor and tier
   (`coder`/opus, this session, `text-broker`), dependencies. Tracks that share a file are one
   track. A track's verification command is something `runner` can execute without judgement.
6. **Handoff**: what is done, what is next, what was decided alone and how to reverse it. Keep
   it current after every track so a fresh session continues from the file alone.

**L work gets a challenger before code**: run the `go-reviewer` (and `honesty-auditor` when a
figure is involved) on the work file itself with the instruction to attack the design — the
denominator, the layer, the migration, the surface it forgets. Fold the findings into the
decisions section, with the ones you rejected and why.

End with the path of the file and the first track to build. Do not start building unless the
task said to (`/build` does that).
