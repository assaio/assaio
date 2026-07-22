# Roadmap

**This roadmap is direction, not commitment.** It sketches where `assaio` is headed and
why — no dates, no fixed order, no guarantee that any candidate below ships as described,
or ships at all. `assaio` is pre-1.0, and the most useful input to what comes next is real
feedback from people running it against their own repos and teams. Read "exploring,"
"likely," and "candidate" literally. Expect this document to change, sometimes a lot.

This file is the **narrative** — the themes and the reasoning behind them. Three companions
hold the specifics, and this roadmap deliberately does not repeat them:

- **[FEATURES.md](FEATURES.md)** — what exists today, and since which release.
- **[CHANGELOG.md](CHANGELOG.md)** — the per-release delta.
- **[BACKLOG.md](BACKLOG.md)** — the ranked pool of concrete candidate work items, each with
  an id (`B18`), an effort estimate, and its honesty caveat. Where a theme below names a
  `B`-id, that is where the actionable detail lives.

## The north star

`assaio` answers one question: **is an organization's use of AI coding tools actually worth
it** — in cost, in code that reaches and survives production, in quality, in developer
experience? Today it measures **output and efficiency** honestly: how much AI produced, how
much it cost, how efficiently. It does not yet measure **outcome**: whether that code was
any good. Closing that gap — without ever fabricating a number to do it — is the throughline
of everything below.

## Themes we're pursuing

### 1. Outcome & quality — the big one

The measurements that matter most to the north star are the ones local logs cannot honestly
answer alone. Whether AI-written lines **survived** in `main` after review and rewrites,
whether they **caused bugs** (only ever against age-matched human code, never a raw
AI-vs-human split), whether they held up as **maintainable** work alongside DORA- and
DX-style signals — all of it needs correlation against git history and an issue tracker.
That is the real reason a server exists in this project, not a growth-stage default. The
path we're exploring: a local `survival` MVP first (correlate AI-heavy days against
`git log`/blame in the same repo, heavy error bars, explicitly directional — `B18`), then
server-side correlation proper (GitHub first; GitLab, Bitbucket, Jira as breadth once the
core works). Everything here ships with its error bars or it does not ship.

### 2. Coverage & truth

Two directions, one goal: every tool a team actually uses should be counted, and every `$`
should be reconcilable against reality.

- **More tools.** The activity gap (some tools contribute cost but no line/edit signals) is
  the priority, because closing it multiplies every activity metric. Cline-family logs carry
  the diffs needed for line extraction (`B39`); `opencode` stores structured
  additions/deletions per edit and is the richest next target (`B52`); the Cline forks Roo
  and Kilo Code (`B60`) and the Gemini fork Qwen (`B61`) reuse most of an existing parser;
  Aider (`B14`) and Copilot CLI (`B53`) round out the shortlist. Out-of-tree
  [exec-plugin connectors](docs/extending.md) already let anyone add a source in any
  language without waiting for a core release — a tool used by one organization is usually
  better served that way.
- **Cost truthfulness.** Every `$` is an estimate at public pay-as-you-go API prices, which
  a flat-rate subscription makes structurally different from real spend. Two candidates
  close the gap: opt-in **vendor billing reconciliation** (pull Anthropic/OpenAI usage
  APIs, show the estimate-vs-actual delta with a confidence band — `B19`), and **tiered /
  context-length pricing** (long-context and distinct cache read/write rates a single
  per-model number can't capture — `B16`).

### 3. Richer local views

A large shortlist of metrics can be honestly supported from data already stored — no git, no
issue tracker, no schema change. These are the fast, frequent wins: turn-efficiency
("am I getting more per prompt?"), subscription-fit (is Max/Pro paying off vs API?),
cost concentration across projects, a workload rhythm heatmap (never an attendance view),
session-outcome taxonomy, throughput per focused hour, rework/thrash bursts over time, and a
self-relative "skill curve" progress panel (you vs you four weeks ago, never cross-person).
Each is one self-registering file and each carries the caveat that keeps it honest — see the
`B01`–`B37` cluster in the backlog.

### 4. Team & server hardening

The team server is a deliberate MVP and says so in its own code. Making it fit for more than
a trusted network: TLS or clear reverse-proxy guidance, real per-member auth and access
control, chunked and resumable sync for large backfills, and configurable retention (`B22`).
On top of that, team-shaped views that never become a leaderboard: adoption evenness
(Lorenz/Gini across pseudonymized members with a minimum-cohort guard), tool coverage and
shadow-tool detection, an onboarding curve in aggregated bands, and server-side computation
of metric-plugin results (safely, never per unauthenticated request). Pushing to the server
is already config-driven — `sync` sends to any host with a token — so repointing at a
hardened server, or an eventual **managed cloud**, is a settings change, not a rebuild; that
cloud is where this theme's hardening (per-member auth, retention, TLS) ultimately lands.

### 5. Ecosystem & extensibility

`assaio` is built to be extended, and the extension surfaces should become dependable
enough to build a community on. Directions: a third exec-plugin protocol for **rule** units
(read a window's verdicts, emit alerts with severity — `B13`, ADR 0005 to come); an
in-process Go **plugin API** (`plugin/metric|rule|connector`, no subprocess, no rebuild —
`B24`); a published, versioned freeze of the plugin protocols and the SQLite schema under
semver (`B23`); JSON Schemas and a scaffolder for plugin authors; and a community registry
page once a few plugins exist. The core rule holds: `internal/` never depends on `plugin/`
or `ee/`.

### 6. Developer experience

The daily-habit and integration layer: an ambient status-line one-liner (today's cost, burn
rate, budget remaining), an interactive TUI, an MCP server so you can ask your own usage
questions from an agent, a weekly markdown digest fit for cron, a packaged GitHub Action
(the `check` gate plus a PR comment), shell completions and man pages, data exports
(OpenMetrics for Grafana, ndjson/parquet for data teams), and growing the dashboard's
i18n scaffold into real locales as people ask.

### 7. Scale

Reserved for when the current shape stops being enough, not before: a Postgres backend once
a single SQLite file no longer suffices for a central store (`B25`), and dashboard depth
(multi-window tabs, sparklines, top-N drilldowns) as reports carry more history.

## Principles that don't change

No theme above is allowed to weaken these:

- **Measure value, not people.** Aggregated and pseudonymized views are the default at every
  stage; a per-person view is only ever a deliberate, governed opt-in in team mode — never
  silent, never a leaderboard, never built for individual performance evaluation.
- **Honest statistics or nothing.** Every domain fact carries its provenance and confidence;
  attribution and effectiveness claims ship with their error bars; a directional signal is
  labeled directional.
- **The refusals hold regardless of demand:** no "estimated time saved" headline (the logs
  contain no counterfactual), no lines-of-code or token leaderboard, nothing ranked per
  named individual, and no cohort/percentile comparison without a minimum cohort size and
  explicit consent.
- **The core stays Apache-2.0** throughout; commercial modules, if any, stay isolated under
  `ee/`.

## How we prioritize

Order follows real-world feedback, pull requests, and bug reports — not this document's
sequence, and things not listed anywhere can land first when a PR or a bug report makes the
case. To weigh in: open a feature-request issue or a Discussion, or add weight to a tracked
item by referencing its `B`-id. Connectors additionally follow the
[connector intake flow](docs/extending.md#the-intake-path-open-a-connector-issue-first) —
open an issue with a redacted sample before building, so the format is verified first.
