# Contributing to assaio

Thanks for helping build assaio. This document is the authoritative set of rules for
contributions — code style, git workflow, and pull requests. It is shared by humans and
by AI assistants working in this repo (see `AGENTS.md`, which defers to this file).

Get oriented first: `README.md` for what assaio does, `AGENTS.md` for the code
philosophy and layout, the Architecture Decision Records under `docs/adr/`, and
`ROADMAP.md` + `BACKLOG.md` for direction and open work.

---

## Principles

assaio is written the way a senior engineer writes: **minimal, SOLID, clean**. We
optimize for code that the next person can read and change with confidence. When in
doubt, choose the smaller, clearer option.

We also make a product promise — **measure value, not people; honest statistics or
nothing** — and that promise constrains code too (see "Honesty rules" below).

---

## Code style

**Small and focused.**
- One file, one responsibility. If a file passes ~200 lines or starts doing two things,
  split it.
- One metric = one file in `internal/analyze/`. One data source = one package in
  `internal/parser/`. Out-of-tree, both are exec plugins declared in config
  ([`docs/extending.md`](docs/extending.md)). The planned in-process `plugin/` tree
  (`plugin/metric/`, `plugin/rule/`, `plugin/connector/` — see `ROADMAP.md`) keeps the
  same one-unit-one-file law when it lands.
- Small functions, intention-revealing names, obvious control flow.

**Self-explaining, not commented.**
- The code is the documentation. Names and structure carry the meaning.
- Comment **only** to state a constraint the code cannot express, and keep it to a
  single line. Do not write comments that narrate what the code does, justify a change,
  or record where code came from. Such comments are noise the moment the PR merges.

**SOLID and layered.**
- Program to interfaces. Dependencies point inward: the `internal/` core never imports
  `plugin/` or `ee/`. Plugins depend on the core, never on each other.
- No speculative abstraction. Build for the requirement in front of you (YAGNI). No
  dead code, commented-out blocks, or TODO dumps.

**Tested.**
- Table-driven tests. Golden files (`testdata/`) for log parsers, using real captured
  samples — vendor formats change, and golden files catch it.
- Every metric and rule must be testable in isolation.

### Code standards

What tooling can check, it checks. `.golangci.yml` is the source of truth for the linter
canon: the standard set plus bodyclose, errorlint, gosec, misspell, revive, unconvert,
unparam, gocritic (diagnostic + performance), noctx, copyloopvar, intrange, usestdlibvars,
perfsprint, nolintlint, and depguard (`internal/` must not import `plugin/` or `ee/`).
Formatting is gofumpt (a stricter gofmt) plus goimports, run through `golangci-lint fmt`.

Complexity and file size (~200 lines) are review norms, not lint gates: we omit
`gocyclo`/`gocognit`/`funlen`/`lll`/`goconst` on purpose so reviewers, not a threshold,
hold that line. Coverage is reported in CI; there is no hard gate — reviewers look at the
uncovered branches, not the number.

Every new parser must ship a native `FuzzParse` (invariants: no panic, no negative token
counts, non-empty dedupe key) with a seed corpus; see [`docs/extending.md`](docs/extending.md).

**Honesty rules (product-critical).**
- Every domain fact carries provenance and confidence; never blend session-level hook
  data with daily vendor aggregates without tagging both.
- Attribution and effectiveness results ship with their error bars. A directional
  signal must be labeled directional.
- AI-line bug density is only compared against age-matched human code.
- Individual metrics are never presented as performance-evaluation inputs;
  pseudonymized is the default privacy mode.

---

## Git workflow

We keep the history **clean and linear**. That imposes two rules with no exceptions.

**1. One commit per pull request.**
Your PR must contain exactly one commit. Squash your work before requesting review:

```bash
git rebase -i main      # squash your commits into one
git push --force-with-lease
```

Multi-commit PRs are not accepted. This keeps `main` a readable sequence of meaningful
changes, one per merged PR.

**2. Conventional Commits with concrete subjects.**

