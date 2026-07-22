# Backlog

The ranked pool of concrete candidate work items — the finer-grained counterpart to
[ROADMAP.md](ROADMAP.md)'s narrative direction. **Nothing here is a commitment or a
schedule.** The release buckets below are a working hypothesis of order; real feedback
from people running `assaio` reorders them, and any item can be reshaped or dropped.

**How this file works**

- One item = one checkbox with a stable id (`B01`) — reference it in issues, PRs, and
  commit bodies. Ids are never reused.
- When an item ships, in the same PR: add a user-facing entry to `CHANGELOG.md` under
  `[Unreleased]`, add or update its row in [FEATURES.md](FEATURES.md), and delete the
  line here. If only part ships, split the item. The three files together are the
  lifecycle: BACKLOG (candidates) → CHANGELOG (per-release delta) → FEATURES (current
  state, with the release each capability arrived in).
- Every item inherits the honesty rules ([CONTRIBUTING.md](CONTRIBUTING.md)):
  directional framing, `—` over a fabricated number, aggregate and pseudonymized by
  default, never a leaderboard. The refusals at the bottom hold no matter what ships.
- Effort: **S**mall / **M**edium / **L**arge. Scope: **solo** / **team** / **both**.
- Want to pick one up? Comment on or open an issue first so the approach is agreed
  before the work — connectors additionally follow the
  [connector intake flow](docs/extending.md#the-intake-path-open-a-connector-issue-first).

## v0.2 — "Know thyself" (fast wins, no schema change)

Theme: a solo user sees their own curve within two weeks of installing. All items read
existing store columns; none needs a migration.

- [ ] **B02 · cache-hygiene trend** — S · both — add the day-over-day cache-read-share
  trend to the shipped `cache-hygiene` validator, which already reports the current-window
  share and the cache-write-waste flag. Caveat: vendor cache TTLs are invisible; day-grain
  approximation.
- [ ] **B05 · statusline** — S · solo — `assaio-agent statusline`: one line (today's
  estimated `$`, tokens, budget remaining) plus a Claude Code statusline integration
  recipe. The daily-visibility habit loop.
- [ ] **B06 · metric-plugin scaffolder + schemas** — S · both — `metrics init --lang
  python|node|sh` writes a working plugin skeleton; publish JSON Schemas for the
  metric envelope/result under `docs/schemas/`. Lowers the barrier the moment the
  protocol is public.
- [ ] **B07 · explain** — S · both — `assaio-agent explain <validator>`: the long-form
  "how to read this metric, and what to do about it" page in the terminal.
- [ ] **B08 · dashboard PL locale** — S · both — a second language through the
  existing `localeStrings` seam, proving the i18n scaffolding with a real locale.
- [ ] **B58 · format-drift heuristics** — S/M · both — per-source canary metrics after
  every `backfill` (files vs last run, records/file vs history, zero-token share,
  skipped ratio) → `warning: possible format drift in <tool>` + a `doctor` section.
  Closes the silent-underreporting gap described in
  [docs/format-resilience.md](docs/format-resilience.md).

## v0.3 — "Team & ecosystem"

- [ ] **B09 · team-evenness** — M · team — new validator + dashboard Team panel:
  Lorenz/Gini spread of usage across pseudonymized members with a minimum-cohort
  (≥5) guard. Answers "broad adoption, or two power users?" — never a ranked list.
- [ ] **B10 · tool-coverage** — S · team — members per tool on a central store;
  shadow-tool and unused-seat detection.
- [ ] **B11 · weekly digest** — S/M · both — `digest --weekly`: markdown summary
  (top movers, verdict changes, anomalies) fit for cron/launchd; delivery stays the
  user's own script (mail, Slack, …).
- [ ] **B12 · GitHub Action** — M · team — packaged action running `check` as a gate
  plus a PR comment with movers/effectiveness for the changed window.
- [ ] **B13 · exec rule plugins (ADR 0005)** — M · both — third protocol completing
  processors → analyzers → rules: a rule reads the window's verdicts (all Results +
  totals as JSON on stdin) and emits alerts with severity; assaio prints, sets exit
  code, or forwards. Config `rules:`, same opt-in and boundary-validation posture as
  ADR 0003/0004.
