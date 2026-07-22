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

## [0.2.0] - 2026-07-22

### Added
- **Three `analyze` validators.** `coverage` (Coverage & Confidence) — the provenance meter:
  what share of a window's tokens come from tools with full activity capture vs cost-only
  sources, and what share is priced. `cache-hygiene` (Cache Hygiene) — prompt-cache reuse
  (cache-read share of billed input) with an honest cache-write-waste flag. And
  `subscription-fit` (Subscription Fit) — for flat-plan users (Claude Max/Pro, ChatGPT
  Plus/Pro): projects the window's API-equivalent estimate to a month and compares it against
  your configured `pricing.monthly_subscription_cost`, so the estimate reads as plan value
  (a "137x — paying off" verdict) instead of a meaningless spend figure.
- **Four behavioral `analyze` validators** from session and per-turn data: `session-taxonomy`
  (conversational / light-edit / heavy-edit session mix), `turn-efficiency` (one-shot rate,
  median turns per code-producing session, output per turn), `model-right-sizing` (premium-model
  turns that produced little output — downgrade candidates, reframed as speed/limits on a flat
  plan), and `reasoning-share` (extended-thinking share of output, honest about which tools
  report it). Twelve built-in validators in total.
- **`survival` command** — the first local outcome signal: for a git repository, how much of
  the window's commits still live in `HEAD` (via `git blame`), shown beside the AI lines the
  store recorded for that project. Directional and honest — it never attributes specific lines
  to AI (assaio counts lines, not code) — and the stepping stone toward server-stage git/issue
  correlation. Plus [`docs/automation.md`](docs/automation.md): git-hook and scheduled recipes
  to keep the store fresh, push it to a self-hosted team server, and run `survival` on a timer.
- **Dashboard unpriced honesty.** Cost figures that exclude usage on unpriced models are
  now marked `*` (main cost basis and per-member team costs), with a colophon note —
  matching the CLI tables instead of showing a silent floor.
- **Cline discovery across editors.** Cline task data is now found under VS Code Insiders,
  VSCodium, and Cursor (not just stable VS Code), using the same `saoudrizwan.claude-dev`
  global storage — so Cline usage in any of those editors is counted, not silently missed.

### Fixed
- **Coverage rounding.** In the `coverage` validator, a small but nonzero token share (e.g. a
  few Codex sessions dwarfed by Claude's cache-read volume) now reads `<1%` instead of `0%`
  (which looked absent), and a share just under whole reads `>99%` instead of a gap-hiding
  `100%` — the honesty backbone must not round either edge away.
- **Gemini session ids.** The real Gemini CLI recording carries the session id only on the
  file's header line, not on every message, so message records were left with an empty
  session id. The parser now carries the header's id forward (an older per-line shape still
  works), and skips `$set`/`$rewindTo` control records without miscounting them.
  **Compatibility:** this changes the dedupe key for header-only Gemini logs, so if you
  ingested Gemini usage under v0.1.x, run `assaio-agent clear --tool gemini-cli --yes` then
  `assaio-agent backfill` once after upgrading to avoid double-counting those records.
- **Reasoning tokens no longer double-counted.** Codex and Gemini report reasoning as a
  subset of output; the grand token totals added it a second time, inflating every token
  count (cost was unaffected). Totals now count it once.
- **Anonymized dashboard no longer leaks real subpath names.** The drill-down's
  repository-subpath table was passed through verbatim under `--anonymize`, exposing paths
  like `apps/mobile` beside a pseudonymized project; subpaths are now pseudonymized too.
- **Team server rejects an empty `session_id`/`dedupe_key`.** An empty dedupe key collapsed
  every such row into one under `ON CONFLICT DO NOTHING`, silently undercounting a member.
- **Config validation is enforced on every command.** A typo'd honesty-relevant setting
  (e.g. a misspelled `pricing.mode`) or a duplicate plugin name now errors instead of
  silently reverting; `config` still validates-and-warns so it can display a broken file.
- **Period-over-period movers are deterministic.** Tied cost deltas (common when groups are
  all unpriced) kept a random order across runs; they now sort stably with a name tiebreak.
