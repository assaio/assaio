# AGENTS.md — assaio

Canonical guidance for AI assistants (and humans) working in this repo, following the
[AGENTS.md](https://agents.md) convention. It overrides default assumptions. Keep it
accurate. Claude Code reads it via the `@AGENTS.md` import in `CLAUDE.md`.

Roadmap: `ROADMAP.md`
Architecture decisions: `docs/adr/`
Contributor rules (authoritative, shared with the community): `CONTRIBUTING.md`

## What this is

`assaio` measures how an organization uses AI coding tools and whether it is worth it:
cost/tokens, how much AI-written code reaches production, quality/bug impact, DevEx.
This repository ships one binary, `assaio-agent`: an offline-first CLI (Go, embedded
SQLite) that reads the local session logs of Claude Code, Codex CLI, Gemini CLI, and
Cline and turns them into reports (`report`, `effectiveness`), diagnostics (`analyze`,
`check`, `doctor`, `status`), and the self-contained Assay HTML dashboard. Out-of-tree
exec plugins extend it in any language — parsers via `plugins:` (ADR 0003), metrics via
`metrics:` (ADR 0004). A team-server MVP (`serve` + `sync`) pools a team's usage on
self-hosted infrastructure; the deeper org-analytics server (git/issue-tracker
correlation for survival/bug/quality) is future roadmap — see `ROADMAP.md`.

## Code philosophy (non-negotiable)

Write as a senior engineer would: minimal, SOLID, clean. The bar is high.

**Always:**
- One file, one responsibility. Keep files short — if a file grows past ~200 lines or
  starts doing two things, split it.
- One metric = one file in `internal/analyze/`. One data source = one package in
  `internal/parser/`. Out-of-tree, both are exec plugins (`docs/extending.md`). The
  planned in-process `plugin/metric/`, `plugin/rule/`, `plugin/connector/` tree keeps
  the same one-unit-one-file law when it lands (`ROADMAP.md`).
- Self-explaining code: intention-revealing names, small functions, obvious control
  flow. The code is the documentation.
- Program to interfaces; keep dependencies pointing inward (`internal/` core never
  imports `plugin/` or `ee/`).
- Every validator (metric/rule) is independently readable and testable in isolation.
- Table-driven tests; golden files for parsers (real captured samples).

**Never:**
- Long or narrative comments. Comment only to state a constraint the code cannot show,
  and keep it to one line. No "what the next line does", no "why my change is correct",
  no attribution notes.
- Speculative abstraction or config for a need that does not exist yet (YAGNI).
- Dead code, commented-out code, or TODO dumps left in place.
- Reach into another plugin's internals. Plugins talk to the core only.
- Blend session-level (hook) data with daily vendor-aggregate data without tagging
  provenance and confidence.

## Code standards (enforced + human-reviewed)

The bar is split in two: what tooling can check, it checks and gates on; the rest is a
review norm, not a lint rule. Do not turn the review norms into lint gates — that trade
is deliberate (see `docs/adr/0002-code-standards-and-enforcement.md`).

**Enforced by tooling** (run these before you push):
- Formatting — `make fmt` (`golangci-lint fmt`: gofmt + gofumpt + goimports with the
  `github.com/assaio/assaio` local prefix). Must produce no diff.
- Lint — `make lint` (`gofmt -l`, `go vet`, `golangci-lint run`). The linter canon lives
  in `.golangci.yml` (source of truth): the standard set plus bodyclose, errorlint,
  gosec, misspell, revive (exported, context-as-argument, error-strings,
  indent-error-flow, superfluous-else, var-naming), unconvert, unparam, gocritic
  (diagnostic + performance), noctx, copyloopvar, intrange, usestdlibvars, perfsprint,
  nolintlint (explanation + specific required), and depguard (`internal/` must not import
  `plugin/` or `ee/`).
- Tests — `make test` (`go test ./...`; CI adds `-race` and a coverage profile).
  Stdlib-only (no testify), table-driven, golden files regenerated with `-update`.
- Fuzzing — `make fuzz` for any parser change. Every parser ships a native `FuzzParse`
  (Cline: `FuzzParseTask`) with a seed corpus under `testdata/`.
- Vulnerabilities — `make vuln` (`govulncheck`); also a CI job.
- Hooks — `make hooks` installs the opt-in lefthook hooks (pre-commit: gofumpt on staged
  files + `go vet` + lint of new issues; commit-msg: Conventional Commits + `Signed-off-by`).

**Human-reviewed norms** (not lint gates — reviewers hold this line):
- File size (~200 lines) and single-responsibility: split a file that grows past the
  budget or starts doing two things. Deliberately no `funlen`/`lll`.
- Cyclomatic and cognitive complexity: kept low by review, not by `gocyclo`/`gocognit`.
- Self-explaining code and one-line constraint-only comments (see philosophy above);
  deliberately no `goconst`.
- Coverage is reported in CI, never gated on a number (Goodhart). Reviewers look at the
  uncovered branches, not the percentage; weak packages get attention in review.
- Parser contract: single-root `Discover(root string)`, skip-and-count corrupt-line
  policy, the shared `internal/parser` scanner (`MaxLineBytes`) and `NonNeg`, and the
  `FuzzParse` requirement are documented in `docs/extending.md` — read it before adding
  a parser.
- Exec-plugin protocol: out-of-tree parsers are subprocesses speaking handshake + JSONL
  over stdout, opt-in via config only, validated at the boundary, stored as
  `plugin:<name>` — see `docs/extending.md` and ADR 0003.

## Honesty rules (product-critical)

- Every domain fact carries its provenance and confidence.
- Attribution and effectiveness claims ship with their known error bars. Prefer a
  signal labeled "directional" over a precise-looking number that is wrong.
- Bug-density on AI lines is only ever compared against **age-matched** human code.
- Never present individual metrics as performance-evaluation inputs. Pseudonymized is
  the default privacy mode.

## Git & commits (keep the tree clean)

- **Do not commit automatically.** Commits are made **by hand after a big milestone**,
  to keep the history clean. Do the work; leave committing to the maintainer unless
  explicitly asked.
- **Conventional Commits**, concrete subjects: `feat(agent): parse Codex rollout logs`,
  not `wip` or `updates`. Body explains *why* when it is not obvious.
- **One commit per PR.** The author squashes their branch before merge; multi-commit
  PRs are not accepted. `main` is protected; PR + green CI + ≥1 review required.
- Sign every commit with DCO (`Signed-off-by`).

## Layout

```
cmd/assaio-agent/        CLI entrypoint (report, analyze, dashboard, serve, sync, …)
internal/analyze/        one-file-per-metric validator framework behind assaio analyze
internal/cli/            command wiring and flag handling; one file per command
internal/config/         defaults + YAML file + ASSAIO_-prefixed env vars
internal/dashboard/      builds + renders the offline Assay HTML dashboard
internal/ingest/         discovers session files, parses them, upserts into the store
internal/parser/         shared scanner + NonNeg; one package per tool below
internal/parser/claude/  parses Claude Code session logs into usage records
internal/parser/cline/   parses Cline task directories into usage records
internal/parser/codex/   parses Codex CLI rollout logs into usage records
internal/parser/gemini/  parses Gemini CLI chat logs into usage records
internal/paths/          resolves data/config/tool-log filesystem locations
internal/plugin/         runs out-of-tree exec plugins (parser + metric protocols),
                         validating everything at the boundary
internal/pricing/        loads the vendored LiteLLM price table, prices usage records
internal/projectid/      resolves a session's cwd to its git repository root + subpath
internal/report/         aggregates stored usage into priced rows; renders table/JSON/CSV
internal/server/         self-hosted team server: usage collection + served dashboard
internal/store/          embedded SQLite persistence for usage records
internal/usage/          normalized representation of AI-tool usage events
internal/version/        build-time version metadata
docs/adr/                Architecture Decision Records
```

Future stages (the deeper org-analytics server, the in-process `plugin/` architecture,
a richer web UI) are described in ROADMAP.md.
