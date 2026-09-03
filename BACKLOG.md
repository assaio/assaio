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
  [connector intake flow](docs/extending/data-source.md#the-intake-path-open-a-connector-issue-first).

## Shipped — "Trust the dataset" and "Intent"

Two promises are closed and their items deleted per the lifecycle rule above. Kept here as
one line so a reader knows why the numbering starts where it does: format-drift canaries,
`doctor --strict`, granularity provenance, the confidence envelope on every result, `init`,
the source-depth matrix (`B58`, `B59`, `B69`, `B81`, `B82`, `B83`, all v0.5), and session
annotations with per-kind-of-work stratification (`B80`, v0.6) alongside the GitHub Copilot
CLI parser (`B53`, v0.6). See [CHANGELOG.md](CHANGELOG.md) and [FEATURES.md](FEATURES.md).

## Shipped — "Correctness lockdown"

Nothing new is built on top of a surface that is known to be wrong. Every item here is a
defect in something already shipped, each with the reproduction that found it, and they are
listed in the pool below. They close first. The reason is v0.12: the flagship parser was 2x
wrong for eleven releases and every honesty guard this project owns said it was fine.

The eight that carried a wrong number or destroyed data closed in v0.13 (`B119`–`B123`,
`B127`, `B131`, `B132`). The remaining ten closed in v0.14 (`B124`–`B126`, `B128`–`B130`,
`B133`–`B136`), together with four defects the same review found that had no id yet: a
repeated Claude transcript line counted twice, a superseded sub-agent aggregate left in the
store, a team-server push that could never correct a partial figure, and an activity
correction that could not reach a stored row at all. **The milestone is closed** — what a
later review finds opens a new pool below rather than reopening this one.

## Alongside everything — "Somebody outside this repository"

[ROADMAP.md](ROADMAP.md#the-next-milestones)'s stage 0, and the only milestone in this file
whose work is not this repository's to do. Two sources and the reconciler are calibrated
against a sample written in the source's own shape rather than a real one, and no amount of
local work changes that: the missing input is a file somebody else has. `B144` is the half that
already had an id; below are the door a capture arrives through, the half that never had an id,
and the reason a silence here would mean anything.

- [ ] **B191 · a capture contribution has no door** — S · both — the repository takes bug,
  connector and feature issues and disables blank ones, so somebody who runs Gemini CLI and is
  willing to hand over a redacted chat log has to pick a template written for a different
  purpose, or open a Discussion and guess at the rules. A **Capture contribution** template
  states them instead: which source, the tool's version, the field allowlist every checked-in
  trace was redacted under (numbers verbatim, identifiers and paths replaced by stand-ins,
  bodies replaced by the same number of placeholder lines, nothing the parser does not read
  copied at all), and — the part a contributor should never have to infer — that the file is
  committed to a public Apache-2.0 repository, so it is redacted for publication rather than for
  a private handoff.

- [ ] **B192 · the reconciler has never read a real export** — S · both — `B19` shipped the
  reader and the arithmetic against a **constructed** CSV whose column names and magnitudes were
  invented ([`internal/reconcile/testdata/README.md`](internal/reconcile/testdata/README.md)),
  which is why it binds columns by header alias and prints what it bound instead of shipping a
  named vendor profile. The alias list is therefore a guess about every vendor. One redacted
  export — date, model, token count, amount, currency, and nothing else — turns that sample
  `real` and settles the aliases for the vendor it came from. It has been a trailing clause on
  `B144` with no id of its own, which left a v1.0 condition tracked under an item about two
  other sources.

- [ ] **B193 · the ask has to reach somebody who has not already found us** — S · solo —
  publishing the call in [ROADMAP.md](ROADMAP.md) and `README.md` reaches people who are already
  here, and that is by definition the population that has not appeared. Take it once to where
  the users of the two uncalibrated tools are — those tools' own issue trackers and community
  channels, and the places the 2026-08-11 launch post already reached — and record, per channel,
  what was asked and what came back. That record is what makes stage 0's kill criterion
  readable: "nobody contributed" decides nothing unless somebody was asked, and a silence with
  no attempt behind it is evidence about neither the ask nor the reach.

## Now — "Calibrated measurement"

The guard the v0.12 class of bug needs, which provenance and coverage cannot provide. `B137`,
the conservation and metamorphic suite, shipped in v0.14 as `internal/calibration`, and `B19`,
the offline reconciliation against a vendor's own export, shipped after it — and `B139`, the
price table nothing watched, which was the largest error left in the `$` figure. All three are in
[CHANGELOG.md](CHANGELOG.md). One item remains and it cannot be closed here at all: `B144` needs
a redacted real capture, the same contribution `B19`'s column aliases need — which now carries
its own id (`B192`) and a milestone to arrive through, above. The milestone also
depends on `B116` and `B118`, which live in the code-health pool below because they are
corrections to a shipped mechanism rather than calibration work; `B116` is additionally a v1.0
condition.

- [ ] **B184 · a trend on the shared card** — S · both — the shipped `share` renders point
  figures only: tokens, lines, `$`/100 lines, the model mix, the archetype and the one reserve.
  `B149` also asked for the token/cost *trend*, and that half did not ship. It is the half that
  makes a second card worth posting — "down 12% and the same output" is a story, while a second
  month of point figures is the same picture again. The data is already there (`report --compare`
  and `internal/digest` both compute it), so this is a renderer change, not a measurement one.
  The honesty constraint travels with it: `digest` already declares when a comparison is weak —
  overlapping windows, unequal lengths, a parser that changed between the two runs — and a card
  that shows a delta without that declaration would publish a movement the store cannot support.

- [ ] **B144 · calibrate Gemini CLI and Cline against a real capture** — S · both — both are
  calibrated today against a *constructed* sample in the source's shape, because the maintainer's
  machine holds neither a Gemini chat log carrying token counts nor a Cline install. A
  constructed trace proves the reading; only a real one also proves the shape is still what the
  vendor writes, and each trace already declares which it is (`capture: real|constructed`). Needs
  one redacted capture per source from anybody who runs them.

## Shipped — "Worth opening twice"

Everything that decided whether the analysis already shipped reaches anybody: findings that
arrive ordered with their reasons shown (`B148`), a digest that reports what *moved* and
declares when the comparison itself is weak (`B11`), a label derived through a rule engine
rather than typed (`B152`), the extension surface no longer weaker than the core it extends
(`B155`), and an assay that can be posted in public with redaction as the feature rather than
a flag (`B149`). All five are in [CHANGELOG.md](CHANGELOG.md). One piece of `B149` was dropped
deliberately rather than deferred: the item asked for repositories as a count *and a sorted
spend distribution* ("5 repositories, the heaviest 43% of spend"), and that distribution is a
re-identification surface — the ordering and proportions are exactly what somebody who knows
the setup reads a pseudonym back out of, which is the reason the same item refuses pseudonyms.
The count ships; the distribution does not, and will not. What is still missing is `B184`.

## Then — "Everything the logs already say"

Depth before breadth, and before correlation. This milestone is the half of the product that
needs no server, no credential and no repository — and the half every other conclusion rests
on, since a link to a merged pull request is only as good as the session it links.

- [ ] **B147 · agent behaviour trace: the session as a sequence, not a total** — L · both —
  every figure assaio publishes today is an aggregate, and an aggregate cannot say *why* a
  session was expensive. The logs already carry the order: plan, search, read, edit, test
  failure, repeated tool call, compaction, retry, abandon. Store a content-free timeline —
  step kind, ordinal, model, token delta, outcome — and read detectors off it: retry loop,
  tool thrashing, the same file read N times, search with no following edit, a large edit
  before any test, compaction recovery, model switch after failure, dead-end run. No prompt,
  no code, no file contents; a path is a category, never a name. The refusal that has to
  hold: a detector fires on a *pattern*, and a pattern is not a fault — a hard bug
  legitimately looks like thrashing, so every detector ships with what it cannot distinguish.
  **Store cost is settled, and this item's own guess about it was right.** Measured on the
  maintainer's store after a full rebuild under the horizon: 1.88 stored step rows per usage
  record (335,527 against 178,016) — but a row multiplier is not a size measurement, and in
  bytes the table and its indexes are **101.9 MB against `usage_record`'s 58.3 MB**, roughly
  1.7x the table they describe. Hence a horizon
  (`trace.horizon_days`, default 30) pruned on every ingest rather than by a tidy-up command;
  30 days is also all that is recoverable, since Claude deletes transcripts at 30 by default --
  the history horizon and `doctor`'s retention line that state so shipped in v0.21.
  **Two things this item did not anticipate, both found by measuring before designing.** A
  session-grained denominator is a trap: 86% of sessions on the audited machine are SDK calls
  contributing 5% of rows, so "89% of sessions end without an edit" would be true, precise and
  worthless — every detector declares its scope from `entrypoint`/`sidechain` and renders the
  excluded share beside its figure. And a sub-agent needs its own timeline: 103 sessions hold
  39% of all rows, and blending them would make a sub-agent's thrashing read as a person's.
  **Landed** in v0.20.0: the step contract, the Claude reading, migration `0012`, the horizon and
  its prune. **Landed** in the release after it: the two detectors (`edit-loops`, `recovery`), the
  scope vocabulary and its shared denominator, the capability row and its three signals, the plugin
  boundary, and the widened target that made a read and a failed edit nameable at all. **Landed** in
  v0.26.0: the Codex reading, and what it settled is below. What remains is the two detectors this
  substrate can now carry but does not yet: the same file read N times (`docs/recipes/extensions.md` publishes it as a
  worked example rather than a built-in) and a large edit with nothing run after it, which needs a
  command *class* the store does not hold. A step carries no command identity, so "a command ran" is
  all any detector sees; making "was it a test" answerable is a closed vocabulary and its own ADR. Two things the readings do **not**
  do yet, stated because the ADR would otherwise imply they do: nothing reads
  `toolUseResult.interrupted`, `userModified` or Codex's `turn_aborted` (`B111`, `B113` — all
  measured at or near zero on the audited corpus), and the outcome vocabulary deliberately has
  no member they could fill.
  **What the Codex reading settled (v0.26.0)**, measured on its own 2,625 rollouts and 70,522
  lines rather than assumed from Claude's, none of it guessable from the 76 files this item was
  written against. The row multiplier is **1.66** (9,555 steps against 5,765 records) and in bytes
  **2.00 MB for 8,096 steps, 260 B each**, measured on a store built from Codex alone rather than
  differenced across two runs — stated because this item's own correction above is that a row
  multiplier is not a size. Codex *does* state how it was started (`session_meta.source`: `exec`
  on 2,604 rollouts, `cli` on 13), so `internal/trace`'s claim that it writes no entrypoint was
  wrong on a current install and its sequences are scoped rather than landing in `unstated`. Codex
  states an outcome on a patch and on nothing else — `completed` means a call returned, and 57 of
  840 commands exited non-zero under it — so the depth row claims `ai.steps.count` and
  `ai.step.target`, refuses `ai.step.outcome`, and `recovery` now narrows to the sources that
  answer it rather than counting Codex's silence as a clean run. A patch applies to 1.91 files at
  once, so the sequence records one step per file where the record counts one write per call: two
  answers to two questions, and the only way a repeat-edit reading sees the file that was returned
  to. **The substrate is thinner than the corpus suggests**, and the honest figure is the second
  one: 1,904 of Codex's 9,555 steps (19.9%) are a person's terminal session, and after the 30-day
  horizon only 608 of them in 5 sequences reach a detector. Of the 34 rollouts that edit anything,
  17 return to one file three or more times and the heaviest patches one file 36 times — a
  measurement over the whole corpus, not a figure any shipped detector publishes, since most of
  those rollouts are `exec` and out of every detector's declared scope.
- [ ] **B107 · Codex cache-write tokens are never read** — S · solo —
  `payload.info.total_token_usage.cache_write_input_tokens` is reported on every Codex
  `token_count` and `usage.Record` gets no value for it. **Measure before claiming a
  correction**: on the audited corpus the field carried a value on 238 events and was zero on
  every one, so this fixes no number here — the reason to read it is that a zero nobody looked
  at and a zero the vendor reported are different facts, and a different plan or model may not
  share it. Two things have to be settled first, and neither is guessable from that corpus:
  whether `input_tokens` already contains the write (it does contain `cached_input_tokens`,
  which the parser subtracts), and how history gets the value, since the local restate covers
  activity and granularity but deliberately not tokens. A `backfill --full` alone will not do
  it.
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
  fresh session before touching the parser. Not establishable from this machine — it needs a
  redacted capture from someone running Gemini CLI, which is `B144`'s standing caveat.
  **The second half shipped in v0.22 as the `barren` canary, and this item's stated cause for it
  was wrong.** It read "each needs a 20-file sample floor"; two of the four floors are 50 and the
  discovery canary's total-loss branch has none, and an A/B with all four set to `1` produced
  `no canary fired` on both builds. The blocker was never the floor but the baseline guard: a
  source that has always yielded zero has a baseline of zero and there is no drop to detect. A
  fix written from the wording above would have shipped green and changed nothing. Recorded
  because the corrected cause is the reusable part.
- [ ] **B111 · the human correction, recorded rather than proxied** — S · solo — **the claim
  this item was written on is false, measured 2026-08-12 and corrected here rather than
  discovered after shipping.** `toolUseResult.userModified` was described as marking an edit the
  person changed after the AI wrote it "on a meaningful share of Claude edit results". The
  *field* is on a meaningful share — 2,691 of 13,038 tool results in a 400-transcript sample —
  and its *value* is `false` on every one: `"userModified":true` appears in **0 of 5,706
  transcripts, ~5 GB**. A signal built on it would be structurally zero and nobody would notice.
  The field is now read into the step timeline's outcome so a different plan, model or user
  reveals it (the `B107` template: a zero nobody looked at and a zero the vendor reported are
  different facts), and it stays out of the signal catalog until one does. `rework` remains the
  proxy it says it is.
- [ ] **B112 · attribution beyond skill and sub-agent, and the build that wrote the line** — M ·
  solo — Claude stamps `attributionPlugin` and `attributionMcpServer`/`attributionMcpTool` on a
  large share of turns, which would widen `skill-economics` from two dimensions to four and give
  MCP servers a cost of their own. Alongside it, `version` on every line is the harness-version
  cohort input `B96` assumes needs a server — it is already on disk, offline, per turn.
- [ ] **B113 · how a turn ended** — S/M · both — **measured 2026-08-12; the "stopped" half is
  near-empty and the item is downgraded accordingly.** Across the whole corpus:
  `"interrupted":true` in **0 of 5,706** transcripts, `"stop_reason":"max_tokens"` in **5 of
  5,706**, Codex `turn_aborted` **9 occurrences across 2,625 rollouts** (all `reason: interrupted`; the
  76-file figure this line carried is superseded by `B147`'s corpus). A
  signal promising to "distinguish a turn that finished from one that was stopped" would ship a
  permanently empty bucket. What does fire is `toolDenialKind` (automode-blocked, user-rejected,
  automode-unavailable, permission-rule) and `stop_reason` as a continuation-versus-finish
  discriminator (tool_use 18,330 against end_turn 512). All of it is read into the step
  timeline's outcome vocabulary; none of it claims a signal yet. Codex's
  `changes.<path>.{type,move_path}` is untouched by this measurement and still belongs here: an
  add, an update and a rename are one undifferentiated edit today.
- [ ] **B114 · Codex states its own limits** — M · both — `payload.info.model_context_window` is
  the model's context limit on every `token_count`, which is what `B16` proposes to vendor a
  table for; and `payload.rate_limits.{plan_type,primary.used_percent,primary.resets_at,credits}`
  is the only place any source says how close a session ran to a plan's ceiling. That is the
  subscription question `subscription-fit` answers from a configured number today, answered
  from the vendor's own accounting instead. Both are per-session state rather than usage, so the
  storage shape needs deciding first.

## After that — "Evidence Graph: from session to shipped change"

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
  deprecation window and a conformance fixture, and **lands with the protocol freeze (`B23`)**
  rather than before it — settled by [ADR 0016](docs/adr/0016-usage-is-a-store-row-not-an-event.md),
  which put stage 2 behind the freeze rather than in front of stage 3. See [ADR 0008](docs/adr/0008-signal-catalog.md).
  **Three requirements came out of building the git collector** rather than being guessed at:
  (1) a validator receives `[]store.UsageRow`, so there is no shape in which it could read a
  commit observation at all — the context has to serve *heterogeneous* observation streams
  keyed by type, which is a bigger job than swapping the row types; (2) `signals coverage`
  weights support by tokens (`analyze.TokensByTool`), so a source that has no tokens cannot
  appear in it, which is why no `vcs.*` signal is registered yet and what a presence notion
  would have to replace; (3) capability lives on `parser.Answers`, and a collector is not a
  parser — the registry has to move out of `internal/parser` or learn about non-parser
  sources before the first `vcs.*` signal is declared.
  **v0.24 took the first step and deliberately not the whole one**: `analyze.Capability` is the
  closed vocabulary of what an analyzer reads, `Needs` is how a validator declares it, and the
  runner records what a window could not supply (`Result.Withheld`) instead of leaving each
  validator to invent a sentence about an empty field. Four validators declare; the rest declare
  nothing and are held to nothing, which is the property that makes this migratable. What is
  still open is the part this item is named for: `Input` still carries the store row types, and
  retiring them is still a breaking change to the metric-plugin wire. The capability set is now
  the seam that change would move through — and it already earns its keep on the wire, where
  `needs:` gates the 44 MB step timeline.
- [ ] **B103 · commit observations that outlive one pass** — M · both — the half of `B91`
  deliberately not shipped: a configured repository list (rather than one `--repo` at a time),
  a keyed pseudonymous commit digest for the syncable case, and durable storage. Each waits
  for the consumer that decides its shape — `B85` needs commits queryable *across* passes to
  link a session to one, `B100` decides what identity may leave the machine — and each costs a
  migration, a size bound and a cleanup path, so none is worth guessing at. Path-level storage
  stays out entirely until something needs it: `B91` never records a path, so there is no
  opt-in to design yet. See [ADR 0009](docs/adr/0009-local-git-evidence-collector.md).
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
- [ ] **B153 · what a change costs after it is written** — M/L · both — "AI lines" counts
  what was produced, never what it took to land. Once a session links to a pull request, the
  downstream burden becomes countable as named signals rather than adjectives: review rounds
  and requested changes (`review_tax`), CI failures and the repair cycles after them
  (`ci_repair_tax`), revert rate, time to merge, and the two ratios that make cost mean
  something — `cost_per_merged_change` and `cost_per_surviving_change`. Each with its own
  coverage and confidence, and each **age-matched against human-authored changes of the same
  age**, which is already the standing rule for bug density and is the only way these are not
  a smear. They are observations about a process, never a measure of a person; the
  per-person refusal holds here exactly as everywhere else.
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

## Then — "Harness intelligence & verified improvement"

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
- [ ] **B150 · which model for which kind of work, from your own history** — M/L · both —
  a person running Claude, Codex, Gemini, Copilot and Cline has no basis for choosing between
  them beyond feel. The parts are already stored: task class (`mark`), model, cost, retries
  and failed tools, rework, and — once outcomes link — review and survival. Compare models
  *within* a task class and repository class, never globally, and publish the comparison with
  its sample size. The rule that keeps it honest is a floor, not a disclaimer: below a
  minimum of comparable sessions it reports insufficient evidence and names no winner. A
  global "best model" is never emitted; benchmarks measure other people's work.
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

## After that — "Team adoption, without surveillance"

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

## Later — "Cost truth, policy & interoperability"

Every `$` today is an estimate at public pay-as-you-go prices, which a flat-rate
subscription makes structurally different from real spend — and assaio's own model should
speak a standard rather than only its own dialect.

- [ ] **B138 · credentialed billing pull** — M/L · both — the network half of the
  reconciliation `B19` moved up to "Calibrated measurement": an opt-in pull of the Anthropic/OpenAI usage and cost
  APIs so the reconciliation runs unattended instead of from a downloaded export. Network- and
  credential-gated; pulls vendor aggregates only, never uploads logs. Deliberately second: the
  offline import carries most of the value and needs nobody's key.
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

## v1.0 — "Measurements worth depending on"

The one milestone that keeps a version number, because there the number *is* the promise. It is
**not** only the semver guarantee any more: after v0.12 the bar leads with calibration, because
a frozen contract over an uncalibrated measurement is a stable wrong answer, which is worse than
a breaking correct one. The six conditions are spelled out in
[ROADMAP.md](ROADMAP.md#what-v10-has-to-mean); the contract freeze is the last of them, not the
first. `B24` used to sit here and no longer does — its own entry says it is not a v1.0
condition, so it moved to the reserved pool below rather than contradicting itself in place.

- [ ] **B23 · protocol freeze** — M · both — declare the exec plugin protocols (parser,
  metric, rule), the canonical event and signal contracts and the sync API stable under semver,
  with conformance fixtures and deprecation windows. **The SQLite schema is deliberately
  excluded**: v0.12 needed a migration that rewrote stored rows to correct a semantic error and
  it will not be the last, so freezing the store would make every future correction a breaking
  change and create pressure to leave a wrong number in place. It stays internally versioned
  and migratable — a promise about correctability, not the absence of one.

## Pool — reserved: waiting on a condition, not on time

Neither of these is scheduled, and neither is dropped. Each waits for a fact to become true —
a scale that does not exist yet, and a mechanism that has not been chosen — so listing them
anywhere else would misrepresent them as work anyone is about to pick up. `B24` in particular
was under the v1.0 heading while its own text said it is not a v1.0 condition.

- [ ] **B25 · Postgres backend** — L · team — once a single SQLite file stops being enough
  for a central store. Not before: the current shape has not been shown to hurt.
- [ ] **B24 · in-process plugin API (research first)** — L · both — **no longer a v1.0
  condition**: it is a research question, not a readiness bar. The mechanism is **not**
  decided, and native Go plugins are explicitly not a v1 requirement: they need cgo, do not
  work on Windows, and require host and plugin built with identical toolchain and dependency
  versions, which contradicts shipping one static binary per platform. Evaluate a sandboxed
  WebAssembly/WASI component instead — it brings capability limits and resource budgets — and
  keep exec plugins as the universal baseline either way. Deliverable is a decision record,
  not code.

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
  share, the cache-write-waste flag, the 1-hour write share and the top stated miss cause
  (`B108`, v0.12). Day-grain approximation stands.

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

## Pool — from the 2026-08 needs research

Sourced from public evidence rather than from this machine, which is the point: the roadmap
had been growing from one maintainer's dogfooding, and that is a sample of one.

- [ ] **B157 · Cline's second root** — S · both — the standalone Cline CLI stores tasks under
  `~/.cline/tasks/<id>/`, overridable by `--data-dir` / `CLINE_DATA_DIR`
  ([docs.cline.bot/cli/cli-reference](https://docs.cline.bot/cli/cli-reference), retrieved
  2026-08-11); assaio discovers only the VS Code root, so a Cline-CLI user gets zero rows and
  no warning. Discovery work, not new parsing.
- [ ] **B158 · Copilot's OTel file exporter** — M · both — `COPILOT_OTEL_ENABLED=true` with a
  file exporter writes token, cache and reasoning counts to `~/.copilot/otel/*.jsonl`
  ([ccusage.com/guide/copilot](https://ccusage.com/guide/copilot/); GitHub shipped
  enterprise-managed OTel export [2026-07-08](https://github.blog/changelog/2026-07-08-enterprise-managed-opentelemetry-export-for-vs-code-and-cli/)).
  A second, opt-in path beside the `session.shutdown` totals already parsed — and the template
  for every tool that speaks OTel, which is where cloud-executed sessions surface at all.
  Carries a calibration caveat: Claude Code 2.1.216 fixed telemetry double-counting on streams
  emitting multiple cumulative `message_delta` frames, so OTel figures predating it are inflated.
- [ ] **B159 · an AI-tool inventory an auditor accepts** — S/M · both — ISO 42001 has moved
  from standard to procurement requirement, and buyers now ask for an inventory of the AI
  systems and services a supplier relies on. `report` already knows tools, models, projects and
  first/last-seen; a stable `inventory` export (tools × models × repos × window, counts only,
  no persons, no paths) answers that questionnaire offline. Aggregate-only by construction, so
  it clears the never-a-performance-metric refusal without a flag. The EU AI Act angle was
  checked and is **not** applicable: Article 50 targets generated content and deepfakes.
- [ ] **B160 · the reopen window: cost per accepted change** — M · both — the measure that
  decides whether AI improved delivery sits just outside session data: not tokens, but the
  second and third attempt at the same change. What ships today stops at the session boundary —
  `rework` counts within-session churn and rejections and says so, and `survival` checks git
  blame directionally. Neither answers how often a file an agent touched is touched again
  within two weeks, which is what turns cost per session into cost per accepted change — the
  number an engineering lead can defend at a budget review. This is the **git engine**, separate
  from log parsing by construction, and its first universal analyser. Raised in public feedback
  on the launch post, 2026-08-11; converges with the evidence-graph milestone below.
- [ ] **B163 · a digest that can tell a prune from a change** — M · both — v0.17 made `clear`
  drop the digest's comparison basis, which fixes the case assaio itself causes. It does not
  cover a store edited by anything else, or the general question: the digest compares two
  reports and has no way to know the population under them changed. A row count per snapshot
  would catch a shrink, but cannot distinguish a prune from a genuine fall in usage — which is
  why this is a design question rather than a patch, and why the shipped answer is a caveat
  about what could not be checked instead of a claim about what happened.
- [ ] **B162 · a destructive command that cannot be pointed at the wrong store** — S/M · both —
  `clear` has no `--db`: it always acts on the default local store, and there is no environment
  variable either. v0.17 made it print the path and record count before deleting, which is the
  cheap half. The open question is the other half — whether a command that deletes should accept
  a target at all (making the mistake possible in a new way), or should refuse when the store it
  found is not the one the invocation implies. Raised by a real incident: a tooling run set a
  variable that does not exist, expected `clear --all --yes` to apply to a copy, and emptied a
  513,617-row store. `--yes` skipping the prompt is the other half of why it was silent.

## Pool — from the 2026-08-12 needs research

A second sweep, sourced the same way as the pool above: public evidence, each claim carrying
the URL it was read from. Where it contradicted an item already here, the correction went onto
that item (`B60`) rather than becoming a new one.

- [ ] **B164 · the autonomy validator** — M · both — Anthropic published a vocabulary for how
  autonomously an agent ran — turn duration, auto-approve and interrupt rate, agent-initiated
  stops — and states that the API has "no reliable way to link independent requests into
  coherent agent sessions"
  ([anthropic.com](https://www.anthropic.com/research/measuring-agent-autonomy)). Local
  transcripts do link them, which is the whole argument. Every input is either stored or one
  field away: `toolDenialKind`'s `automode-blocked`/`automode-unavailable` is the auto-approve
  half, already parsed. Depends on `B147`'s timeline; it is the first analyser that should read
  it, because the definition is citable rather than invented here.
- [ ] **B172 · a project name is read with the host's path rules** — S · solo — `carryForward.project`
  takes `filepath.Base(cwd)`, where `cwd` comes from the transcript and `filepath` answers for the
  platform assaio runs on. A Windows transcript read on Linux therefore yields the whole
  `C:\\w\\app` string as the project name rather than `app`, and every per-project figure splits
  along the platform that read the log. Found while fixing the same host-dependence in the step
  target reading, which CI caught on Windows (`targetKey` is the shape the fix should take here).
  Pre-existing, not introduced by that change, and it needs a `backfill` to restate the names it
  already stored.
- [ ] **B171 · the served dashboard cannot show the detectors** — S/M · both — `serve` renders
  `GET /` unauthenticated and rebuilds it on every request with no cache, so it deliberately does
  not read the step sequences: `store.Timelines` is a full scan of the step table plus a GROUP BY
  over the window's records, about 2.5s on a 339,000-step store, and it costs that before it can
  know whether a step exists. On the store this surface is built for the answer is always none --
  `sync` carries usage records and not sequences (ADR 0012) -- so today the detectors correctly
  report that and say why. Making them work on a self-hosted store that also ingests locally needs
  the thing this handler lacks and the CLI does not need: a built snapshot with a lifetime, which is
  the same machinery a cached dashboard would want anyway.
- [ ] **B170 · a call that runs nothing is classified as a command** — S · solo — `parser.StepKind`
  maps `BashOutput` and `KillShell` onto the same `command` kind as `Bash`, because the tool-purpose
  classification predates the sequence reading and only had to answer "was this a shell tool".
  `edit-loops` adopts CodeBurn's definition, which turns on *a shell command having run* between two
  edits of one file, so polling a background shell's buffered output counts as one and inflates the
  repeat rate for anyone running a dev server or a long test in the background. Measured: zero
  `BashOutput`/`KillShell` calls across ~1,500 recent transcripts on the maintainer's machine
  against 35,215 `Bash` calls, which is exactly why the cross-check against SQL could not catch it —
  both readings key on the same stored `kind`. Fixing it properly means splitting the kind, which
  moves `tool_commands` on every usage record and needs a backfill plus a re-derivation of the
  calibration corpus; the detector states the limitation in the meantime.
- [ ] **B169 · the cleanup orphans a sub-agent transcript** — S · solo — measured on the
  maintainer's machine 2026-08-12: of 5,774 Claude Code transcripts, 286 are older than the 30-day
  `cleanupPeriodDays` boundary and **every one of them is a sub-agent file** under
  `<session>/subagents/`, with no session transcript older than the boundary at all. So the cleanup
  walks sessions and leaves their sub-agents behind. Consequences to check rather than assume: a
  sub-agent sequence whose parent's own rows were never ingested carries whichever entrypoint its
  own lines state, and `trace.Scope` classifies it as a sub-agent regardless, so nothing is
  misfiled today. What is unmeasured is whether an orphan is *discovered* at all once the parent
  directory holds no session file, and whether the 45-day floor observed here is the cleanup's real
  behaviour or an artifact of when it last ran.
- [ ] **B165 · OpenCode and Kilo Code: one parser, two tools** — M · both — OpenCode keeps a
  relational store at `~/.local/share/opencode/opencode.db` whose `session` table carries
  `cost`, a five-way token split and `directory`, with per-message cost/tokens beside it
  (verified against
  [session/sql.ts @v1.18.16](https://raw.githubusercontent.com/sst/opencode/v1.18.16/packages/core/src/session/sql.ts)).
  Kilo Code v7 rebased onto the same schema at `~/.local/share/kilo/kilo.db` (see `B60`). One
  reading covers the largest unparsed population found. Ships with the double-count guard both
  need: the pre-v7 Kilo era coexists on disk and v7's importer can restate it.
- [ ] **B166 · Goose: the cleanest per-request ledger in the field** — M · both —
  `~/.local/share/goose/sessions/sessions.db` holds a literal `usage_ledger`: one row per
  request with model, a cache read/write split, `cost`, **a `cost_source` provenance column**
  and an **`is_compaction` flag**, with `sessions.working_dir` giving the project. The
  provenance column is the notable part — it is the only surveyed source that states where its
  own cost figure came from, which is exactly what assaio's honesty rules ask every fact to
  carry. Its `is_compaction` also feeds `B164`. Note `B154`'s Goose entry is stale: Goose left
  session JSONL at v1.10.0 and the legacy files linger unmanaged, so discovery must not read
  both eras as separate sessions.
- [ ] **B167 · Aider is an approximate source, not an exact one** — S · solo — correction to
  `B14`, whose premise is wrong: there is no default `~/.aider/analytics.jsonl`, exact values
  need the opt-in `--analytics-log`, and the default chat history **rounds anything ≥10k tokens
  to whole thousands**. Aider can therefore only ever be an approximate-confidence source, and
  its depth row has to say so before a parser exists rather than after. Last upstream push
  2026-05-22.

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
- [ ] **B06 · JSON Schemas for the metric envelope and result** — S · both — publish
  machine-readable schemas under `docs/schemas/` for the Input a metric plugin is sent and the
  Result it returns, so a plugin can be validated by a generic tool rather than by reading
  prose. `docs/conformance/` already publishes accept/refuse vectors for the same boundary; a
  schema is the other half, and the two must not be able to disagree. The scaffolder this item
  used to carry shipped in v0.25 as `plugins init --kind parser|metric|rule --lang
  go|python|sh`.
- [ ] **B08 · a second locale** — S · both — add one `i18n.Catalog` beside `en` and a case in
  `i18n.For`, proving the scaffolding with a real language. v0.4 built the catalog (dashboard
  chrome, statusline words, the 18 explain pages); what remains is translating it and choosing
  how a locale is selected. Data-derived validator text stays out of scope: translating it
  needs message templates for the interpolated numbers.
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

Each follows the [connector intake flow](docs/extending/data-source.md#the-intake-path-open-a-connector-issue-first);
a tool used by one organization is usually better served by an out-of-tree
[exec plugin](docs/extending/parser-plugin.md#write-a-plugin-any-language).

- [ ] **B52 · opencode** — M — `~/.local/share/opencode/storage/message/**` JSON (plus a
  newer relational `opencode.db`). Assistant messages carry `tokens{input,output,reasoning,
  cache{read,write}}`, `cost`, `modelID`, and — richest of any candidate — the `edit` tool
  persists a structured `filediff{additions,deletions,patch}`, so lines +/- are stored
  directly, no diff parsing. The best activity target after the current four.
- [ ] **B194 · Antigravity's two readable figures, and the two questions they raise** — M ·
  both — the `agy` parser ships (ADR 0017) reading the transcript only, and three findings from
  the 2026-09-02 corpus sweep are parked here rather than lost. **(a) Observed context
  occupancy.** `gen_metadata` field path `1.9.10.1` is prompt/context tokens used: monotonic
  across steps in **198/198** conversations with more than one row, and `used ≤ window` in
  **638/638** observations against `1.9.10.4`, the model's context window, which takes exactly
  {80000, 128000, 160000, 256000} and pairs with the model name in the same blob **218/218**
  times. `store.SessionRow.PeakContextTokens` is *derived* on every other source
  (`cache_read + input`); this would be *observed*, and one column holding both with no field
  to say which is which is the blend AGENTS.md forbids. Needs a provenance flag on the column
  first. **(b) The model name.** Field paths `3.28` / `1.19` hold it (`gemini-3.7-flash-high`
  ×208, `gemini-3.7-flash` ×45, `claude-opus-4-6-thinking` ×52, plus `gpt-oss-120b-medium`,
  `claude-sonnet-4-6`, `gemini-pro-agent`) — so this source runs models from three vendors and
  assaio reports none of them. Reading it means opening another tool's live WAL-mode SQLite
  file and walking an unnamed protobuf field, for a name covering 218 of 500 conversations and
  often several per conversation with no request count to choose between them. **(c) A
  session-level failure signal.** 384 of 1,537 transcript entries are `ERROR_MESSAGE` — a
  stream or platform failure, not a tool call returning an error, and the catalog has no id for
  it. `ai.tool_errors.count` is the wrong home; a new signal is a catalog decision. All three
  were verified against Antigravity CLI 1.1.23, whose binary self-updates.
- [ ] **B55 · Cursor (Admin API)** — M — local storage verified to lack token counts;
  vendor-aggregate granularity, tagged as such.
- [ ] **B195 · Cursor's local store — the format passes and the corpus does not** — S · both —
  measured 2026-09-02 against the same two-part test every source has to pass: a stable
  discoverable format **and** a real captured corpus. The format half passes and is the most
  discoverable of any candidate — `~/.cursor/ai-tracking/ai-code-tracking.db` is a plain SQLite
  file at a fixed path with named columns and no protobuf:
  `ai_code_hashes(hash, source, fileExtension, fileName, requestId, conversationId, timestamp,
  model)` and `scored_commits(commitHash, branchName, scoredAt, linesAdded, linesDeleted,
  tabLinesAdded, tabLinesDeleted, composerLinesAdded, composerLinesDeleted, humanLinesAdded,
  humanLinesDeleted, blankLinesAdded, blankLinesDeleted, commitMessage)`. The corpus half fails
  on the audited machine: 1,493 `ai_code_hashes` rows spanning 2026-07-15..2026-07-29 — 14 days
  that stopped five weeks before the measurement — 85 `scored_commits`, and `conversation_summaries`
  and `tracked_file_content` both empty. Summed over all 85 scored commits, `linesAdded` = 53,059
  while `composerLinesAdded` = 6,636, `tabLinesAdded` = 0 and `humanLinesAdded` = 0: the AI-versus-
  human split the columns promise is not written here. **No token count and no cost column exists
  anywhere in the schema, on any machine**, so Cursor can never answer the token half — `B55`
  covers that through the Admin API instead. **What would change the answer:** a machine where
  `humanLinesAdded` and `tabLinesAdded` are populated across a real span. `scored_commits` is, in
  shape, exactly what ROADMAP stage 3 wants — AI versus human lines *per commit*, already
  attributed by the vendor — which makes it worth reading for the survival question the day
  somebody brings that corpus, and never for cost.
- [ ] **B151 · what a plugin declares, and a badge that goes red when a vendor moves** — M ·
  both — two halves of the same gap. A plugin today is verified for *protocol* conformance
  (`plugins verify`, `metrics verify`) and declares nothing about itself: whether it touches
  the network, what it reads, which schema version it supports, what it emits. A manifest
  carrying permissions, privacy class, input and output signals, supported schema and a
  checksum makes an install a decision instead of a leap. The second half is a compatibility
  run over **public fixtures** per source, so a vendor changing a format turns a badge red
  and notifies the owner rather than being discovered by a wrong number months later — the
  in-tree drift canaries already do this for the six built-in sources and stop at the repo
  boundary. Blocked on nothing except the fixture question: a public fixture is a real log,
  so what may be in one has to be settled before any are accepted.
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
  **Half of this item is out of date, checked 2026-08-12 against pinned upstream source.** Kilo
  Code v7 is no longer a Cline fork: it rebased onto OpenCode and moved to
  `~/.local/share/kilo/kilo.db` with a byte-identical OpenCode schema, so Kilo belongs beside
  OpenCode as *one parser for two tools*, not here. Both eras coexist on disk and v7 ships an
  importer, which is a **double-count hazard** whichever parser claims it. Roo is still a Cline
  fork and is the cheap half — five `globalStorage` roots plus a `customStoragePath` override —
  and Roo-Code is archived (last push 2026-05-15), so it is a frozen, low-risk target.
- [ ] **B61 · Qwen Code (Gemini family)** — M — a Gemini CLI fork, but **not** the same
  on-disk shape: chats live at `~/.qwen/projects/<hash>/chats/<id>.jsonl` (not
  `tmp/*/chats`), and tokens are in a raw-API `usageMetadata` object, not Gemini's
  normalized `tokens{…}`. It also persists tool-call args, so unlike Gemini it carries edit
  activity. Needs its own parser, not the Gemini one; verify a real sample first.
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
- [ ] **B154 · unverified connector candidates — one intake, not five entries** — M · both —
  five tools whose local storage nobody here has read: **Factory droid** (session-granularity
  logs), **Kiro**, **Continue** (`~/.continue` dev-data event logs reportedly carry token
  counts), **Goose** (local session JSONL reportedly carries usage) and **Amp / Crush** (local
  thread/session storage, token presence unverified for both). Each held its own line for
  months and each said the same sentence — *verify a real sample first* — which is a wish list
  wearing a backlog's clothes, and it made the connector pool the longest in this file while
  none of it was actionable. They collapse into one intake: a candidate leaves this line and
  earns its own id **when somebody opens a connector issue with a redacted sample showing token
  counts exist**, and not before. Supersedes `B54`, `B56` and `B62`–`B64`, whose ids are
  retired rather than reused. The candidates above keep their entries because a verified
  on-disk format is a different kind of fact from a rumour about one.

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
  `analyze`'s two formatters (`compactCount`, `money`) moved onto it byte-identically. v0.11
  folded in `report.formatCompactTokens` (the `status` peak-context figure now reads `85.0K`
  rather than `85k`) and moved every percentage onto `humanize.Percent`. v0.13 took
  `dashboard.formatCompactUSD` with `B131`, since the rounding that made it a duplicate was
  also what printed a real cost as `$0`; the survivor is `humanize.USDCompact`. Still out:
  `humanize.USD` and `USDCompact` disagree about the same amount — `USD` keeps cents below
  $100 and has no M tier, `USDCompact` keeps them below $1 and has one — so `status` and the
  dashboard still render one figure two ways, and reconciling them changes shipped output.
- [ ] **B77 · readWhenEnough** — S · both — factor the "gate the favorable read behind a
  minimum-sample floor, else neutral" block shared verbatim by session-taxonomy,
  turn-efficiency, and model-right-sizing into one helper beside `readFor`.
- [ ] **B106 · two files past the size budget outside the analysis packages** — S · both —
  `internal/i18n/en_explain_work.go` (221) and `internal/cli/analyze.go` (208) each carry more
  than one responsibility at more than the ~200-line budget. (`internal/plugin/metric_input.go`
  came off this list in v0.11, split into the envelope and its wire mapping when the capability
  field landed.) The i18n one is the interesting case rather than the largest: it is a block of
  prose per metric, so splitting it by metric group is a different judgement from splitting
  code, and doing it badly makes the catalog harder to translate (`B08`), not easier.
- [ ] **B118 · does a cross-source line rate get to keep its denominator?** — M · both — a
  source recording no changed line contributes a true zero to a line *total*, so `AI lines
  total` is honest. A **rate** is the open question: `throughput`'s lines/active-day counts
  days on which only a cost-only source ran, `model-fit`'s lines/1M tok counts its tokens,
  `concentration` divides by the window's whole `Totals.Lines`, and `$`/100 lines — the
  `status` headline and all of `effectiveness` — divides a cost from every source by lines
  from only some. Each denominator is then larger than the population that could have fed the
  numerator, which understates the rate. **Measured:** zero effect on the audited store, whose
  two sources both record lines; it bites the moment a Gemini or Cline window exists. This is
  a decision before it is a patch — gating the denominator makes `$`/100 lines answer "per
  line-visible spend" rather than "per dollar", which is a different and arguably less useful
  question — so it wants a paragraph in [ADR 0011](docs/adr/0011-capability-gated-metrics.md)
  and its own real-data proof, not a quiet change to a figure people quote. The generic
  invariant test deliberately leaves `LinesAdded` out until this is settled.
  **Settled, pending the patch:** the denominator is gated by capability, and the share it
  excludes is quantified beside the figure — the shape `B139` established for unpriced tokens.
  `$`/100 lines then answers "per line-visible spend" and says so; a window where sources
  recording no lines carry a material share of the cost states that share rather than letting
  the rate read as if it covered everything. Leaving the denominator and disclosing an
  understatement was rejected: it keeps a known-wrong number on screen, and the honest rate is
  the one a reader can act on. Still needs the ADR 0011 paragraph and its own proof — zero
  effect on the audited store means the proof will be `constructed`.
- [ ] **B117 · `metrics verify` prints a blank confidence label** — S · both — the verify path
  renders a plugin's `Result` without `analyze.Stamp`, so the label the summary leads with is
  empty (`Confidence:  · 3 usage rows · activity coverage 0%`) while the coverage axes beside
  it read as a real zero. Confirmed against v0.10 as well, so it is not new. Either stamp the
  window like `analyze` does, or render the unstamped case as what it is — a conformance
  check on the document, not a verdict about anyone's data.
- [ ] **B116 · `model` is the one column a re-read still cannot correct** — S · solo — the rest
  of this item shipped in v0.24: `ts`, `project`, `subpath`, `entrypoint` and `git_branch` now
  follow the rule `granularity` answered yes to — **and the downward watch does not cover them**:
  `lowerRestateSQL` reads the thirteen activity counters only, so a parser regression on those
  five is writable and uncounted, where the fill-only rules made it unwritable. Widening the
  predicate (or renaming the counter to "restated") is part of this item now, `session_step.ts` and `kind` move with them, and
  ingest counts the rows a re-read moves *down* so a corrected rule and a parser regression are
  no longer indistinguishable from the store's side. What is left is `model`, which is still
  fill-only — `CASE WHEN model = '' THEN ? ELSE model END` — because a source can legitimately
  emit a record before it knows the model (Cline reads it from a sidecar that may not exist yet)
  and a first read that saw none must not pin those tokens as unpriceable forever. That reasoning
  argues for filling a blank; it does not argue against *correcting* a wrong one, and a model
  name decides the price, so a fix to model extraction reaches nothing today. The question is
  whether a later read of the same file is the authority on a name an earlier read already had.

## Pool — from the v0.13 whole-codebase review

Same rule as the pool below: reproduced against the maintainer's own store before being
written down. What v0.13 fixed is in [CHANGELOG.md](CHANGELOG.md) and not repeated here. `B139`
left this pool for the calibration milestone above — an unwatched price table is a figure checked
against nothing, which is that milestone's subject rather than a cleanup — and shipped from it.

- [ ] **B140 · a Claude `<synthetic>` message is counted as a turn** — S · solo — Claude Code
  writes locally-generated assistant messages (API errors, refusals to continue) with
  `"model": "<synthetic>"` and an all-zero usage block. **Measured:** 551 of 163,976 records
  (0.34%) on the audited store carry zero tokens, zero tool calls and zero lines — 565 counting
  the other zero-token Claude rows. **One live consequence already found**: `<synthetic>` has no
  price, so it sets `Row.HasUnpriced` while carrying no spend, and the first cut of v0.13's cost
  gate keyed on that flag and failed every run on a fully priced store. The gate now keys on
  unpriced tokens; every *other* reader of `HasUnpriced` — `report`, `status`, the dashboard —
  still prints the "includes unpriced usage" asterisk on a window where nothing is missing.
  Beyond that they are real events but not API requests, so
  every per-record denominator (tokens per turn, the confidence envelope's "N usage rows",
  the coverage weighting) is 0.34% larger than the population that could feed its numerator.
  **Below the threshold of a wrong number here and recorded so it is not rediscovered as one**;
  it matters if the share ever grows, which a rate-limited or erroring session would do. The
  decision is whether a synthetic turn is a turn — `B113`'s "how a turn ended" is the same
  question from the other side, so they probably answer together.

## Pool — from the v0.14 whole-codebase review

The defects this review found are fixed and listed in [CHANGELOG.md](CHANGELOG.md). What it
found that was **not** a defect — three places where the code did exactly what it said, and the
question was whether what it said was still the right call — was decided rather than deferred,
and all three shipped in v0.14:

- `B141` — the team panel's per-member row now shows engagement only; output and spend are the
  team's total. A pseudonym is not anonymous to a colleague who knows the roster.
- `B142` — a projected "per month" is a calendar month, not thirty *active* days, which had been
  inflating the figure a flat plan is compared against.
- `B143` — the parser plugin protocol rejects a field it does not define, as the metric and rule
  protocols already did.

## Pool — from the v0.22 whole-codebase review

What this review found and v0.22 fixed is in [CHANGELOG.md](CHANGELOG.md) and not repeated
here. These are the items it measured and deliberately did not fix: two too large for the
release, five that are judgement calls about invented thresholds rather than defects, and one
file-size norm.

- [ ] **B173 · `usage_record` is unbounded and has no measured bound anywhere** — M · solo —
  measured on the maintainer's store: **1.31 MB/day over 44.7 days → ~476 MB/year**, with no
  horizon, no automatic prune, and only a manual `clear --older-than` to reduce it. Today's big
  table is the *bounded* one — `session_step` is capped at 30 days ≈ 136.3 MB steady state,
  re-measured for v0.26 — and
  the unbounded one crosses it in about 78 more usage-days and never comes back down. The rate is
  one machine's; the unboundedness is structural. What makes this hard rather than a second
  horizon: a usage record is the only thing a re-import can rebuild *and* the only thing a report
  reads, so a retention rule has to answer what a five-year cost trend is worth against a store
  nobody can carry. Deciding that is the work; the size is not in dispute.
  **v0.24 made it visible without deciding it**: `doctor` projects the store's own measured
  growth from live bytes (2.9 MB/day → at most 1.0 GB/year on the maintainer's store, stated as
  an upper bound because the rate averages the horizon-bounded timeline with this table) and
  takes `--db`, so the central
  store an operator cannot walk over to is diagnosable. What remains is the decision this item
  is named for -- what to keep, at what resolution, for how long -- which is a policy, not a
  line of SQL.
- [ ] **B190 · read the uncovered branches, not the coverage number** — M · solo — measured on
  this build: `internal/parser` 47.7%, `internal/recommend` 60.6%, `internal/usage` 70.0%,
  `internal/reconcile` 70.7%, `internal/config` 72.0%, `internal/vcs` 74.1%, `internal/cli`
  75.8%. The number is not the finding and no gate is wanted (Goodhart, and
  [ADR 0002](docs/adr/0002-code-standards-and-enforcement.md) says so): the work is to open the
  uncovered branches in the packages that carry a *contract* rather than a rendering — the
  shared parser limits, reconciliation's failure paths, CLI error paths, and the recommendation
  families whose abstention rules are the whole point — and to give each one a test or a written
  reason. `internal/layer` and `internal/trace` report no direct package coverage at all and are
  exercised only through consumers, which is a different risk from a low percentage.

- [ ] **B187 · attribute usage to the MCP server that caused it** — M · both — Claude Code's own
  `/usage` breaks recent consumption down by skill, sub-agent, plugin **and MCP server**; assaio
  has the first two (`skill-economics`) and not the last. An MCP tool call is named
  `mcp__<server>__<tool>` in the transcript, so the server is recoverable from the tool name
  without any new source — the work is a vocabulary decision (is an MCP server a fourth
  attribution dimension or a facet of the tool taxonomy?), a store column, and the same
  minimum-volume rules the skill dimension already carries. Verified against Claude Code
  2.1.238.

- [ ] **B186 · one assistant message, two transcripts, two different counts** — M · solo — found
  by the downward-restatement canary `B116` asked for, on its first run against the maintainer's
  store. A Claude sub-agent's transcript and its parent session both carry the same assistant
  `msg_` id, and the two files see different amounts of it: for
  `msg_011Ce6Ytmoyd4FXNBViS2XzB` the parent holds 4 `tool_use` blocks and the sub-agent file
  holds 1. The dedupe key is the message id alone, so whichever file the walk reads last wins
  the restate, and the stored `tool_calls` oscillates between 4 and 1 forever — the canary
  reports the same 6 rows on every `backfill --full`, unchanged while the store gains new
  records around them. **Measured: 6 of 192,277 records (0.003%), a swing of ~18 tool calls
  against a window of 216,790**, which is why this is a defect to fix rather than a figure to
  distrust. The decision is which file is the authority on a message that appears in two: the
  parent transcript is the authority on the parent's turn and the sub-agent file on its own, so
  the key probably has to carry the scope it was read in — the same shape the sync path already
  uses to keep two members' rows apart. Whatever the answer, the canary is the regression test:
  it must reach zero and stay there.

- [ ] **B185 · give the withdrawn verdicts a line derived from your own history** — L · both —
  v0.24 withdrew the good/bad call from fourteen metrics because their thresholds were numbers
  picked once (`B176`–`B178`, `B180`, and the same class in `reasoning-share` and
  `session-taxonomy`). The figures stayed; what is gone is a colour this project could not
  defend. The credible line is not a published constant — there is none for most of these — but
  **the same store's earlier windows**: "your tool-error rate is 22%, against 9% over your own
  prior 90 days" cites a source, derives from a distribution, and is the claim a reader can act
  on. It needs per-metric history that assaio does not keep: today only the *current* window is
  recomputed, and `digest` stores a bounded basis per run rather than a per-metric series.
  So the work is (a) decide what a metric's own history is and what it costs to store — `B173`'s
  size question, one table smaller — (b) restate it when a parser fix corrects the past, and (c)
  a floor of history below which the comparison is withheld rather than made against two weeks.
  Until it exists, `analyze` leads on the four metrics whose lines *are* derived or cited, and
  that shortness is the honest state, not a gap to paper over.

- [ ] **B196 · attach a published threshold to a metric, once one fits** — M · solo — the
  register (`internal/threshold`) refuses every candidate today, and
  `TestNoRegisteredCandidateGradesAnything` keeps it that way until someone changes it
  deliberately. What would let one attach: a churn or survival figure whose numerator,
  denominator, window, data source and layer match a metric assaio already computes — or an
  assaio-side change that closes the gap, chiefly age-matched survival (a fixed post-commit
  horizon rather than "until now") and AI attribution on the git side, which together would
  make `survival` comparable to GitClear's two-week churn on all five properties instead of
  three.

- [ ] **B197 · `report --format csv` publishes a cost its own columns cannot explain** — S ·
  solo — the export carries `in`, `out`, `cache_read`, `cache_write` and `cost`, but no
  `cache_write_1h`, and the 1-hour cache-write tier is billed at its own rate. Recomputing the
  cost from the four published token columns therefore misses exactly
  `cache_write_1h × (1h rate − standard write rate)` — measured per model against the
  maintainer's store, it agrees to six decimals for all eight priced models, and for
  `claude-opus-5` alone the unexplainable remainder is **$2,582.38**. A machine surface whose
  own figures do not reconcile is the class of defect this project fixes rather than documents.
  Found while building an independent check for `reprice`, which had to bypass the export and
  price straight from `internal/pricing/litellm.json` to verify anything.

- [ ] **B198 · `make fuzz` flakes on `agy` at the `-fuzztime` boundary** — S · solo — measured
  2026-09-02: `internal/parser/agy FuzzParseTranscript` fails roughly one run in three with
  `context deadline exceeded` at exactly the deadline, writing **no crasher** — three consecutive
  25s runs passed with zero new corpus entries, and replaying the seed corpus takes 0.269s, so
  there is no slow input. It is Go's worker-shutdown race, and it grows likelier as the cached
  corpus does (487 baseline inputs and climbing). This matters because the nightly CI fuzz job
  added this cycle opens an issue on a finding: a target that fails without producing one will
  file noise, and a gate that cries wolf gets muted, which is worse than not having it. Options:
  raise the per-target `-fuzztime` so the drain has room, treat "failed with no new corpus entry"
  as a retry rather than a finding in the workflow, or both.

- [ ] **B199 · the opt-in commit hook is stricter than the gate it guards** — S · solo — measured
  2026-09-02 while committing v0.25.0: `make fmt` (`golangci-lint fmt` v2.12.2) left three test
  files that the pre-commit hook's standalone `gofumpt` 0.10.0 rejected, so the commit was refused
  on code CI accepts. The embedded gofumpt inside the pinned golangci-lint does not enforce the
  newer trailing-comma rule the standalone binary does. Anyone who runs `make hooks` inherits a
  formatter stricter than `.golangci.yml`, and the disagreement is silent until a commit is
  blocked. Pick one authority: pin the hook to the same gofumpt golangci-lint embeds, or raise
  golangci-lint until the two agree. Do not fix it by loosening the hook — being stricter is not
  the defect, disagreeing is.

- [ ] **B200 · `internal/analyze/analyze.go` is over the file budget** — S · solo — 261 lines
  against the ~200 the review norm sets, after the duplicate-name panic landed in v0.25.0. It is
  three responsibilities in one file: the `Input`/`Result` contract, the registry, and the
  `Validator` interface with its scoping rules. Splitting it is mechanical but touches the
  package every validator imports, so it wants its own change rather than a ride-along.

- [ ] **B201 · two per-record relationships are checked only at the server boundary** — S · solo
  — `Sidechain ∈ {0,1}` and "the tool-purpose split sums to `ToolCalls`" live in
  `internal/server/validate.go`, although `internal/usage/bounds.go`'s own header says it exists
  because the plugin and sync boundaries were drifting. Not reachable today: the plugin wire
  record carries neither field, so `CheckCounts` is never asked about them from that direction.
  It is latent — the wire has grown before, and a field added to it would inherit no check.
  Related: a rejection names the rule but not the offending field (`negative count field`), which
  on a `sync` push tells a team member what was wrong but not where.

- [ ] **B202 · `skill` and `agent` labels reach the store with no length bound** — S · solo —
  they are the only log-sourced *text* that is stored, and on the local path they are taken
  verbatim from the vendor's `attributionSkill`/`attributionAgent`. The 512-byte clamp exists
  only where a record arrives from outside the process (`server.maxStringField`,
  `plugin.maxWireStringLen`). PRIVACY.md calls them "short text labels", which describes vendor
  behaviour rather than an enforced invariant. Note `parser.VocabularyToken` is the **wrong**
  clamp here — its charset excludes `-`, so it would drop `code-review`; a plain length bound is
  the fitting hardening.

- [ ] **B203 · the codex yield canary cannot separate a format break from a changed habit** — S ·
  solo — it fires today on the maintainer's store (`3.5 record(s)/file across 1170 file(s),
  history is ~22.9`) and `doctor --strict` exits 1 on it. Investigated 2026-09-02 and it is **not
  drift**: 4142 `token_count` events in the corpus against 4054 parsed records is 97.9% yield, and
  1159 of 1170 files carry `originator=codex_exec` — hundreds of short one-shot batch runs instead
  of a few long interactive sessions. The canary is right to fire, because it cannot tell the two
  apart; what is missing is a second figure that can. Events-per-file beside records-per-file
  would have answered it in one line.

- [ ] **B204 · `codex` writes an ordered step timeline nobody reads** — M · solo — measured
  2026-09-02: `item_completed` events appear in 984 of 1170 rollout files, up to 98 per file,
  under the `codex exec` originator that now dominates this corpus. The depth matrix says "no
  ordered step sequence is read from its logs yet, so the behaviour detectors have nothing to read
  for it" — that gap is now closable without a new source. Wants a golden fixture and a fuzz seed
  from a real capture, and a check that the sequence is genuinely ordered rather than emission
  order.

- [ ] **B205 · retention is reported for `claude-code` only** — S · solo —
  `internal/cli/doctor_store.go` hardcodes it, so the line that tells a reader "35 of those days
  came from transcripts already deleted and exist only here now" is silent about the other five
  sources. Circumstantial evidence that `agy` rotates: `brain/` holds exactly 500 conversation
  directories and the earliest transcript entry is 30 days after the installation id was created.
  Not proof — but if it rotates, agy history older than the vendor's window exists only in the
  store, and nothing says so before a deletion.

- [ ] **B206 · README's `effectiveness` worked example prints a caveat the binary no longer
  emits** — XS · solo — `README.md:252` shows "Not every source records changed lines; the ones
  that do not contribute cost but no line counts". `effCoverageNote` now emits one of four
  sentences and none of them is that one; for the all-`claude-code` data in the example it would
  print "Every source in this table records changed lines." A worked example is the thing a reader
  checks their own output against, so a stale one reads as a bug in their store. Left as an item
  rather than fixed in the v0.25.0 tree deliberately: it is prose, not a wrong figure, and it
  wants a pass over *every* sample block in README at once rather than a single line.

- [ ] **B207 · the shared ingest path still documents itself as Cline-only** — XS · solo —
  `internal/ingest/state.go:30` ("one discovered file or Cline task directory"), `:52` ("dirInput
  signs a Cline task directory"), `internal/ingest/parse.go:23` ("one file's (or cline
  directory's)"). `dirInput` now also signs an Antigravity conversation, whose tree includes
  `transcript_full.jsonl` — a file the parser never reads, so a change to it invalidates the
  signature. That is correct conservative behaviour, but the comment no longer names what it
  signs. `docs/extending/data-source.md` was generalised when `agy` landed; the Go comments were
  not.

- [ ] **B208 · `adoption` asserts a capability where it measured the data** — S · solo —
  `internal/analyze/adoption.go:62` derives `breadthKnown` from `inv.Unattributed < len(in.Usage)`
  ("no row named a project") and then prints "no source here records a working directory". Those
  are different statements, and the purity effect is real — a breadth of 0 is replaced by
  `neutralPurity`. Not a live defect: only `claude-code`, `codex` and `copilot-cli` write `Cwd`,
  so today every window where the two coincide really is one whose sources record no cwd. It
  becomes one the day a cwd-carrying source is added, and the failure will be silent because the
  note asserts the conclusion rather than the premise. The rest of v0.25.0 moved these decisions
  onto `parser.Answers`; this one was left because it needs a project signal id that does not
  exist yet.

- [ ] **B209 · `CollapseForTable` is an invariant held by a doc comment and one caller** — XS ·
  solo — `internal/cli/report.go` must call it before `RenderTable` or the day table prints the
  duplicate rows v0.25.0 removed. `Aggregate` already returns rows unchanged for `by == "day"`, so
  `RenderTable` could call it itself and the invariant would be unbreakable. Also worth noting:
  `collapse_test.go`'s fixtures carry no `Tokened` row, so every token cell in them renders `—`
  and a regression in the token columns would not show there.

## Refusals (will not build, regardless of demand)

- No "estimated time saved" headline — the logs contain no counterfactual.
- No lines-of-code or token leaderboards; nothing ranked per named individual, ever.
- No per-person analytics outside a deliberate, governed team-mode opt-in.
- No cohort/percentile comparisons without a minimum cohort size and explicit consent.
- No wrapping `analyze --format json` in an object. The ordering (`B148`) wanted a
  `{worthAttention, results}` document and it would read better, but the array is what every
  script already reads and a shape change breaks them all silently. The ordering rides on the
  results it promotes instead, as `lead`. Decided during v0.16; revisit only at a major, and
  only with the break stated in the changelog before anyone hits it.
