# Documentation map

Where everything lives, in one place.

## Using assaio

- [`README.md`](../README.md) — install, quick start, the command table, supported
  tools and accuracy caveats.
- [`FEATURES.md`](../FEATURES.md) — the maintained inventory of what exists today,
  with the release each capability arrived in.
- [`extending.md`](extending.md) — the contract for every extension surface: in-tree
  parsers and validators, exec parser plugins (any language), exec metric plugins
  (any language), custom log paths, the team server, and direct SQL on the store.
- [`config.example.yaml`](../config.example.yaml) — every config key, documented, with
  defaults.
- [`PRIVACY.md`](../PRIVACY.md) — exactly what is read, extracted, stored, and never
  touched.

## Project direction

- [`ROADMAP.md`](../ROADMAP.md) — the narrative direction: what shipped in v0.1 and
  the themes being explored next. Conceptual, not contractual.
- [`BACKLOG.md`](../BACKLOG.md) — the ranked pool of concrete candidate items
  (`B01`–…): proposals and effort estimates, not commitments.
- [`CHANGELOG.md`](../CHANGELOG.md) — what actually shipped, per release
  (Keep a Changelog format).

Lifecycle: an item graduates BACKLOG → CHANGELOG `[Unreleased]` → a row in FEATURES,
all in the shipping PR.

## Contributing and policies

- [`CONTRIBUTING.md`](../CONTRIBUTING.md) — the authoritative rules: code style, the
  local gate, git workflow, honesty rules.
- [`GOVERNANCE.md`](../GOVERNANCE.md) — how decisions are made.
- [`CODE_OF_CONDUCT.md`](../CODE_OF_CONDUCT.md) — Contributor Covenant 2.1.
- [`SECURITY.md`](../SECURITY.md) — private vulnerability disclosure.
- [`RELEASING.md`](../RELEASING.md) — maintainers only: versioning, the immutable-
  migrations rule, cutting a release.
- [`format-resilience.md`](format-resilience.md) — how vendor log-format drift is
  detected and fixed: current defenses, known gaps, and the report → fixture → patch
  release loop (`format-drift` label).

## Architecture Decision Records

- [ADR 0001](adr/0001-language-and-architecture.md) — Go, one static binary, the
  `internal/` core and dependency direction.
- [ADR 0002](adr/0002-code-standards-and-enforcement.md) — code standards: what
  tooling gates vs what review holds.
- [ADR 0003](adr/0003-exec-plugin-protocol.md) — exec plugin protocol for out-of-tree
  **parsers** (subprocess, handshake + JSONL).
- [ADR 0004](adr/0004-exec-metric-plugin-protocol.md) — exec plugin protocol for
  out-of-tree **metrics** (Input on stdin → one Result), and why the team server never
  executes them.
