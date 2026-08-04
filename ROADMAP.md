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

What that makes `assaio`, stated once so the rest of this document has a subject: **the
open-source, privacy-first evidence layer for AI-assisted software engineering — reading
what the agents on your machine already recorded, and connecting it to commits, review, CI
and durable outcomes.**

Those are two halves and they are not equal. Correlation is what makes an outcome claim
possible; **the analysis of the logs themselves is what makes any of it worth reading**, and
it is the half that has to be strongest. A perfect link from a session to a merged pull
request says nothing useful if the session was parsed shallowly, if a format change silently
halved the numbers, or if the tool read three fields out of a log carrying twenty. The local
half also has properties the correlated half never will: it works with no server, no
credentials, no network and no repository access, it is where the majority of users will
stay, and it is the part a competitor cannot copy by wiring up one more API. So the order of
priority is: read the logs exhaustively and prove the reading is correct, then connect what
was read to what shipped. Both get built; when they compete for attention, depth of analysis
wins.

That framing is a deliberate correction. `$ per 100 AI lines` is a useful ratio and it is
staying, but it was carrying too much weight as the headline answer to "is AI delivering".
Lines are an **output** measure (see the four layers below); promoting one to an outcome
claim is the most likely way this project could start lying. It is also the part every
first-party vendor dashboard already does for its own tool — where a cross-vendor,
local-first tool has something nobody else does is the evidence *between* AI activity and
what shipped.

## Four layers, never relabeled

Everything `assaio` reports sits on exactly one of four layers, and the product says which:

- **Activity** — tokens, turns, tool calls, edits. What happened.
- **Output** — changed lines, commits, PRs, tests. What was produced.
- **Outcome** — merged, survived, passed CI, review burden, reverted. Whether it held.
- **Impact** — value to a user or a business. Needs context `assaio` does not have.

The `$/100 AI lines` headline lives on **output**, and that is the whole reason it is framed
as cost efficiency rather than productivity. Lines reward greenfield and boilerplate, punish
debugging and analysis, say nothing about tests, review rounds, or whether the code shipped —
and can grow with unnecessary complexity. Promoting an output metric to an outcome claim is
the most likely way this project could start lying, so the layer label is not decoration.

## The next milestones

The themes below are the direction; these are the next **promises to a user**, each with what
would have to be true to call it done. Same caveat as everything else here: no dates, and a
milestone can be reshaped or skipped. One maintainer, pre-1.0 — so each is deliberately small
enough to ship and prove on real data.

A milestone deliberately carries **no version number** until it ships. The table's order is
the sequence we intend to attempt, shipped rows name the release they landed in, and the
first unshipped row is what is being worked on now — but a small release can land between
two of them, and pre-assigning `v0.7` to a promise would make this document schedule work it
explicitly says it does not schedule. `v1.0` is the exception, because there it is the
promise: the semver stability guarantee itself.

