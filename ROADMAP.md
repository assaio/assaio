# Roadmap

**This roadmap is conceptual, not contractual.** It sketches a direction, not a
schedule: no committed dates, no fixed order, and no guarantee that a candidate below
ships as described, or ships at all. `assaio` is pre-1.0 and pre-first-release, and the
most useful input to what comes after v0.1 is real feedback from people running it
against their own repos and teams — not a plan written in a vacuum before anyone has.
Expect this document to change, sometimes a lot, as that feedback arrives. Read
"we're exploring," "likely," and "candidate" literally: they mean exactly that, not a
softer way of saying "planned."

This document is the narrative direction. Two finer-grained companions track the
concrete state: [BACKLOG.md](BACKLOG.md) holds the ranked pool of candidate work items
(ids, effort, honesty caveats), and [FEATURES.md](FEATURES.md) inventories what exists
today and since which release; shipped items graduate through
[CHANGELOG.md](CHANGELOG.md).

One section below is a statement of fact rather than direction: what has actually
shipped in v0.1.

## v0.1 — the offline agent, now (shipped)

v0.1 started as an offline token/cost reporter. Over this cycle it grew into a fuller
offline diagnostic agent, plus a first, honest team-server MVP. This section describes
what is actually in the binary today, not what is planned.

**Coverage and accounting**
- Four tool parsers: Claude Code, OpenAI Codex CLI, Gemini CLI, and Cline.
- Activity extraction — AI lines added/removed, edits, tool calls, compaction,
  within-session rework — for **Claude Code and Codex**. Gemini CLI and Cline
  contribute cost and token counts only, not activity, until their logs get the same
  treatment.
- Correct cost/token accounting, including sub-agent (Task) token usage, which used to
  be invisible — a correctness fix, not a new feature — and exact, delta-based Codex
  accounting read from Codex's own cumulative counters, not estimated.
- Project identity resolved to the git repository root, so a monorepo's subdirectories
  (e.g. `apps/mobile`) roll up into one project instead of fragmenting.

**Diagnostics**
- `effectiveness` — AI output over cost, `$` per 100 AI lines, per project. A
  directional diagnostic, not a performance score.
- `analyze` — a validator framework (adoption, model fit, context health, throughput,
  rework) behind one command. Each validator is a self-registering, independently
  testable file, and the framework is the extension point for community-contributed
  metrics.
- **Exec metric plugins** — your own analyzer out-of-tree, in any language, declared
  under `metrics:` in config: it reads the same prepared data every built-in validator
  reads and renders beside them in `analyze` and the local dashboard, namespaced
  `plugin:<name>`, with `metrics verify` as the conformance tool (ADR 0004). The team
  server deliberately does not execute them.
- Insights — hot / going-stale / dormant-tool / inventory views over the same data.
- Session analytics — turns, peak context, resume-safe active minutes, compaction
  rate, and code-producing vs. conversational session share.
- Rework/churn — the within-session thrash proxy: AI-added lines removed again in the
  same transcript.
- **The Assay** — a self-contained, offline HTML dashboard: light and dark, a
  "how to read" explainer per section, a bounded project drill-down, and an
  i18n-scaffolded template (English only today, but structured for another locale).

**Cost honesty and budgeting**
- Every `$` is disclosed as an **estimate at public pay-as-you-go API prices**, not your
  actual spend — token counts are computed server-side, and a flat-rate subscription
  (Claude Pro/Max, ChatGPT Plus/Pro) makes the effective cost-per-token entirely
  different. `config.pricing` lets a subscription or negotiated-rate user declare their
  real basis (an effective `$/token` or a monthly plan cost), so reports show a truer
  figure alongside the estimate.
- `check` — an exit-code budget gate for CI or a pre-push hook: non-zero when usage
  exceeds a **token** budget (the plan-independent default) or an optional, clearly
  labeled API-equivalent `$` budget.
