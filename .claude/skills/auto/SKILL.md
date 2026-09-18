---
name: auto
description: Autonomous, non-interactive run of the whole chain — work → plan → build → gate → review → ship up to the PR — for a task or B-id, without questions. Takes the most reversible decision when unsure and records it in the work file as "decided alone … reverse by …". Stops only at hard lines (merge, tag, push to main, money, protected contracts). Exits cleanly when nothing is left to do.
argument-hint: "<task or B-id>"
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
disable-model-invocation: true
---

Task: $ARGUMENTS

Run `/work`, then `/plan`, `/build`, `/gate full`, `/review`, `/ship pr` — in that order,
without asking anything. The rules of each skill apply unchanged; this one only removes the
pauses between them.

**When unsure, take the most reversible option** and write it into the work file's Decisions
as `decided alone: … — reverse by …`. Reversible means: additive over destructive, a new file
over an edited shipped one, a `—` over a guessed number, a caveat over a claim, a smaller
scope over a wider one.

**Hard lines — stop, write the Handoff, report:**

- merging, tagging, pushing to `main`, anything that publishes (site, release, Homebrew tap);
- editing a shipped migration, a plugin protocol, the sync protocol, or any contract in
  `docs/compatibility.md`;
- more than 5 calls through the GPT/Gemini door, or any `claude -p`;
- a threshold or figure nobody supplied a source for;
- writing to the maintainer's real store;
- a red gate you cannot make green without widening the task.

**Exit cleanly when there is nothing to do**: the `B` id is already deleted from `BACKLOG.md`,
an open PR already covers the task, or the described behaviour is already the code's. Say so
in three lines and stop — do not manufacture work.

**Continue rather than duplicate**: an open work file on the topic is resumed from its
Handoff; an open branch on the topic is checked out, not re-created.

Every 3 tracks, checkpoint: update the Handoff, run the full gate. The final report is the
`/ship` report plus the list of decisions taken alone.