The first two unshipped rows are the exception to "one at a time": **evidence graph** and
**everything the logs already say** run in parallel, because they improve different things
and one of them is the floor under the other. When the two compete for a release's attention,
depth of analysis wins — see [why](#why-local-analysis-is-the-stronger-half-not-the-smaller-one)
below.

| Milestone | Promise | Done when |
|---|---|---|
| **Trust the dataset** ✓ | Before `assaio` advises anything, it shows what it read, what it missed, and how sure it is | shipped in v0.5: every validator carries coverage + confidence as data, format drift is detected instead of silently under-reporting, `doctor --strict` fails a cron job, every source publishes its real depth, and a first run needs no config (`B58`, `B59`, `B69`, `B81`, `B82`, `B83`) |
| **Intent** ✓ | Metrics can be read per kind of work | shipped in v0.6: a session can be labeled task class / outcome / difficulty, any metric stratified by it, and unlabeled data stays fully counted (`B80`) |
| **Evidence graph** | You can see which AI sessions produced commits and pull requests, what happened in review and CI, and how sure `assaio` is about every link | the canonical event contract ([ADR 0007](docs/adr/0007-canonical-event-contract.md)) and the signal catalog ([ADR 0008](docs/adr/0008-signal-catalog.md)) exist, both shipped; local git commit metadata is a content-free observation, shipped, and GitHub PR/review/check metadata is not yet; the conformance corpus that defines what an honest link is, shipped ahead of the engine ([ADR 0010](docs/adr/0010-attribution-conformance-corpus.md)); attribution edges carry method, confidence, alternatives and ambiguity; ambiguous stays ambiguous; `outcomes` shows the funnel with its unattributed share and `evidence explain` says why a link was or was not made (`B92`, `B94`, `B85`, `B18`, `B21`, `B100`–`B104`) |
| **Everything the logs already say** | Without a server, a repository or a credential, `assaio` reads every signal the local logs carry, proves the reading is right, and turns it into more conclusions than any vendor dashboard draws from the same file | every source's unread fields are inventoried and either extracted or documented as deliberately skipped; the activity gap closes where a log carries the data (`B39`, `B72`); the local-view metrics land as one file each with their caveats (`B28`–`B37`, `B02`, `B78`, `B79`); and a parser proves itself against captured real-world samples rather than only against fixtures, so a format change fails a test before it reaches a report (`B105`, `B20`) |
| **Harness intelligence** | You learn which agent configuration and workflow changes actually helped, and each finding comes with one reversible experiment | agent config artifacts are inventoried without storing their content; cohorts can be compared before and after a harness version changed; recommendations are deterministic rules carrying evidence, one action, a follow-up metric and a review window, and abstain when outcome coverage is thin — never an LLM narrating a dashboard (`B95`, `B96`, `B84`, `B97`, `B87`, `B17`, `B44`) |
| **Team, without surveillance** | A team self-hosts it, sees adoption and delivery outcomes, and cannot build a leaderboard from it | authenticated, resumable sync with retention and roles; adoption read as activation → retention → breadth, in bands with a cohort floor (`B22`, `B09`, `B40`, `B41`) |
| **Cost truth & interoperability** | The `$` figure can be reconciled against what was actually billed, and the data speaks a standard | opt-in vendor billing reconciliation shows the estimate-vs-actual delta with a confidence band; pricing models long-context and cache tiers; canonical fields map to OpenTelemetry GenAI conventions with content dropped by default; a connector SDK makes a community source possible without touching core (`B19`, `B16`, `B98`, `B99`) |
| **v1.0 · Stable platform** | Plugin authors and a managed cloud can build on it without breakage | plugin protocols, the event and signal contracts, the sync API and the schema frozen under semver, with conformance fixtures (`B23`) |

### Why local analysis is the stronger half, not the smaller one

It would be easy to read the milestone table as "the local part is done, now for the
interesting work". It is not done. Every source publishes a depth row precisely because each
one leaves something on the table, and the honest reading of that matrix is that `assaio`
today extracts a fraction of what these logs contain — a Codex rollout, a Claude transcript
and a Cline task directory each carry structure no report has ever rendered. Closing that is
cheap, needs nobody's permission, ships as one self-registering file at a time, and improves
every existing figure rather than adding a new page beside them.

It is also where the honesty rules bite hardest. Correlation is the part that most obviously
*could* fabricate, so it gets the ADRs and the conformance corpus — but a shallow parse
fabricates too, more quietly: a metric computed over a field that turned out to mean
something else looks exactly like a metric that works. That is why this milestone's "done
when" includes proving the reading against captured real samples, not only counting new
signals.

The two halves reinforce each other in one direction more than the other. Better parsing
makes every attribution method better, because a link is only as good as the activity it
links. A better attribution engine does not improve the parsing at all.

### Why outcome attribution moved ahead of recommendations

The earlier plan reached recommendations and team adoption before connecting a session to
the change it produced. That order is backwards for this product: a recommendation resting
on activity and output proxies alone is a guess delivered in a confident voice, and the one
thing `assaio` cannot afford is manufactured certainty. Linking a session to a commit, a
pull request, review and CI is also the part that first-party vendor dashboards structurally
cannot do — each sees only its own tool — so it is where a cross-vendor, local-first tool has
something to say that nobody else does. Recommendations get materially more credible once
they can name an outcome rather than a proxy, which is why they follow rather than lead.

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

Two things have to exist before any of it is honest, and both are now tracked:

- **Attribution as evidence, not a foreign key** (`B85`). A session-to-commit or
  session-to-PR link is a claim, so it carries its method (explicit marker, branch, temporal
  proximity, issue id, inferred), its confidence, and the alternatives it beat. One session
  is never silently forced onto one PR, ambiguous stays ambiguous, and re-running attribution
  after new git data arrives must never mutate the underlying usage events. Outcome metrics
  then pick a minimum confidence rather than pretending links are facts.
- **Task intent** (`B80`, shipped). "Did AI-written code survive" means nothing averaged over
  research, a refactor and a greenfield feature. Intent is not in the logs and will not be
  recovered by reading prompts; it comes from an explicit, optional label — `assaio-agent mark`,
  three closed vocabularies, local only ([ADR 0006](docs/adr/0006-session-annotations.md)).

### 2. Coverage & truth

Four directions, one goal: every tool a team actually uses should be counted, every `$`
should be reconcilable against reality, and a report should state both its own limits and
the real depth of each source it read.

- **Confidence as a first-class field** (`B81`, shipped v0.5). A validator's limits travel as
  structure rather than as prose a reader can skip — activity, pricing and turn-level
  coverage, the sample it rests on, when the data was read and by which build — so a
  low-coverage verdict cannot be quoted as a solid one, and so `check` and the recommendation
  engine (`B84`) have a floor to refuse to fire below. Deliberately **not** collapsed into one
  opaque score: a display label may summarize, the components stay inspectable. What remains
  is extending the same envelope to outcome evidence, where attribution confidence becomes a
  fourth axis.
- **A depth matrix instead of a supported/unsupported list** (`B83`, shipped v0.5). "Supports
  Gemini CLI" is misleading when that means tokens but no edits. Every source publishes its
  per-field depth in three tiers — **deep** (tokens + activity + attribution), **standard** (reliable usage,
  documented activity gaps), **import-only** (billing or aggregate data that cannot support
  session-level conclusions) — plus parser freshness and known gaps. Tool count is not the
  goal; high-confidence signal coverage is.
- **Depth over count.** Connector count is explicitly *not* the scoreboard, and chasing it
  would be a losing race: an OSS competitor already advertises tracking across dozens of
  tools, and a shallow integration that contributes cost but no activity makes every activity
  metric weaker, not stronger. The strategy is a small set of **deep** sources maintained in
  core, **standard** and **import-only** tiers stated honestly (`B83` already publishes the
  matrix), and everything else served by an SDK and conformance kit (`B99`) so a community
  connector needs no core release. The activity gap stays the priority inside that set,
  because closing it multiplies every activity metric: Cline-family logs carry the diffs
  needed for line extraction (`B39`), `opencode` stores structured additions/deletions per
  edit and is the richest next target (`B52`), the Cline forks Roo and Kilo Code (`B60`) and
  the Gemini fork Qwen (`B61`) reuse most of an existing parser, and Aider (`B14`) rounds out
  the shortlist. Vendor exports, org APIs and OTLP (`B98`) cover the sources whose local files
  cannot honestly be reverse-engineered — imported as what they are, never dressed up as a
  deep parser.
- **Cost truthfulness.** Every `$` is an estimate at public pay-as-you-go API prices, which
  a flat-rate subscription makes structurally different from real spend. Two candidates
  close the gap: opt-in **vendor billing reconciliation** (pull Anthropic/OpenAI usage
  APIs, show the estimate-vs-actual delta with a confidence band — `B19`), and **tiered /
  context-length pricing** (long-context and distinct cache read/write rates a single
  per-model number can't capture — `B16`).

### 3. Deeper local analysis — the half that has to be strongest

Three layers, in the order they pay off.

**Read what is already in the file.** Every parser turns a rich log into a fixed record, and
each one currently drops fields it does not yet know what to do with. Nobody has systematically
asked what those are — `B105` is that audit, source by source, ending in one of two outcomes
per field: extracted, or documented as deliberately skipped with the reason. Where a log
carries activity the parser does not extract yet, closing it multiplies every activity metric
at once rather than adding a figure: Cline-family logs carry the diffs needed for line
extraction (`B39`), and Gemini's OpenTelemetry path is the honest route to its missing edits
(`B72`).

**Turn it into conclusions.** A large shortlist of metrics is honestly supportable from data
already stored — no git, no issue tracker, no schema change: throughput per focused hour,
rework/thrash bursts over time, the full day × hour rhythm heatmap (never an attendance view),
long- versus short-session efficiency, entrypoint mix, model freshness and lock-in, sub-agent
delegation economics and its tool-call split, compaction recovery cost, and a self-relative
"skill curve" panel (you vs you four weeks ago, never cross-person). Each is one
self-registering file carrying the caveat that keeps it honest — the `B28`–`B37` cluster,
plus `B02`, `B20`, `B78` and `B79`.

**Prove the reading is right.** This is the part that separates depth from guesswork. Format
drift canaries and per-source depth shipped in v0.5, every parser ships a fuzz target and a
golden corpus, and the signal catalog states what a zero means for each figure. What is still
thin: goldens are captured samples chosen by hand, so a vendor's format change in a shape
nobody captured stays invisible until a number moves. Widening that corpus, and asserting a
parser's output against it rather than against fixtures written from the same reading, is
what makes "we extract twenty signals" a claim rather than a count.

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
enough to build a community on. The third exec-plugin protocol, **rule** units gating
`check` (ADR 0005), has landed; what remains: a published, versioned freeze of the plugin
protocols and the SQLite schema under semver (`B23`); JSON Schemas and a scaffolder for
plugin authors (`B06`); and a community registry page once a few plugins exist (`B57`).
Exec plugins stay the universal baseline — any language, no ABI, no rebuild.

The in-process plugin API (`B24`) needs its mechanism revisited before it is built. Go's
native `plugin` package fights everything this project promises: it needs cgo, does not work
on Windows, and requires plugin and host to be built with identical toolchain and dependency
versions — which is incompatible with shipping one static binary per platform. A sandboxed
**WebAssembly/WASI** component is the more plausible in-process path, with capability limits
and resource budgets as a bonus rather than an afterthought. Until that is evaluated, `B24`
is a research item, not a commitment. The core rule holds regardless: `internal/` never
depends on `plugin/` or `ee/`.

### 6. Developer experience

The daily-habit and integration layer. The ambient `statusline` and the incremental backfill
that makes it cheap shipped in v0.4; what remains: a first run that needs no config at all
(`init` — detect the tools, show exactly what will be read, backfill, print the dashboard
path, stay network-free — `B82`), an interactive TUI (`B45`), a read-only MCP server so you
can ask your own usage questions from an agent (`B44`), a weekly digest fit for cron (`B11`),
a packaged GitHub Action (the `check` gate plus a PR comment — `B12`), shell completions and
man pages (`B46`), data exports (OpenMetrics for Grafana, ndjson/parquet for data teams —
`B47`), and growing the i18n scaffold into real locales as people ask (`B08`).

Every one of those is a **surface over the same analysis**, never a second implementation of
it. A number that differs between `report`, the dashboard, the TUI and MCP is a bug, and the
cheapest way to avoid it is to keep the computation in `internal/analyze` and `internal/report`
and let surfaces only render.

### 7. Harness intelligence

A repository increasingly carries configuration written *for* coding agents — `AGENTS.md` and
other context files, rules, skills, sub-agents, hooks, commands, MCP declarations, permission
and model policies. Research on thousands of repositories finds these artifacts spreading fast
and used shallowly, and that skills in particular deserve to be evaluated like software rather
than assumed to work because they exist.

`assaio` is unusually well placed to ask whether any of it helped: it already sees the token
and retry cost a configuration imposes, and after the evidence graph it will see the outcome
on the other side. The plan is an **inventory, not a copy** (`B95`) — artifact type, scope,
keyed hash, size band, modified time, and whether it was observably invoked, with the content
deliberately not stored — followed by before/after cohorts around a harness version change
(`B96`) that state their sample size and confounders and stay labelled observations. This is
also what gives the recommendation engine (`B84`) something to recommend that is not a
restatement of a token count.

### 8. Interoperability

Claude Code already exports OpenTelemetry, and the OTel **GenAI semantic conventions**
standardize model, token, tool, agent, session and operation concepts. Inventing an isolated
schema and deferring interoperability to the end would be a mistake `assaio` can cheaply
avoid: map the canonical fields to those conventions while the canonical model is still small
(`B98`). Two payoffs — OTLP becomes a source for tools whose local files cannot or should not
be parsed, and anything `assaio` emits can land in infrastructure a team already runs. The
non-negotiable part: prompt, completion, tool-input and tool-result attributes are dropped by
default, and the dropped set is documented rather than implied.

### 9. Scale

Reserved for when the current shape stops being enough, not before: a Postgres backend once
a single SQLite file no longer suffices for a central store (`B25`), and dashboard depth
(multi-window tabs, sparklines, top-N drilldowns) as reports carry more history.

### 10. From a finding to a checked change

The gap between "here is a number" and "here is what to do" is where an analytics tool either
earns its place in someone's week or gets uninstalled. The intended shape (`B84`):

```text
explain → suggest → act → verify
```

A suggestion is a **deterministic rule over signals** — context bloat and repeated
compaction, a premium model on low-output turns, cache behaviour that never pays off, retry
and tool-error loops, long sessions that correlate with rework, a plan that does not pay for
itself — and it carries what was observed, on how much data, at what confidence, one concrete
action, the metric that will show whether it worked, and the window after which to look again.
It can be dismissed as not relevant, and that dismissal sticks.

The order matters: facts → metrics → confidence → rule → suggestion, and only then an
optional LLM *explaining* a suggestion that already exists. Never raw numbers → LLM →
confident-sounding advice; that route manufactures certainty the data does not have, which is
the one thing this project cannot afford. Verifying whether a change helped is a
before/after comparison with its limits stated — never a causal claim from observational
data (the pre/post window is a candidate `B87`, and it is what makes "did the suggestion
work?" answerable at all).

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
- **Never relabel a layer.** Activity is not output, output is not outcome, outcome is not
  impact (see above). A metric states which one it is.
- **Privacy is a protocol property, not a policy page.** "Prompts and code are never
  collected" has to stay true in local storage, the sync wire format, plugin inputs and
  outputs, exports, and error output — every one of those is a place it could leak, and each
  is where the guarantee is actually kept or broken.
- **The core stays Apache-2.0** throughout; commercial modules, if any, stay isolated under
  `ee/`. The boundary that keeps that honest: **open source owns the data, the schemas, the
  core computation and self-hosting; anything paid would monetize operations, managed
  integrations, scale and governance** — never unlocking a chart the local binary could
  compute. A local and self-hosted user must keep a complete, useful product with no cloud
  requirement, or the open-source promise is bait.

## How we prioritize

Order follows real-world feedback, pull requests, and bug reports — not this document's
sequence, and things not listed anywhere can land first when a PR or a bug report makes the
case.

GitHub stars are not the measure of whether this is working. The questions worth tracking are:
how long from install to a first useful insight; what share of users have high-confidence
**activity** data rather than tokens only; whether someone runs a second report in the
following week; how often a finding is acted on, and whether the metric it named actually
moved; how often the honest answer was "insufficient evidence"; and, for the ecosystem,
whether a plugin author gets something working and whether parser drift gets fixed quickly.
An "insufficient evidence" rate that is *too low* is a warning sign, not a win.

To weigh in: open a feature-request issue or a Discussion, or add weight to a tracked
item by referencing its `B`-id. Connectors additionally follow the
[connector intake flow](docs/extending.md#the-intake-path-open-a-connector-issue-first) —
open an issue with a redacted sample before building, so the format is verified first.
