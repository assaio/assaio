# Features

The maintained inventory of what `assaio` does **today**, and since which release. This
is the current-state counterpart to the other two lifecycle documents: candidates live
in [BACKLOG.md](BACKLOG.md), the per-release delta lives in [CHANGELOG.md](CHANGELOG.md),
and every shipped user-facing capability gets (or updates) a row here in the same PR.

`v0.1` below means the v0.1.0 line — the first public release; the column starts
mattering the moment a second release exists.

## Commands

| Command | Since | What it does |
|---------|-------|--------------|
| `init` | v0.5 | A first run end to end: detects tools, prints the exact directories it will read before reading them, imports, writes the report, names what to run next. Writes no config when defaults work; `--non-interactive` for scripted setup. |
| `demo` | v0.1 | Full reports on bundled sample data — no logs needed. |
| `backfill` | v0.1 | Import all historical local session logs into the store. Since v0.4, incremental: an input already parsed unchanged by this build is skipped and reported as `unchanged=`; `--full` forces a complete re-parse. |
| `report` | v0.1 | Token/cost report; `--by day\|project\|tool\|model\|entrypoint\|member`, `--format table\|json\|csv`, `--compare` top movers. |
| `effectiveness` | v0.1 | AI output vs cost — AI lines, edits, rejections, **`$`/100 AI lines** — per project by default; `--compare`. |
| `analyze` | v0.1 | Runs the metric validators below plus configured exec metric plugins; `--list`, `--format text\|json`, `[name...]` subset. Since v0.5 every result carries a confidence envelope (coverage, sample size and unit, data age, parsing build, and a `high\|medium\|low\|insufficient` label). |
| `check` | v0.1 | Budget gate with non-zero exit: `--max-tokens` (default basis) or `--max-cost` (labeled API-equivalent). Since v0.3, also the host for exec **rule** plugins: an `error` alert — or a rule that could not be evaluated — fails the gate. |
| `dashboard` | v0.1 | Writes the self-contained offline Assay HTML report; pseudonymized by default, `--no-anonymize` opt-out. |
| `serve` | v0.1 | Self-hosted team server: collects pushed usage, serves the aggregated team dashboard. |
| `sync` | v0.1 | Pushes local usage to a team server; pseudonymous by default, `--member` is an explicit opt-in. |
| `doctor` | v0.1 | Detected tools, resolved log roots, store inventory and size, health, format-drift canaries, accuracy caveats. `--strict` (v0.5) exits non-zero on suspected drift or a configured source with no inputs, for cron/CI. |
| `status` | v0.1 | Terminal overview: inventory, headline `$`/100 lines, hot / going-stale projects, session stats. |
| `statusline` | v0.4 | One ambient line for a status bar: today's tokens, AI lines, cost basis, and data age. Local day (timestamp range, not a UTC bucket); read-only; never fails loudly. See [automation](docs/automation.md). |
| `explain` | v0.4 | The long-form page for a metric — what it measures, how to read it, what to do about it, its limits. Needs no store; no argument lists every metric. |
| `survival` | v0.2 | Directional local outcome check: how much of a git repo's window survives in `HEAD` (`git blame`), beside the AI lines the store recorded. See [automation](docs/automation.md). |
| `clear` | v0.1 | Deletes stored data; requires an explicit scope and `--yes`. Reports what stays held by the store afterwards. |
| `compact` | v0.5 | Reclaims disk space the store freed but still holds (`VACUUM` + WAL truncate); prints the size before and after. Deleting rows never shrinks a SQLite file on its own. |
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
| `burn-anomaly` | v0.3 | Which days burned far outside the window's typical day (robust median/MAD outlier test) — catching a runaway loop or an agent left running. | A spike is a prompt to look, not a fault; needs 7+ active days for a baseline. |
| `cache-hygiene` | v0.2 | Prompt-cache reuse: cache-read share of billed input, and whether cache writes are reused. | Cost signal, not quality; a big one-shot task legitimately shows low reuse. |
| `concentration` | v0.3 | How token spend spreads across projects, and where a project's share of tokens outruns its share of AI-written lines. | Concentration itself is neither good nor bad; undefined below 2 projects. Project-level only, never people. |
| `context` | v0.1 | Are sessions healthy: turns, peak context, focused minutes, compaction rate? | Neutral below 3 sessions — no verdict from thin data. |
| `coverage` | v0.2 | How much of the window is high-confidence data: token share from tools with full activity capture, share priced, and — when a window mixes them — the share from per-turn rather than whole-session records. | Says how complete the data is, never whether the work was good. |
| `explore-produce` | v0.3 | What the tool calls were for: reading and searching the codebase vs writing code in it, with reads-per-write. | Exploring earns the right to write; only flags the extreme. Covers Claude Code and Codex; states its own coverage, and un-backfilled history reads as unclassified. |
| `friction` | v0.3 | How often tool calls fail outright, beside how often a human declines one — the tokens spent on calls that produced nothing. | Some failure is normal probing. Rates cover only tools that mark the outcome of every call (Claude Code today); Codex reports it for file edits alone and is excluded rather than counted as successful. |
| `model-fit` | v0.1 | Premium vs cheaper token share (by real price tier), lines-per-token contrast, sub-agent delegation share, upper-bound routing savings. | Savings figure is an upper bound, never a switch recommendation. |
| `model-right-sizing` | v0.2 | Premium-model turns that produced little output — downgrade candidates a cheaper/faster model might handle. | Task difficulty is invisible; a prompt to review, not a verdict. On a flat plan it's about speed/limits, not $. |
| `reasoning-share` | v0.2 | Extended-thinking (reasoning) share of output among tools that report it, and how much of your output that covers. | Only Codex/reasoning models report it today; Claude Code doesn't. |
| `rework` | v0.1 | Within-session churn (AI lines undone in the same transcript) and human rejection rate. | Directional friction proxy; healthy iteration churns too. |
| `rhythm` | v0.3 | When sessions run: off-hours and weekend share, the time-of-day shape of the work, and how long the longest focused sessions last (p95). | Aggregate workload signal, not an individual measure; hours read in the machine's local timezone. |
| `session-taxonomy` | v0.2 | The mix of session kinds: conversational (no edits), light-edit, heavy-edit — how you actually use AI. | Descriptive, not a scorecard; conversational is real work. A thrash bucket needs per-session rework (not stored yet). |
| `skill-economics` | v0.3 | Which skills and sub-agents the tokens went to, and how much code each produced — where shared tooling quietly concentrates spend. | Only Claude Code labels turns with a skill/sub-agent today; other tools are absent from the split, not zero. |
| `subscription-fit` | v0.2 | For flat-plan users: the window's API-equivalent projected to a month vs the configured plan cost — a value multiple and an "is it paying off?" verdict. | API-equivalent is an estimate at public prices, not your bill; needs `pricing.monthly_subscription_cost`. |
| `throughput` | v0.1 | Total AI lines, lines per active day, top projects, week-over-week trend. | Activity rate, never a productivity score. |
| `turn-efficiency` | v0.2 | Getting more per prompt: one-shot rate, median turns per code-producing session, output tokens per turn. | Task size is invisible; directional, never a per-person score. |

