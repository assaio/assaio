# <title>

Work file for one task: spec, plan, journal and handoff in one place. Deleted by `/ship`.
Parked work moves to `docs/work/parked/` with the Handoff section explaining what remains.

## Goal

What changes for a reader or a contributor, and which decision it serves. For a measurement:
layer, what a zero means, what the reader does differently.

## Open questions (with the default the work proceeds on)

- Q: … — default: …

## Decisions

- D1: … — alternatives considered: …; chosen because …
- decided alone: … — reverse by …

## Blast radius (each box is a completion criterion)

- [ ] tests (table-driven; golden `-update` read and accepted)
- [ ] `CHANGELOG.md` `[Unreleased]` entry
- [ ] `FEATURES.md` row / `BACKLOG.md` line deleted
- [ ] `README.md` / `site/index.html` / `site/llms.txt` sentences still true
- [ ] `PRIVACY.md` (new directory, field, or a source that reads less)
- [ ] `make docs` regenerated and committed
- [ ] ADR (a commitment a future contributor could undo)
- [ ] corpus proof (`corpus-prover`) for any figure that moved
- [ ] `/ui-check` for any dashboard or site change

## Tracks

| # | Files | Behaviour change | Tests | Verify | Executor | Depends on |
|---|---|---|---|---|---|---|
| 1 | | | | `go test ./internal/... -run X` | coder | — |

## Journal

- <date>: …

## Handoff

- Done: …
- Next: …
- Decided alone: … — reverse by …
- Verification state: gate quick/full last run at …, result …