- [ ] **B14 · Aider connector** — M · both — in-tree parser. The parseable token source is
  the opt-in `~/.aider/analytics.jsonl` (`message_send` events with `properties.{main_model,
  prompt_tokens,completion_tokens,total_tokens,cost}`, `time` in epoch seconds, no session
  id or cache split); `.aider.chat.history.md` is markdown-only. No structured per-edit field
  — Aider auto-commits, so lines +/- come from git, not the logs.
- [ ] **B15 · explore-vs-produce** — M · both — first schema extension (migration
  `0002_*.sql` — shipped migrations are immutable): split `tool_calls` into
  read/search/command/write counts → a validator explaining *why* `$`/100 lines
  differs (debugging-heavy vs building-heavy projects).
- [ ] **B16 · context-utilization** — M · both — vendored model context-window table
  (like the price table) → peak context vs model limit, near-limit share, and honest
  right-sizing hints ("paying long-context rates for 50k contexts").
- [ ] **B17 · progress ("skill curve")** — M · solo — the headline composite: you vs
  you four weeks ago across adoption breadth, turn-efficiency, cache-hygiene, rework —
  a small panel of deltas, deliberately not a single score. Strictly self-relative,
  never cross-person. Needs a careful design pass before code.

## v0.4 — "Outcome & truth"

- [ ] **B18 · survival: per-day correlation + age-matching** — M · both — the shipped
  `survival` command reports window-level survival of a repo's commits beside the store's
  AI lines. Still to add: per-day AI-heavy vs quiet-day survival comparison (does code from
  AI-heavy days survive at a different rate?), an age/settle threshold so recent commits
  aren't counted as "survived", and rename-following in blame. Server-side git/issue-tracker
  correlation across the team remains the larger "Outcome & quality" stage.
- [ ] **B19 · vendor billing reconciliation** — M/L · both — opt-in pull of
  Anthropic/OpenAI usage/cost APIs; estimate-vs-actual delta with a confidence band.
  Network- and credential-gated; pulls vendor aggregates only, never uploads logs.
- [ ] **B20 · compaction-recovery-cost** — M · both — tokens spent in the turns right
  after a compaction vs baseline: the true price of overflowing context.
- [ ] **B21 · test-touch** — M · both — share of AI edits touching test files via
  privacy-safe category counts (test/source/docs/config) classified at parse time —
  paths are never stored. Needs a PRIVACY.md note. First quality-adjacent signal
  without a server.
