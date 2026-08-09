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
| `report` | v0.1 | Token/cost report; `--by day\|project\|tool\|model\|entrypoint\|member` plus `task\|outcome\|difficulty` (v0.6), `--format table\|json\|csv`, `--compare` top movers. |
| `effectiveness` | v0.1 | AI output vs cost — AI lines, edits, rejections, **`$`/100 AI lines** — per project by default; `--compare`. Since v0.6 also groupable by session label, so cost per line is comparable per kind of work. |
| `analyze` | v0.1 | Runs the metric validators below plus configured exec metric plugins; `--list`, `--format text\|json`, `[name...]` subset. Since v0.5 every result carries a confidence envelope (coverage, sample size and unit, data age, parsing build, and a `high\|medium\|low\|insufficient` label). Since v0.6, `--task\|--outcome\|--difficulty` recompute every validator over just the sessions carrying that label. Since v0.9 the envelope also carries **signal coverage** — how much of the window a metric's own subject reaches — so a figure read off a sliver cannot travel as a fully covered verdict. |
| `check` | v0.1 | Budget gate with non-zero exit: `--max-tokens` (default basis) or `--max-cost` (labeled API-equivalent). Since v0.3, also the host for exec **rule** plugins: an `error` alert — or a rule that could not be evaluated — fails the gate. Since v0.13 the same rule holds for `--max-cost`: a window carrying usage the price table cannot price fails rather than passing on a partial figure. |
| `dashboard` | v0.1 | Writes the self-contained offline Assay HTML report; pseudonymized by default, `--no-anonymize` opt-out. |
| `serve` | v0.1 | Self-hosted team server: collects pushed usage, serves the aggregated team dashboard. |
| `sync` | v0.1 | Pushes local usage to a team server; pseudonymous by default, `--member` is an explicit opt-in. |
| `doctor` | v0.1 | Detected tools, resolved log roots, store inventory and size, health, format-drift canaries, accuracy caveats. `--strict` (v0.5) exits non-zero on suspected drift, on a configured source with no inputs, or (v0.13) on a store it cannot read. Since v0.10 a root it could not read is reported as a failed discovery rather than counted as zero files. |
| `status` | v0.1 | Terminal overview: inventory, headline `$`/100 lines, hot / going-stale projects, session stats. Since v0.10 each session figure reads only the sources that record it, says so when that is fewer sessions than the window holds, and prints "not recorded" rather than a zero nobody measured. |
| `statusline` | v0.4 | One ambient line for a status bar: today's tokens, AI lines, cost basis, and data age. Local day (timestamp range, not a UTC bucket); read-only; never fails loudly. See [automation](docs/automation.md). |
| `explain` | v0.4 | The long-form page for a metric — what it measures, how to read it, what to do about it, its limits. Needs no store; no argument lists every metric. |
| `survival` | v0.2 | Directional local outcome check: how much of a git repo's window survives in `HEAD` (`git blame`), beside the AI lines the store recorded. Since v0.8 it reads commit observations, so it also reports what the window changed by file category and how many commits were reverts. Since v0.9 merge commits are reported apart from the rate — git publishes no line counts for a merge, so a hand-resolved conflict is counted in neither figure rather than inflating survival. Since v0.14 a file `git blame` cannot read is counted and named rather than passed over, so the rate never reads as covering files it never saw, and a non-ASCII path is decoded from git's quoted form instead of vanishing from it. See [automation](docs/automation.md). |
| `signals` | v0.7 | What assaio can report and what your own data supports. `list` names every signal; `describe <id>` says what it counts, where it is honest, and **what a zero means**; `coverage` reads your store and reports per signal whether it is fully, partly or not at all supported, naming the sources that can answer it. |
| `mark` | v0.6 | Labels a session with a task class, outcome and difficulty — category values only, never free text, never synced. Defaults to the newest session in the current repository; `--last`, an id prefix, `--list`, `--unmark`. |
| `clear` | v0.1 | Deletes stored data; requires an explicit scope and `--yes`. Reports what stays held by the store afterwards. Session labels survive `--all` and are removed only by `--labels` (v0.6), which since v0.13 follows the same scope and only takes the label of a session the clear removes entirely. Since v0.13 a clear that is not time-scoped also forgets it read those inputs, so a plain `backfill` rebuilds; `--older-than` keeps them and says so. |
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
| `cache-hygiene` | v0.2 | Prompt-cache reuse: cache-read share of billed input, whether cache writes are reused, what share of writes bought the 1-hour lifetime (v0.12), and the vendor's own top reason for a miss (v0.12). | Cost signal, not quality; a big one-shot task legitimately shows low reuse. The tier and the miss cause are read from Claude Code only; a window whose sources state neither says so rather than reading zero. |
| `concentration` | v0.3 | How token spend spreads across projects, and where a project's share of tokens outruns its share of AI-written lines. | Concentration itself is neither good nor bad; undefined below 2 projects. A project running entirely on a source that records no lines is excluded from the gap rather than read as producing nothing. Project-level only, never people. |
| `context` | v0.1 | Are sessions healthy: turns, peak context, focused minutes, compaction rate? | Neutral below 3 sessions — no verdict from thin data. Since v0.10 each figure is read only from the sessions whose source records it, the reach is stated, and a window whose sources never mark a compaction gets no health verdict at all. |
| `coverage` | v0.2 | How much of the window is high-confidence data: token share from tools with full activity capture, share priced, and — when a window mixes them — the share from per-turn rather than whole-session records. | Says how complete the data is, never whether the work was good. |
| `explore-produce` | v0.3 | What the tool calls were for: reading and searching the codebase vs writing code in it, with reads-per-write. | Exploring earns the right to write; only flags the extreme. Covers Claude Code and Codex; states its own coverage, and un-backfilled history reads as unclassified. |
| `intent` | v0.6 | How much of the window carries a task label, and whether enough classes have enough sessions to compare kinds of work at all. | Reads readiness to stratify, never diligence: it has no unfavorable verdict, and unlabeled sessions stay fully counted everywhere. |
| `friction` | v0.3 | How often tool calls fail outright, beside how often a human declines one — the tokens spent on calls that produced nothing. | Some failure is normal probing. Rates cover only tools that mark the outcome of every call (Claude Code today); Codex reports it for file edits alone and is excluded rather than counted as successful. |
| `model-fit` | v0.1 | Premium vs cheaper token share (by real price tier), lines-per-token contrast, sub-agent delegation share, upper-bound routing savings. | Savings figure is an upper bound, never a switch recommendation. |
| `model-right-sizing` | v0.2 | Premium-model turns that produced little output — downgrade candidates a cheaper/faster model might handle. | Task difficulty is invisible; a prompt to review, not a verdict. On a flat plan it's about speed/limits, not $. |
| `reasoning-share` | v0.2 | Extended-thinking (reasoning) share of output among tools that report it, and how much of your output that covers. | Only Codex/reasoning models report it today; Claude Code doesn't. |
| `rework` | v0.1 | Within-session churn (AI lines undone in the same transcript) and human rejection rate. | Directional friction proxy; healthy iteration churns too. The rejection rate divides only by calls whose source records a refusal. |
| `rhythm` | v0.3 | When sessions run: off-hours and weekend share, the time-of-day shape of the work, and how long the longest focused sessions last (p95). | Aggregate workload signal, not an individual measure; hours read in the machine's local timezone. Session length reads only the sources that record focused minutes; timing still covers every session. |
| `session-taxonomy` | v0.2 | The mix of session kinds: conversational (no edits), light-edit, heavy-edit — how you actually use AI. | Descriptive, not a scorecard; conversational is real work. Only sessions whose source records edits are bucketed — elsewhere a zero is the tool's silence, not a conversation. A thrash bucket needs per-session rework (not stored yet). |
| `skill-economics` | v0.3 | Which skills and sub-agents the tokens went to, and how much code each produced — where shared tooling quietly concentrates spend. | Only Claude Code labels turns with a skill/sub-agent today; other tools are absent from the split, not zero. The share is of attributed tokens, and how small a slice of the window those are is declared as its signal coverage. |
| `subscription-fit` | v0.2 | For flat-plan users: the window's API-equivalent projected to a month vs the configured plan cost — a value multiple and an "is it paying off?" verdict. | API-equivalent is an estimate at public prices, not your bill; needs `pricing.monthly_subscription_cost`. |
| `throughput` | v0.1 | Total AI lines, lines per active day, top projects, week-over-week trend. | Activity rate, never a productivity score. |
| `turn-efficiency` | v0.2 | Getting more per prompt: one-shot rate, median turns per code-producing session, output tokens per turn. | Task size is invisible; directional, never a per-person score. Reads only sessions whose source records edits, so "no code-producing sessions" never means "no source that could tell". |

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
| JSON/CSV pipes | v0.1 | `report`/`effectiveness` `--format json\|csv` into your own tooling. Since v0.14 the CSV carries `task`, `outcome` and `difficulty`, so a row grouped by a session annotation names the group it belongs to. |
| Per-team configuration | v0.1 | `config.yaml` + `ASSAIO_*` env vars (sources, plugins, metrics, privacy, server, sync, pricing). |

