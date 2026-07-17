# Features

The maintained inventory of what `assaio` does **today**, and since which release. This
is the current-state counterpart to the other two lifecycle documents: candidates live
in [BACKLOG.md](BACKLOG.md), the per-release delta lives in [CHANGELOG.md](CHANGELOG.md),
and every shipped user-facing capability gets (or updates) a row here in the same PR.

Pre-release note: `v0.1` below means the upcoming first public release; the column
starts mattering the moment a second release exists.

## Commands

| Command | Since | What it does |
|---------|-------|--------------|
| `demo` | v0.1 | Full reports on bundled sample data — no logs needed. |
| `backfill` | v0.1 | Import all historical local session logs into the store. |
| `report` | v0.1 | Token/cost report; `--by day\|project\|tool\|model\|entrypoint\|member`, `--format table\|json\|csv`, `--compare` top movers. |
| `effectiveness` | v0.1 | AI output vs cost — AI lines, edits, rejections, **`$`/100 AI lines** — per project by default; `--compare`. |
| `analyze` | v0.1 | Runs the metric validators below plus configured exec metric plugins; `--list`, `--format text\|json`, `[name...]` subset. |
| `check` | v0.1 | Budget gate with non-zero exit: `--max-tokens` (default basis) or `--max-cost` (labeled API-equivalent). |
| `dashboard` | v0.1 | Writes the self-contained offline Assay HTML report; pseudonymized by default, `--no-anonymize` opt-out. |
| `serve` | v0.1 | Self-hosted team server: collects pushed usage, serves the aggregated team dashboard. |
| `sync` | v0.1 | Pushes local usage to a team server; pseudonymous by default, `--member` is an explicit opt-in. |
| `doctor` | v0.1 | Detected tools, resolved log roots, store inventory, health, accuracy caveats. |
| `status` | v0.1 | Terminal overview: inventory, headline `$`/100 lines, hot / going-stale projects, session stats. |
| `clear` | v0.1 | Deletes stored data; requires an explicit scope and `--yes`. |
| `config` | v0.1 | Prints the effective merged configuration and its source path. |
| `plugins` | v0.1 | `list` / `verify` for exec **parser** plugins (protocol conformance, nothing stored). |
| `metrics` | v0.1 | `list` / `verify` for exec **metric** plugins (runs on your real window, prints violations + rendered result). |
| `version` | v0.1 | Prints the version (also `--version`). |

## Metric validators (`assaio analyze` + dashboard)

Each validator is one file in `internal/analyze/`, self-registered, rendered
generically by the CLI, JSON output, and the dashboard.

| Validator | Since | Question it answers | Built-in caveat |
|-----------|-------|--------------------|-----------------|
| `adoption` | v0.1 | How broad is AI usage (sessions, active days, project/tool breadth) and is it growing? | Breadth, not quality. |
| `context` | v0.1 | Are sessions healthy: turns, peak context, focused minutes, compaction rate? | Neutral below 3 sessions — no verdict from thin data. |
| `model-fit` | v0.1 | Premium vs cheaper token share (by real price tier), lines-per-token contrast, sub-agent delegation share, upper-bound routing savings. | Savings figure is an upper bound, never a switch recommendation. |
| `rework` | v0.1 | Within-session churn (AI lines undone in the same transcript) and human rejection rate. | Directional friction proxy; healthy iteration churns too. |
| `throughput` | v0.1 | Total AI lines, lines per active day, top projects, week-over-week trend. | Activity rate, never a productivity score. |

Exec **metric plugins** (below) render beside these, namespaced `plugin:<name>`.

## Extension surfaces

| Surface | Since | Contract |
|---------|-------|----------|
| In-tree parser (new data source) | v0.1 | One Go package under `internal/parser/` + golden & fuzz tests ([guide](docs/extending.md#add-a-data-source)). |
| Exec **parser** plugin, any language | v0.1 | `<command> scan`, handshake + JSONL records on stdout; `plugins:` in config (ADR 0003). |
| Exec **metric** plugin, any language | v0.1 | `<command> analyze`, Input JSON on stdin → handshake + one Result on stdout; `metrics:` in config (ADR 0004). |
| In-tree metric validator | v0.1 | One file implementing `Validator`, registered via `init()` ([guide](docs/extending.md#adding-a-metric-validator)). |
| Direct SQL on the store | v0.1 | Documented `usage_record` schema, any SQLite client. |
| JSON/CSV pipes | v0.1 | `report`/`effectiveness` `--format json\|csv` into your own tooling. |
| Per-team configuration | v0.1 | `config.yaml` + `ASSAIO_*` env vars (sources, plugins, metrics, privacy, server, sync, pricing). |

## Data sources (parsers)

| Tool | Since | Tokens & cost | Activity (lines/edits/rework) |
|------|-------|---------------|-------------------------------|
| Claude Code | v0.1 | ✔ (incl. sub-agent turns) | ✔ full, incl. rejections |
| OpenAI Codex CLI | v0.1 | ✔ (exact, delta-based) | ✔ except rejections |
| Gemini CLI | v0.1 | ✔ | — (cost only, see ROADMAP) |
| Cline | v0.1 | ✔ (recomputed from tokens) | — (cost only, see ROADMAP) |
| Exec parser plugins | v0.1 | ✔ (validated at the boundary) | — (protocol carries tokens only) |

Shared parser guarantees: skip-and-count on corrupt lines, deterministic dedupe keys
(re-runs never double-count), `Granularity` honesty (session-level data never
masquerades as per-turn), fuzz tests with committed corpora.

## The Assay dashboard

| Element | Since |
|---------|-------|
| Verdict faceplate (one gauge cell per validator, incl. metric plugins) | v0.1 |
| Ledger (figures, "how to read", bars, caveats, takeaway per validator) | v0.1 |
| Top-project drill-down + subpath breakdown | v0.1 |
| Team section (per-member, appears only on a central store) | v0.1 |
| Cost-basis footnote + honesty colophon | v0.1 |
| Light/dark theme, fully self-contained single file, i18n scaffold (EN) | v0.1 |
| Pseudonymized project/member names by default | v0.1 |

## Team server (MVP)

| Capability | Since |
|-----------|-------|
| `POST /v1/usage` — bearer-token ingestion, per-member dedupe namespacing | v0.1 |
| `GET /` — served team dashboard, always anonymized (no auth on read — run behind a reverse proxy) | v0.1 |
| `GET /healthz` | v0.1 |
| Team-aware CLI: `--db` against a central store, `--by member` | v0.1 |
| Explicitly not executed server-side: exec metric plugins (ADR 0004) | v0.1 |

## Cost honesty & budgeting

| Capability | Since |
|-----------|-------|
| Every `$` disclosed as an estimate at public pay-as-you-go API prices | v0.1 |
| Unpriced models: `—` / `null` / excluded from totals — never a fake `$0` | v0.1 |
| `config.pricing` — subscription / negotiated-rate basis shown alongside the estimate | v0.1 |
| Token-first budgets in `check` (plan-independent), `$` budgets labeled API-equivalent | v0.1 |
| Model-routing savings shown only as a labeled upper bound | v0.1 |

## Cross-cutting guarantees

- No network at runtime for local analysis; no telemetry; prompts and code are never
  read — counts only ([PRIVACY.md](PRIVACY.md)).
- Aggregate and pseudonymized by default; per-person is a deliberate, governed opt-in;
  never a leaderboard.
- Plugins (parser and metric) are config-declared only — never PATH-scanned, never
  downloaded; everything they emit is validated at the boundary and namespaced
  `plugin:<name>`.
- Schema self-heal: an existing local database migrates itself forward.
