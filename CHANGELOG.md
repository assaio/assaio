# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

**How to read this file.** `[Unreleased]` means merged to `main` but not yet part of
any tagged release — installing the latest release does *not* give you those entries.
At release time the whole `[Unreleased]` section becomes the new version's section
(enforced by `make release`, see [RELEASING.md](RELEASING.md)), so a version heading
always describes exactly what its tag contains. The headings link to the tag or diff.

This file records only what has actually shipped. What's *coming* is tracked in
[BACKLOG.md](BACKLOG.md) — ranked **proposals and effort estimates, not commitments**:
the actual order follows real-world feedback, pull requests, and bug reports, so items
there can be reshaped, reordered, or dropped, and things not listed anywhere can land
first when a PR or a bug report makes the case. To suggest something new — or add
weight to a tracked item (reference its `B` id) — open a feature-request issue or a
Discussion.

## [Unreleased]

### Added
- CI now runs the test suite on macOS and Windows alongside Linux; POSIX-only
  plugin-script tests are skipped on Windows.
- Per-platform install instructions in the README: Windows (PowerShell), Linux/macOS
  tarball, Homebrew/Linuxbrew, `go install`, and attestation verification.

## [0.1.0] - 2026-07-19

### Added
- Four tool parsers — Claude Code, OpenAI Codex CLI, Gemini CLI, Cline — with activity
  extraction (AI lines, edits, rejections, compactions, within-session rework) for
  Claude Code and Codex, and sub-agent (Task) token usage counted.
- `report` and `effectiveness` (`$`/100 AI lines) with
  `--by day|project|tool|model|entrypoint|member`, `--format table|json|csv`, and
  period-over-period `--compare` top movers.
- `analyze` — the validator framework: adoption, model fit (with an upper-bound
  model-routing savings estimate), context health, throughput, rework; one
  self-registering file per metric.
- **Exec metric plugins** — out-of-tree analyzers in any language, declared under
  `metrics:` in config, rendered beside the built-ins in `analyze` and the dashboard;
  `metrics list|verify` conformance tooling (ADR 0004).
- Exec parser plugins — out-of-tree data sources in any language, declared under
  `plugins:` in config; `plugins list|verify` (ADR 0003).
- **The Assay** — a self-contained, offline HTML dashboard: light/dark, per-section
  "how to read" explainers, a bounded project drill-down, a per-member team section;
  project and member names pseudonymized by default.
- Team-server MVP — `serve` (shared-bearer-token collection endpoint plus the served
  team dashboard) and `sync` (pseudonymous-by-default push; `--member` is an explicit
  opt-in), with `--db`/`--by member` for team-aware reads.
- `check` — a token (default) or API-equivalent-`$` budget gate with a non-zero exit
  for CI and pre-push hooks; `config.pricing` declares a subscription or negotiated
  cost basis shown alongside the API estimate.
- `demo` — the full reports on bundled sample data; plus `doctor`, `status`,
  `backfill`, `clear`, and `config`.
- Cost honesty throughout: every `$` disclosed as an estimate at public
  pay-as-you-go API prices; unpriced models render an honest blank, never a fake `$0`.

[Unreleased]: https://github.com/assaio/assaio/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/assaio/assaio/releases/tag/v0.1.0
