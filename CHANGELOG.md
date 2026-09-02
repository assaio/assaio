# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

**How to read this file.** `[Unreleased]` means merged to `main` but not yet part of
any tagged release — installing the latest release does *not* give you those entries.
At release time the whole `[Unreleased]` section becomes the new version's section
(enforced by `make release`, see [RELEASING.md](RELEASING.md)), so a version heading
always describes exactly what its tag contains. The headings link to the tag or diff.

Every entry sits under one of seven headings — **Breaking**, **Added**, **Changed**,
**Fixed**, **Removed**, **Deprecated**, **Security** — and is one to three lines. A
breaking change carries its migration path.

**Where the post-mortems live.** assaio corrects figures it has already published, and
[docs/corrections.md](docs/corrections.md) is the register of every one of them: what was
wrong, since when, what the wrong number showed a reader, what the fix changed, and what a
later review overruled. An entry whose full story lives there is marked `(correction)` and
links to it. **No correction was dropped when this file was split**: the register holds the long
form of each and this file keeps its line. That promise is about corrections only — feature
entries were condensed in the same pass, and the register is not their archive.

This file records only what has actually shipped. What's *coming* is tracked in
[BACKLOG.md](BACKLOG.md) — ranked **proposals and effort estimates, not commitments**:
the actual order follows real-world feedback, pull requests, and bug reports, so items
there can be reshaped, reordered, or dropped, and things not listed anywhere can land
first when a PR or a bug report makes the case. To suggest something new — or add
weight to a tracked item (reference its `B` id) — open a feature-request issue or a
Discussion.

## [Unreleased]

## [0.25.0] - 2026-09-02

### Breaking