- Model-routing savings — the `model-fit` litmus turns its premium/cheaper split into a
  directional, **upper-bound** estimate ("premium-tier tokens repriced on the cheapest
  cheaper model ≈ $X/mo"), explicitly a prompt to review, never an auto-applied switch.
- Period-over-period `--compare` on `report` and `effectiveness` — top movers by cost and
  AI lines vs. the previous equal window, so "what changed" leads.
- `demo` — the full reports on bundled sample data, no logs needed, for an instant first
  look before importing your own history.

**Team and sharing**
- A team-server MVP: `serve` runs a self-hosted collection endpoint behind a shared
  bearer token, `sync` pushes an agent's local usage to it, pseudonymized by default
  (`--member` is an explicit opt-in to a real name).
- A per-member Team section on the Assay dashboard, and a team-aware CLI (`--db` to
  point any read command at a central store, `--by member` to group by it) —
  aggregated and pseudonymized by default, the same governed-opt-in posture as the
  rest of `assaio`.

**Robustness**
- Schema self-heal: an existing local database migrates itself forward, no manual
  step.
- User-configurable log-source paths, for non-default install locations.
- Anonymization on by default for anything meant to be shared (the dashboard); the
  interactive CLI still shows real names, since those never leave your machine.

This is still, honestly, an **output and efficiency** measurement, not an **outcome**
one: it says how much AI produced and how efficiently, never whether that code was any
good. Closing that gap is the whole subject of what's next.

## What's next

Everything below is a direction we're exploring, grouped by theme rather than pinned
to a version number — what actually comes next follows from what pilot users ask for,
not from this list's order.

### Deeper impact & quality — the big one

Needs git and issue-tracker correlation, so it needs a server, not the local agent.
v0.1 can tell you AI *produced* 300 lines; it cannot tell you whether those lines:

- **Survived** in `main` after review, rewrites, and reverts — not just that they were
  written.
- **Caused bugs**, measured only against age-matched human code, never a raw
  AI-vs-human split.
- Held up as **quality and maintainable** code, alongside DORA- and DX-style
  engineering metrics.

None of this is honestly answerable from local logs alone — that is the actual reason
a server exists in this project, not a growth-stage default. It would correlate synced
usage against git history first, and an issue tracker next (GitHub to start; GitLab,
Bitbucket, and Jira are candidate breadth once the core correlation works).

### Server hardening

The `serve`/`sync` MVP that shipped this cycle is an honest MVP, and says so in its own
code: one shared bearer token, no TLS, no chunked or resumable sync, no retention
policy — built to run behind a reverse proxy on a trusted network, not exposed to the
open internet. Likely next: TLS or clear reverse-proxy guidance, real per-member auth
and access control, chunked and resumable sync for large backfills, and configurable
retention.

### Cost truthfulness — reconciliation and real rates

v0.1 is honest that every `$` is an *estimate at public pay-as-you-go API prices*, not
your actual spend: token counts are computed server-side and are structurally
unobservable to an offline client, and a subscription (Claude Pro/Max, ChatGPT Plus/Pro)
bills a flat rate that makes the effective cost-per-token entirely different. `config.pricing`
already lets a subscription or negotiated-rate user declare their real basis, and `check`
gates on tokens — the plan-independent primitive — by default. Two directions close the
rest of the gap, both candidates:

- **Vendor billing reconciliation.** Where a vendor exposes actuals (Anthropic's and
  OpenAI's usage/cost APIs), an opt-in reconciliation would pull those aggregates and
  show the estimate-vs-actual delta with a confidence band — the honest close on "is my
  estimate right?", and the real answer to the subscription gap. Strictly opt-in,
  network- and credential-gated; it pulls vendor aggregates only and never uploads logs.
- **Tiered and context-length pricing.** The embedded price table is flat per model; real
  pricing has tiers — long-context premiums (e.g. a `[1m]` 1M-context rate) and distinct
  cache read/write rates a single per-model number can't capture. Modeling those would
  sharpen the estimate for the heaviest, long-context sessions, where it matters most.

### Richer views (candidates from research)

We ran a research pass across the DORA / SPACE / DX Core 4 / METR / Faros literature
asking one question: what else can the same local logs honestly support, with no git
and no issue tracker? Its ranked shortlist is longer than this; these are the
candidates we're most likely to explore first — each still an idea under evaluation,
not a commitment, and each carrying the caveat that keeps it honest:

- **Provenance/confidence coverage meter** — one figure for how much of a given report
  is high-confidence data vs. thin or partial coverage; the honesty backbone the rest
  of this list leans on.
- **Cache-reuse efficiency trend** — cache-read share over time. High reuse is
  cheaper, not "better work"; a big one-shot task legitimately shows low reuse.
- **Session-outcome taxonomy** — conversational / light-edit / heavy-edit / thrash
  buckets. Conversational is not waste — design and debugging sessions are real work.
- **Time-of-day / day-of-week rhythm** — a workload heatmap with an after-hours/weekend
  guardrail. Explicitly never an attendance or "who works late" view.
- **Cost concentration (Pareto/Lorenz)** — which projects drive most of the spend.
  Project-level only, never computed across named people.
- **Model right-sizing & in-session switching** — flags a possibly-overpowered model
  on a small turn. Task difficulty is invisible in logs, so this is a prompt to look,
  not a verdict.
- **Throughput per active hour** — the shipped `throughput` validator tracks raw
  AI-line volume and its week-over-week trend; this candidate would normalize by
  focused time instead. Labeled an activity rate, not a productivity score.
- **Rework/thrash bursts over time** — the shipped rework rate, broken out per-session
  and over time to see where churn clusters. Rework is not failure; healthy iteration
  churns too.

The refusals hold regardless of what ships from this list: no "estimated time saved"
headline (logs contain no counterfactual), no lines-of-code or token leaderboard, and
nothing ranked per named individual, ever.

### More tool coverage

- Activity extraction (AI lines, edits, rework) for Gemini CLI and Cline, closing the
  gap with Claude Code and Codex.
- More exec-plugin connectors — the protocol (ADR 0003) already lets anyone add one
  out-of-tree, in any language, without a core release. Candidates on our radar:
  opencode and Copilot CLI and Factory droid at session granularity, Cursor via its
  Admin API only (its local storage has been verified to lack token counts), and Kiro
  if its logs turn out to carry real token data.

### DX niceties

- The dashboard's i18n scaffolding (one `localeStrings` struct, English only today)
  was built with a language switcher in mind. Whether it grows one depends on whether
  anyone asks.

---

Further out and rougher than the above: usage-anomaly alerts, a weekly digest, an MCP
server for querying your own usage data, an ambient status-line one-liner (today's cost,
burn rate, budget remaining), a CI pull-request comment and job-summary, honest
cohort/percentile context (never a rank, with a minimum cohort size), a stabilized plugin
API for connectors, metrics, and rules, and a Postgres backend once a single SQLite file
stops being enough. A few minor, non-blocking polish items surfaced in the pre-1.0 review
are deferred here rather than gating the release. None of this is designed yet; it's
further-out direction, not a queue.

The core stays Apache-2.0 throughout. One design principle holds regardless of what
above changes: **measure value, not people, by default.** Aggregated and pseudonymized
views are what `assaio` produces out of the box at every stage; a per-person view is
only ever a deliberate, governed opt-in in team mode — never silent, never a
leaderboard, never built for individual performance evaluation.
