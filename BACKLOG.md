# Backlog

The ranked pool of concrete candidate work items — the finer-grained counterpart to
[ROADMAP.md](ROADMAP.md)'s narrative direction. **Nothing here is a commitment or a
schedule.** The milestones below are a working hypothesis of order — they mirror the
promises in [ROADMAP.md](ROADMAP.md#the-next-milestones), and the pools after them hold
everything not yet attached to one. Real feedback from people running `assaio` reorders all of it, and any item
can be reshaped or dropped.

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

## Shipped — "Trust the dataset" and "Intent"

Two promises are closed and their items deleted per the lifecycle rule above. Kept here as
one line so a reader knows why the numbering starts where it does: format-drift canaries,
`doctor --strict`, granularity provenance, the confidence envelope on every result, `init`,
the source-depth matrix (`B58`, `B59`, `B69`, `B81`, `B82`, `B83`, all v0.5), and session
annotations with per-kind-of-work stratification (`B80`, v0.6) alongside the GitHub Copilot
CLI parser (`B53`, v0.6). See [CHANGELOG.md](CHANGELOG.md) and [FEATURES.md](FEATURES.md).

## Next — "Evidence Graph: from session to shipped change"

The outcome layer, pulled ahead of recommendations: a suggestion built on activity and
output proxies alone is a guess with a confident voice. First connect a session to the
change it produced — commit, pull request, review, CI, merge, survival — carrying the
confidence and the ambiguity of every link. Everything here ships with its error bars or it
does not ship.

- [ ] **B101 · a sub-agent aggregate's project is decided by parse order** — S/M · solo — a
  completed Claude sub-agent is keyed `agent:<id>` alone, and `usage_record` is
  `UNIQUE(tool, dedupe_key)` with `ON CONFLICT DO NOTHING`, so when the same sub-agent id is
  seen with two different projects the first file ingested wins and the second is dropped
  silently. Found while proving the canonical event contract against 324,416 real records:
  **404** of them collide, identical in tokens, timestamp and session but disagreeing on
  `project`. Cost is neither lost nor double-counted — only the project attribution wobbles,
  on ~0.12% of records — but attribution edges (`B85`) link by project, so this becomes a
  wrong answer rather than a rounding error. Fixing it means deciding what the key should be
  (adding the project changes a shipped dedupe contract) and whether the parser should even
  emit the aggregate twice; the event contract deliberately mirrors today's store behaviour
  rather than diverging from it, so this is one fix in one place, not two.
- [ ] **B102 · AnalyzerContext: retire the store types from the analyzer surface** — L · both
  — the half of `B90` deliberately not shipped with the catalog. Validators already do not
  query SQLite; what leaks is *types*, since `analyze.Input` carries `[]store.UsageRow`,
  `[]store.SessionRow` and `[]store.AttributionRow`, and those same shapes are the published
  metric-plugin wire (ADR 0004). Replacing them is therefore a breaking change to a public
  surface for no user-visible gain, so it waits for a real second consumer to design against
  — the git evidence collector (`B91`) — rather than being guessed at now. Ships with a
  deprecation window and a conformance fixture, and is a candidate to land with the protocol
  freeze (`B23`) rather than before it. See [ADR 0008](docs/adr/0008-signal-catalog.md).
  **Three requirements came out of building the git collector** rather than being guessed at:
  (1) a validator receives `[]store.UsageRow`, so there is no shape in which it could read a
  commit observation at all — the context has to serve *heterogeneous* observation streams
  keyed by type, which is a bigger job than swapping the row types; (2) `signals coverage`
  weights support by tokens (`analyze.TokensByTool`), so a source that has no tokens cannot
  appear in it, which is why no `vcs.*` signal is registered yet and what a presence notion
  would have to replace; (3) capability lives on `parser.Answers`, and a collector is not a
  parser — the registry has to move out of `internal/parser` or learn about non-parser
  sources before the first `vcs.*` signal is declared.
- [ ] **B103 · commit observations that outlive one pass** — M · both — the half of `B91`
  deliberately not shipped: a configured repository list (rather than one `--repo` at a time),
  a keyed pseudonymous commit digest for the syncable case, and durable storage. Each waits
  for the consumer that decides its shape — `B85` needs commits queryable *across* passes to
  link a session to one, `B100` decides what identity may leave the machine — and each costs a
  migration, a size bound and a cleanup path, so none is worth guessing at. Path-level storage
  stays out entirely until something needs it: `B91` never records a path, so there is no
  opt-in to design yet. See [ADR 0009](docs/adr/0009-local-git-evidence-collector.md).
- [ ] **B104 · the usage→event adapter still has no caller** — S/M · solo — `deadcode
  ./cmd/...` reports seven unreachable functions, all of them the AI half of the canonical
  contract: `event.FromRecord`, `Event.with`, and both `Usage` and `Edit` payloads. `B91` gave
  `internal/event` its first real consumer, but only for `vcs.commit.observed`; nothing in the
  binary ever adapts a parsed usage record into an observation. Two things hang on this rather
  than on taste. The error posture is still split — the collector skips and counts, while
  `FromRecord` fails a whole batch on the first rejected record — and [ADR 0009](docs/adr/0009-local-git-evidence-collector.md)
  says explicitly that having both is not defensible and the log parsers' skip-and-count is
  the precedent; that is a decision the adapter's first caller gets to make. And the contract's
  claim that an analyzer reads AI usage and git commits the same way is currently unproven for
  half of it. The likely first caller is `B85`, which needs sessions and commits in one stream.
- [ ] **B92 · GitHub connector v1** — L · both — read-only, least-privilege metadata only:
  PR lifecycle, commits in a PR, review states and requested-changes rounds, check suites and
  runs, merge time and method, detectable revert relations. GitHub Cloud first, with the
  interface shaped so Enterprise Server and GitLab can follow without contaminating the core
  model. Credentials never reach reports or plugins.
- [ ] **B85 · attribution engine + edges (session → commit / PR)** — L · both — versioned,
  confidence-bearing candidate and confirmed edges carrying their method (explicit marker,
  repo/branch match, bounded temporal proximity, file-category compatibility, identity
  compatibility, parent/child session, manual confirmation), the alternatives they beat, and
  their ambiguity status. Never force one session onto one PR; many-to-many stays
  many-to-many; an unattributed session stays unattributed; user correction wins and survives
  replay after an algorithm change. **Inherits from B80**: the optional issue-id or branch
  reference was deliberately cut from session annotations rather than shipped as a bare
  string, because a session→change pointer is a claim that must carry its method and
  confidence, not a category label — see [ADR 0006](docs/adr/0006-session-annotations.md).
  **The conformance corpus (`B93`, shipped) is the specification to build against**, and
  writing it surfaced a requirement that was not obvious: "identity compatibility" is not an
  available method today, because a commit observation carries no author at all. The
  `overlapping-users` scenario therefore *requires* ambiguity, and separating those sessions
  means changing what an observation carries — a privacy decision belonging with `B100`,
  not a better ranking. See [ADR 0010](docs/adr/0010-attribution-conformance-corpus.md).
- [ ] **B94 · `outcomes` funnel + `evidence explain`** — M · both — the first visible path:
  sessions → sessions with edits → linked commits → linked PRs → passing CI → merged →
  surviving, sliced by tool, model, project, task annotation and confidence band, always
  showing the unattributed and insufficient-evidence share. `evidence explain <edge>` says why
  a link was or was not made. No causal language anywhere in it.
- [ ] **B18 · survival: per-day correlation + age-matching** — M · both — per-day AI-heavy
  vs quiet-day survival comparison, an age/settle threshold so recent commits are not counted
  as "survived", and rename-following in blame. Only ever age-matched.
- [ ] **B21 · test-touch** — M · both — share of AI edits touching test files via
  privacy-safe category counts (test/source/docs/config) classified at parse time — paths are
  never stored. Needs a PRIVACY.md note. The first quality-adjacent signal without a server,
  and deliberately a floor rather than a verdict: research finds agent-written tests lean on
  mocks more than human ones, so "tests were touched" is not "the change was verified".
- [ ] **B100 · privacy threat model for correlation** — S/M · both — correlation creates
  risks the local-only store did not have: commit and PR metadata reveal work patterns,
  branch and label names can leak client or project identity, timing correlation slides toward
  employee monitoring, and combined datasets can re-identify a pseudonym. Ships with the
  connector, not after it: local-only defaults, field-level sync policy, retention, minimum
  cohort for any server view, and a test asserting no ranking surface exists.

## After that — "Harness intelligence & verified improvement"

Agent configuration — `AGENTS.md`, rules, skills, sub-agents, hooks, MCP servers — is a
versioned engineering input, not magic that is assumed to help. Once outcomes are linkable,
the question "did changing it actually improve anything" becomes answerable, and a
recommendation can rest on outcome rather than on activity.

- [ ] **B95 · harness inventory v1** — M · both — detect the configuration artifacts a
  repository carries and store only their shape: type, scope, keyed hash/version, size band,
  modified time, tool compatibility, and whether the artifact was observably invoked. Content
  is **not** stored by default — an `AGENTS.md` is a document, and this is an inventory.
- [ ] **B96 · harness-to-outcome cohorts** — L · both — compare repository or time cohorts
  before and after a harness version changed: token and cache profile, retry and failed-tool
  behaviour, time to first edit and first test, CI-repair burden, review tax, delivery yield,
  survival. Every result states sample size, confounders and attribution confidence, and is
  labelled an observation, never proof.
- [ ] **B84 · deterministic recommendations** — M/L · both — one concrete next step per
  finding: rules over verdicts and outcomes emitting observed pattern, evidence, confidence,
  one action, the follow-up metric and a review window; dismissable, and the dismissal sticks.
  No LLM anywhere in the decision path — an LLM may later only explain a suggestion that
  already exists. Abstains when outcome coverage is inadequate; that abstention rate being
  *too low* is a warning sign, not a win.
- [ ] **B97 · experiment framework v1** — L · both — what turns a recommendation into
  evidence: named hypothesis, target cohort, baseline window, treatment version, success and
  guardrail metrics, a minimum-observation requirement, pre/post or alternating-period mode,
  and a result carrying its confidence and caveats. Nothing mutates a config automatically.
- [ ] **B87 · pre/post window comparison** — M · both — the measurement half of B97 on its
  own: compare a metric across matched windows around a deliberate change, with sample-size
  and seasonality warnings and mandatory association-not-causation framing.
- [ ] **B20 · compaction-recovery-cost** — M · both — tokens spent in the turns right after
  a compaction vs baseline: the true price of overflowing context.
- [ ] **B17 · progress ("skill curve")** — M · solo — you vs you four weeks ago across
  adoption breadth, turn-efficiency, cache-hygiene, rework — a small panel of deltas,
  deliberately not a single score. Strictly self-relative, never cross-person.
- [ ] **B44 · read-only MCP/query interface** — M · solo — ask your own usage questions from
  an agent: source coverage, recent cost and workflow changes, active experiments, the
  evidence behind a finding. Never exposes prompts, code, secrets, or — in anonymized mode —
  project names.

## Then — "Team adoption, without surveillance"

A team self-hosts it, sees cross-tool adoption and delivery outcomes, and cannot build a
leaderboard from it.

- [ ] **B22 · server hardening** — M · team — TLS/reverse-proxy guidance, per-member tokens
  and roles, retention and deletion policy, chunked/resumable sync, health and migration
  checks, backup/restore, bounded ingestion queues.
- [ ] **B09 · team-evenness** — M · team — Lorenz/Gini spread of usage across pseudonymized
  members with a minimum-cohort (≥5) guard. "Broad adoption, or two power users?" — never a
  ranked list.
- [ ] **B10 · tool-coverage** — S · team — members per tool on a central store; shadow-tool
  and unused-seat detection, only where the evidence is explicit.
- [ ] **B40 · onboarding-curve** — M · team — usage growth vs weeks-since-first-sync, in
  aggregated bands, pseudonymized.
- [ ] **B41 · team efficiency spread** — M · team — distribution bands of turn-efficiency
  signals across members; "the team needs prompting practice" without naming anyone.
- [ ] **B25 · Postgres backend** — L · team — once a single SQLite file stops being enough
  for a central store.

## Later — "Cost truth, policy & interoperability"

Every `$` today is an estimate at public pay-as-you-go prices, which a flat-rate
subscription makes structurally different from real spend — and assaio's own model should
speak a standard rather than only its own dialect.

- [ ] **B19 · vendor billing reconciliation** — M/L · both — opt-in pull of Anthropic/OpenAI
  usage/cost APIs; estimate-vs-actual delta with a confidence band and an unexplained-delta
  report. Network- and credential-gated; pulls vendor aggregates only, never uploads logs.
- [ ] **B16 · context-utilization** — M · both — vendored model context-window table (like
  the price table) → peak context vs model limit, near-limit share, and honest right-sizing
  hints. Prerequisite for pricing long-context and cache tiers instead of one flat rate. For
  Codex the table is unnecessary: the log states `model_context_window` on every `token_count`
  (`B114`), which is a stronger source than a snapshot assaio maintains.
- [ ] **B98 · OpenTelemetry GenAI mapping + content-free ingest** — M/L · both — publish a
  field mapping between assaio's canonical events and the OTel GenAI semantic conventions
  (model, tokens, tool calls, agent and operation identity, session), then ingest only the
  subset needed to prove it against a Claude Code telemetry fixture. Prompt, completion, tool
  input and tool result attributes are **dropped by default**, and the dropped set is
  documented rather than implied. Opens a path to sources whose local files cannot be parsed.
- [ ] **B99 · connector SDK + conformance kit** — M · both — generated scaffold, JSON
  Schemas, typed SDKs for a couple of languages, a fixture redaction helper, a depth-manifest
  validator, parser-drift tests and a privacy manifest. What makes a community connector
  possible without touching core packages. Supersedes the "more parsers in core" reflex.

## v1.0 — "Stable open evidence platform"

The one milestone that keeps a version number, because there the number *is* the promise:
`v1.0` is the semver stability guarantee.

Plugin authors and any future managed cloud can build on it without breakage.

- [ ] **B23 · protocol & schema freeze** — M · both — declare the exec plugin protocols
  (parser, metric, rule), the canonical event and signal contracts, the sync API and the
  SQLite schema stable under semver, with conformance fixtures and deprecation windows.
- [ ] **B24 · in-process plugin API (research first)** — L · both — the mechanism is **not**
  decided, and native Go plugins are explicitly not a v1 requirement: they need cgo, do not
  work on Windows, and require host and plugin built with identical toolchain and dependency
  versions, which contradicts shipping one static binary per platform. Evaluate a sandboxed
  WebAssembly/WASI component instead — it brings capability limits and resource budgets — and
  keep exec plugins as the universal baseline either way. Deliverable is a decision record,
  not code.

## Next — "Everything the logs already say"

Depth before breadth, and before correlation. This milestone is the half of the product that
needs no server, no credential and no repository — and the half every other conclusion rests
on, since a link to a merged pull request is only as good as the session it links.

- [ ] **B107 · Codex cache-write tokens are never read** — S · solo — the audit's clearest
  gap: `payload.info.total_token_usage.cache_write_input_tokens` is reported on every Codex
  `token_count` and `usage.Record` gets no value for it, so Codex cost is a floor rather than an
  estimate and `cache-hygiene` cannot see a Codex cache write at all. One field on the token
  struct, one delta, one golden — and a `backfill --full` to restate history.
- [ ] **B108 · Claude's cache is explainable, not opaque** — M · solo — two fields the audit
  found on every assistant turn: `message.usage.cache_creation.ephemeral_5m_input_tokens` /
  `ephemeral_1h_input_tokens` (the TTL tier a write bought) and
  `message.diagnostics.cache_miss_reason.type` (a six-value vocabulary: `messages_changed`,
  `model_changed`, `previous_message_not_found`, `system_changed`, `tools_changed`,
  `unavailable`). `cache-hygiene` currently ships the caveat "vendor cache TTLs are invisible"
  (`B02`); they are not. Turns a cache-read share into a cause a person can act on, and the
  1-hour tier is priced differently from the 5-minute one, so it also sharpens cost.
- [ ] **B109 · Copilot CLI is deeper than its matrix row** — M · both — the audit found named
  tool calls (`data.toolRequests[].name`), per-call line counts
  (`data.toolTelemetry.metrics.linesAdded`/`linesRemoved`), per-model request counts
  (`data.modelMetrics.<model>.requests.count`), a published cache TTL
  (`data.modelCacheState[].cacheTtlSeconds`) and sub-agent parentage
  (`data.parentAgentTaskId`) — every one of them a signal its depth row currently declares it
  cannot answer. Per-call lines are the interesting half: they would end the "credited whole to
  the model with the most requests" compromise. The corpus behind this is three sessions, so
  verify against a real one before writing the row.
- [ ] **B110 · Gemini CLI: the parsed shape is missing from a current install** — S/M · solo —
  the two files matching the discovery glob on the audited machine contain no token field at
  all (`files=2 records=0`), and the wider `~/.gemini` tree carries a different vocabulary
  (`CHECKPOINT`, `PLANNER_RESPONSE`, `CONVERSATION_HISTORY`) with no `tokens` object anywhere.
  Either the recording moved or these files were never the token source; settle which against a
  fresh session before touching the parser. **The second half is ours, not Gemini's**: a source
  this small is invisible to every drift canary, since each needs a 20-file sample floor. A
  source that has always had a handful of files needs a canary that can fire on "discovered
  files, parsed zero records", which none of the four does.
- [ ] **B111 · the human correction, recorded rather than proxied** — S · solo —
  `toolUseResult.userModified` marks an edit the person changed after the AI wrote it, on a
  meaningful share of Claude edit results. `rework` today infers correction from add-then-remove
  churn within a transcript and says it is a proxy; this is the thing itself. Ships as its own
  signal beside `ai.rework.lines`, never folded into it — they mean different things.
- [ ] **B112 · attribution beyond skill and sub-agent, and the build that wrote the line** — M ·
  solo — Claude stamps `attributionPlugin` and `attributionMcpServer`/`attributionMcpTool` on a
  large share of turns, which would widen `skill-economics` from two dimensions to four and give
  MCP servers a cost of their own. Alongside it, `version` on every line is the harness-version
  cohort input `B96` assumes needs a server — it is already on disk, offline, per turn.
- [ ] **B113 · how a turn ended** — S/M · both — three fields for one signal: Claude's
  `message.stop_reason` (`max_tokens` marks a truncated answer), `toolUseResult.interrupted` (a
  command the human cut short), and Codex's `turn_aborted` with `reason: interrupted`. Together
  they distinguish a turn that finished from one that was stopped, which every efficiency figure
  currently averages together. Codex's `changes.<path>.{type,move_path}` belongs here too: an
  add, an update and a rename are one undifferentiated edit today.
- [ ] **B114 · Codex states its own limits** — M · both — `payload.info.model_context_window` is
  the model's context limit on every `token_count`, which is what `B16` proposes to vendor a
  table for; and `payload.rate_limits.{plan_type,primary.used_percent,primary.resets_at,credits}`
  is the only place any source says how close a session ran to a plan's ceiling. That is the
  subscription question `subscription-fit` answers from a configured number today, answered
  from the vendor's own accounting instead. Both are per-session state rather than usage, so the
  storage shape needs deciding first.

## Pool — validators from data already stored (unscheduled)

- [ ] **B28 · rhythm: day × hour heatmap** — S · both — the shipped `rhythm` validator
  reports off-hours/weekend share, four time-of-day bands, and p95 focused minutes; still
  to add the full day-of-week × hour heatmap. Explicitly never an attendance view.
- [ ] **B29 · session-taxonomy: thrash + trend** — M · both — the shipped `session-taxonomy`
  validator buckets conversational / light-edit / heavy-edit; still to add a thrash bucket
  (needs per-session rework, not stored yet) and the mix's week-over-week trend.
- [ ] **B30 · delegation: trends and per-session economics** — S · solo — `sidechain` now
  marks sub-agent turns exactly and `skill-economics` reports per-agent tokens and lines;
  still to add the delegation-share trend and tokens per delegated vs main-loop session.
  Task difficulty is invisible — a prompt to look, not a verdict.
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
- [ ] **B36 · marathon: long- vs short-session efficiency** — S · solo — p95 session
  active minutes ships in `rhythm`; still to compare output and thrash between long and
  short sessions, since thrash correlates with marathons.
- [ ] **B37 · branch-mix** — S · both — AI lines on the default branch vs feature
  branches; a process signal (direct-to-main AI work), local only.
- [ ] **B02 · cache-hygiene trend** — S · both — add the day-over-day cache-read-share
  trend to the shipped `cache-hygiene` validator, which already reports the current-window
  share and the cache-write-waste flag. The caveat this item used to carry — "vendor cache TTLs
  are invisible" — turned out to be false: Claude splits a cache write by TTL tier and Copilot
  publishes the TTL in seconds (`B105` audit, now `B108` and `B109`). Day-grain approximation
  stands.

## Pool — needs a schema or parser extension

- [ ] **B79 · local-day bucketing** — M · both — `UsageRow.Day` is `substr(ts,1,10)` over
  timestamps normalised to UTC, so every day-based figure (active days, lines/active-day,
  week-over-week, burn-anomaly's baseline) counts UTC calendar days while `rhythm` reads
  session starts in local time. Blocked on data, not effort: Claude Code and Codex both log
  `Z` timestamps, so the offset a session actually happened in **does not exist in the
  source** and cannot be recovered for history. Converting to the reporting machine's zone
  would help a solo user but make the team server arbitrary (bucketing every member in the
  server's zone) where UTC is at least uniform. A real fix needs the tools to log an offset,
  or an explicit reporting-timezone setting with the trade-off stated. Disclosed meanwhile
  as caveats on `burn-anomaly` and in `doctor`.
- [ ] **B78 · sub-agent tool-call split** — S · solo — a Claude sub-agent's aggregate record
  keeps `ToolCalls=0` and an all-zero purpose split, so its `toolStats` read/search/bash/edit
  counts never reach `explore-produce`. Populating them means also setting `ToolCalls`, which
  today would double-count against the parent turn — needs the accounting decided first. The
  `B105` audit named the fields and confirmed they are always present together:
  `toolUseResult.toolStats.{readCount,searchCount,bashCount,editFileCount,otherToolCount}`
  beside the `linesAdded`/`linesRemoved` already read, plus `totalToolUseCount`. Nothing is
  missing but the decision.
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

## Pool — team

- [ ] **B42 · server-side exec metrics** — M · team — lift the ADR 0004 non-goal
  safely: compute metric-plugin results on sync-write or a TTL cache, never per
  unauthenticated request.
- [ ] **B12 · GitHub Action** — M · team — packaged action running `check` as a gate
  plus a PR comment with movers/effectiveness for the changed window.

## Pool — CLI & DX

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
- [ ] **B06 · metric-plugin scaffolder + schemas** — S · both — `metrics init --lang
  python|node|sh` writes a working plugin skeleton; publish JSON Schemas for the
  metric envelope/result under `docs/schemas/`. Lowers the barrier the moment the
  protocol is public.
- [ ] **B08 · a second locale** — S · both — add one `i18n.Catalog` beside `en` and a case in
  `i18n.For`, proving the scaffolding with a real language. v0.4 built the catalog (dashboard
  chrome, statusline words, the 18 explain pages); what remains is translating it and choosing
  how a locale is selected. Data-derived validator text stays out of scope: translating it
  needs message templates for the interpolated numbers.
- [ ] **B11 · weekly digest** — S/M · both — `digest --weekly`: markdown summary
  (top movers, verdict changes, anomalies) fit for cron/launchd; delivery stays the
  user's own script (mail, Slack, …).
- [ ] **B86 · contributor entry points** — S · both — public issues cut from the top
  backlog items, labels (`good first issue`, `help wanted`, `connector`, `metric`,
  `parser-drift`, `research-needed`, `privacy-review`), Discussions categories, and a short
  architecture map showing where a parser, metric, rule, connector or dashboard change belongs.

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
- [ ] **B54 · Factory droid** — M — session-granularity local logs (per ROADMAP).
- [ ] **B88 · Antigravity — activity-only, no cost** — M · both — research verified
  (2026-07-31), and the answer is unusual enough to record: Google Antigravity stores a lot
  locally and **no token counts anywhere**. Its CLI keeps one SQLite database per
  conversation under `~/.gemini/antigravity-cli/conversations/<uuid>.db` (175 of them, 191 MB
  on the inspected machine) whose every payload column — `steps.metadata`,
  `steps.step_payload`, `gen_metadata.data` — is an **undocumented protobuf blob**; no column
  or field named `token` exists in any of them, nor in `log/`, nor in `brain/`. What *is*
  readable is `conversation_summaries.db`: `conversation_id`, `step_count`,
  `last_modified_time`, `last_user_input_time`, `workspace_uris` (→ project), `agent_name`,
  `status`, `nesting_depth`. That is enough for adoption and rhythm signals and **nothing
  else** — no cost, no model, no lines. Two consequences to settle before building: a
  token-less source would report zero tokens forever, which trips the B58 zero-token canary
  and would need the canary to read a source's declared depth rather than assume tokens; and
  under B83 it fits no current tier, so the matrix needs an `activity-only` row or an honest
  reason to refuse the source. Decision record first, code second.
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
- [ ] **B14 · Aider connector** — M · both — in-tree parser. The parseable token source is
  the opt-in `~/.aider/analytics.jsonl` (`message_send` events with `properties.{main_model,
  prompt_tokens,completion_tokens,total_tokens,cost}`, `time` in epoch seconds, no session
  id or cache split); `.aider.chat.history.md` is markdown-only. No structured per-edit field
  — Aider auto-commits, so lines +/- come from git, not the logs.

## Pool — code health (from earlier reviews)

Deferred cleanups surfaced by the max-effort review of the 0.2 work; no behavior change, so
they wait behind features but keep the growing metric surface maintainable.

- [ ] **B73 · validator meta** — S · both — embed a `meta{name,title,describe,howToRead}`
  value in each validator so the eighteen metrics stop repeating the three trivial interface
  methods and the duplicated `Result` header; one source of truth per metric. The long-form
  explain text is deliberately *not* part of this: it lives in the i18n catalog so it can be
  translated, and keeping it there is what let `internal/analyze` stay free of any
  presentation dependency.
- [ ] **B74 · member aggregate** — S · team — collapse `dashboard/team.go`'s four parallel
  per-member maps (`lines`/`cost`/`hasCost`/`unpriced`) into one `map[string]memberAgg`, so a
  member's totals travel as a unit.
- [ ] **B75 · humanize helpers** — S · both — `internal/humanize` landed in v0.4 with
  `analyze`'s two formatters (`compactCount`, `money`) moved onto it byte-identically. Still
  to fold in: `report.formatCompactTokens` and `dashboard.formatCompactUSD`, which round and
  case differently ("1.0K" vs "1k"), so unifying them *changes rendered output* and needs a
  deliberate decision plus golden updates -- which is why v0.4 left them alone.
- [ ] **B77 · readWhenEnough** — S · both — factor the "gate the favorable read behind a
  minimum-sample floor, else neutral" block shared verbatim by session-taxonomy,
  turn-efficiency, and model-right-sizing into one helper beside `readFor`.
- [ ] **B106 · three files past the size budget outside the analysis packages** — S · both —
  `internal/plugin/metric_input.go` (217), `internal/i18n/en_explain_work.go` (221) and
  `internal/cli/analyze.go` (208) each carry more than one responsibility at more than the
  ~200-line budget. The i18n one is the interesting case rather than the largest: it is a
  block of prose per metric, so splitting it by metric group is a different judgement from
  splitting code, and doing it badly makes the catalog harder to translate (`B08`), not
  easier.

## Refusals (will not build, regardless of demand)

- No "estimated time saved" headline — the logs contain no counterfactual.
- No lines-of-code or token leaderboards; nothing ranked per named individual, ever.
- No per-person analytics outside a deliberate, governed team-mode opt-in.
- No cohort/percentile comparisons without a minimum cohort size and explicit consent.