- **The metric-plugin envelope is a projection, and the plugin declares what it reads**
  (`assaio_metric_input: 4`, [ADR 0004](docs/adr/0004-exec-metric-plugin-protocol.md)). A metric
  answers `<command> describe` with `{"needs":[...]}`, optionally narrowed by `fields` (columns)
  and `where` (rows), and receives exactly that; the `needs:` config key becomes the reader's veto
  over that declaration rather than its source. On a real 30-day window the same envelope falls
  from 1.18 MB to 43 KB for a token-share metric and from 53.28 MB to 7.18 MB for one reading the
  step timeline. **Migration:** bump the handshake to `{"assaio_metric": 4, ...}` and add a
  `describe` branch — `{"needs":["usage","sessions","trace","attribution","turn-sizing",
  "cache-misses","prices"]}` reproduces protocol 3's payload exactly. `assaio-agent plugins init
  --kind parser|metric|rule --lang go|python|sh` prints a conforming skeleton, and
  `docs/conformance/` publishes the vectors that judge one.

### Added

- **Antigravity CLI (`agy`) is the sixth source** (`B88`) — conversation transcripts under
  `~/.gemini/antigravity-cli/brain/<id>/.system_generated/logs/transcript.jsonl`, contributing
  sessions, turns, tool calls and edits. Verified against Antigravity CLI 1.1.23.
- **`activity-only` joins the depth tiers** ([ADR
  0017](docs/adr/0017-a-source-without-a-token-counter.md)): a source that records sessions and
  what was done in them and publishes no token counter anywhere in its format. `agy` is the
  first; every token and cost figure withholds its verdict for it rather than counting a zero.
- A register of published figures a verdict could rest on (`internal/threshold`), each
  carrying its definition, source, population, measurement date and a mandatory expiry.
  No metric qualifies for a line today: `rework` and `survival` were both weighed against
  GitClear's churn study and refused, and `analyze rework` now names that study and the
  properties that differ rather than leaving a silence a reader fills with it.
- **`reprice`: which plan and which model mix this window's own numbers support.** Re-prices
  the turns already in the store against another entry in the same price table — arithmetic over
  observed events, never a counterfactual — states what it holds fixed and what share it could
  not price, and ends in one experiment with its rollback and follow-up
  ([ADR 0015](docs/adr/0015-structured-recommendations.md)).
- **A reproducible release build, and an SBOM per archive.** `.goreleaser.yaml` builds from the
  module as the Go proxy serves it at the tag, strips the builder's absolute paths (`-trimpath`)
  and pins archive mtimes to the commit's own timestamp — the three inputs that otherwise made
  two builds of one tag differ. Each archive also ships an SPDX document beside its checksum and
  provenance attestation. That is the reproducibility half of ROADMAP's v1.0 condition 7; the
  tested-upgrade half stays open.
- **Four documents the repository was missing**: [`docs/corrections.md`](docs/corrections.md),
  the register every "a parser fix reaches stored history" claim rests on;
  [`docs/architecture.md`](docs/architecture.md), the path through the internal packages;
  [`docs/threat-model.md`](docs/threat-model.md), beside `SECURITY.md` and `PRIVACY.md`; and
  [`SUPPORT.md`](SUPPORT.md).

### Changed

- **`--help` groups the commands by what you would be doing** — Start here, Read, Act,
  Maintain, Team, Extend — instead of one alphabetical list. A command that serves another
  command's workflow rather than a job of its own stays under "Additional Commands".
- **Every token- and dollar-denominated figure now reads only the sources that count tokens.**
  `status` peak context and output-per-session, `burn-anomaly`'s daily burn, `turn-efficiency`'s
  output-per-turn, and `report`'s four token columns each state that a source records none
  instead of averaging its silence in. `report --format json|csv` gained a `tokened` field.
- **`adoption` withholds project breadth where no row in the window names a project.** Breadth
  comes from a session's working directory; a source that writes none left every row
  unattributed and the verdict read "usage is narrow and flat" off that silence. The
  week-over-week trend half still stands, and `doctor`'s inventory no longer counts an unnamed
  model as a model.
- **The zero-token drift canary skips a source whose depth row declares no token counter.**
  Such a source is 100% zero-token on a healthy run, so the canary would fire on every backfill
  and fail `doctor --strict` on correct data. A source outside the matrix is still judged.
- **The comparison against the built-in stats leads with the structural claim, and names
  Codex.** Codex CLI 0.140.0 ships its own `/usage`, so a table measured only against Claude
  Code read as out of date; it gains a Codex column and concedes what an account-side view does
  better. The four limits no release closes — and the refusal to rank named individuals, which
  was only ever stated in `PRIVACY.md` — now sit above the feature table in `README.md`,
  `site/index.html` and `site/llms.txt`.
- **`reprice` carries the cost-estimate disclosure as a stated assumption**, so `--format json` carries
  it too, and where a flat subscription is configured it says beside the deltas that nothing in the
  `vs observed` column is spend that would stop.
- **The metric-plugin envelope withholds `totals` and `delegation` from a narrowed window.** A `where`
  predicate on `usage`, `byModel` or `byProject` no longer ships the prepared views: they span the whole
  window, so `sum(usage.in)/totals.tokens` was a share of a population the delivered rows did not
  describe. The window's own denominator stays available as `projection.rows[<section>].available`.
- **The vendored LiteLLM price table is refreshed to its 2026-09-02 snapshot: 3,111 -> 3,518
  entries.** `claude-fable-5-1` gains a price, so a window holding it stops excluding those
  tokens; `gpt-5.6` and `gpt-5.6-sol` fall ~20% on every rate, and 155 further entries re-price.
  Eleven were withdrawn upstream (`xai/grok-2*`, `xai/grok-beta`, `xai/grok-vision-beta`, two
  RunwayML video models, one GigaChat), so a window naming one of those now reads as unpriced.

### Fixed

- **`needs: [trace]` — the key v0.24 required — made `config.yaml` fail to load.** Every
  `metrics:` entry was held to the parser-and-rule rule, whose last check refuses `needs:`, so
  the documented migration path was the one edit that could not be made, and every command
  reading `config.yaml` exited on it.
  ([correction](docs/corrections.md#needs-key-made-the-config-unloadable))
- **`CITATION.cff` named four parsers and version 0.1.1 for twenty-three releases**, and nothing
  in review opened it. Both halves are now mechanical: `consistency.yml` fails when the cited
  version is neither the newest tag nor the one `CHANGELOG.md` is preparing, when
  `date-released` is not that tag's date, or when the abstract's tool list is not the parser set
  the binary reports in `docs/reference.json`.
- **`report` and `effectiveness` totalled `$0.00` for a window in which nothing priced**, and
  `effectiveness` printed "Every source in this table records changed lines" over a table whose
  only source records none. Both totals and the caveat now withhold.
- **A GitHub Copilot CLI session reporting a model with no name emitted a record nothing could
  identify.** The dedupe key is `<session>:<model>`, so every unnamed model in one session
  collapsed onto a single stored row and could be neither priced nor grouped; it is now counted
  as skipped, and the session's code changes go to a named model. Found by a fuzz seed.
- **`report --by day` now prints one table row per day+tool+model.** The store also keys a group
  by project, entrypoint, member and granularity, so a single day arrived as several
  identical-looking rows and the first of them read as the day's total. `--format json|csv`
  still carries every group.
- **The team panel ranks and names nobody.** `dashboard --no-anonymize` rendered real member
  names ordered by session count with proportional bars, under a caption reading "this panel is
  not a scoreboard". Members are now pseudonymous on every render and ordered alphabetically by
  label; `--no-anonymize` reveals project names only, and raw names stay behind `report --identify`
  ([correction](docs/corrections.md#team-panel-ranked-named-individuals)).
- **`reprice --against <model>` names a target the price table cannot cost** instead of dropping it.
  Naming two models and pricing one printed a table that read as the complete answer.
- **An expired citation now expires.** The register's renderer tested fit before validity, so every
  registered candidate took the unfit branch forever and the expiry branch was unreachable — a figure
  past its `ValidUntil` was still quoted with nothing saying the source may have restated it.
- **`survival` renders the register's refusal of GitClear's churn figure.** A bare survival percentage
  beside a published churn percentage invites the subtraction the register exists to prevent.
- **The unpriced disclosure names both reasons when both hold.** A window with tokens on an unlisted
  model *and* rows from a source that publishes no token counter got only "a refreshed table ships with
  each release" — a fix for part of the gap presented as the fix for all of it.
- **`agy`: one model response written as two transcript entries is now one turn, not two.** A `GENERIC`
  and a `PLANNER_RESPONSE` sharing a `created_at` are two halves of one response; 653 entries across the
  500-conversation capture become 645 turns, in three conversations, with tool calls and edits unchanged.
- **`effectiveness` withheld a figure `agy` measures and fabricated one it does not.** The EDITS column
  blanked on line capability rather than edit capability, the TOTAL summed edits only over line-recording
  groups (so an edit-recording source's edits vanished from it), and the REJ column and total printed `0`
  for a source that records no refusal.
- **`effectiveness --format csv` carries `line_capable`, `edit_capable`, `refusable` and `tokened`**, so a
  withheld zero is no longer byte-identical to a measured one on the format a reader loads into a sheet.
- **`burn-anomaly` says "days with token data", not "active days"** — `adoption` prints a different
  number under that label — and its coverage is a day share rather than a row share.
- **`turn-efficiency`'s output/turn states its own narrower basis** instead of borrowing the coverage of
  the two figures beside it, and `status`'s sessions basis line names which figure is the narrowest,
  omitting itself when every candidate already printed "not recorded".
- **`agy`: a transcript that cannot be opened reports the reason without the absolute path.**
- **Four surfaces kept warning that the 1-hour cache-write tier was not priced**, three releases
  after v0.12.0 started pricing it. `doctor`, `explain cost`, README and the signal catalog all
  said so — and the catalog's line renders one row above the same page's "billed at its own higher
  rate". Only the long-context half of those caveats was still true
  ([correction](docs/corrections.md#cache-tier-caveat-outlived-its-defect)).


### Removed

- **The canonical event contract no longer models AI usage** (`B104`, [ADR
  0016](docs/adr/0016-usage-is-a-store-row-not-an-event.md)). `event.FromRecord` and the
  `ai.usage.observed` / `ai.edit.observed` payloads had no caller in eighteen releases, so the
  binary carried two canonical models of the same token counts; `usage_record` stays the one,
  and the contract now covers only the domains that have no store row. Nothing user-facing
  changes — the deleted code never ran outside its own tests.
- **`model-fit` no longer prints an `est. savings (upper bound)` figure.** A saving is a claim
  about an alternative history, which this binary refuses everywhere else; `reprice` states the same
  arithmetic as what the window *cost* against another table, with the span, the carried-over cache
  mix and the unpriced margin the deleted figure had none of.

## [0.24.0] - 2026-08-23

### Breaking

- **The team server's dashboard route now requires the bearer token.** `GET /` was open while
  the write route was guarded, which protected the wrong direction: the page carries a whole
  team's usage. ([correction](docs/corrections.md#serve-open-dashboard-route))
- **`serve` refuses a shared token shorter than 16 characters.** A secret short enough to guess
  is not one. A deployment using a short token must lengthen it — or move to `server.members`,
  where a shared token is no longer required at all.
- **A metric plugin gets the step timeline only if it declares `needs: [trace]`, and the
  handshake moves to 3** (`B168`). Add `needs: [trace]` to the `metrics:` entry and emit
  `"assaio_metric": 3`. ([correction](docs/corrections.md#metric-handshake-version-3))
- **`report --format json|csv` now emits a member pseudonym, not the synced name** (`B182`).
  Scripts that joined on the raw name need `--identify`; a local store has no member to label.
  ([correction](docs/corrections.md#report-export-carried-the-synced-member-name))

### Added

- **`doctor --db`, and a storage-growth projection on every store** (`B174`). Measured in live
  bytes -- on the maintainer's store 162.6 MB over 56 days = 2.9 MB/day, and *at most* 1.0
  GB/year. A store younger than a week says why it will not project.
- **`runtime inspect` (experimental)** -- a read-only snapshot of a self-hosted vLLM server's
  and an NVIDIA DCGM exporter's own metrics endpoints, where what is unavailable is never
  rendered as zero. **A feasibility slice behind a demand gate, and it may be removed.**
- **`recommend`: advice as a typed record, not a sentence** ([ADR
  0015](docs/adr/0015-structured-recommendations.md)). Each experiment carries its evidence, its
  risk and how to undo it. **Abstention is the default.**
- **A metric plugin declares what it reads, and the 44 MB step timeline is sent only when it
  does.** Without `needs: [trace]` the section is absent and the envelope's new `withheld` field
  says so, rather than an empty array a plugin cannot tell from a window with no sequences.
- **Validators can declare the evidence they read** (`B102`). `analyze.Capability` is the closed
  vocabulary, and one implementing `Needs` has its missing inputs disclosed as a caveat. A
  validator declaring nothing behaves exactly as before.
- **`docs/compatibility.md`: one answer to what v1.0 freezes.** Four published files each
  carried a different version of the promise, and every check passed because each was internally
  consistent. ([correction](docs/corrections.md#v1-freeze-policy-contradiction))
- **`survival` states the age of what it measured** (`B179`). The rate is monotonic in commit
  age -- 99% over `--since 7d` at a median age of 3 days here, 92% over `365d` at 13 -- so every
  run names the median age and the span.
- **`backfill` reports how many stored rows a re-read moved *down*.**
  ([correction](docs/corrections.md#backfill-downward-restatement))

### Changed

- **The vendored LiteLLM price table is refreshed to its 2026-08-23 snapshot: 3,040 -> 3,111
  models, none removed.** Fifteen entries changed price, so a window containing them re-prices
  on upgrade.

### Fixed

- **README and the site say what the built-in stats cannot do, instead of a claim that had
  stopped being true** (`B188`, `B189`): the opening said vendor dashboards count "spend, never
  output". ([correction](docs/corrections.md#vendor-stats-comparison-claim))
- **`digest` would have reported fourteen verdict changes as findings after upgrading.**
  ([correction](docs/corrections.md#digest-verdict-changes-after-upgrade))
- **`model-fit`'s two figures and its sentence used different denominators**, so a window with
  any unpriced usage read "60% premium" above a sentence saying 80%.
  ([correction](docs/corrections.md#model-fit-two-denominators))
- **A lines-per-million-tokens rate off a trivial token base.** The dashboard printed 5,298,857
  lines/1M tokens beside a real 6,386 -- an 830x contrast off a 3,500-token base.
  ([correction](docs/corrections.md#lines-per-million-trivial-base))
- **`concentration` rendered one quantity in two units.** The figure printed the spend gap in
  share points (`89pp`) and the takeaway printed the same number as a percent (`89%`), which
  invites the relative reading the figure avoids. One helper renders it everywhere.
- **`session_step.ts` stayed pinned while `usage_record.ts` became correctable.**
  ([correction](docs/corrections.md#session-step-ts-pinned))
- **`recovery` compared the aftermath of a failure against a baseline containing it** (`B175`),
  compressing the ratio toward `CONTAINED`. Measured here: 1.02x -> 1.03x.
  ([correction](docs/corrections.md#recovery-baseline-contained-the-aftermath))
- **A collapse in AI-line output read as "too few lines to call".** Output falling from 201,347
  lines to zero was reported as unreadable rather than as the direction it is.
  ([correction](docs/corrections.md#throughput-volume-floor-one-side))
- **A parser fix could not reach `ts`, `project`, `entrypoint` or `git_branch`.**
  ([correction](docs/corrections.md#identity-columns-uncorrectable))
- **A step's `kind` could not be corrected at all.** `restateStepSQL` did not touch it, so no
  path including `backfill --full` could change a stored classification: under the default
  30-day horizon a wrong one aged out, under `trace.horizon_days: 0` it was permanent (`B183`).

### Removed

- **Fourteen metrics stopped publishing a verdict they could not source** (`B176`, `B177`) -- a
  15% error rate, a 20% compaction rate, each picked once. 5 of 21 reads still carry one.
  ([correction](docs/corrections.md#unsourced-verdicts-withdrawn))
- **`rhythm` no longer judges when the work happened, and prints the band it counts against**
  (`B178`). "off-hours: 31% [WATCH]" was a workload judgement about an individual.
  ([correction](docs/corrections.md#rhythm-off-hours-verdict))
- **`throughput` no longer reads more AI lines as good news** (`B180`). A rising line count is
  an output measure, and colouring it favourably promoted output to value. The count and its
  direction are reported without a verdict.

### Security

- **The team server authenticates reads, and can derive who a member is from their token.**
  `server.members` puts it in server-derived identity mode, where the member is whoever holds
  the secret; requests are rate limited per secret, 120/minute by default.
- **The team server read the whole request body before checking the token.**
  ([correction](docs/corrections.md#server-read-body-before-token))
- **The rate limiter could be bypassed by varying the header, and the bypass grew the map it
  walked.** ([correction](docs/corrections.md#rate-limiter-header-bypass))
- **Every GitHub Actions reference is pinned to a commit SHA, as `SECURITY.md` always claimed.**
  Five jobs across two workflows used `actions/checkout@v4` -- a mutable tag -- while the policy
  said otherwise. A test now holds the claim to the workflows instead of review.
- **The supported build toolchain is Go 1.26.6 or newer, and `make vuln` is clean.** The
  documented floor was 1.25, on which `govulncheck` reports six reachable standard-library
  advisories. ([correction](docs/corrections.md#toolchain-floor-reachable-advisories))
- **A metric plugin was handed raw member names.** The usage rows, session rows and step
  timelines sent to an out-of-tree subprocess carried the name each member synced under.
  ([correction](docs/corrections.md#metric-plugin-raw-member-names))

## [0.23.1] - 2026-08-17

### Fixed

- **`share`: the profile frame printed its own caveat over the card's footer.**
  ([correction](docs/corrections.md#share-caveat-overflowed-the-footer))
- **`share`: the suggested post is formatted for where it gets pasted.**
  ([correction](docs/corrections.md#share-post-formatting))

## [0.23.0] - 2026-08-17

### Added

- **`assaio-agent share` -- an assay fit to post in public** (`B149`): one window as a 1:1 or
  9:16 reel, a still poster, and the post text beside them. **Redaction is structural**: no
  field the renderer reaches holds a repository, member, path, branch, skill or sub-agent name.

### Changed

- The vendored LiteLLM price table is refreshed to its 2026-08-17 snapshot (20 models added since
  the previous one).

## [0.22.0] - 2026-08-15

### Breaking

- **A metric-plugin result must declare its `layer`, and the protocol is now `2`** (`B155`).
  Emit `{"assaio_metric": 2, …}` and add `"layer": "activity"` or `"layer": "output"`.
  ([correction](docs/corrections.md#metric-protocol-2-layer))

### Added

- **A new `barren` drift canary: files found, and no run on record has read a usage record out
  of them.** It is the only canary judged on a condition rather than a rate, and deliberately
  carries no sample floor. ([correction](docs/corrections.md#barren-canary))
- **Migration immutability is enforced, in name and content.**
  ([correction](docs/corrections.md#migration-immutability-unenforced))
- **The harness this repo is developed with is checked in** under
  [`.claude/`](.claude/README.md): a `PreToolUse` guard, a permission split, six reviewers and
  three commands. A convenience, never a substitute for the gate in `CONTRIBUTING.md`.

### Changed

- **Every figure states its measurement layer** ([ADR
  0013](docs/adr/0013-measurement-layers.md)). Of twenty-one built-in validators, **eighteen are
  activity and three are output -- not one claims an outcome**.

### Fixed

- **A "seven-day" window was divided by eight days**, understating `subscription-fit` by 12.5%
  at 7 days and 3.2% at the default 30, and reporting the gap as `reconcile`'s `Unexplained`.
  ([correction](docs/corrections.md#seven-day-window-divided-by-eight))
- **A withheld verdict drew an empty gauge.**
  ([correction](docs/corrections.md#withheld-verdict-drew-an-empty-gauge))
- **`rework` flagged a window it could not measure, and could render above 100%.** Two defects
  in one verdict. ([correction](docs/corrections.md#rework-flagged-an-unmeasurable-window))
- **`humanize.USD` chose its unit before rounding, could never reach "M", and rendered a real
  cost as `0.00`.** `USD(1,000,000)` printed `1000.0K`.
  ([correction](docs/corrections.md#humanize-usd-unit-before-rounding))
- **Gemini CLI's output token total could overflow to zero, leaving reasoning above it.**
  ([correction](docs/corrections.md#gemini-output-token-overflow))
- **A billing export over 64 MiB was silently truncated and reconciled as complete.**
  ([correction](docs/corrections.md#billing-export-silently-truncated))
- **The `skipped` drift canary divided lines by records and called the result a line share**:
  2,000 lines yielding 500 records and 60 failures reported 10.7% against a true 3%.
  ([correction](docs/corrections.md#skipped-canary-lines-per-record))
- **The dashboard's project panel claimed the store's history could not be read.**
  ([correction](docs/corrections.md#drill-panel-lost-history-start))
- **The dashboard's subpath table showed one row per member, with no member column.**
  ([correction](docs/corrections.md#subpath-table-one-row-per-member))
- **`metrics verify` ran a plugin with a plan price of zero.**
  ([correction](docs/corrections.md#metrics-verify-zero-plan-price))
- **`edit-loops` project bars reordered between identical runs.**
  ([correction](docs/corrections.md#edit-loops-bars-reordered))
- **`subscription-fit` described a projection method its divisor does not use.**
  ([correction](docs/corrections.md#subscription-fit-projection-span))
- **`clear --labels` with no other scope deleted every label with nothing said first.**
  ([correction](docs/corrections.md#clear-labels-deleted-silently))
- **`backfill` could free 70 MB and never mention `compact`.** Tightening the trace horizon
  moved 70.3 MB onto the freelist and left the 170.1 MB file exactly the same size.
  ([correction](docs/corrections.md#backfill-freed-bytes-silently))
- **A structural silence no longer reads as an AI that produced nothing.** ADR 0011 was applied
  throughout `internal/analyze` and to none of the surfaces that sum `LinesAdded` directly.
  ([correction](docs/corrections.md#structural-silence-read-as-zero-lines))
- **A step's stored token total and outcome can be corrected downward.**
  ([correction](docs/corrections.md#step-columns-kept-not-assigned))
- **`doctor` states the timeline's own horizon, and says plainly when there is none.**
  `trace.horizon_days: 0` costs a measured 3.40 MB/day with no prune and no upper bound.
  ([correction](docs/corrections.md#trace-horizon-cost-undisclosed))
- **The step table's size bound is age-matched.** Dividing a pruned table by an unbounded one
  answered 1.88x where the age-matched figure is **2.19x**.
  ([correction](docs/corrections.md#step-table-size-bound-not-age-matched))
- **The digest names its output layer and withholds a line count nothing measured.** It ships an
  AI-line trend into a mailbox -- the one surface designed to be read out of context -- and
  carried no caveat about what lines measure.
- **`PRIVACY.md` names all five parsed sources and the two settings files v0.21 began reading.**
  ([correction](docs/corrections.md#privacy-md-named-three-of-five-sources))
- **`README.md`'s "Every command" includes `survival`**, shipped in v0.2; its validator count
  reads twenty-one rather than nineteen; and the roadmap tool list matches `ROADMAP.md`.
  ([correction](docs/corrections.md#readme-commands-and-counts))
- **`site/index.html` no longer contradicts itself on the live page.**
  ([correction](docs/corrections.md#site-contradicted-itself))
- **Every published in-tree validator example compiles against the interface again.**
  ([correction](docs/corrections.md#validator-examples-stopped-compiling))
- **`docs/format-resilience.md`, `docs/automation.md` and `docs/extending/data-source.md`
  describe what ships.** ([correction](docs/corrections.md#guides-described-what-does-not-ship))
- **`docs/extending/query-your-data.md` describes the store that exists.**
  ([correction](docs/corrections.md#query-your-data-described-a-different-store))
- **`docs/README.md` maps every ADR.** It listed 0001–0005 of thirteen, and `AGENTS.md` and
  `llms.txt` both call it the place architecture decisions live. A test now fails when one is
  missing. `AGENTS.md`'s own package map gained the five packages it had stopped listing.
- **`ROADMAP.md`, `FEATURES.md` and `demo` stop disagreeing with the binary.** `FEATURES.md`
  gains the rows it had stopped adding since v0.17, and `demo` no longer calls `analyze` "the
  five-dimension litmus" five lines before saying "of 21 reads".

### Removed

- **`report --by member` and `effectiveness --by member` are refused, not caveated.** Grouped by
  member they printed each person's spend, AI lines and cost-per-100-lines -- the ranking
  `BACKLOG.md`'s Refusals rule out. ([correction](docs/corrections.md#report-by-member-refused))

## [0.21.0] - 2026-08-13

### Added

- **The step timeline has readers**: `edit-loops`, on the definition [CodeBurn
  publishes](https://github.com/getagentseal/codeburn) so the figure means the same thing in
  both tools, and `recovery`, what a failed call or a lost context costs next.
- **Detectors declare their scope in the interface, not in prose.** A validator reading
  sequences names one of `interactive`, `sub-agent`, `programmatic` or `unstated`, so a rate
  spanning two scopes cannot be printed as if it described one.
- **The sequence crosses the metric-plugin boundary**, scope precomputed per sequence, so a
  plugin can write the detectors the core just gained (`trace`, `historyStart` on the wire).
- **A history horizon on every trend, and retention in `doctor`** (`B156`). A validator
  implementing `analyze.Trending` is stamped with where the store's history starts. Here: 44
  days stored against 30 the source still keeps.
- **Three signals for the sequence** (`ai.steps.count`, `ai.step.outcome`, `ai.step.target`) and a
  `step` grain in the catalog, with the capability row that carries them — pinned by a new
  adjudicated trace, since a depth row nothing calibrates is a promise nothing checks.

### Changed

- **A step's target is read from the call's own arguments instead of its result.** Over 5,741
  real transcripts, reads went from 0 to 34,980 of 37,373 (93.6%) and edits from 94.4% to 100%.
  Run `backfill` to apply it.
- `store.HistoryStart` and the retention line take a tool, because each source keeps its logs for
  its own length of time and comparing one tool's retention against every tool's span answers
  about a set nobody measured.
- The "far from typical" rule — median, median-absolute-deviation, modified z-score — is shared by
  every metric that asks the question instead of restated per metric.
- `analyze` reads the stored sequences, which costs about 2.5s on a 339,000-step store. The read
  is skipped entirely when no registered validator wants it, and `trace.horizon_days` bounds how
  much there is to read.
- The two detectors report nothing until a `backfill` has stored sequences, and only Claude Code
  records them; every other source's depth row says so rather than reporting a zero.
- The served dashboard (`serve`) deliberately does not read sequences: `GET /` is unauthenticated
  and rebuilds per request, and a store filled by `sync` holds none anyway, since the team-server
  contract carries usage records and not sequences. Its detectors say so (`B171`).

### Fixed

- **A re-read no longer keeps the higher of two parsers' target numbers.**
  ([correction](docs/corrections.md#target-ref-restated-with-max))

## [0.20.0] - 2026-08-12

### Added

- **A session is stored as a sequence, not only as a total.** A Claude Code transcript now also
  yields its step timeline as content-free rows -- no prompt, no code, no file name; a target is
  an integer assigned in first-seen order within one sequence.
- **`trace.horizon_days` (default 30) bounds how much of that timeline the store keeps.** Not
  housekeeping: here the timeline and its indexes measure 101.9 MB against the usage table's
  58.3 MB.
- `doctor` reports the timeline's size and the window it covers, beside the store's own size.
- `backfill` prints `steps-pruned=` when a run drops stored steps for being past the horizon,
  so assaio deleting your history is never silent.

### Changed

- `clear` now erases the step timeline under the same scope as the records it erases.
- A step older than the horizon is never inserted, rather than inserted and pruned moments
  later — the round trip grew the SQLite file permanently and made `backfill` print a count for
  rows that no longer existed when it returned.
- Steps refused at the store's vocabulary boundary are counted as skipped instead of dropped in
  silence.
- **The store gains a table** (migration `0012`). Nothing existing is rewritten and no reader
  changes, so an older binary keeps working against an upgraded store — it simply ignores the
  new table and never prunes it.
- **Expect the store to grow by roughly 1.7x the usage table** on first backfill after
  upgrading, bounded by `trace.horizon_days`. `assaio-agent compact` reclaims the space if you
  then set the horizon lower or to a shorter window.
- **Widening `trace.horizon_days` later does not bring pruned steps back on its own.** Ingest
  skips transcripts it has already read, so recovering them takes `backfill --full`, and only
  while the tool still has the files on disk.

## [0.19.0] - 2026-08-12

### Added

- **The documentation is published, and the recipes in it are executed.**
  [assaio.dev/docs](https://assaio.dev/docs) renders the Markdown it is written in -- the
  Markdown stays the only copy, and `make test` fails when the two disagree.
- **A cookbook of recipes that are run, not merely printed.** Each YAML recipe is loaded and its
  derivations asserted; each plugin is executed against a fixture window. Where a recipe can
  only be shape-checked, the page says which.
- **Four more ways for a document to be wrong, now checked**: a command line naming a flag the
  binary lacks, a helper no package declares, a link that does not resolve, and a recipe
  claiming a stronger check than the suite performs.

### Changed

- **`site.yml`'s guards walk the whole served tree**, not the top directory: twenty pages are
  published where two were.

### Fixed

- **How each recipe is checked is classified in code, and the pages print the counts.**
  ([correction](docs/corrections.md#recipe-execution-claim-uncounted))
- **A published CI recipe would have passed at any spend, forever.** Its "fetch the team window"
  step set two environment variables and fetched nothing — and no command could have: `sync` is
  push-only. ([correction](docs/corrections.md#ci-recipe-passed-at-any-spend))
- **`make docs` could not regenerate eighteen of the twenty pages it is named as the fix for.**
  Its `-run 'TestCommittedReference'` matched neither guide-page test, while the failing test,
  `guidepage.go` and `docs/site.md` all told a contributor to run it.
- **A recipe told a reader `statusline` exits non-zero when the store is stale.** It never exits
  non-zero on any path; the paragraph under it said so, and the block above it contradicted that.
  `doctor --strict` is the command with the exit code.
- **The test behind the cookbook's flagship claim could not fail.**
  ([correction](docs/corrections.md#cookbook-test-could-not-fail))
- **The Go examples were described as shape-checked, and nothing checked them.** They are now
  parsed and their method set held to the `Validator` interface, which catches a renamed method
  and a changed signature.
- **The invocation check abandoned any line whose command it did not recognize**, so a renamed
  command took its flags out of scope with it and the whole line passed.
  ([correction](docs/corrections.md#invocation-check-abandoned-a-line))
- **Fourteen pages shipped a meta description cut mid-sentence**, because it was taken from the
  first hard-wrapped line rather than the first sentence, and one leaked raw link syntax.
- **`docs export --format html` lost the reference's own navigation** when the page moved into
  the documentation: with no guide set to show, the sidebar fell to a single entry and the
  standalone export became a wall of tables. It lists its own sections again.
- **`/reference` became a live 404 with no redirect**, and two of our own documents still linked
  it. A static host cannot redirect, so the URL keeps a generated page that says where the
  reference went.
- **Four pages recommended `shareOrDash` for a divide-by-zero, and no package had such a
  function.** The helper is `humanize.PercentOrDash`. This is the drift the new helper check was
  written for, found by writing it.
- **A test tripped `SA5011` in CI and not locally**, on the same pinned linter version and the
  same Go minor: staticcheck read the dereference after `t.Fatal` as a possible nil. An explicit
  `return` satisfies both. ([correction](docs/corrections.md#sa5011-in-ci-only))
- **A published document named releases as `v0.6.0`**, which the `noversion` guard reads as a
  version stamp — correctly, since it cannot tell one from the other. The sentence means the same
  with `v0.6`.

## [0.18.0] - 2026-08-11

### Added

- **The published surfaces are generated from the binary, or checked against it** (`B161`).
  `docs export` projects every live register into one document; `docs/reference.json` and
  `site/reference.html` are generated and committed.
- **The website declares which of its claims are checkable, and a test holds it to them.** It
  runs in both directions, and the second is the one that matters: a claim with nothing behind
  it fails, **and so does a shipped capability the page never names**.
- **The reference publishes an environment variable only where one works.** 17 of the 35
  configuration keys have none, so those rows carry a blank rather than a name that bricks the
  binary.
- **The metric-contract tables describe shape, not obligation.** A "Required" column derived
  from `,omitempty` would have marked fields the core overwrites or discards. They now state
  what reflection can answer.

### Changed

- **`docs/extending.md` is a map, not a manual.** 1,923 lines in one file became an index, with
  one page per surface under `docs/extending/`.
- **`site/llms.txt` states no counts.** It pointed at "nineteen metric validators" in a file no
  count check can reach; it now names the generated reference and tells the assistant reading it
  to answer capability questions from there.

### Fixed

- **`digest` and `mark --suggest` shipped in v0.17.0 and the website said neither.**
  ([correction](docs/corrections.md#site-never-named-v0170-commands))
- **Six contract fields were absent from the documentation a metric author reads.**
  ([correction](docs/corrections.md#metric-input-fields-undocumented))
- **Two documentation anchors had never resolved**, in `docs/extending.md`'s data-source section
  and in `README.md`'s link to the statusline automation guide. Both were found by checking every
  Markdown link and anchor in the repository while splitting the extension docs.
- **`site.yml`'s guards cover every served page**, not `index.html` alone, and the `noversion`
  guard masks dotted quads first -- `reference.html` publishes `127.0.0.1:8787`, which carries a
  three-component prefix. ([correction](docs/corrections.md#site-guards-read-one-page))
- **The site deploy no longer resolves its own tooling at build time.**
  ([correction](docs/corrections.md#site-deploy-resolved-wrangler-at-build-time))
- **Every internal link and canonical names a path the host actually serves.**
  ([correction](docs/corrections.md#canonical-pointed-at-a-redirect))

### Removed

- **The website's roadmap section is gone** -- 6,519 bytes and eight cards duplicating
  `ROADMAP.md`, which is linked instead. It was the one place on the page permitted to describe
  things that do not exist.
- **854 bytes of stylesheet for the deleted roadmap section** left `site/index.html` with it.

## [0.17.0] - 2026-08-11

### Added

- **Six prepared `Input` fields now cross the metric-plugin boundary** (`B155`): `windowStart`,
  `planMonthlyCost`, `skills`, `agents`, `turnSizing`, `cacheMisses`. All six are additive.
  ([correction](docs/corrections.md#plugin-input-missing-six-fields))
- **A label nobody has to type, as a rule engine rather than one convention** (`B152`). `mark
  --suggest` derives a label from what the store recorded and shows its evidence. Sources that
  disagree derive **nothing** for that axis.
- **A digest that reports what moved** (`B11`). `digest --weekly` writes markdown for cron, and
  states when the comparison itself is weak -- overlapping windows, unequal lengths, or a parser
  that changed between the two runs.

### Fixed

- **`clear` names the store it is about to empty.** Found the hard way: a run set an environment
  variable that does not exist, expected a copy, and emptied a real 513,617-row store.
  ([correction](docs/corrections.md#clear-did-not-name-the-store))
- **`clear` drops the digest's comparison basis with the records it described.** The next digest
  would have reported a pruned store as "tokens -62%, this model gone".
  ([correction](docs/corrections.md#clear-dropped-the-digest-basis))
- **`mark <id> --accept-suggested` no longer ignores the session it was given.**
  ([correction](docs/corrections.md#mark-accept-suggested-ignored-its-target))
- **A suggestion never reaches a session anyone has already annotated.** Clearing a single axis
  leaves the row in place, so a blank there is a decision rather than a gap; treating it as
  unanswered let `--accept-suggested` re-derive exactly what somebody had removed.
- **The digest stops reporting things that did not move.**
  ([correction](docs/corrections.md#digest-reported-what-did-not-move))
- **The digest's cost figure carries the same error bar as every other cost surface.** It stored
  no notion of what the price table could not cost, so a week where a new model went unpriced
  rendered as a fall in spend. ([correction](docs/corrections.md#digest-cost-had-no-error-bar))
- **The digest's stored basis is bounded and matched per window.** The bound trimmed the newest
  12 across all windows while the read filtered by window, so twelve daily runs could evict a
  monthly basis and silently turn that digest into a first run.
- **`labels.rules` is validated where every other config section is.** A malformed rule set
  passed `config` and `doctor` and only failed when `mark --suggest` ran.
- **The metric-plugin parity canary can no longer pass on a field it never maps.** It compared
  field names only, so a field present on the envelope but never filled by `buildMetricInput`
  would have reached every plugin as an honest-looking empty.

### Security

- **Derived labels respect the member boundary.**
  ([correction](docs/corrections.md#derived-labels-crossed-the-member-boundary))
- **Project names in a digest are pseudonymized by default**, like every other file assaio
  writes to be shared (`privacy.anonymize`, default true).
  ([correction](docs/corrections.md#digest-project-names-unpseudonymized))

## [0.16.0] - 2026-08-11

### Added

- **A cost figure now says how much of itself is missing** (`B139`). The `*` disclosing an
  unpriced model read identically at 0.1% and at the 45.5% that once left an estimate $15,452.42
  short for five weeks. ([correction](docs/corrections.md#unpriced-share-was-one-asterisk))
- **Nineteen reads stop arriving as nineteen equals** (`B148`). `analyze`, `demo` and the
  dashboard lead with the findings worth a week's attention. An ordering and never a score: a
  window whose reads are all fine promotes none of them.
- **`https://assaio.dev/llms.txt`** states what assaio measures, what it refuses to measure, and
  links the documents that answer the rest, so an assistant reads a summary instead of scraping
  the page.