- [ ] **B22 · server hardening** — M · team — TLS/reverse-proxy guidance, per-member
  tokens, retention policy, chunked/resumable sync (per ROADMAP's own MVP caveats).

## v1.0 — stability

- [ ] **B23 · protocol & schema freeze** — M · both — declare the exec plugin
  protocols (parser, metric, rule) and the SQLite schema stable under semver.
- [ ] **B24 · in-process Go plugin API** — L · both — the dynamically loaded
  `plugin/metric|rule|connector` tree sketched in CONTRIBUTING.md; no rebuild, no
  subprocess.
- [ ] **B25 · Postgres backend** — L · team — once a single SQLite file stops being
  enough for a central store.

## Pool — validators from data already stored (unscheduled)

- [ ] **B26 · burn-anomaly** — S/M · both — daily cost/token spikes vs a trailing
  robust median (MAD z-score); the local foundation for later alerting rules.
- [ ] **B27 · concentration** — S · team — Pareto/Lorenz of cost across projects
  (project-level only, never people).
- [ ] **B28 · rhythm** — S · both — day-of-week × hour heatmap plus after-hours
  share; explicitly never an attendance view.
- [ ] **B29 · session-taxonomy: thrash + trend** — M · both — the shipped `session-taxonomy`
  validator buckets conversational / light-edit / heavy-edit; still to add a thrash bucket
  (needs per-session rework, not stored yet) and the mix's week-over-week trend.
- [ ] **B30 · delegation** — M · solo — sub-agent economics: delegation share trend,
  tokens per delegated vs main-loop session, lines-per-token with vs without
  delegation. Task difficulty is invisible — a prompt to look, not a verdict.
- [ ] **B31 · throughput-per-hour** — S · both — AI lines per focused hour
  (ActiveMinutes); labeled an activity rate, never a productivity score.
- [ ] **B32 · rework-bursts** — S/M · both — rework clustered over time and per-session
  p90; healthy iteration churns too.
- [ ] **B33 · reasoning-share: per-project + trend** — S · solo — the shipped
  `reasoning-share` validator reports the overall reasoning share of reporting-tool output
  plus its coverage; still to add the per-model/project breakdown and a week-over-week trend.
- [ ] **B34 · model-freshness / lock-in** — S · both — single-model dependence and
  share of usage on unknown/legacy-priced models.
- [ ] **B35 · entrypoint-mix** — S · both — CLI vs IDE vs hook usage and where
  friction (rejections) differs; the stored `entrypoint` column is unused today.
- [ ] **B36 · marathon** — S · solo — p95 session active minutes and long- vs
  short-session efficiency; thrash correlates with marathons.
- [ ] **B37 · branch-mix** — S · both — AI lines on the default branch vs feature
  branches; a process signal (direct-to-main AI work), local only.

## Pool — needs a schema or parser extension

- [ ] **B38 · friction-events** — M · both — tool errors / API errors / retries
  extracted from logs → an error-rate trend; needs per-tool log research.
- [ ] **B39 · Cline activity extraction** — M · both — close the "cost but no lines" gap
  for Cline: `ui_messages.json` `say:"tool"` payloads (`newFileCreated` /
  `editedExistingFile` / `appliedDiff`) and `api_conversation_history.json` tool_use blocks
  carry the paths and diffs, so lines added/removed are derivable. Multiplies every activity
  validator. Gemini CLI's default recording carries only tool-call *names*, not diffs — its
  edit activity needs the opt-in telemetry export (B72), so it is split out.
- [ ] **B72 · Gemini activity via OpenTelemetry** — M · both — Gemini CLI's structured edit
  data (`ToolCallEvent.model_added_lines`/`model_removed_lines`, `FileOperationEvent`) lives
  only in its **opt-in** OTel export, not the default session JSONL. An optional connector
  could read a user-configured OTLP file export where enabled; strictly opt-in and clearly
  labeled, since most installs won't have it on.

## Pool — robustness

- [ ] **B59 · doctor --strict** — S · both — non-zero exit when B58's drift heuristics
  fire (or a configured source discovers zero files), so a cron/CI job can alert on
  vendor format drift instead of a human noticing shrunk numbers.
- [ ] **B68 · live-session ingest consistency** — M · both — activity counts (lines /
  edits / rework, not tokens or cost) can double-count (Codex trailing-flush) or
  under-count (Claude post-hoc attribution) when a session is ingested while still being
  written, because inserts are idempotent (first-write-wins) and activity is attributed
  after the fact. Needs upsert-on-activity semantics (`ON CONFLICT ... DO UPDATE` for the
  derived columns only) with care around the sync dedupe contract. Disclosed as a `doctor`
  caveat until fixed.
- [ ] **B69 · usage-granularity provenance in reports** — S/M · both — `Store.Usage`
  blends session-granularity plugin rows with per-turn rows without surfacing
  `granularity`; carry it into `UsageRow` and mark mixed-granularity totals so session
  data never silently reads as per-turn (honesty rule).
- [ ] **B70 · subpaths composite index** — S · both — add `(project, ts)` index for the
  dashboard drill query (`WHERE project = ? AND ts >= ?`); today it range-scans `idx_usage_ts`.
  Perf only at local scale; ship with the next schema migration.

## Pool — team

- [ ] **B40 · onboarding-curve** — M · team — usage growth vs weeks-since-first-sync,
  in aggregated bands, pseudonymized.
- [ ] **B41 · team efficiency spread** — M · team — distribution bands of
  turn-efficiency-style signals across members; "the team needs prompting practice"
  without naming anyone.
- [ ] **B42 · server-side exec metrics** — M · team — lift the ADR 0004 non-goal
  safely: compute metric-plugin results on sync-write or a TTL cache, never per
  unauthenticated request.

## Pool — CLI & DX

- [ ] **B43 · hook / incremental ingest** — S/M · both — a Claude Code session-end
  hook recipe (or `backfill --incremental`) so data stays fresh without a daemon.
- [ ] **B44 · MCP server** — M · solo — `assaio-agent mcp`: query your own usage from
  Claude ("what did that refactor cost?"); also on ROADMAP's further-out list.
- [ ] **B45 · TUI** — L · both — interactive terminal dashboard (validators +
  project drill), the flagship DX piece once the small wins land.
- [ ] **B46 · completions + man pages** — S · both — cobra generators, shipped via
  goreleaser.
- [ ] **B47 · exports** — M · team — OpenMetrics endpoint on `serve` for Grafana;
  ndjson/parquet dump for data teams.
- [ ] **B66 · Scoop bucket (Windows)** — S · both — `scoops:` block in goreleaser
  publishing to a new `assaio/scoop-bucket` repo. Prerequisite: extend the release
  PAT's repository access to that bucket, or the next tag's release fails at the
  publish step.
- [ ] **B67 · winget manifest** — M · both — automated manifest PR to
  `microsoft/winget-pkgs` on release (goreleaser supports it); a heavier review loop
  than Scoop, so Scoop first.

## Pool — dashboard

- [ ] **B48 · sparklines** — M · both — per-day series for key figures as inline SVG;
  the dashboard stays fully self-contained.
- [ ] **B49 · multi-window tabs** — M · both — 7d/30d/90d generated into one HTML
  with a client-side toggle.
- [ ] **B50 · top-N drilldowns** — S/M · both — accordion drill for the top 3–5
  projects instead of only the top one.
- [ ] **B51 · print stylesheet** — S · team — a PDF-able layout for management
  readouts.

## Pool — connectors

Each follows the [connector intake flow](docs/extending.md#the-intake-path-open-a-connector-issue-first);
a tool used by one organization is usually better served by an out-of-tree
[exec plugin](docs/extending.md#write-a-plugin-any-language).

- [ ] **B52 · opencode** — M — `~/.local/share/opencode/storage/message/**` JSON (plus a
  newer relational `opencode.db`). Assistant messages carry `tokens{input,output,reasoning,
  cache{read,write}}`, `cost`, `modelID`, and — richest of any candidate — the `edit` tool
  persists a structured `filediff{additions,deletions,patch}`, so lines +/- are stored
  directly, no diff parsing. The best activity target after the current four.
- [ ] **B53 · Copilot CLI** — M — `~/.copilot/session-state/<id>/events.jsonl` (+ a
  `session-store.db`). Dir layout is solid, but per-turn token counts and edit-diff fields
  are not in the public schema; tokens appear only in an opt-in OTel export
  (`~/.copilot/otel/*.jsonl`). Verify field shapes on a real session before building.
- [ ] **B54 · Factory droid** — M — session-granularity local logs (per ROADMAP).
- [ ] **B55 · Cursor (Admin API)** — M — local storage verified to lack token counts;
  vendor-aggregate granularity, tagged as such.
- [ ] **B56 · Kiro** — M — only if its logs turn out to carry real token data.
- [ ] **B57 · community plugin registry page** — S — a docs page listing community
  exec plugins (parsers and metrics) once a few exist, seeded with the weekend-usage
  example.
- [ ] **B60 · Roo Code + Kilo Code (Cline family)** — S/M — both are Cline forks with the
  same task-directory storage under their own VS Code `globalStorage` publisher ids: Roo is
  `rooveterinaryinc.roo-cline` (format confirmed identical to Cline's `api_req_started`
  shape, plus an ignored `apiProtocol` field); Kilo is `kilocode.kilo-code` (inferred from
  lineage, token shape not yet source-verified). Parameterize the existing Cline parser over
  publisher roots **and a per-fork tool name** — so Roo/Kilo attribute distinctly, not as
  `cline` — instead of writing new parsers. Still needs a verified sample per fork.
- [ ] **B61 · Qwen Code (Gemini family)** — M — a Gemini CLI fork, but **not** the same
  on-disk shape: chats live at `~/.qwen/projects/<hash>/chats/<id>.jsonl` (not
  `tmp/*/chats`), and tokens are in a raw-API `usageMetadata` object, not Gemini's
  normalized `tokens{…}`. It also persists tool-call args, so unlike Gemini it carries edit
  activity. Needs its own parser, not the Gemini one; verify a real sample first.
- [ ] **B62 · Continue** — M — `~/.continue` dev-data event logs reportedly carry
  token counts; verify shape via a connector issue before building.
- [ ] **B63 · Goose** — M — local session JSONL reportedly carries usage; verify.
- [ ] **B64 · Amp / Crush** — M — local thread/session storage; token presence
  unverified for both; research first (connector issues).
- [ ] **B65 · Cursor local activity source** — M — `~/.cursor/ai-tracking/`'s
  AI-code-tracking database is a potential **activity-only** source (AI-attributed
  code, no token counts — Cursor's local storage is verified to lack them). Would ship
  as lines/activity with `granularity`/provenance honestly tagged, complementing the
  Admin-API cost path (B55). Research item.

## Pool — code health (0.3 candidates, from the pre-0.2 review)

Deferred cleanups surfaced by the max-effort review of the 0.2 work; no behavior change, so
they wait behind features but keep the growing metric surface maintainable.

- [ ] **B73 · validator meta** — S · both — embed a `meta{name,title,describe,howToRead}`
  value in each validator so the twelve metrics stop repeating the three trivial interface
  methods and the duplicated `Result` header; one source of truth per metric.
- [ ] **B74 · member aggregate** — S · team — collapse `dashboard/team.go`'s four parallel
  per-member maps (`lines`/`cost`/`hasCost`/`unpriced`) into one `map[string]memberAgg`, so a
  member's totals travel as a unit.
- [ ] **B75 · humanize helpers** — S · both — unify the four K/M/B number formatters
  (`compactCount`, `money`, `formatCompactUSD`, `formatCompactTokens`) behind one shared core
  so token and USD glance-figures read consistently across CLI and dashboard.
- [ ] **B76 · cli/common.go** — S · both — split the shared cross-command helpers
  (`addDBFlag`, `resolveDBPath`, `openReportStore`, `emptyStoreHint`, `emptyStatusHint`,
  `compareFormatConflict`, `parseSinceAt`) out of the single-command `report.go` (now past the
  ~200-line budget) into a `cli/common.go`.
- [ ] **B77 · readWhenEnough** — S · both — factor the "gate the favorable read behind a
  minimum-sample floor, else neutral" block shared verbatim by session-taxonomy,
  turn-efficiency, and model-right-sizing into one helper beside `readFor`.

## Refusals (will not build, regardless of demand)

- No "estimated time saved" headline — the logs contain no counterfactual.
- No lines-of-code or token leaderboards; nothing ranked per named individual, ever.
- No per-person analytics outside a deliberate, governed team-mode opt-in.
- No cohort/percentile comparisons without a minimum cohort size and explicit consent.
