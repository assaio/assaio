---
name: gate
description: Runs this repo's local gate quietly — fmt, lint, test, build, docs regeneration, the harness checks, fuzz when a parser changed, vuln with full — one line per step, the tool's verdict verbatim, log tail only when red. Use before a commit, after a build track, "run the gate", "is it green", "check everything".
argument-hint: "[quick|full]"
allowed-tools: Bash, Read
---

Run `bash .claude/checks/gate.sh $ARGUMENTS` (empty means `quick`; `full` adds `make fuzz` and
`make vuln` unconditionally).

Report the script's output as it is: one line per step, then only the failing tails it printed.
On green, one line naming the mode and the steps that ran. On red, the failing step, its verdict
lines and the log path — nothing else, and **fix nothing here**: this skill reports.

What the steps mean when they fail:

- `fmt` — `golangci-lint fmt --diff` printed a diff: run `make fmt`, then re-run.
- `docs` — `make docs` changed `docs/reference.json`, `site/reference.html` or `site/docs/`: a
  registry grew and the published surface did not; commit the regenerated files with the change.
- `harness` — `.claude/checks/setup.py`, `ops_catalogue.py` or `guard_test.sh` is red: the
  harness itself drifted (`.claude/rules/harness.md`).
- `fuzz` — a crasher was written under `testdata/fuzz/`; it is the reproduction, keep it.

Scope during iteration is the track's own `go test` command; the gate is for before a commit.