Exec **metric plugins** (below) render beside these, namespaced `plugin:<name>`.

## Extension surfaces

| Surface | Since | Contract |
|---------|-------|----------|
| In-tree parser (new data source) | v0.1 | One Go package under `internal/parser/` + golden & fuzz tests ([guide](docs/extending.md#add-a-data-source)). |
| Exec **parser** plugin, any language | v0.1 | `<command> scan`, handshake + JSONL records on stdout; `plugins:` in config (ADR 0003). |
| Exec **metric** plugin, any language | v0.1 | `<command> analyze`, Input JSON on stdin → handshake + one Result on stdout; `metrics:` in config (ADR 0004). |
| Exec **rule** plugin, any language | v0.3 | `<command> evaluate`, the window's verdicts on stdin → handshake + one alerts document on stdout; `rules:` in config; gates `check` (ADR 0005). |
| In-tree metric validator | v0.1 | One file implementing `Validator`, registered via `init()` ([guide](docs/extending.md#adding-a-metric-validator)). |
| Direct SQL on the store | v0.1 | Documented `usage_record` schema, any SQLite client. |
| JSON/CSV pipes | v0.1 | `report`/`effectiveness` `--format json\|csv` into your own tooling. |
| Per-team configuration | v0.1 | `config.yaml` + `ASSAIO_*` env vars (sources, plugins, metrics, privacy, server, sync, pricing). |

## Data sources (parsers)

"Supported" is not one thing: a source that reports tokens but no edits cannot answer the
same questions as one that reports both. Every source therefore publishes its **depth**
(v0.5), which `doctor` prints for the tools it finds on your machine and which the
`coverage` validator reads instead of keeping its own list.

| Tool | Since | Depth | Tokens & cost | Activity (lines/edits/rework) | Attribution (skill / sub-agent) |
|------|-------|-------|---------------|-------------------------------|---------------------------------|
| Claude Code | v0.1 | **deep** | ✔ (incl. sub-agent turns) | ✔ full, incl. rejections | ✔ |
| OpenAI Codex CLI | v0.1 | standard | ✔ (exact, delta-based) | ✔ except rejections | — |
| Gemini CLI | v0.1 | standard | ✔ | — (cost only, see ROADMAP) | — |
| Cline | v0.1 | standard | ✔ (recomputed from tokens) | — (cost only, see ROADMAP) | — |
| Exec parser plugins | v0.1 | declared per record | ✔ (validated at the boundary) | — (protocol carries tokens only) | — |

- **deep** — tokens, per-turn activity, and the labels that say what the work was.
- **standard** — reliable per-turn usage whose activity gaps are documented, not implied.
- **import-only** — billing or aggregate figures that cannot support session-level
  conclusions. No in-tree parser is here today; a source that emits whole-session records
  lands here, which is why granularity travels with every report row.

A plugin is absent from the tiers by design: its depth is whatever its author implemented,
so it is read from the records it emits rather than promised by a table assaio maintains.

Shared parser guarantees: skip-and-count on corrupt lines, deterministic dedupe keys
(re-runs never double-count), `Granularity` honesty (session-level data never
masquerades as per-turn), fuzz tests with committed corpora. Cline task data is
discovered across VS Code, VS Code Insiders, VSCodium, and Cursor.

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
| Explicitly not executed server-side: exec metric plugins (ADR 0004), exec rule plugins (ADR 0005) | v0.1 |

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
- Plugins (parser, metric, and rule) are config-declared only — never PATH-scanned, never
  downloaded; everything they emit is validated at the boundary and attributed to the
  plugin that emitted it (`plugin:<name>`).
- Schema self-heal: an existing local database migrates itself forward.