## Data sources (parsers)

"Supported" is not one thing: a source that reports tokens but no edits cannot answer the
same questions as one that reports both. Every source therefore publishes its **depth**
(v0.5), which `doctor` prints for the tools it finds on your machine and which the
`coverage` validator reads instead of keeping its own list.

| Tool | Since | Depth | Tokens & cost | Activity (lines/edits/rework) | Attribution (skill / sub-agent) |
|------|-------|-------|---------------|-------------------------------|---------------------------------|
| Claude Code | v0.1 | **deep** | ✔ (incl. sub-agent turns) | ✔ full, incl. rejections | ✔ |
| OpenAI Codex CLI | v0.1 | standard | ✔ (exact, delta-based) | ✔ except rejections; since v0.13 a created file contributes its lines, which it did not before | — |
| Gemini CLI | v0.1 | standard | ✔ | — (cost only, see ROADMAP) | — |
| GitHub Copilot CLI | v0.6 | standard | ✔ (exact, per model, incl. reasoning) | ✔ lines added/removed per session; edit and tool-call counts are in the log but not extracted yet ([audit](docs/extending.md#what-each-sources-log-carries-and-what-assaio-reads)) | — |
| Cline | v0.1 | standard | ✔ (recomputed from tokens) | — (cost only, see ROADMAP) | — |
| Exec parser plugins | v0.1 | declared per record | ✔ (validated at the boundary) | — (protocol carries tokens only) | — |

- **deep** — tokens, per-turn activity, and the labels that say what the work was.
- **standard** — reliable per-turn usage whose activity gaps are documented, not implied.
- **import-only** — billing or aggregate figures that cannot support session-level
  conclusions. No in-tree parser is here today; a source that emits whole-session records
  lands here, which is why granularity travels with every report row.

A plugin is absent from the tiers by design: its depth is whatever its author implemented,
so it is read from the records it emits rather than promised by a table assaio maintains.

Within a tier the answer is per signal, not per column: reasoning tokens are reported by
Codex, Gemini CLI and Copilot CLI and by neither Claude Code nor Cline, a declined tool call
only by Claude Code and a context compaction only by Claude Code and Codex, and Copilot's
per-session totals carry changed lines but no edit, tool-call, turn or rework count. Run
`assaio-agent signals coverage` for what your own mix supports; that matrix row is also what
makes a source syncable and targetable by `clear --tool`.

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
- A figure is computed only over the sources that record its field, states how much of the
  window that reaches, and withholds its verdict rather than averaging in a source's silence
  (v0.10, [ADR 0011](docs/adr/0011-capability-gated-metrics.md)). Since v0.11 that holds for
  rates over stored columns as well as per-session figures, and an out-of-tree metric plugin
  is told each source's capability on the wire (`answers`) so it can apply the same rule.
  What each source's log carries and what is deliberately skipped is inventoried per source
  ([the audit](docs/extending.md#what-each-sources-log-carries-and-what-assaio-reads), v0.10).
- Shares never round in the direction that hides a signal: a small but nonzero share reads
  `<1%` rather than `0%`, and a share just under whole reads `>99%` rather than `100%`
  (v0.11). A whole sub-agent run is never counted as one turn (v0.11).
- Aggregate and pseudonymized by default; per-person is a deliberate, governed opt-in;
  never a leaderboard.
- Plugins (parser, metric, and rule) are config-declared only — never PATH-scanned, never
  downloaded; everything they emit is validated at the boundary and attributed to the
  plugin that emitted it (`plugin:<name>`).
- Schema self-heal: an existing local database migrates itself forward.
