---
description: Run the full local gate (fmt, lint, test, build, docs, and fuzz when a parser changed) and report only what failed
argument-hint: "[full]"
allowed-tools: Bash, Read, Grep, Glob
---

Run this repo's gate and report the result. Argument: $ARGUMENTS (`full` also runs
`make fuzz` and `make vuln` regardless of what changed).

1. `make fmt` — must produce no diff. If it changed files, say which and stop.
2. `make lint`
3. `make test`
4. `CGO_ENABLED=0 go build ./...`
5. `make docs` — then `git status --porcelain docs/reference.json site/reference.html`.
   A diff here means a registry grew and a published surface did not; commit the
   regenerated files with the change that caused them.
6. If `git diff --name-only main...HEAD` (or the working tree) touches
   `internal/parser/`, `internal/reconcile/` or `internal/plugin/`, also run `make fuzz`.
   With `full`, run `make fuzz` and `make vuln` unconditionally.

Report: one line per step with pass/fail, then the failing output verbatim and nothing else.
On a fully green gate say so in one line and name what you ran. Do not fix anything —
this command reports.
