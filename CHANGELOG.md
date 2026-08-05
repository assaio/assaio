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

### Fixed
- **The field audit's Codex cache-write row overstated its own consequence.** It said Codex cost
  is a floor because the cache-write count is unread. The count is unread, but on the audited
  corpus it carried a value on 238 events and was **zero on every one**, so no figure is
  currently wrong. The row and `B107` now say what was measured rather than what would follow if
  it were non-zero — the same standard the audit applies to a vendor's fields applies to its own
  claims.

## [0.10.0] - 2026-08-05

### Added
- **The unread-field audit, source by source** (`B105`) — a table per source in
  [docs/extending.md](docs/extending.md#what-each-sources-log-carries-and-what-assaio-reads)
  putting every field a tool writes into exactly one of two states: **extracted** (a catalog
  signal is computed from it and a golden covers it) or **skipped, with the reason written
  down**. A field the vendor does not document is skipped with that stated, never guessed at
  from its name. Each table names the corpus it was built from, because a field the corpus does
  not contain is the one most likely to be missing from the table — and two of the five say
  outright that their corpus is too thin to be conclusive.

  What it found, now tracked: Codex reports a token class assaio never reads (`B107`); the
  cache TTL tiers and miss reasons `cache-hygiene` calls invisible are in the Claude log
  (`B108`); Copilot CLI names its tool calls, counts lines per call and records the commit its
  session started at, none of which its depth row claims (`B109`); the human's own correction
  of an AI edit is recorded rather than needing the churn proxy (`B111`); and the Claude Code
  build that wrote each line is on disk, which is the harness-cohort input a later milestone
  assumed would need a server (`B112`). It also found Gemini CLI producing **zero records from
  two discovered files** on a current install, invisible to every drift canary because each
  needs a twenty-file sample floor a two-file source can never reach (`B110`).

### Fixed
- **A metric no longer reads a source's silence as a zero.** Four validators and the `status`
  Sessions block read per-session fields — edits, turns, focused minutes, compaction — that
  three of the five in-tree sources never record, and counted every one of those sessions as a
  zero. A Gemini CLI, Cline or Copilot CLI window therefore read as **100% conversational, 0%
  produced code, 0 marathons, 0% compaction** and carried a verdict on all four; on a Claude
  window the two Copilot sessions did the same in miniature. Each figure is now computed over
  the sessions whose source can answer it, declares that reach as its signal coverage, prints
  `—` where nothing in the window records it, and withholds the verdict rather than certifying
  a silence. Affects `session-taxonomy`, `context`, `rhythm` and `turn-efficiency`; `status`
  names the narrower basis on its own line.
- **`insufficient` now says which of the three ways a verdict rests on nothing.** One sentence
  covered all of them, so `explore-produce`, `friction` and `skill-economics` printed
  "nothing to measure in this window" over a store holding 119,896 records — directly beneath
  their own caveat saying the opposite. The three are different facts and now read as such:
  *nothing in this window can answer it* (the metric declared zero reach), *0 tool calls* (it
  counted none of its own unit), and *no stated basis* — which is what an exec metric plugin
  omitting `confidence.samples` means.
- **A completed Claude sub-agent is a session total, not a turn.** One record summarizes a
  whole sub-agent run, and labelling it `turn` made every per-turn figure count it as a single
  very large one — 1,015 such rows on the maintainer's machine, averaging 2.4× a real turn's
  output. It is now `session` grain, so `coverage` reports the mixed window it always was and
  `model-right-sizing` stops counting an aggregate as a turn.
- **The rejection rate gets its own denominator.** It divided by every tool call in the
  window, including calls from sources that record no refusal at all, so a Codex-heavy window
  read as calmer than it was. `friction` and `rework` now divide by the calls whose source
  records a refusal, and say so.
- **`concentration` stops blaming a project for a source that writes no lines.** A project
  running entirely on a cost-only source writes zero lines by construction, so its whole token
  share read as the widest spend-versus-output gap in the window. Those projects are excluded
  from the gap and counted in a caveat instead.
- **`skill-economics` states its reach, and a lone label no longer reads as zero tokens.** The
  concentration share is of attributed tokens, which are a slice of the window — 18% of it on
  the maintainer's store — and that is now the metric's declared signal coverage. A window with
  a single skill prints `—` for the largest share rather than a share of nothing beside
  "attributed tokens: 0".
- **`doctor` reports a failed discovery instead of printing it as "not detected".** A root it
  could not read counted as zero files, while `backfill` reported the same condition as an
  error.
- **`make fuzz` runs the Copilot CLI parser's fuzzer.** It shipped one in v0.6.0 and the target
  never listed it, so the guarantee that every parser is fuzzed was true of the code and not of
  the gate. It passes; no crasher was found.

### Changed
- **Two signals join the catalog**: `ai.compactions.count` and `ai.rejected.count`, both
  answered by Claude Code (compaction also by Codex). They existed as stored columns and as
  the subject of two verdicts without ever being declarable, which is why the metrics over
  them could not tell absence from zero. `signals list|describe|coverage` cover them like any
  other.
- **Every remaining hardcoded tool list in user-facing text now reads from the depth matrix**:
  `skill-economics`'s provenance caveat and the `explore-produce` explain page named tools in
  prose that the next parser would have made wrong.
- **[ADR 0011](docs/adr/0011-capability-gated-metrics.md)** records the rule the fixes above
  share, because a future validator could undo it by reading a session field directly: a metric
  computes only over the sources that record its field, and a generic test asserts every
  registered validator returns the same result whether or not those fields carry a value for a
  source that cannot fill them.

### Compatibility
- **Migration `0006_subagent_session_grain.sql`** relabels stored Claude sub-agent aggregates
  from `turn` to `session`. A re-parse cannot reach them: once a sub-agent has its own
  transcript file the parent's aggregate is suppressed at parse time, so the row already
  written is never offered to the store again. The migration is an `UPDATE` over
  `dedupe_key LIKE 'agent:%'` (and the sync server's `<member>:agent:%`) scoped to
  `claude-code` — no row is added and no column widened, so the store does not grow and there
  is nothing to clean up afterwards.
- `InsertLocal` now restates `granularity` from the current parse rather than keeping the
  stored value. It is the one column assigned instead of maximised, because a build that
  learns a record summarizes a whole run has to be able to say so; `Insert` — the sync path —
  stays first-write-wins and never relabels another member's record.

## [0.9.0] - 2026-08-04

### Added
- **Every verdict now says how much of your window it actually describes.** The confidence
  envelope gained a fourth axis beside coverage, pricing and granularity: `signalCoverage`,
  the share of the window a metric's own subject reaches. The three existing axes describe the
  *window*; this one describes the *question*, and only the metric can answer it. `analyze`
  names it when it is the weakest one — `Confidence: low · 43 active days · signal coverage
  <1%` — and a metric declaring it covers none of the window reads `insufficient`, which is a
  different fact from a thin answer. Exec metric plugins declare it with the same key, and one
  that omits it keeps claiming the whole window, exactly as before
  ([docs/extending.md](docs/extending.md#write-a-metric-plugin-any-language)).
- **`survival` reports what merges hold.** `git log --numstat` prints no diff for a merge, so
  a conflict resolution is a hole in git's own reporting rather than a change of size zero.
  The report now names it: `merges: 1 commit(s) holding 50 line(s) in HEAD, counted in neither
  figure below`.
- **The attribution conformance corpus** (`B93`, [ADR 0010](docs/adr/0010-attribution-conformance-corpus.md)).
  Ten scenarios — 1:1, N:1, 1:N, wrong branch, overlapping users, delayed commit, sub-agent,
  genuinely ambiguous, manual correction, replay after an algorithm change — each building a
  real git repository read back through the real collector, each stating what any attribution
  engine (`B85`) must conclude and where it must refuse to. Ambiguous fixtures are verified to
  be *structurally* ambiguous rather than merely declared so, and two stand-in engines run
  against the corpus in its own tests: the nearest-commit heuristic, which must fail, and an
  honest one, which must pass. Nothing user-facing yet; it is the specification the engine
  will be built against.

### Fixed
- **`survival` counted merge lines as survivors that were never counted as added.** `git
  blame` names a merge for every line of a hand-resolved conflict while `numstat` reports none
  of them, so the rate divided a number by a total it was never part of — on a fixture with a
  50-line resolution that printed `50 surviving of 3 added (100%)`. Both sides now count the
  same commits, the merge is reported separately, and the clamp that hid the contradiction
  behind a flat 100% is gone.
- **A figure computed from a sliver of the window carried a confident envelope.** On real data
  `reasoning-share` read a 20% share off under 1% of the output and reported `high`, while
  `signals coverage` called the same signal partially supported — two surfaces disagreeing
  about one number. `reasoning-share`, `friction` and `explore-produce` now declare their own
  reach, and a window where no source can answer at all reads `insufficient` instead of `high`.
- **A real coverage share no longer rounds to an absent-looking `0%`** in the confidence line;
  it reads `<1%`, the same honest rounding every other percentage in the reports already used.
- **The README's headline and the exec-plugin schema table omitted GitHub Copilot CLI**, which
  has been a supported source since v0.6.0 — the README said "five sources" three paragraphs
  further down.

### Compatibility
- `analyze --format json` gains an optional `confidence.signalCoverage` field. It is absent
  when a metric does not declare one, so a consumer reading the envelope keeps working.
- Exec metric plugins (ADR 0004) may set the same key. Omitting it means what it meant before,
  so every released plugin is unaffected.
- Survival rates change for repositories containing merge commits: the figure was previously
  inflated by blamed merge lines and is now computed over non-merge commits only. No stored
  data changes — `survival` reads git directly and persists nothing.

## [0.8.0] - 2026-08-03

### Added
- **The local git evidence collector: what your commits changed, never what they changed it
  to.** `internal/vcs` reads a repository and emits `vcs.commit.observed` — the first
  observation assaio produces that is not parsed out of an AI tool's log, and the first
  consumer the canonical event contract has ever had. A commit observation carries parent
  count, changed files, added and removed lines, a **six-way file-category split**
  (test / source / docs / config / generated / other) and a revert flag. There is no field for
  a path, a branch name, a commit message or a diff, the split must add up to the file count
  beside it, and the whole thing is classed `local-only` until the correlation threat model
  (`B100`) says otherwise (`B91`,
  [ADR 0009](docs/adr/0009-local-git-evidence-collector.md)).
- **`survival` reports what the window actually changed.** It no longer shells out to git for
  its own commit set: it reads those observations, so the survival rate now travels beside the
  mix of files the window touched and the number of commits git itself labelled a revert.
  Categories come from a naming heuristic and say so, and unreadable commits are counted and
  named rather than silently dropped.

### Fixed
- **Copilot CLI was only half-wired, and three surfaces were the proof.** The parser landed
  in v0.6.0, but three places still kept their own list of tool names instead of reading the
  depth matrix: `sync` rejected every `copilot-cli` record as an unknown tool, so a team
  member using it synced nothing and was told nothing; `clear --tool copilot-cli` failed the
  same way, which meant its data could not be deleted per source; and `reasoning-share`
  skipped it while printing "Only Codex and Gemini CLI report reasoning tokens today",
  though Copilot has reported them all along and the store held them unread.
- **The signal catalog claimed every source reports reasoning tokens.** `ai.tokens.reasoning`
  was bundled into the cost signals every row inherits, so `signals coverage` reported it as
  fully supported on a Claude-only machine — where the real answer is *none*, since Claude
  Code never surfaces a thinking count. On the maintainer's own store the figure moved from
  "100% of tokens" to "<1%, codex", which is the honest number. Reasoning is now declared per
  source, like every other capability.
- **A source that records lines and nothing else no longer passes for full activity coverage.**
  The confidence envelope on every verdict was computed from the tier table's one-bit
  `Activity` axis, which reads true for Copilot CLI — so a Copilot-only window reported
  *activity coverage 100%* while `signals coverage` correctly said its edit, tool-call and
  rework signals were unsupported. Two surfaces, two answers to one question. The envelope now
  asks the per-signal question ([ADR 0008](docs/adr/0008-signal-catalog.md)), and `coverage`
  separates the two facts it used to conflate: "cost only" means a source contributes no
  changed lines, and a source with lines but no edit counts is named as partial rather than
  hidden behind either verdict.
- **A session annotation no longer selects another member's usage.** On a central store the
  label filter matched on `session_id` alone while every other label query joins on
  `(session_id, member)` — the label's own primary key — so two members whose locally
  generated session ids collided could be filtered into each other's results. Local stores,
  where `member` is always empty, were never affected.
- **A Copilot session with no id or no timestamp is skipped instead of stored.** Its dedupe
  key is `<session>:<model>`, so an unidentifiable session collapsed every other one like it
  into a single row under `ON CONFLICT DO NOTHING`, and an undated one was stored at year
  one. Both are now counted as skipped, the same skip-and-count policy every parser follows.

### Changed
- **What a source can answer has exactly one place.** `parser.Answers(tool, signal)` is now
  the only capability question in the codebase, and `parser.Tools()` the only list of source
  names; `friction`, `reasoning-share`, `coverage`, the confidence envelope, `signals
  coverage`, `sync` validation and `clear --tool` all read them instead of keeping a private
  copy. The exec-plugin floor moved there too, and dropped its claim on the per-turn signals:
  a plugin declares turn or session grain per record, so a turn count can no more be assumed
  from it than from a session-total source.
- **Caveats stopped naming sources.** The line-coverage, tool-call, failure-capture,
  project-attribution and reasoning notes across `analyze`, `effectiveness`, the dashboard,
  the `explain` pages and the binary's own `--help` spelled out "Claude Code and Codex" or
  "Gemini CLI and Cline" — a sentence a new parser makes wrong, and most of them already
  were. They now describe the property and point at `assaio-agent signals coverage`, or name
  the window's own sources from the matrix. The `coverage` validator also stops printing a
  cost-only caveat when the window contains no cost-only source, and no longer blames
  cost-only sources for a window whose gap is a source recording lines and nothing else.

## [0.7.0] - 2026-08-02

### Added
- **`signals` — what assaio can tell you, and what your own data supports.** The source-depth
  matrix answers "what can this tool tell me"; this answers the question people actually have,
  "can it tell me *this*, here". `signals list` names all eighteen; `signals describe <id>`
  says what one counts, which grains it is honest at, and — the field that earns the catalog
  its keep — **what a zero means**, because "no rework happened" and "this source never
  recorded rework" are different facts and every metric that confuses them lies confidently.
  `signals coverage` reads the window in your store and reports each signal as fully, partly
  or not at all supported, naming the sources that can answer it. Support is computed from
  your real token mix and what each parser declares it can answer, never from a claim the
  catalog makes about itself, and a partial share never rounds up to a whole 100% (`B90`,
  [ADR 0008](docs/adr/0008-signal-catalog.md)).
- **The source-depth matrix now declares capability per signal, not per axis.** `deep`,
  `standard` and `import-only` still summarise a source, but "has activity" turned out to be
  one bit over a source that records changed lines and nothing else — which is what Copilot
  CLI is. Its first run on real data reported sixteen of eighteen signals as fully supported
  when the honest answer is ten, because Copilot totals a whole session and so carries no
  turn count, no edit count, no tool calls and no rework. Each parser now lists the signals it
  actually answers, a test asserts that list never contradicts the tier axes, and adding a
  parser means answering "which of these can you produce" instead of one yes-or-no.

### Changed
- `mark` no longer answers an ambiguous session prefix with every match it found — on a real
  store a one-character prefix printed 48 ids on one line. It now lists six and counts the
  rest, the way git reports an ambiguous short revision.

### Internal
- The dashboard's project drill builds an `Input` without the window-only fields the CLI and
  the server fill (`Skills`, `Agents`, `TurnSizing`), so a validator reading one of those
  while not being `WindowScoped` would report "no attribution" as a fact about the project.
  That has never happened, because all three such validators are `WindowScoped` — but nothing
  enforced it, and a twentieth validator would have broken it silently into a plausible-looking
  verdict rather than a crash. A test now asserts it across every project-scoped validator.
- `internal/store/label.go` grew past the file budget doing two jobs; naming a session is now
  `session_ref.go`, which is where the git evidence collector will look for it.
- **The canonical event contract, the first piece of the evidence graph.** Collectors that are
  not log parsers — a local git reader, a pull-request connector — need a shape to emit and
  analyzers need one to read, or every new evidence source teaches every analyzer a new record
  and the fields that make this project defensible get re-invented per source. `internal/event`
  is that shape: a versioned envelope carrying source, source version and the build that read
  it, the two clocks and how much to trust them, grain, privacy class and provenance, wrapping
  one closed payload. Today's `usage.Record` adapts into it — one usage observation always, one
  edit observation only when the record carries activity, so a source with no edit extraction
  emits nothing rather than a row of zeros. It is an **interface contract, not a storage
  format**: no event table, no migration, and nothing user-facing moves. Content is impossible
  by construction — payloads are closed structs of counts, closed vocabularies and identifiers
  the source itself assigned, a test walks the contract and fails on any string field nobody
  accounted for, and paths and branch names are dropped at the adapter (`B89`,
  [ADR 0007](docs/adr/0007-canonical-event-contract.md)).
- Proving it against 324,416 real records changed the design twice, which is the point of
  proving it. Rejecting an event whose source timestamp is newer than the batch's reading time
  dropped 51 real records from sessions still being written, so the two clocks are no longer
  ordered against each other — a reading time is not a causality claim, and the evidence path
  must not be stricter than the store it mirrors. The same run surfaced `B101`: 404 sub-agent
  aggregates whose id collides across projects, which the store has always resolved by keeping
  whichever file was ingested first.

## [0.6.0] - 2026-08-02

### Added
- **Sessions can be labeled with what the work actually was.** `assaio-agent mark` attaches
  a task class, an outcome and a difficulty to a session — the one fact session logs never
  contain, and one that cannot be recovered by reading prompts. With no session named it
  marks the most recent session in the repository you are standing in, so labeling is a
  command you run right after the work; `--last` takes the newest session anywhere, a git-style
  id prefix names one exactly, `--list` shows what is labeled and what is not, and `--unmark`
  undoes it. Setting an axis later merges rather than replaces, so a task set today survives
  an outcome recorded tomorrow (`B80`).
- **Every metric can now be read per kind of work.** `analyze --task refactor` (and
  `--outcome`/`--difficulty`) recomputes every validator over just those sessions, and
  `report`/`effectiveness --by task|outcome|difficulty` group the same way — which is what
  makes `$`/100 AI lines on bugfixes comparable against features instead of averaged into
  one number nobody can act on. A filter narrows all five window queries at once, never
  just the usage rows, and the verdicts that describe the whole window rather than a slice
  of it (subscription fit, pooled attribution, per-model turn counts) are skipped and named
  as skipped rather than restated over a subset they do not describe.
- **`intent` validator** — how much of the window carries a label and whether that is
  enough to compare kinds of work at all. It has no unfavorable verdict by design: scoring
  how diligently someone labels, or judging a work mix that is genuinely all one kind, is
  not something this tool does.
- **`clear --labels`** — the deliberate way to delete session labels. `--all`,
  `--older-than` and `--tool` now leave them alone and report how many they kept, because
  labels are the only data in the store that no re-import can rebuild.
- **GitHub Copilot CLI is a supported source.** Its session logs
  (`$COPILOT_HOME`, else `~/.copilot/session-state/<id>/events.jsonl`) carry a complete
  accounting per model — input, output, cache-read, cache-write and reasoning tokens, plus
  the session's added and removed lines — which assaio now reads. Two modelling decisions
  are worth stating because both could quietly produce wrong numbers: Copilot's
  `usage.inputTokens` is the **whole** prompt including tokens written to cache, so the
  uncached share is read from `tokenDetails` instead — using the total beside the
  cache-write count would have roughly doubled the estimated cost of a cached session. And
  code changes are reported once per session with no per-model split, so they are credited
  whole to the model that made the most requests rather than divided by a ratio nobody
  measured. Copilot only totals a session when it ends, so records are session-granularity:
  they are marked `‡` in reports, lower the turn-level coverage the `coverage` validator
  reports, and carry their gaps in `doctor`'s depth matrix (`B53`).

### Changed
- Labeling a session cannot move a figure anyone was already reading: the annotation join
  is opt-in per query, so `analyze`, `status`, `dashboard`, the exec metric-plugin input
  (ADR 0004, unchanged) and every unfiltered report run exactly the SQL they ran before.
  A regression test asserts the unfiltered output is byte-identical before and after
  labeling.
- `report --by` and `effectiveness --by` accept `task`, `outcome` and `difficulty`. Usage
  from unlabeled sessions is always rendered as its own `unlabeled` group, never hidden.

### Documentation
- **The roadmap now leads with outcome evidence, not recommendations.** Connecting a session
  to the change it produced — commit, pull request, review, CI, merge, survival — moves ahead
  of the recommendation engine, because a suggestion resting on activity and output proxies
  alone is a guess delivered in a confident voice. The milestone table is re-sequenced
  accordingly (evidence graph → harness intelligence → team → cost truth and interoperability
  → the v1.0 stability promise), with two new themes: measuring whether agent configuration
  (`AGENTS.md`, skills, hooks, MCP) actually helps, and mapping the canonical fields onto
  OpenTelemetry GenAI conventions. Connector strategy is stated as depth over count.
- **A milestone no longer carries a version number until it ships.** The table's order is the
  sequence, shipped rows name the release they landed in, and `v1.0` is the one exception
  because there the number *is* the promise. Pre-assigning `v0.7` to a promise made this
  document schedule work it explicitly says it does not schedule — and it broke the moment a
  small release landed between two milestones, which is exactly what this one is.
- **The stated positioning changed.** `$` per 100 AI lines remains a signal but is no longer
  the headline answer to "is AI delivering" — it is an *output* measure, and promoting one to
  an outcome claim is the most likely way this project could start lying.
- README no longer describes the tool as "the v0.1 local agent — the only thing that ships
  today", five releases after that stopped being true; `site/index.html` was three releases
  behind and now describes this one.
- **Four surfaces were counting four sources and eighteen validators.** README, the website,
  `AGENTS.md` and `FEATURES.md` are corrected to five parsers and nineteen validators, and
  Copilot CLI is listed as a source rather than as a roadmap item. `FEATURES.md` had also
  dated the Copilot parser to v0.5, a release it missed by two commits. Miscounting your own
  inventory is the cheapest possible way for an honesty-first tool to be wrong about itself.
- Shipped backlog items are deleted rather than left checked off, as that file's own
  lifecycle rule requires.

### Internal
- Two CI workflows guard the documentation lifecycle mechanically instead of by review
  memory: `site` publishes the page on every push to `main` and fails when it names a
  version other than the latest tag, and `consistency` fails on a completed backlog item, a
  duplicate backlog id, a tagged version with no changelog section, or a changelog section
  claiming a version that was never tagged. Both make one exception, for the same reason:
  the release being prepared. `RELEASING.md` requires the changelog section and the website
  to be updated *before* the tag is cut, and `main` is protected — so a rule that admitted no
  pending version would have made every release-prep commit unmergeable, and the tag it was
  preparing impossible to create. Exactly one untagged version is allowed, and only when it
  is newer than the latest tag.

### Compatibility
- **Migration `0005_session_labels.sql`** adds the `session_label` table (session id,
  member, the three vocabulary values, and when it was marked) and the composite
  `(project, ts)` index on `usage_record` that project-scoped window queries want (`B70`).
  Both apply automatically on the next run. Nothing is rewritten and no existing column
  changes, so an older build reads the same store unchanged — it simply ignores the new
  table. Category values are validated in Go rather than by a SQL `CHECK`, so adding a
  category later is not a migration.
- The store grows by roughly 80 bytes per session you mark by hand. It does not grow with
  the volume of logs ingested, and nothing prunes labels automatically.

## [0.5.0] - 2026-07-31

### Added
- **Format-drift canaries.** Every vendor log format assaio parses is internal and can change
  without notice, and the failure that mattered was never a crash — it was plausible-looking
  under-reporting that no error path reports. After each `backfill`, four canaries judge every
  source against its own recent history: files discovered, records per file read, share of
  lines skipped, and share of records that parse but carry no tokens. A breach prints
  `warning: possible format drift in <tool>` and shows up in a new `doctor` drift section.
  Each canary is paired with a minimum-sample floor and abstains below it, and the history
  baseline is a median, so one odd pass cannot move it. Verified on a real 4.5 GB / 5707-file
  Claude Code corpus: **317,354 records, no canary fired**; renaming the vendor's token fields
  in a 60-transcript copy fired `zero-token: 831 of 831 record(s) carry no tokens (100.0%)`
  (`B58`).
- **`doctor --strict`** exits non-zero when a canary fired or a source you configured
  explicitly in `sources:` finds no inputs at all, so a cron or CI job alerts on drift instead
  of a human eventually noticing the numbers shrank. The second case is the one no canary can
  catch: a path that never worked has no history to have shrunk from (`B59`).
- **`compact`** rewrites the store without its free pages and truncates the write-ahead log.
  Deleting records only ever moves bytes onto SQLite's freelist — the file itself never
  shrinks until something reclaims them — so `clear` now says how much is still held, and
  `doctor` gained a `size:` line making growth legible before it becomes a surprise. It is a
  separate command because rewriting needs roughly twice the store's size in temporary disk
  space.

- **A confidence envelope on every verdict.** A metric's limits used to live in prose
  caveats a reader can skip, so a verdict computed on thin or partial data could be quoted as
  a solid one. Every `analyze` result now carries structure instead: the window's activity,
  priced and turn-level coverage, how many observations the metric counted and what they are,
  when the data was read and by which parsing build, and a derived
  `high | medium | low | insufficient` label. Deliberately not one opaque score — the label
  summarizes, the components stay inspectable, and the text report names the axis that held
  the label down (`low · 1 active days · priced coverage 1%`) because "medium" alone tells
  nobody what to do. `insufficient` is distinct from a neutral read: one says the data gave
  no answer, the other that there was not enough data to ask (`B81`).
- **`init`** — one command for a first run. It detects which supported tools have logs here,
  prints the exact directories it will read **before** reading anything and states that only
  counts are extracted, imports, writes the report, and names the three commands worth
  running next. It writes no config file when the default locations work, and when nothing is
  found it prints the `sources:` skeleton and exits successfully — a machine with no AI tools
  installed is not an error. Network-free; `--non-interactive` for packaging smoke tests
  (`B82`).
- **A source depth matrix.** "Supported" was one word for two very different things: a source
  that reports tokens but no edits cannot answer the questions one that reports both can.
  Every source now publishes its depth — **deep** (tokens + per-turn activity + attribution),
  **standard** (reliable usage with documented gaps), **import-only** (aggregates that cannot
  support session-level conclusions) — with the specific gaps spelled out below the top tier.
  `doctor` prints it for the tools found on this machine, FEATURES.md carries the full matrix,
  and the `coverage` validator reads it rather than keeping its own list (`B83`).

### Changed
- **Reports no longer blend per-turn and whole-session records silently.** `granularity`
  travels from the store through every report row: sources that emit one record per session
  (exec plugins today) are marked `‡` and footnoted, and a grouped total that merges both
  units reads `mixed-granularity total` instead of quietly presenting session aggregates as
  per-turn data. The `coverage` validator replaces its "not yet surfaced" caveat with a real
  turn-level share, `report --format csv` gained a `granularity` column, and the metric-plugin
  envelope carries the field so a plugin summing usage rows can see it too (`B69`).
- **Per-input ingest state no longer grows with install age.** `ingest_file` kept a row for
  every transcript ever seen, including ones the vendor's own retention had long since
  deleted — measured here, the corpus fully rotates in under two months, so the table
  accumulated roughly 34,000 dead rows a year. Rows for inputs no longer on disk are now
  dropped after each pass, *unless* that source's discovery canary fired: "the files are gone"
  and "we stopped being able to find them" are indistinguishable from there, and discarding
  state during exactly the failure being diagnosed would destroy the evidence.

### Compatibility
- **New migration `0004_ingest_source.sql`** adds an `ingest_source` table, applied
  automatically on first open. Like `ingest_file` it holds no usage and can be dropped at any
  time at the cost of losing the drift baseline; unlike it, the table is bounded by
  construction — only the newest runs per tool are kept, pruned inside the same transaction
  that writes them, so its size follows how many tools are configured, never how long assaio
  has been installed. `usage_record` is untouched.
- **The canaries need a baseline, so the first `backfill` after upgrading fires nothing.**
  History-relative canaries (discovery, yield) stay silent until a second pass exists to
  compare against; the two absolute ones (skipped lines, zero-token records) work from the
  first run.
- **`report --format csv` gained a `granularity` column** between `member` and `in`. A
  consumer that reads columns by position rather than by header name needs updating.
- **The metric-plugin envelope gained `usage[].granularity`.** The addition is backward
  compatible and `assaio_metric_input` stays at `1`: a plugin that ignores the field behaves
  exactly as before.
- **A metric plugin should now declare `confidence.samples` and `confidence.samplesUnit`** in
  its result. Coverage and provenance are stamped for it from the same window the built-in
  metrics use, but only the plugin knows how many observations its verdict counted. One that
  omits the field reads as `insufficient` — the honest label for "did not say what it rests
  on" — rather than borrowing a number assaio would have to invent. The result protocol stays
  at `assaio_metric: 1`.
- **`doctor` replaced its `activity:` line with a `depth:` section** and dropped the two
  caveats the matrix now owns; they are printed per source, and only for sources present on
  this machine.

### Fixed
- **The README's manual install instructions pointed at v0.1.0**, three releases behind, so
  anyone following them got a binary without `statusline`, `explain`, incremental backfill or
  the last two rounds of correctness fixes. Both the shell and PowerShell snippets now resolve
  the latest tag before downloading.

## [0.4.0] - 2026-07-29

### Added
- **`backfill` is incremental.** An input this build already parsed unchanged is skipped
  and reported as `unchanged=`, so keeping the store current stops being a chore. Measured
  on a real 4.7 GB / 6262-file Claude Code history: the pass that reads everything took
  **68 s**, the next one **0.13 s** — and that fast pass still picked up the 291 records a
  session had written in between. Nothing is skipped silently, and nothing is skipped that
  could hide a problem:
  a file that failed to parse is never recorded, so it keeps being retried and keeps
  appearing in `failed=`. The stored state is keyed on the parsing build's identity, so a
  new version re-reads everything once — that pass is what lets history gain signals an
  older parser could not extract, and it is now automatic instead of implicit. Use
  `backfill --full` to force it on the same build, which is what you want while working on
  a parser. Closes B43's mechanism.
- **`statusline`** — one ambient line for an editor or shell status bar: today's tokens,
  AI lines, cost basis, and how fresh the data is. The day is the machine's **local** day,
  computed as a timestamp range rather than the store's UTC day bucket, so work after
  local midnight counts where it happened. Figures route through the same aggregation and
  pricing path as `report`, so the two cannot disagree. On a flat plan the money segment
  shows month-to-date beside the plan price as two raw numbers (`$412/$200 mo`) rather
  than a percentage, which would read as "consumed" when a plan only pays off *above* its
  price. It only ever reads: it never creates or writes the store, and any error prints
  nothing and exits 0 rather than breaking your prompt. Closes B05.
- **`explain <validator>`** — the long-form page for each of the 18 metrics: what it
  measures, how to read it, what to do about it, and the limits that keep it honest.
  Documentation, not measurement: it never opens the store, so it works before any data
  exists. `explain` with no argument lists every metric. Closes B07.
- **`doctor` reports ingest freshness per source**, so "why are my numbers stale" is
  answerable without guessing.

### Fixed
- **A session ingested while it was still being written froze a half-attributed turn.** A
  turn's failed tool calls, denials and edit counts are attributed by a *later* line in the
  log, so ingesting a live transcript stored the turn with its calls but without its
  outcomes — and nothing ever corrected it, because a re-read only ever repaired rows that
  carried no signals at all. `friction` therefore reported an error rate it could prove was
  wrong. Re-reading a file the store owns now restates that turn's activity columns, taking
  `MAX(stored, offered)` so a repair can never lower a stored figure. Records pushed to a
  team server keep the opposite contract — first-write-wins, so one member's push cannot
  restate another's row. A count that first read too *high* is still not corrected; `doctor`
  says so. Closes B68, and with it the reason a per-turn hook was not recommended.
- **`doctor` under-reported Claude Code by thousands of files.** It counted only top-level
  transcripts while `backfill` reads those *and* every sub-agent transcript beneath them —
  on a real machine, 1993 files reported against 6398 actually read. The diagnostic that
  answers "what will be read" now reports both counts separately.

### Compatibility
- **New migration `0003_ingest_state.sql`** adds an `ingest_file` table. It is applied
  automatically on first open and holds no usage: it records only which inputs were parsed,
  at what size, mtime, and by which build. Nothing else reads it, so it can be dropped at
  any time at the cost of one slow re-parse. `usage_record` is untouched and the sync wire
  protocol is unchanged, so an older team server accepts pushes from this build unchanged.
- **The first `backfill` after upgrading is a full one**, by design: state written by a
  different build is never trusted, and that pass is what lets history gain the activity
  signals an older parser could not extract. Runs after it are incremental.
- Stores carried over from an earlier release have no ingest state until that first
  backfill, so `statusline` reports the data's age as unknown rather than guessing, and
  `doctor` says no ingest has been recorded yet.

### Changed
- Static user-visible text moved into a new `internal/i18n` catalog — dashboard chrome,
  the statusline's words, and the explain pages — so adding a language becomes one new
  catalog rather than an edit spread across the codebase. No wording changed; the
  dashboard's rendered output is byte-identical. Data-derived text (a validator's read,
  takeaway, and figures) deliberately stays with the validator, since translating it
  means templating interpolated numbers. This is the scaffolding B08 needs.

### Internal
- The four duplicated K/M/B number formatters are collapsing into one `internal/humanize`
  package; `analyze`'s two call sites moved with byte-identical output (part of B75).
- `internal/ingest` split into `ingest.go`, `discover.go`, and `state.go`, which brings
  the package back under the file-size budget.
- The dashboard golden fixtures are built in `time.Local` rather than UTC. `rhythm` reads
  session starts in the machine's own zone, so a UTC fixture placed the same session in a
  different time-of-day band depending on where the test ran -- the rendered goldens passed
  on a CET laptop and failed in CI. Verified identical across five zones.

## [0.3.0] - 2026-07-26

### Added
- **Three `analyze` validators, all on data the store already holds** (no schema change,
  no re-backfill). `concentration` (Spend Concentration) — how token spend spreads across
  projects and, more usefully, where a project's share of the tokens outruns its share of
  the AI-written lines; concentration alone is reported as neither good nor bad, since one
  project can legitimately own the work. `rhythm` (Work Rhythm) — the off-hours and weekend
  share of sessions, the time-of-day shape of the work, and the p95 focused-session length;
  an aggregate workload signal, never an individual measure. `burn-anomaly` (Burn Anomaly) —
  the days that burned far outside the window's typical day, by a robust median/MAD outlier
  test, to catch the spike nobody noticed (a runaway loop, a re-ingested backfill, an agent
  left running) rather than to fault a heavy day. Closes B26, B27, and the after-hours and
  p95-session parts of B28/B36.
- **Tool-call purpose capture, and three validators over it.** The Claude Code and Codex
  parsers now classify each tool call into read / search / command / write / other against a
  shared allowlist (`internal/parser/toolclass.go`) and count the ones that came back an
  error. The tool's *name* is classified during parsing and then dropped — neither it nor any
  tool input is ever stored. On top of it: `explore-produce` (Explore vs Produce) — how much
  of the work was looking around versus changing code, with reads-per-write; `friction`
  (Friction) — how often calls fail outright, reported apart from how often a human declines
  one, because the two have different fixes; and `skill-economics` (Skill & Agent Economics)
  — which skills and sub-agents the tokens went to and how much code each produced, the place
  shared tooling quietly concentrates spend. Closes B15 and B38.
- **Sub-agent turns are now marked at the source.** Records carry `sidechain`, read from the
  log's own marker instead of inferred from the dedupe key, so the delegation share is exact.
  The served team dashboard now computes delegation and attribution too; it previously
  rendered delegation as a blank because it never queried them.
- **Exec rule plugins — your own CI gate, in any language (ADR 0005).** The third
  out-of-tree protocol, completing parsers → metrics → rules. A rule plugin declared under
  `rules:` is invoked as `<command> evaluate`, reads this window's validator verdicts on
  stdin (exactly what `analyze --format json` prints — no usage rows, no sessions, no
  prices) and emits alerts with a severity of `info`, `warn`, or `error`. `assaio-agent
  check` runs them: alerts print under the budget report, an `error` alert exits non-zero,
  and a rule that could not be evaluated fails the gate too rather than passing silently.
  Thresholds are organizational, so they belong in your script, not in our tree. Everything
  is validated at the boundary — whitelisted severity, required `rule` and `message`, 50
  alerts max, length caps, no control characters, unknown JSON fields rejected — and a
  violating document is dropped whole. Closes B13.

### Changed
- **Record schema (migration `0002_activity_signals.sql`).** Nine columns added:
  `tool_reads`, `tool_searches`, `tool_commands`, `tool_writes`, `tool_other`, `tool_errors`,
  `sidechain`, `skill`, `agent`. All default to `0`/`''`, so rows written by an older build
  stay valid and simply read as "not captured".
- **The first backfill after upgrading restates old rows.** History parsed by a build that
  could not extract these signals would otherwise keep zeros forever behind
  `ON CONFLICT DO NOTHING`. `Insert` now fills them in on a stored row that carries none of
  them, and never overwrites a signal already recorded — so a steady-state re-run still
  writes nothing.

### Breaking
- **`Result.BarsAreProjects` is now `Result.BarsPseudonym`** (JSON `barsAreProjects` →
  `barsPseudonym`), a string naming what kind of user-authored label the bars carry:
  `"project"`, `"skill"`, or `""` for a fixed vocabulary that must never be pseudonymized.
  A boolean could not express that skill and sub-agent names need pseudonymizing too, which
  is how they reached an anonymized dashboard verbatim. Exec **metric** plugins may keep
  emitting the old key — it is accepted and maps to `"project"` — but should migrate, since
  the new field is the only way to say `"skill"`.

### Compatibility
- Exec **parser** plugins cannot emit the new activity fields yet; records they push read as
  "not captured" (`0`/`''`), not as a real zero. The wire protocol is unchanged.
- Metrics over the new fields state their own coverage. Gemini CLI and Cline logs do not name
  their tool calls, so their usage is excluded from the explore/produce split rather than
  counted as zero, and only Claude Code labels turns with a skill or sub-agent today.

### Fixed
- **A metric plugin written against the pre-rename `barsAreProjects` key lost its
  pseudonymization**, publishing real repository names into an anonymized dashboard. The
  legacy key is accepted again and maps to `"project"`; every *other* unknown field is now
  rejected outright, so a misspelled key can never quietly disarm a setting again.
- **The project drill re-ran window-scoped metrics against one project's rows.** Subscription
  Fit divided the whole plan price by a single project's spend and printed "API could be
  cheaper" beside the window-level "your plan pays off" on the same page; Skill & Agent
  Economics showed window-wide totals under a project heading. A validator can now declare
  itself `WindowScoped` and the drill skips it — see `docs/extending.md`.
- **Codex's tool-purpose split reported 0% produced whatever the agent did**, because Codex
  applies file edits through a `patch_apply_end` event that no tool call names. Those edits
  now count as write calls (and a failed one as an errored call), with a guard so a version
  that *does* name its patch calls cannot double-count them.
- **`friction` counted calls that cannot report a failure.** Codex marks the outcome of file
  edits only — a shell command that exits non-zero looks identical to one that succeeded — so
  its calls padded the denominator and diluted the error rate. Rates are now taken over tools
  that mark every call, and the caveat says which.
- **Sub-agent tokens were double-counted** on a store holding both a pre-upgrade `agent:`
  aggregate row and the per-turn rows later parsed from the same transcript. A window now
  picks one definition: the `sidechain` marker when any row carries it, the older dedupe-key
  shape otherwise.
- **`concentration` returned a green ALIGNED when no project was large enough to compute a
  gap** — a passed check for an examination that never ran — and computed its headline shares
  over the whole window while the Gini score used only named projects, so the two figures on
  one line disagreed. Both now read as shares of attributable spend.
- **`rhythm` asserted an off-hours finding its own verdict had refused**, printing "a large
  share of sessions runs off-hours" beside a neutral read on a two-session window.
- **`burn-anomaly` named calendar dates as spikes below its own day floor**, computed from a
  baseline the same entry declared unusable.
- **`skill-economics` rendered a share and a total from different dimensions** side by side,
  pointing the reader at a skill worth 1% of the window. Both now come from one dimension.
- **`check` double-counted reasoning tokens** against the budget, so a CI gate could fail a
  window that `report` and `analyze` showed as under budget. The same double-count is fixed
  in the delegation query.
- **The sync boundary accepted a tool-purpose split that contradicts its own tool-call
  count**, and a `sidechain` outside 0/1. Both are rejected now; an all-zero split with calls
  present stays valid, since that is the documented "not captured" state.
- **`StructuredOutput` counted as code production**, inflating the produce share of any turn
  that answered structurally. It writes no file and is classified as "other".
- **`assaio-agent demo` left three of eighteen panels blank** — the tool's own showcase — for
  want of the new signals on its bundled records. Its sample sessions are also pinned to a
  working weekday morning now: they used to inherit the invocation clock, so the identical
  fixture read as ordinary hours before lunch and as off-hours work after dinner, and its
  rhythm verdict changed with the reader's timezone.
- **Skill and sub-agent names reached anonymized reports verbatim.** The team server's
  unauthenticated dashboard pooled them across every member beside pseudonymized project
  and member names; skill names are user-authored and can name a client. They are now
  pseudonymized like project names.
- **Rule plugins were handed the ranked bar lists**, which carry project, skill, and
  sub-agent names — more than PRIVACY.md said they receive. Bars are stripped from the
  envelope; a rule gates on a verdict, not on which repository produced it.
- **The nine columns migration 0002 adds bypassed the sync boundary's numeric check**, so a
  pushed record could store a negative count (rendering impossible percentages) or an
  overflow-magnitude one (breaking `SUM()` for the whole team). A reflection-based test now
  fails the build if any numeric record field is left unbounded.
- **`check` failed open in two ways.** A metric plugin that failed was dropped with a
  warning, so a rule gating on its verdict passed on a verdict set it never saw; and a rule
  emitting `null` or `{}` — what a jq filter prints when it stops matching — was read as
  "no alerts" instead of a rule that evaluated nothing. Both now fail the gate, matching
  what the command already promised.
- **`friction` reported a fabricated 0.0% and a green verdict** on windows where no call
  could record a failure at all, and impossible percentages (`150%` errors, `-50%` clean)
  when errors outnumbered counted calls. Rates are now taken over calls whose failure state
  was actually recorded, coverage is stated, and incoherent counts render `—`.
- **`rhythm` raised a WATCH alarm on windows too small to judge** — an amber badge beside
  its own "too few sessions to call the rhythm" takeaway. It now returns the neutral read
  every sibling validator uses.
- **`skill-economics` flagged anyone using a single skill**, whose share is 100% by
  construction. A concentration verdict now needs at least two labels to compare.
- **`concentration` treated the unattributed bucket as a project**, so usage from tools that
  log no working directory became the widest spend gap and inflated the project count. It is
  excluded from every statistic and disclosed as a caveat instead.
- **`backfill` and `sync` counted restated rows as new records**, inflating `inserted=` to
  the whole store on exactly the upgrade run. Restatement is now a separate repair step and
  the count means new rows again.
- **The dashboard's project drill dropped four inputs** its caller had populated, so
  Subscription Fit demanded a plan cost that was already configured and Skill & Agent
  Economics and Model Right-Sizing rendered blank on every dashboard.
- `burn-anomaly` now discloses that days are bucketed in UTC, which splits an evening that
  runs past midnight across two of them.
- `explore-produce`'s coverage caveat blamed tools that name no tool calls; those record no
  calls at all, so they cannot lower coverage. It now names the real cause — history
  ingested before the capture existed — and points at `backfill`.
- `PRIVACY.md`'s exhaustive "what it extracts" list did not mention migration 0002's nine
  columns, two of which are text labels rather than counts.
- `FEATURES.md` listed every validator that shipped in 0.2.0 as "Unreleased"; they now
  carry their real release.

### Internal
- Percentile interpolation is now one shared helper (`percentileAt`) instead of a median
  open-coded in `context.go`, so every median and p95 figure uses the same method.
- The metric and rule protocols share one subprocess runner (`docProtocol`), so timeout,
  stdout cap, stderr prefixing, and handshake handling exist once. As a side effect a
  metric plugin that floods stdout is now killed on the breach instead of at the timeout.
- The CLI tests that assert `analyze --list` output no longer hard-code the validator
  count; they derive it from the registry, so registering a metric cannot break them.

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

[Unreleased]: https://github.com/assaio/assaio/compare/v0.10.0...HEAD
[0.10.0]: https://github.com/assaio/assaio/compare/v0.9.0...v0.10.0
[0.9.0]: https://github.com/assaio/assaio/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/assaio/assaio/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/assaio/assaio/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/assaio/assaio/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/assaio/assaio/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/assaio/assaio/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/assaio/assaio/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/assaio/assaio/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/assaio/assaio/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/assaio/assaio/releases/tag/v0.1.0
