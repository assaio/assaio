# 2. Code standards and enforcement

## Status
Accepted (2026-07-13)

## Context
A repo whose selling point is honest measurement has to be readable and trustworthy
itself. We wanted a high, consistent bar without spending review time on things a machine
can decide, and without gaming the parts a machine decides badly.

## Decision
Split the bar in two: enforce what tooling checks well, keep the rest a review norm.

- **Linter canon** in `.golangci.yml` (source of truth): standard set plus bodyclose,
  errorlint, gosec, misspell, revive, unconvert, unparam, gocritic (diagnostic +
  performance), noctx, copyloopvar, intrange, usestdlibvars, perfsprint, nolintlint, and
  depguard (`internal/` must not import `plugin/` or `ee/`). Formatting is gofumpt +
  goimports via `golangci-lint fmt`. **Deliberately absent:** gocyclo, gocognit, funlen,
  lll, wsl, goconst — complexity and file size (~200 lines) are human-reviewed norms, not
  gates.
- **Testing:** stdlib-only (no testify), table-driven, golden files with `-update`. Every
  parser ships a native `FuzzParse` with a seed corpus (invariants: no panic, no negative
  tokens, non-empty dedupe key). CI runs `-race`.
- **Coverage** is reported, never gated on a number — a percentage target is a Goodhart
  trap; reviewers read uncovered branches instead.
- **Hooks** are opt-in lefthook (`make hooks`), never auto-installed: pre-commit runs
  gofumpt + vet + lint of new issues, commit-msg checks Conventional Commits + DCO.
- **Supply chain:** CI actions are SHA-pinned, workflows use minimal permissions,
  govulncheck runs on every push, dependabot tracks gomod + actions, OpenSSF Scorecard
  runs weekly, and releases carry build-provenance attestations.
- **Agent guidance** is canonical in `AGENTS.md` (agents.md convention); `CLAUDE.md` is a
  one-line `@AGENTS.md` import so Claude Code and other tools read one source.

## Consequences
Contributors run `make fmt lint test` (plus `fuzz` for parser work) and get the same
answer CI does; reviewers spend their attention on complexity, coverage gaps, and the
honesty rules rather than on formatting. The deliberate linter exclusions mean file-size
and complexity drift can only be caught in review — an accepted cost. One canonical
standards document (`AGENTS.md`) keeps human and AI contributors on identical rules.
