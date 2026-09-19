---
name: refactor
description: Behaviour-preserving restructuring under this repo's norms — split a file past ~200 lines or doing two things, remove dead code with evidence, untangle a dependency pointing outward — with golden files and the corpus as the proof nothing moved. Use "refactor", "split this file", "this is dead code", "simplify". Deleting code is proposed with evidence, not done unasked.
argument-hint: "<path or package>"
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

Target: $ARGUMENTS

## Evidence first

- `wc -l` on the target; the two responsibilities named when claiming a split.
- Dead code: `grep -rn` for every symbol, plus `go vet` and `golangci-lint run` (`unused`,
  `unparam`); a symbol only tests reference is dead in the product. List candidates with the
  evidence — **deleting them is a separate decision the maintainer takes**, unless the task said
  to delete.
- `git log --oneline -10 -- <path>` to know why the shape is what it is before changing it.

## Rules

- Behaviour identical: `make test` green before and after; golden files unchanged (a golden
  diff means this was not a refactor); for anything on the figure path, `corpus-prover` A/B on
  a pinned window shows identical output.
- One file, one responsibility, ~200 lines; small functions; names that say what.
- Dependencies point inward; `internal/` never imports `plugin/` or `ee/` (depguard).
- No new abstraction for a need that does not exist; no TODO left behind; comments kept only
  where they state a constraint the code cannot show.
- `make docs` afterwards: a moved command or flag changes the generated reference.

Finish with `bash .claude/checks/gate.sh`, then `/review diff` with the go-reviewer's file-size
and single-responsibility judgement stated explicitly.
