---
description: Self code review before a release — four narrow reviewers over the change (or the whole repo), findings ranked, nothing edited
argument-hint: "[diff|repo|<path>]"
allowed-tools: Bash, Read, Grep, Glob, Agent
---

Review target: $ARGUMENTS (`diff` = uncommitted + unpushed work, the default; `repo` = the
whole codebase; a path = that subtree).

Establish the target first and state it — `git status --porcelain`, `git log --oneline
origin/main..HEAD`, `git diff --stat` — so the reviewers are not guessing what changed.

Then run these agents in parallel, each on the same target:

- **go-reviewer** — correctness and the norms `.golangci.yml` deliberately does not lint.
- **honesty-auditor** — every figure a reader could act on: layer, provenance, confidence,
  scope denominator, error bars, the refusals.
- **surface-auditor** — whether the published prose still describes this binary.
- **store-steward** — only if a migration, a stored field, or a write path is in the target.

**Edit nothing during the review.** A mid-review edit changes the tree the other reviewers are
reading and their findings stop being about the same code. Collect everything first.

Then merge the findings yourself: drop duplicates, drop anything the linter already catches,
and rank by whether a user would act on a wrong result. For each survivor give file:line, the
concrete failure (inputs → wrong output), and the smallest fix. Say explicitly which findings
you could not verify.

End with one line: is this safe to tag, and if not, what is the shortest path to yes.