- **Throughput top-project bars cover the whole window**, matching the "AI lines total"
  headline instead of a recent-only sub-window with no label.
- **Cline recovers a truncated `ui_messages.json`.** A read racing Cline's live rewrite
  lost the whole task; the array is now streamed, keeping every message before the break
  (skip-and-count, per the parser contract).
- **Plugin I/O robustness.** A plugin's newline-free stderr flood is bounded instead of
  growing memory unbounded; a stdout-cap breach is now reported as such and the child
  killed promptly, instead of being misreported as a timeout. String fields on pushed and
  plugin-emitted records are length-capped at the boundary.
- **Codex timestamps.** A `session_meta` whose payload omits a timestamp no longer resets
  the record timestamp to the zero time. Cline model resolution is now deterministic.
- **Schema migrations apply atomically** with their bookkeeping row, so a crash mid-migration
  can't leave a half-applied migration that re-runs next boot.
- **`clear` guards.** An unknown `--tool` value (e.g. `claude` for `claude-code`) now errors
  instead of silently deleting nothing, and `--all` combined with `--older-than`/`--tool`
  is rejected as contradictory rather than silently narrowing the deletion.
- **`--db`-aware empty-store hints.** `effectiveness`, `status`, and `analyze` no longer tell
  a `--db` user to run `backfill` (which only writes the local store). `--compare` now errors
  instead of silently ignoring `--format json|csv`. `doctor` reports a store-count error
  instead of printing `ok`.

## [0.1.1] - 2026-07-20

### Fixed
- **Claude Code sub-agent accounting.** Background/async sub-agent (Task) token usage was
  not counted at all, and a completed sub-agent's cost was read from a last-turn summary
  in the parent transcript rather than its full per-turn record — a large under-count.
  assaio now reads each sub-agent's own transcript as the source of truth and suppresses
  the redundant parent summary, so sub-agent usage is counted once and in full. This is
  the accounting behavior the 0.1.0 notes claimed but did not fully deliver.
- **Codex double-count guard.** A rate-limit-only `token_count` update (`info:null`) no
  longer resets the cumulative-token baseline, which could make the next update re-count
  the whole session.
- **Negative-token clamp (Claude Code, Cline).** A malformed line carrying a negative
  token count is clamped to zero, matching the parser contract; it can no longer deflate
  stored totals.
- **Gemini messages with a zero `total`** but non-zero input/output tokens are counted
  instead of being silently dropped.
- **Period-over-period windows.** `--compare` and the week-over-week trend now span an
  equal number of days on each side; the recent window previously covered one extra day,
  biasing every movement upward.
- **Config from the environment.** `ASSAIO_` variables for keys that contain underscores
  (e.g. `ASSAIO_PRICING_EFFECTIVE_PER_TOKEN`) now apply instead of being silently ignored.
- **Explicit `--config`.** A `--config` path that does not exist now errors instead of
  silently falling back to built-in defaults; `config` prints the path actually in use.
- **Team-server timestamps.** A pushed record with a zero or out-of-range timestamp is
  rejected, so a far-future record can no longer sit permanently in the shared dashboard's
  recent windows.
- **`sync` pseudonyms** widened from 16 to 40 bits, matching report pseudonyms, so two
  members no longer collide into one on a shared store.
- **`demo`** closing hint uses the real `assaio-agent` binary name (was `assaio`, which
  fails with "command not found"), and `demo --dashboard` writes to a private
  per-invocation temp dir instead of a predictable shared path.
- **Unpriced marking.** `report --compare` movers and `status` mark a group whose cost
  excludes unpriced usage with `*` and a footnote, instead of a bare or fabricated `$0`.
- **Diagnostics.** A misconfigured exec plugin's failure reason is printed to stderr; a
  file whose usage lines are all corrupt is counted as skipped; and a Ctrl-C during a
  plugin run is no longer misreported as a plugin timeout.

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

[Unreleased]: https://github.com/assaio/assaio/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/assaio/assaio/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/assaio/assaio/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/assaio/assaio/releases/tag/v0.1.0