```
<type>(<scope>): <imperative summary>

<why this change exists, when not obvious>

Signed-off-by: Your Name <you@example.com>
```

- Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`, `ci`.
- Scope is the area touched: `agent`, `server`, `connector/github`, `metric/churn`, …
- Subjects say what changed: `feat(agent): parse Codex rollout logs`. Never `wip`,
  `updates`, `fixes`, or `misc`.
- The commit message drives the changelog and versioning — write it for a reader.

**3. Sign off (DCO).**
Every commit needs `Signed-off-by:` (`git commit -s`). By signing you certify the
[Developer Certificate of Origin](https://developercertificate.org/). We use DCO, not a
CLA.

**4. `main` is protected.**
A PR and green CI are required for every change. A contributor PR also needs an
approving review from a maintainer. (While there is a single maintainer, they self-merge
their own changes after green CI — GitHub does not allow self-approval — and the review
requirement applies to everyone else and takes full effect once there is a second
maintainer.)

---

## Before you open a PR

Run the full local gate:

```bash
make fmt              # gofumpt + goimports; must produce no diff
make lint             # gofmt -l, go vet, golangci-lint run
make test             # go test ./... (CI adds -race)
make fuzz             # only when you touch a parser
make vuln             # govulncheck; optional locally, gated in CI
```

Coverage is reported in CI; there is no hard gate — reviewers look at the uncovered
branches, not the number.

Hooks are optional but recommended: `make hooks` installs the lefthook pre-commit hook
(gofumpt on staged files + `go vet` + lint of new issues) and commit-msg hook
(Conventional Commits + `Signed-off-by`), so the gate runs before you push.

Your PR description must include: what changed and why, the linked issue, how it was
tested, and any documentation impact. The PR template will prompt for these.

For a user-facing change, documentation impact concretely means: an entry in
`CHANGELOG.md` under `[Unreleased]`, and a new or updated row in
[`FEATURES.md`](FEATURES.md) — the maintained inventory of what exists and since which
release. If the work came from [`BACKLOG.md`](BACKLOG.md), delete its line there in the
same PR (that file's header describes the full lifecycle).

---

## Adding a connector, metric, or rule

These are the common contributions and the architecture is built for them. The
authoritative walkthroughs live in [`docs/extending.md`](docs/extending.md); this is the
map. Looking for something valuable to pick up? [`BACKLOG.md`](BACKLOG.md) is the
maintained pool of candidate items — comment on or open an issue before starting a
larger one.

- **Connector** (new data source): today, one Go package under
  `internal/parser/<name>/` with golden and fuzz tests, wired into ingest and doctor —
  see [Add a data source](docs/extending.md#add-a-data-source), and open a
  [connector issue](.github/ISSUE_TEMPLATE/connector.yml) first. A tool only your
  organization uses is usually better served by an out-of-tree
  [exec plugin](docs/extending.md#write-a-plugin-any-language) (any language, no PR).
- **Metric**: today, one file under `internal/analyze/<name>.go` implementing
  `Validator`, self-registered from its own `init()` — it appears in `analyze` and the
  dashboard automatically; see
  [Adding a metric validator](docs/extending.md#adding-a-metric-validator). A
  company-specific metric can instead ship as an out-of-tree
  [metric plugin](docs/extending.md#write-a-metric-plugin-any-language), no fork needed.
- **Rule** (alerting/policy units): not a shipped surface yet. The planned in-process
  `plugin/rule/` tree is roadmap (`ROADMAP.md`); if you need one today, open an issue
  describing the signal so the interface is shaped by real cases.

If your change alters the data schema or a plugin protocol, follow the
versioning/compatibility policy (semver + deprecation window), call it out in the PR,
and remember release notes must flag it (see `RELEASING.md`).

---

## Reporting security issues

Do not open a public issue for vulnerabilities. See `SECURITY.md`.

---

## Code of conduct

Participation is governed by `CODE_OF_CONDUCT.md`.
