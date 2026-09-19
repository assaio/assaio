---
name: build
description: Executes the tracks of a work file (docs/work/*.md) — in this session or through parallel coder agents when file sets are disjoint — checking every return with a diff and a re-run, a checkpoint per track and a full gate every third. Use after /plan, "build it", "implement the tracks", "continue the work file".
argument-hint: "[work file] [track numbers]"
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

Work file: $ARGUMENTS (default: the only open file in `docs/work/`; two open files means ask
which).

Read the Handoff section first and continue from it — never restart a track marked done.

For each track, in dependency order:

1. **Executor per the table.** Disjoint file sets may run as parallel `coder` agents, each with
   the track text, the decisions section and nothing else. Overlapping files run in this
   session, sequentially. If a coder would need a file outside its track, it reports and stops;
   you re-cut the tracks rather than let two writers share a file. Risky overlap goes to a
   worktree (`EnterWorktree`), not to hope.
2. **Check the return, do not trust it**: `git diff --stat` shows only the track's files; read
   the diff; run the track's verification command yourself (or through `runner`) and copy the
   verdict into the Journal. A coder's "tests pass" is a claim until the command ran here.
3. **Checkpoint**: update the Handoff (done / next / decided alone) before starting the next
   track. Every third track, `bash .claude/checks/gate.sh` in full and fix red before continuing.
4. **A figure moved?** The track is not done until `corpus-prover` showed it moved on the real
   corpus by the expected amount and the untouched figures stayed put.

Stop conditions: a red gate you cannot explain, a decision the work file leaves open with no
default, a change that would touch a shipped migration or a protocol. Write the state to the
Handoff and say what is needed.

When every track is done: `bash .claude/checks/gate.sh full`, then `/review`.
