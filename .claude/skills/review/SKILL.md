---
name: review
description: Reviews a change (or the whole repo, or a path) in fresh contexts with this repo's four reviewers — go-reviewer, honesty-auditor, surface-auditor, store-steward when a write path is touched — then runs the review → fix → gate → re-review loop over the fixes only, at most three rounds. Use "review this", "self review", before /ship and before a tag.
argument-hint: "[diff|repo|<path>] [--no-fix]"
allowed-tools: Bash, Read, Grep, Glob, Edit, Write, Agent
---

Target: $ARGUMENTS (`diff` = uncommitted + unpushed work, the default; `repo` = the whole
codebase; a path = that subtree). `--no-fix` stops after round one with findings only.

## Round one

Establish the target and state it — `git status --porcelain`, `git log --oneline
origin/main..HEAD`, `git diff --stat` — so the reviewers are not guessing what changed.

Run in parallel, each on the same target, each in its own fresh context:

- **go-reviewer** — correctness and the norms `.golangci.yml` deliberately does not lint.
- **honesty-auditor** — every figure a reader could act on: layer, provenance, confidence,
  denominator scope, error bars, the refusals.
- **surface-auditor** — whether the published prose still describes this binary.
- **store-steward** — only if a migration, a stored field or a write path is in the target.

**Edit nothing while they run.** A mid-review edit changes the tree under the other reviewers.

Merge the findings: drop duplicates and anything the linter already catches; rank by whether
a user would act on a wrong result. For each survivor: file:line, the concrete failure (input
→ wrong output), the smallest fix. Say which findings you could not verify.

## Rounds two and three (unless `--no-fix`)

Fix the confirmed findings, run `bash .claude/checks/gate.sh`, then send **only the fixed
hunks** (`git diff` of those files) back to the reviewer that raised each finding, asking
whether it is closed. A finding still open after round three goes into the work file's
Handoff as a known gap, not into a fourth round.

End with one line: safe to ship / tag, or the shortest path to yes.