### Changed

- **The faceplate shows a verdict and its gauge, without the bare ratio** (`B145`). `STRONG ·
  0.46` read as a contradiction, and purity is a different quantity in every validator, so a
  bare figure invited a meaningless comparison.

### Fixed

- **`insufficient` stops blaming a source for history it does record (`B115`).**
  ([correction](docs/corrections.md#insufficient-blamed-the-source))
- **A count reads the same on every surface (`B146`).** Validator figures group thousands
  through the shared formatter, so the dashboard no longer prints `16400 tokens` beside a report
  table printing `16,400`. ([correction](docs/corrections.md#counts-formatted-two-ways))
- **The website carries a share card.** Every Open Graph tag was present except `og:image`, so a
  link to assaio.dev rendered as a bare grey line wherever it was pasted. CI now fails if the
  tags go missing.

## [0.15.0] - 2026-08-10

### Added

- **`reconcile` -- check the `$` figure against the vendor's own numbers, offline** (`B19`). No
  credential, no network. Overlap scope first, then only evidenced causes, and the residual
  reported as *unexplained delta* -- never adjusted, never rounded away.

### Changed

- **Validator prose renders a real em dash in the dashboard.** `internal/analyze` writes ` -- `
  because the text report prints to terminals that cannot be assumed to render U+2014.

### Fixed

- **Money reads like money.** ([correction](docs/corrections.md#money-rendered-as-0-0000))
- **The Assay dashboard is easier to read.**
  ([correction](docs/corrections.md#dashboard-gauges-misread))

## [0.14.0] - 2026-08-09

### Breaking

- **Your stored activity counts change in both directions, and the upgrade rebuilds them.** On
  5,586 transcripts, lines added 383,579 -> **1,010,406**; migration `0010` makes the first
  import slow. ([correction](docs/corrections.md#stored-activity-counts-rebuilt))
- **`report --format csv` gained three columns** — `task`, `outcome`, `difficulty`, after
  `granularity`. A consumer reading columns by position must be updated; one reading them by
  header name is unaffected.
- **A parser plugin's record line is now rejected for a field the protocol does not define**
  (`B143`), which the metric and rule protocols already did.
  ([correction](docs/corrections.md#parser-plugin-unknown-field-stored-a-zero))
- **A parser plugin's records are now rejected for an out-of-range timestamp**, matching what
  the sync endpoint already enforced on the same shape, and for `reasoning_tokens` above
  `output_tokens`. A conforming plugin is unaffected.
- **A `backfill` may now delete stored rows** -- only Claude Code sub-agent aggregates whose own
  transcript is on disk, which were counting the same work twice. Zero such rows here; a store
  upgraded from v0.1.0 may hold some.

### Added

- **A calibration suite whose expected answers do not come from the parser that produces them**
  (`B137`): eight redacted traces with totals counted from raw bytes, plus invariants and
  metamorphic properties over a real corpus.

### Changed

- A metric plugin's wire input carries `cacheWrite1h` on every usage row and on every price,
  so a plugin re-pricing what it is handed no longer has to bill every cache write at the
  cheaper 5-minute rate and report a cost the core disagrees with.

### Fixed

- **A projected "per month" figure is now a calendar month, not thirty *active* days** (`B142`).
  Here the 30-day API-equivalent moves $107.6K -> $100.6K/mo; every existing headline moves
  down. ([correction](docs/corrections.md#per-month-was-thirty-active-days))
- **Every context compaction was counted twice.** Claude Code writes one overflow as two
  adjacent lines — a `system` boundary and the user-side summary that replaces the context — and
  the counter fired on each. ([correction](docs/corrections.md#every-compaction-counted-twice))
- **`codex` and `gemini-cli` claimed a cache-write signal neither parser can produce.**
  ([correction](docs/corrections.md#cache-write-signal-overclaimed))
- **Every file Claude Code created counted as zero added lines.** On 5,586 real transcripts,
  added lines 383,579 -> **1,010,406**: the corpus had been reporting 38% of what the tool
  wrote. ([correction](docs/corrections.md#created-files-counted-zero-lines))
- **A completed sub-agent's record depended on a field it does not own**, dropping 477 sub-agent
  records with 21,369 of their lines. Caught by the real-corpus A/B before it shipped.
  ([correction](docs/corrections.md#subagent-result-content-union))
- **A repeated transcript line was counted twice.** On 5,597 real transcripts, 329 repeated edit
  results carried 460 added and 656 removed lines: 382,738 added reported against a true
  382,278. ([correction](docs/corrections.md#repeated-transcript-line-counted-twice))
- **A stored sub-agent aggregate outlived the transcript that superseded it.** The parent
  transcript summarizes a completed sub-agent as one row; the sub-agent's own file holds the
  same work per turn. ([correction](docs/corrections.md#stale-subagent-aggregate-survived))
- **The team server could never correct a partial figure.**
  ([correction](docs/corrections.md#team-server-could-not-correct-a-partial-figure))
- **Non-ASCII filenames left the survival rate silently.**
  ([correction](docs/corrections.md#quotepath-dropped-files-from-survival))
- **A failed blame was reported as code that did not survive.**
  ([correction](docs/corrections.md#failed-blame-counted-as-not-surviving))
- **Every worktree session in every repository resolved to a project named `..`** on git 2.48
  and later, which writes the worktree pointer *relative* to the worktree.
  ([correction](docs/corrections.md#worktree-projects-named-dotdot))
- **The "seven-day" recent window covered eight day-buckets.**
  ([correction](docs/corrections.md#recent-window-covered-eight-buckets))
- **Usage that names no project was counted as a project.**
  ([correction](docs/corrections.md#nameless-usage-counted-as-a-project))
- **`report --by task|outcome|difficulty --format csv` emitted rows nothing could tell apart.**
  ([correction](docs/corrections.md#label-dimension-csv-had-no-columns))
- **A metric plugin could inject a record with no timestamp bound**, so a year-9999 row sat
  inside every `--since` window forever (`B133`).
  ([correction](docs/corrections.md#plugin-record-with-no-timestamp-bound))
- **`init --db` imported to one store and reported from another**, so the command printed an
  empty first run against the database it had just told the user about.
  ([correction](docs/corrections.md#init-db-wrote-one-store-and-read-another))
- **`skill-economics` totalled one dimension and ranked another** (`B135`): an 80% share sat
  beside a total it was never taken from.
  ([correction](docs/corrections.md#skill-economics-two-dimensions))
- **`rework` drew a full gauge beside a withheld verdict.**
  ([correction](docs/corrections.md#rework-full-gauge-beside-a-withheld-verdict))
- **A Codex diff could lose a removed line to a comment marker.**
  ([correction](docs/corrections.md#codex-diff-lost-a-removed-line))
- **Codex could store more prompt tokens than its own total gained**, keeping a whole cached
  delta beside a clamped zero. Unobserved on the audited corpus (0 of 1,686 events).
  ([correction](docs/corrections.md#codex-cached-input-exceeded-its-total))
- **A record with no timestamp was stored and then invisible.** Every report, validator and
  dashboard window is bounded by `ts >= ?`, so such a row counted toward the store's totals and
  appeared in no window. ([correction](docs/corrections.md#undated-record-stored-and-invisible))
- **A model name that arrived late could never be applied.**
  ([correction](docs/corrections.md#late-model-name-never-applied))

### Removed

- **The team dashboard's per-member row now shows sessions only** (`B141`).
  ([correction](docs/corrections.md#team-member-row-showed-lines-and-cost))
- **`assaio-agent init` no longer accepts `--db`.** It never honoured it (see below), so the
  flag only ever produced a wrong answer.

## [0.13.0] - 2026-08-08

### Breaking

- **Your stored numbers change, and the upgrade rebuilds them.** Migration `0009` clears two
  watermarks: Codex 6,632 -> 10,604 added lines, Claude Code 37,885 -> 33,752 rework lines.
  ([correction](docs/corrections.md#stored-numbers-rebuilt-v0130))
- **`check --max-cost` can now fail where it used to pass**, on a window carrying tokens the
  price table has no row for. Refresh the price table first: the old behaviour compared your
  budget against a figure missing that model's whole spend.

### Fixed

- **Codex dropped every line of every file it created.** On 61 real rollouts, 54 created files
  carried 3,972 uncounted added lines: 6,632 reported where the true figure is 10,604.
  ([correction](docs/corrections.md#codex-created-files-lost-every-line))
- **The rework cap was a budget nobody spent, so churn could exceed the additions it undoes.**
  On the maintainer's corpus, Claude Code rework 37,885 -> 33,752 lines (13.3% -> 11.9%).
  ([correction](docs/corrections.md#rework-cap-was-a-budget-nobody-spent))
- **A corrected rework rule could not reach a single stored row.**
  ([correction](docs/corrections.md#corrected-rework-rule-could-not-reach-a-row))
- **`clear` left a store that no `backfill` could refill.**
  ([correction](docs/corrections.md#clear-left-a-store-backfill-could-not-refill))
- **`clear --tool codex --labels` deleted every other tool's session labels.**
  ([correction](docs/corrections.md#clear-labels-ignored-its-scope))
- **`doctor --strict` exited 0 on a store it could not open.**
  ([correction](docs/corrections.md#doctor-strict-exited-zero-on-a-broken-store))
- **Nearly half the flagship setup's tokens had no price.** 22.7B of 49.8B tokens -- 45.5% --
  resolved to none, and the window's estimate rose from `$23,750.98` to `$39,203.40` once they
  did. ([correction](docs/corrections.md#half-the-tokens-had-no-price))
- **`check --max-cost` reported OK on a window it could not price.**
  ([correction](docs/corrections.md#check-max-cost-passed-on-an-unpriced-window))
- **A `#` in the store's path opened a different database, and a `?` silently dropped the
  pragmas.** ([correction](docs/corrections.md#store-path-uri-escaping))
- **Compact units were chosen before rounding, so a value printed its own ceiling.**
  `Count(999,999,999)` rendered `1000.0M` instead of `1.0B`, and an exact 0.5% printed `0%`.
  ([correction](docs/corrections.md#compact-units-chosen-before-rounding))
- **The dashboard rendered a real cost as `$0`.**
  ([correction](docs/corrections.md#dashboard-rendered-a-real-cost-as-zero))

## [0.12.0] - 2026-08-07

### Added

- **`cache-hygiene` says *why* a prompt missed the cache.** New signal `ai.cache.miss_reason` --
  and the caveat claiming "vendor cache TTLs are invisible" is gone, because it was false.

### Changed

- `reconcileColumns` now runs before the migration files rather than after, so a migration can
  rely on every column `0001` declares being present. A database with no `usage_record` yet —
  every fresh one — has nothing to heal and returns early.

### Fixed

- **One Claude Code response was counted once per content block, so every token figure for the
  flagship source was roughly double.** On 5,724 real transcripts, cost $53,208 -> $24,339.
  ([correction](docs/corrections.md#response-counted-once-per-content-block))
- **A cache write was priced at the cheap tier whatever lifetime it bought.** 59.7% of the
  audited corpus's cache-write tokens are 1-hour writes, billed at 1.6x, so that component rose
  35.8%. ([correction](docs/corrections.md#cache-write-priced-at-the-cheap-tier))

## [0.11.0] - 2026-08-06

### Added

- **The exec metric envelope (`assaio_metric_input: 1`) gains an `answers` field**, mapping each
  tool in the window to the signal ids it can produce.
  ([correction](docs/corrections.md#plugin-envelope-answers-field))

### Changed

- **`turn-efficiency` gates on both fields it reads.** No source today records one without the
  other, so nothing was wrong; the invariant was held by coincidence rather than by the gate.
- **One implementation per shared question.** "Which sources answer this signal" had three
  implementations and is now `parser.SourcesAnswering`; every median and p95 now shares
  `report.Percentile`.
- No schema change and no migration. `store.Sessions` narrows three of its columns to `turn`-grain
  rows, which moves per-turn session figures on any store holding Claude sub-agent aggregates; no
  stored row is rewritten and re-running `backfill` is not required.

### Fixed

- **A real signal no longer rounds away to "0%".**
  ([correction](docs/corrections.md#rejection-rate-rounded-to-zero))
- **`rework` averaged a source's silence into the churn rate.**
  ([correction](docs/corrections.md#rework-averaged-a-sources-silence))
- **Two more denominators counted work their source never recorded.**
  ([correction](docs/corrections.md#friction-and-explore-denominators))
- **A whole sub-agent run stopped counting as one turn.** On the audited store 65 of 779
  sessions carried 1,015 such rows, one inflated by 89 phantom turns; median turns per code
  session 724 -> 718. ([correction](docs/corrections.md#subagent-run-counted-as-one-turn))
- **`context`'s code-session median read an edit count Cline never writes.**
  ([correction](docs/corrections.md#context-median-read-an-unwritten-edit-count))
- **`rhythm`'s confidence line contradicted its own figures**, printing "insufficient -- nothing
  in this window can answer it" beneath an off-hours share computed from 100% of the window.
  ([correction](docs/corrections.md#rhythm-confidence-contradicted-its-figures))
- **The field audit's Codex cache-write row overstated its own consequence.** It said Codex cost
  is a floor because the cache-write count is unread.
  ([correction](docs/corrections.md#field-audit-overstated-its-own-consequence))
- **The ADR 0011 invariant test now varies both row shapes and both shapes of silent source.**
  ([correction](docs/corrections.md#adr0011-test-had-a-blind-spot))

## [0.10.0] - 2026-08-05

### Added

- **The unread-field audit, source by source** (`B105`): every field a tool writes is
  **extracted** or **skipped, with the reason written down**, and each table names the corpus it
  was built from. What it found is tracked as `B107`-`B112`.
- **Two signals join the catalog**: `ai.compactions.count` and `ai.rejected.count`. They were
  stored columns and the subject of two verdicts without ever being declarable, which is why
  those metrics could not tell absence from zero.
- **[ADR 0011](docs/adr/0011-capability-gated-metrics.md) records the rule the fixes above
  share**, with a generic test asserting every validator returns the same result whether or not
  a source that cannot fill a field carries a value.

### Changed

- **Every remaining hardcoded tool list in user-facing text now reads from the depth matrix**:
  `skill-economics`'s provenance caveat and the `explore-produce` explain page named tools in
  prose that the next parser would have made wrong.
- **Migration `0006_subagent_session_grain.sql`** relabels stored Claude sub-agent aggregates
  from `turn` to `session`. A re-parse cannot reach them, because a sub-agent's own transcript
  suppresses the parent's aggregate at parse time.
- **`InsertLocal` now restates `granularity` from the current parse** -- the one column assigned
  instead of maximised, because a build that learns a record summarizes a whole run has to be
  able to say so.

### Fixed

- **A metric read a source's silence as a zero.** A Gemini CLI, Cline or Copilot CLI window read
  as 100% conversational, 0% produced code, 0 marathons, 0% compaction -- and carried a verdict
  on all four. ([correction](docs/corrections.md#silence-read-as-a-zero))
- **`insufficient` now says which of three ways a verdict rests on nothing.** One sentence
  printed "nothing to measure in this window" over a store holding 119,896 records.
  ([correction](docs/corrections.md#insufficient-had-one-sentence-for-three-causes))
- **A completed Claude sub-agent is a session total, not a turn.** 1,015 such rows on the
  maintainer's machine averaged 2.4x a real turn's output while labelled `turn`.
  ([correction](docs/corrections.md#completed-subagent-labelled-a-turn))
- **The rejection rate gets its own denominator.**
  ([correction](docs/corrections.md#rejection-rate-had-the-wrong-denominator))
- **`concentration` stops blaming a project for a source that writes no lines.**
  ([correction](docs/corrections.md#concentration-blamed-a-lineless-source))
- **`skill-economics` states its reach, and a lone label no longer reads as zero tokens.** The
  concentration share is of attributed tokens -- 18% of the window on the maintainer's store.
  ([correction](docs/corrections.md#skill-economics-did-not-state-its-reach))
- **`doctor` reports a failed discovery instead of printing it as "not detected".** A root it
  could not read counted as zero files, while `backfill` reported the same condition as an
  error.
- **`make fuzz` runs the Copilot CLI parser's fuzzer.** It shipped one in v0.6.0 and the target
  never listed it, so the guarantee that every parser is fuzzed was true of the code and not of
  the gate. It passes; no crasher was found.

## [0.9.0] - 2026-08-04

### Added

- **Every verdict now says how much of your window it actually describes.** The confidence
  envelope gains `signalCoverage`: the other three axes describe the *window*, and only the
  metric can answer this one.
- **`survival` reports what merges hold.** `git log --numstat` prints no diff for a merge, so a
  conflict resolution is a hole in git's own reporting rather than a change of size zero.
- **The attribution conformance corpus** ([ADR
  0010](docs/adr/0010-attribution-conformance-corpus.md)): ten scenarios, each a real git
  repository, stating what an engine must conclude and where it must refuse.
- `analyze --format json` gains an optional `confidence.signalCoverage` field. It is absent
  when a metric does not declare one, so a consumer reading the envelope keeps working.
- Exec metric plugins (ADR 0004) may set the same key. Omitting it means what it meant before,
  so every released plugin is unaffected.

### Changed

- Survival rates change for repositories containing merge commits: the figure was previously
  inflated by blamed merge lines and is now computed over non-merge commits only. No stored
  data changes — `survival` reads git directly and persists nothing.

### Fixed

- **`survival` counted merge lines as survivors that were never counted as added.** On a fixture
  with a 50-line conflict resolution that printed `50 surviving of 3 added (100%)`.
  ([correction](docs/corrections.md#survival-counted-merge-lines))
- **A figure computed from a sliver of the window carried a confident envelope.**
  `reasoning-share` read a 20% share off under 1% of the output and reported `high`.
  ([correction](docs/corrections.md#a-sliver-carried-a-confident-envelope))
- **A real coverage share no longer rounds to an absent-looking `0%`** in the confidence line;
  it reads `<1%`, the same honest rounding every other percentage in the reports already used.
- **The README's headline and the exec-plugin schema table omitted GitHub Copilot CLI**, which
  has been a supported source since v0.6.0 — the README said "five sources" three paragraphs
  further down.

## [0.8.0] - 2026-08-03

### Added

- **The local git evidence collector: what your commits changed, never what they changed it to**
  ([ADR 0009](docs/adr/0009-local-git-evidence-collector.md)). There is no field for a path, a
  branch name, a commit message or a diff.
- **`survival` reports what the window actually changed.** It reads those observations instead
  of shelling out to git, so the rate travels beside the file mix and the commits git itself
  labelled a revert.

### Changed

- **What a source can answer has exactly one place.** `parser.Answers(tool, signal)` is now the
  only capability question in the codebase, and eight surfaces read it instead of keeping a
  private copy.

### Fixed

- **Copilot CLI was only half-wired, and three surfaces were the proof.**
  ([correction](docs/corrections.md#copilot-cli-was-half-wired))
- **The signal catalog claimed every source reports reasoning tokens.** On the maintainer's own
  store the figure moved from "100% of tokens" to "<1%, codex", which is the honest number.
  ([correction](docs/corrections.md#reasoning-signal-claimed-for-every-source))
- **A source that records lines and nothing else passed for full activity coverage.** A
  Copilot-only window reported *activity coverage 100%* while `signals coverage` said its
  signals were unsupported. ([correction](docs/corrections.md#activity-coverage-was-one-bit))
- **A Copilot session with no id or no timestamp is skipped instead of stored.**
  ([correction](docs/corrections.md#copilot-session-with-no-id))
- **Caveats stopped naming sources.**
  ([correction](docs/corrections.md#caveats-named-the-wrong-sources))

### Security

- **A session annotation no longer selects another member's usage.**
  ([correction](docs/corrections.md#label-filter-crossed-members))

## [0.7.0] - 2026-08-02

### Added

- **`signals` -- what assaio can tell you, and what your own data supports** ([ADR
  0008](docs/adr/0008-signal-catalog.md)). `describe` says what a signal counts and -- the field
  that earns the catalog its keep -- **what a zero means**.
- **The canonical event contract, the first piece of the evidence graph** ([ADR
  0007](docs/adr/0007-canonical-event-contract.md)). An **interface contract, not a storage
  format**: no event table, no migration, and content impossible by construction.

### Changed

- `mark` no longer answers an ambiguous session prefix with every match it found — on a real
  store a one-character prefix printed 48 ids on one line. It now lists six and counts the
  rest, the way git reports an ambiguous short revision.
- **The dashboard's project drill is held to the fields it does not receive.** A validator
  reading one while not being `WindowScoped` would report "no attribution" as a fact about the
  project. That has never happened; a test now asserts it.
- `internal/store/label.go` grew past the file budget doing two jobs; naming a session is now
  `session_ref.go`, which is where the git evidence collector will look for it.

### Fixed

- **The source-depth matrix now declares capability per signal, not per axis.**
  ([correction](docs/corrections.md#depth-matrix-overclaimed-per-axis))
- **Proving the event contract against 324,416 real records dropped 51 of them.** Rejecting an
  event whose source timestamp is newer than the batch's reading time discarded sessions still
  being written. ([correction](docs/corrections.md#event-contract-clock-ordering))

## [0.6.0] - 2026-08-02

### Added

- **Sessions can be labeled with what the work actually was** (`B80`). `mark` attaches a task
  class, an outcome and a difficulty -- the one fact session logs never contain. Setting an axis
  later merges rather than replaces.
- **Every metric can now be read per kind of work.** `analyze --task refactor` recomputes every
  validator over just those sessions. Verdicts describing the whole window rather than a slice
  of it are skipped and named as skipped.
- **`intent` validator** -- how much of the window carries a label, and whether that is enough
  to compare kinds of work at all. It has no unfavorable verdict by design.
- **`clear --labels`** — the deliberate way to delete session labels. `--all`,
  `--older-than` and `--tool` now leave them alone and report how many they kept, because
  labels are the only data in the store that no re-import can rebuild.
- **GitHub Copilot CLI is a supported source** (`B53`). Two modelling decisions could have
  produced wrong numbers: `usage.inputTokens` is the **whole** prompt, so the uncached share is
  read from `tokenDetails`; and code changes have no per-model split.
- **Two CI workflows guard the documentation lifecycle instead of review memory**: `site` fails
  when the page names a version other than the latest tag, and `consistency` fails on a
  completed backlog item, a duplicate id, or a version heading with no tag.
- **Migration `0005_session_labels.sql`** adds the `session_label` table and a `(project, ts)`
  index on `usage_record` (`B70`). Category values are validated in Go rather than by a SQL
  `CHECK`, so adding one later is not a migration.

### Changed

- **Labeling a session cannot move a figure anyone was already reading.** The annotation join is
  opt-in per query, so every unfiltered run executes exactly the SQL it ran before, asserted
  byte-identical by a regression test.
- `report --by` and `effectiveness --by` accept `task`, `outcome` and `difficulty`. Usage
  from unlabeled sessions is always rendered as its own `unlabeled` group, never hidden.
- **The roadmap now leads with outcome evidence, not recommendations.** A suggestion resting on
  activity and output proxies alone is a guess delivered in a confident voice.
- **A milestone no longer carries a version number until it ships.** Pre-assigning `v0.7` made
  the roadmap schedule work it explicitly says it does not schedule, and it broke the moment a
  small release landed between two milestones.
- **The stated positioning changed.** `$` per 100 AI lines remains a signal but is no longer
  the headline answer to "is AI delivering" — it is an *output* measure, and promoting one to
  an outcome claim is the most likely way this project could start lying.
- Shipped backlog items are deleted rather than left checked off, as that file's own
  lifecycle rule requires.
- The store grows by roughly 80 bytes per session you mark by hand. It does not grow with
  the volume of logs ingested, and nothing prunes labels automatically.

### Fixed

- README no longer describes the tool as "the v0.1 local agent — the only thing that ships
  today", five releases after that stopped being true; `site/index.html` was three releases
  behind and now describes this one.
- **Four surfaces were counting four sources and eighteen validators.**
  ([correction](docs/corrections.md#surfaces-counted-four-sources))

## [0.5.0] - 2026-07-31

### Breaking

- **`report --format csv` gained a `granularity` column** between `member` and `in`. A
  consumer that reads columns by position rather than by header name needs updating.

### Added

- **Format-drift canaries** (`B58`). The failure that mattered was never a crash but
  plausible-looking under-reporting. Verified on a real 4.5 GB corpus: **317,354 records, no
  canary fired**; renaming the vendor's token fields fired the zero-token canary at 100.0%.
- **`doctor --strict`** exits non-zero when a canary fired or a source configured in `sources:`
  finds no inputs at all (`B59`). The second case is the one no canary can catch: a path that
  never worked has no history to have shrunk from.
- **`compact`** rewrites the store without its free pages. Deleting records only moves bytes
  onto SQLite's freelist, so `clear` now says how much is still held. Separate, because
  rewriting needs twice the store's size in temporary disk.
- **A confidence envelope on every verdict** (`B81`). Every result carries the window's
  coverage, how many observations the metric counted, when the data was read and by which build,
  and a derived label -- deliberately not one opaque score.
- **`init`** -- one command for a first run (`B82`). It prints the exact directories it will
  read **before** reading anything, imports, and names the three commands worth running next. A
  machine with no AI tools installed is not an error.
- **A source depth matrix** (`B83`). "Supported" was one word for two very different things.
  Every source now publishes **deep**, **standard** or **import-only**, with the specific gaps
  spelled out below the top tier.
- **New migration `0004_ingest_source.sql`** adds an `ingest_source` table. It holds no usage
  and is bounded by construction: only the newest runs per tool are kept, pruned inside the same
  transaction that writes them.
- **The metric-plugin envelope gained `usage[].granularity`.** The addition is backward
  compatible and `assaio_metric_input` stays at `1`: a plugin that ignores the field behaves
  exactly as before.

### Changed

- **Per-input ingest state no longer grows with install age.** Rows for inputs no longer on disk
  are dropped after each pass, *unless* that source's discovery canary fired -- discarding state
  during the failure being diagnosed would destroy the evidence.
- **The canaries need a baseline, so the first `backfill` after upgrading fires nothing.**
  History-relative canaries stay silent until a second pass exists to compare against; the two
  absolute ones work from the first run.
- **`doctor` replaced its `activity:` line with a `depth:` section** and dropped the two
  caveats the matrix now owns; they are printed per source, and only for sources present on
  this machine.

### Fixed

- **Reports no longer blend per-turn and whole-session records silently.**
  ([correction](docs/corrections.md#granularity-blended-silently))
- **The README's manual install instructions pointed at v0.1.0**, three releases behind, so
  anyone following them got a binary without `statusline`, `explain` or incremental backfill.
  ([correction](docs/corrections.md#readme-install-pointed-at-v010))

### Deprecated

- **A metric plugin should now declare `confidence.samples` and `confidence.samplesUnit`.** One
  that omits them reads as `insufficient` -- the honest label for "did not say what it rests on"
  -- rather than borrowing a number assaio would invent.

## [0.4.0] - 2026-07-29

### Added

- **`backfill` is incremental.** Measured on a real 4.7 GB / 6262-file history: the pass that
  reads everything took **68 s**, the next one **0.13 s**, and that fast pass still picked up
  the 291 records written in between.
- **`statusline`** -- one ambient line for an editor or shell status bar. The day is the
  machine's **local** day. On a flat plan the money segment shows two raw numbers (`$412/$200
  mo`) rather than a percentage, which would read as "consumed".
- **`explain <validator>`** -- the long-form page for each metric: what it measures, how to read
  it, what to do about it, and the limits that keep it honest. It never opens the store, so it
  works before any data exists.
- **`doctor` reports ingest freshness per source**, so "why are my numbers stale" is
  answerable without guessing.
- **New migration `0003_ingest_state.sql`** adds an `ingest_file` table. It holds no usage --
  only which inputs were parsed, at what size, mtime and by which build -- so it can be dropped
  at the cost of one slow re-parse.

### Changed

- **The first `backfill` after upgrading is a full one**, by design: state written by a
  different build is never trusted, and that pass is what lets history gain the activity
  signals an older parser could not extract. Runs after it are incremental.
- Stores carried over from an earlier release have no ingest state until that first
  backfill, so `statusline` reports the data's age as unknown rather than guessing, and
  `doctor` says no ingest has been recorded yet.
- **Static user-visible text moved into a new `internal/i18n` catalog.** No wording changed and
  the dashboard's rendered output is byte-identical. Data-derived text stays with the validator,
  since translating it means templating interpolated numbers.
- The four duplicated K/M/B number formatters are collapsing into one `internal/humanize`
  package; `analyze`'s two call sites moved with byte-identical output (part of B75).
- `internal/ingest` split into `ingest.go`, `discover.go`, and `state.go`, which brings
  the package back under the file-size budget.

### Fixed

- **A session ingested while it was still being written froze a half-attributed turn.**
  ([correction](docs/corrections.md#live-transcript-froze-a-half-attributed-turn))
- **`doctor` under-reported Claude Code by thousands of files.**
  ([correction](docs/corrections.md#doctor-under-reported-claude-code-files))
- **The dashboard golden fixtures are built in `time.Local` rather than UTC.** `rhythm` reads
  session starts in the machine's own zone, so the goldens passed on a CET laptop and failed in
  CI. ([correction](docs/corrections.md#golden-fixtures-built-in-utc))

## [0.3.0] - 2026-07-26

### Breaking

- **`Result.BarsAreProjects` is now `Result.BarsPseudonym`** (JSON `barsAreProjects` ->
  `barsPseudonym`), a string, because a boolean could not say that skill and sub-agent names
  need pseudonymizing too. The old key is still accepted and maps to `"project"`.

### Added

- **Three `analyze` validators, on data the store already holds**: `concentration` (where a
  project's token share outruns its share of the AI lines, reported as neither good nor bad),
  `rhythm` (an aggregate signal, never individual) and `burn-anomaly`.
- **Tool-call purpose capture, and three validators over it.** Each call is classified during
  parsing and the tool's *name* is then dropped -- neither it nor any tool input is ever stored.
  On top of it: `explore-produce`, `friction` and `skill-economics`.
- **Sub-agent turns are now marked at the source.** Records carry `sidechain`, read from the
  log's own marker instead of inferred from the dedupe key, so the delegation share is exact.
- **Exec rule plugins -- your own CI gate, in any language** ([ADR
  0005](docs/adr/0005-exec-rule-plugin-protocol.md)). An `error` alert exits non-zero, and a
  rule that could not be evaluated fails the gate rather than passing silently.

### Changed

- **Record schema (migration `0002_activity_signals.sql`)**: nine activity columns added, all
  defaulting to `0`/`''`, so rows written by an older build stay valid and read as "not
  captured".
- **The first backfill after upgrading restates old rows.** History parsed by a build that could
  not extract these signals would otherwise keep zeros forever behind `ON CONFLICT DO NOTHING`.
- Exec **parser** plugins cannot emit the new activity fields yet; records they push read as
  "not captured" (`0`/`''`), not as a real zero. The wire protocol is unchanged.
- Metrics over the new fields state their own coverage. Gemini CLI and Cline logs do not name
  their tool calls, so their usage is excluded from the explore/produce split rather than
  counted as zero, and only Claude Code labels turns with a skill or sub-agent today.
- Percentile interpolation is now one shared helper (`percentileAt`) instead of a median
  open-coded in `context.go`, so every median and p95 figure uses the same method.
- The metric and rule protocols share one subprocess runner (`docProtocol`), so timeout,
  stdout cap, stderr prefixing, and handshake handling exist once. As a side effect a
  metric plugin that floods stdout is now killed on the breach instead of at the timeout.
- The CLI tests that assert `analyze --list` output no longer hard-code the validator
  count; they derive it from the registry, so registering a metric cannot break them.

### Fixed

- **The project drill re-ran window-scoped metrics against one project's rows.**
  ([correction](docs/corrections.md#drill-re-ran-window-scoped-metrics))
- **Codex's tool-purpose split reported 0% produced whatever the agent did**, because Codex
  applies file edits through a `patch_apply_end` event that no tool call names.
  ([correction](docs/corrections.md#codex-produce-share-was-zero))
- **`friction` counted calls that cannot report a failure.**
  ([correction](docs/corrections.md#friction-counted-unmarked-calls))
- **Sub-agent tokens were double-counted** on a store holding both a pre-upgrade `agent:`
  aggregate row and the per-turn rows later parsed from the same transcript.
  ([correction](docs/corrections.md#subagent-tokens-double-counted))
- **`concentration` returned a green ALIGNED when no project was large enough to compute a gap**
  -- a passed check for an examination that never ran -- and its two headline figures disagreed.
  ([correction](docs/corrections.md#concentration-passed-an-exam-it-never-ran))
- **`rhythm` asserted an off-hours finding its own verdict had refused**, printing "a large
  share of sessions runs off-hours" beside a neutral read on a two-session window.
- **`burn-anomaly` named calendar dates as spikes below its own day floor**, computed from a
  baseline the same entry declared unusable.
- **`skill-economics` rendered a share and a total from different dimensions** side by side,
  pointing the reader at a skill worth 1% of the window. Both now come from one dimension.
- **`check` double-counted reasoning tokens** against the budget, so a CI gate could fail a
  window that `report` and `analyze` showed as under budget. The same double-count is fixed
  in the delegation query.
- **`StructuredOutput` counted as code production**, inflating the produce share of any turn
  that answered structurally. It writes no file and is classified as "other".
- **`assaio-agent demo` left three of eighteen panels blank** — the tool's own showcase — for
  want of the new signals on its bundled records.
  ([correction](docs/corrections.md#demo-left-three-panels-blank))
- **`check` failed open in two ways.** ([correction](docs/corrections.md#check-failed-open))
- **`friction` reported a fabricated 0.0% and a green verdict** where no call could record a
  failure, and impossible percentages (`150%` errors, `-50%` clean) when errors outnumbered
  counted calls. ([correction](docs/corrections.md#friction-fabricated-a-clean-zero))
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

### Security

- **A metric plugin written against the pre-rename `barsAreProjects` key lost its
  pseudonymization**, publishing real repository names into an anonymized dashboard.
  ([correction](docs/corrections.md#legacy-bars-key-disarmed-pseudonymization))
- **The sync boundary accepted a tool-purpose split that contradicts its own tool-call
  count**, and a `sidechain` outside 0/1. Both are rejected now; an all-zero split with calls
  present stays valid, since that is the documented "not captured" state.
- **Skill and sub-agent names reached anonymized reports verbatim.**
  ([correction](docs/corrections.md#skill-names-reached-anonymized-reports))
- **Rule plugins were handed the ranked bar lists**, which carry project, skill, and
  sub-agent names — more than PRIVACY.md said they receive. Bars are stripped from the
  envelope; a rule gates on a verdict, not on which repository produced it.
- **The nine columns migration 0002 adds bypassed the sync boundary's numeric check**, so a
  pushed record could store a negative count or an overflow-magnitude one, breaking `SUM()` for
  the whole team. ([correction](docs/corrections.md#pushed-records-bypassed-the-numeric-check))

## [0.2.0] - 2026-07-22

### Added

- **Three `analyze` validators**: `coverage` (the provenance meter), `cache-hygiene`
  (prompt-cache reuse with an honest cache-write-waste flag), and `subscription-fit`, which
  reads the API-equivalent estimate as plan value rather than as spend.
- **Four behavioral `analyze` validators**: `session-taxonomy`, `turn-efficiency`,
  `model-right-sizing` (reframed as speed and limits on a flat plan) and `reasoning-share`,
  honest about which tools report it. Twelve built-in validators in total.
- **`survival` command** -- the first local outcome signal: how much of the window's commits
  still live in `HEAD`, beside the AI lines recorded for that project. Directional, and it never
  attributes specific lines to AI.
- **Dashboard unpriced honesty.** Cost figures that exclude usage on unpriced models are
  now marked `*` (main cost basis and per-member team costs), with a colophon note —
  matching the CLI tables instead of showing a silent floor.
- **Cline discovery across editors.** Cline task data is now found under VS Code Insiders,
  VSCodium, and Cursor (not just stable VS Code), using the same `saoudrizwan.claude-dev`
  global storage — so Cline usage in any of those editors is counted, not silently missed.

### Fixed

- **Coverage rounding.** In `coverage`, a small but nonzero token share now reads `<1%` instead
  of `0%` (which looked absent), and a share just under whole reads `>99%` instead of a
  gap-hiding `100%`. ([correction](docs/corrections.md#coverage-share-rounded-to-zero))
- **Gemini session ids** are carried only on the file's header line, so message records had an
  empty id. The dedupe key changes: run `clear --tool gemini-cli --yes`, then `backfill`, once
  after upgrading. ([correction](docs/corrections.md#gemini-session-ids-on-the-header-line))
- **Reasoning tokens no longer double-counted.** Codex and Gemini report reasoning as a
  subset of output; the grand token totals added it a second time, inflating every token
  count (cost was unaffected). Totals now count it once.
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
- **Plugin I/O robustness.** A newline-free stderr flood is bounded; a stdout-cap breach is
  reported as such and the child killed promptly instead of being misreported as a timeout;
  string fields are length-capped at the boundary.
- **Codex timestamps.** A `session_meta` whose payload omits a timestamp no longer resets
  the record timestamp to the zero time. Cline model resolution is now deterministic.
- **Schema migrations apply atomically** with their bookkeeping row, so a crash mid-migration
  can't leave a half-applied migration that re-runs next boot.
- **`clear` guards.** An unknown `--tool` value (e.g. `claude` for `claude-code`) now errors
  instead of silently deleting nothing, and `--all` combined with `--older-than`/`--tool`
  is rejected as contradictory rather than silently narrowing the deletion.
- **`--db`-aware empty-store hints.** `effectiveness`, `status` and `analyze` no longer tell a
  `--db` user to run `backfill`, which only writes the local store. `--compare` now errors
  instead of silently ignoring `--format json|csv`.

### Security

- **Anonymized dashboard no longer leaks real subpath names.** The drill-down's
  repository-subpath table was passed through verbatim under `--anonymize`, exposing paths
  like `apps/mobile` beside a pseudonymized project; subpaths are now pseudonymized too.

## [0.1.1] - 2026-07-20

### Added

- CI now runs the test suite on macOS and Windows alongside Linux; POSIX-only
  plugin-script tests are skipped on Windows.
- Per-platform install instructions in the README: Windows (PowerShell), Linux/macOS
  tarball, Homebrew/Linuxbrew, `go install`, and attestation verification.

### Fixed

- **Claude Code sub-agent accounting.**
  ([correction](docs/corrections.md#subagent-accounting-under-counted))
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

[Unreleased]: https://github.com/assaio/assaio/compare/v0.25.0...HEAD
[0.25.0]: https://github.com/assaio/assaio/compare/v0.24.0...v0.25.0
[0.24.0]: https://github.com/assaio/assaio/compare/v0.23.1...v0.24.0
[0.23.1]: https://github.com/assaio/assaio/compare/v0.23.0...v0.23.1
[0.23.0]: https://github.com/assaio/assaio/compare/v0.22.0...v0.23.0
[0.22.0]: https://github.com/assaio/assaio/compare/v0.21.0...v0.22.0
[0.21.0]: https://github.com/assaio/assaio/compare/v0.20.0...v0.21.0
[0.20.0]: https://github.com/assaio/assaio/compare/v0.19.0...v0.20.0
[0.19.0]: https://github.com/assaio/assaio/compare/v0.18.0...v0.19.0
[0.18.0]: https://github.com/assaio/assaio/compare/v0.17.0...v0.18.0
[0.17.0]: https://github.com/assaio/assaio/compare/v0.16.0...v0.17.0
[0.16.0]: https://github.com/assaio/assaio/compare/v0.15.0...v0.16.0
[0.15.0]: https://github.com/assaio/assaio/compare/v0.14.0...v0.15.0
[0.14.0]: https://github.com/assaio/assaio/compare/v0.13.0...v0.14.0
[0.13.0]: https://github.com/assaio/assaio/compare/v0.12.0...v0.13.0
[0.12.0]: https://github.com/assaio/assaio/compare/v0.11.0...v0.12.0
[0.11.0]: https://github.com/assaio/assaio/compare/v0.10.0...v0.11.0
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
