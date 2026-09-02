# Corrections register

Every figure assaio prints is a claim, and some of them have been wrong. This file is the
record of those: what was wrong, since when, what the wrong number showed a reader, what the
fix changed, and — where a second review pass overturned the first attempt — what the fix
itself overruled.

It exists because the product makes a promise the rest of the tooling cannot keep on its own.
assaio restates history: a parser fix reaches records that were read years of releases ago, so
a number you were shown in one release can be a different number in the next. A tool that
silently improves its own past is indistinguishable, from the inside, from one that drifts. So
each correction is written down, with the release that shipped it, and
[CHANGELOG.md](../CHANGELOG.md) carries a one-line entry pointing here.

**How to read it.** Entries are newest first, grouped by the release that shipped the
correction, and each states that release again so a deep link is self-contained. Anchors are
stable: an entry keeps its `#anchor` even if its wording is edited. The text of every entry is
the text it shipped with — figures, corpus sizes and all — so the measurement that justified a
correction stays attached to it.

**What is here and what is not.** A correction is an entry that describes something being
*wrong* and then *put right*: a wrong number, a fabricated zero, a verdict resting on nothing,
a published claim that had stopped being true, a boundary that let bad data through. A plain
feature addition is not a correction and stays a one-liner in the changelog. Small fixes whose
whole story fits in a changelog line stay there too — this file holds the ones with a
post-mortem to tell.

Two conventions this project holds itself to, visible throughout: a correction states what the
wrong figure *showed*, not merely that something was fixed; and where a figure was measured on
the maintainer's own corpus, the corpus is named beside it.

## [Unreleased]

## [0.25.0] - 2026-09-02

<a id="cache-tier-caveat-outlived-its-defect"></a>

### Four surfaces kept warning that the 1-hour cache-write tier was not priced, three releases after it was

*Corrected in v0.25.0, released 2026-09-02.*

v0.12.0 shipped the 1-hour cache-write tier as a correction of its own
([the entry that shipped it](#cache-write-priced-at-the-cheap-tier)): 59.7% of the audited
corpus's cache-write tokens turned out to be 1-hour writes, and pricing them at the 5-minute
rate had understated that component by 35.8%. `Price.CacheWrite1h` has been billed at its own
rate ever since, the Claude parser populates `CacheWrite1hTokens`, and the signal
`ai.tokens.cache_write_1h` has carried `Status: Observed` throughout.

The caveats written for the defect stayed. `doctor` printed "long-context (e.g. `[1m]`) and
1h-cache premiums are not modeled yet" on **every run**; `explain cost` said "the cache lifetime
a write bought is in some logs but not read yet" and "does not model the 1h-cache premium, so
heavily cached sessions are costed slightly low"; README said the same; and the signal catalog's
`ai.cost.estimated` said "cache tiers are not modelled" — which renders onto the generated
reference page **one row above** `ai.tokens.cache_write_1h`'s own "billed at its own higher
rate". The published reference contradicted itself at a distance of two lines.

The long-context half of every one of those sentences is still true — `NormalizeModel` strips
`[1m]` and the table carries no 1M-context entry — so only the cache clause is withdrawn.

This is the mirror image of the defect the register usually holds. A stale caveat does not
publish a wrong number; it tells a reader to distrust a right one, and it is harder to catch,
because every mechanical guard this project owns asks whether a claim about a *capability* is
true and none asks whether a claim about a *limitation* still is. A correction that closes a
defect has to retire the warning in the same change, and nothing enforced that.

<a id="needs-key-made-the-config-unloadable"></a>

### `needs: [trace]` — the key v0.24 required — made the configuration file fail to load

*Corrected in v0.25.0, released 2026-09-02.*

v0.24.0 made the step timeline conditional on a metric plugin's `metrics:` entry declaring
`needs: [trace]`, published that as a Breaking change with `needs: [trace]` as its migration
path, and documented it in the metric-plugin guide, ADR 0004 and
[the correction that shipped it](#metric-handshake-version-3). `Config.Validate` then held every
`metrics:` entry to `PluginConfig.Validate`, the parser-and-rule rule, whose last check is
`needs: applies to metric plugins only`. So the documented migration was the one edit that could
not be made: a configuration file carrying it was **rejected whole**.

What a reader saw is worse than a broken metric. Validation runs at config load, so the failure
was not scoped to `analyze` — `report`, `backfill`, `doctor`, `check`, every command that reads
`config.yaml`, exited with `metric "<name>": needs: applies to metric plugins only; a parser or
rule plugin reads a different document`, naming the key the release notes had just instructed
the reader to add. The only way back to a working binary was to delete the migration and accept
a plugin that would be handed no timeline.

`metrics:` is now validated by `ValidateMetric`, which is `Validate` with `Needs` cleared — the
one list whose entries may carry the key. The capability *names* are still not checked there:
that vocabulary belongs to `internal/analyze`, and `internal/plugin` owns both halves and checks
them at the boundary.

Two properties of this defect are worth keeping. It was reachable only through the config
**file** — every test that built a `MetricPlugin` in Go skipped `Config.Validate` entirely — and
it was invisible to the whole published-surface guard, because each surface described the key
correctly and consistently. A migration path is a claim about the binary like any other, and
nothing had executed this one.

<a id="team-panel-ranked-named-individuals"></a>

### The team panel ranked named individuals, under a caption saying it was not a scoreboard

*Corrected in v0.25.0, released 2026-09-02.*

`dashboard --no-anonymize` sorted the team panel's members by session count descending and
labelled each bar with the member's **real name**. The caption above it read "Lines and cost are
the team's, never a member's: this panel is not a scoreboard." An ordered list of names with
proportional bars is a scoreboard whatever the caption says.

The refusal it contradicted was not new. `report --by member` has been refused since v0.22 on the
reasoning that "a pseudonym is not anonymous to a colleague who knows the roster" — an argument
that applies to a sorted bar list at least as strongly as to a table — and the dashboard removed
an equivalent figure in v0.14 (`B141`). What changed is that this cycle promoted the refusal from
`PRIVACY.md` to the first screen of three published surfaces, as an unqualified property of the
binary: "nothing ranked per person" (README), "no per-named-individual ranking, in any format"
(`llms.txt`), "*it does not rank individuals* is a very different sentence from *we have promised
not to*" (the site). Publishing that set while the code did the opposite would have made the
sharpest claim on the front page the one least true.

The fix is structural rather than conditional: `buildTeam` no longer takes an `anonymize`
argument, so no value a caller can pass prints a roster of real names. Members are pseudonymous
on every render and ordered alphabetically **by the rendered label** — sorting by the underlying
name and displaying pseudonyms would still hand a colleague who knows the roster each member's
position. The magnitude survives in the bar, where it answers how widely AI use has spread
without answering who is ahead. `--no-anonymize` now reveals project names only, and the one
sanctioned raw-name path is `report --identify`: an unordered export that says on its own face
that it names individuals.

Worth keeping: the claim was correct and the code was wrong, so the caption and the three
published sentences were left exactly as written. The temptation in this shape of defect is to
soften the sentence until it matches the behaviour, which converts a broken promise into a
weaker one nobody notices.

## [0.24.0] - 2026-08-23

<a id="serve-open-dashboard-route"></a>

### The team server's dashboard route now requires the bearer token

*Corrected in v0.24.0, released 2026-08-23.*

`GET /` was open while the write route was guarded, which protected the wrong direction: the
page carries a whole team's usage. An existing bookmark, reverse-proxy rule or read-only monitor
gets a 401 until it presents the token. `/healthz` stays open, and stays exempt from the rate
limit, because it is what an orchestrator polls.

<a id="metric-handshake-version-3"></a>

### A metric plugin gets the step timeline only if its config declares `needs: [trace]`, and the handshake version moves to 3

*Corrected in v0.24.0, released 2026-08-23.*

**A metric plugin gets the step timeline only if its config declares `needs: [trace]`, and the
handshake version moves to 3** (`B168`). The timeline encodes to about 44 MB on a real store
and every plugin received it whether or not it read one. Leaving the version alone would have
had a plugin built against v2 report "no sequences" over a full store — a wrong number with no
error attached — so the handshake now fails loudly and names the version instead. Add
`needs: [trace]` to the `metrics:` entry and emit `"assaio_metric": 3`.

<a id="report-export-carried-the-synced-member-name"></a>

### `report --format json|csv` now emits a member pseudonym, not the synced name

*Corrected in v0.24.0, released 2026-08-23.*

**How this one was disclosed.** Four changes in v0.24.0 broke a working setup on upgrade. Three
fail loudly -- a 401, a refused startup, a rejected handshake. **This one is quiet**, and that is
why it led the release's Breaking list: a pipeline reading `report --format json|csv` keeps
getting a valid document, with different join keys and no error in it. The disclosure goes to
stderr, deliberately, because a line inside the document would corrupt the CSV and the JSON array
every script already parses.

**`report --format json|csv` now emits a member pseudonym, not the synced name** (`B182`). On a
central store the `member` field carried the raw name every sync had pushed, on the default
`--by day` path that never reaches the dimension check — one pivot away from the leaderboard
the `--by member` refusal exists to prevent, and flatly contrary to what the README promised.
Closes `B182`.
Every format now carries a stable `member-xxxx` label instead, and the export says which
identity it holds. Scripts that joined on a raw name need `--identify`, which names
individuals deliberately. A purely local store is unaffected: it has no member to label.

<a id="vendor-stats-comparison-claim"></a>

### README and the site say what the built-in stats cannot do, instead of a claim that stopped being true

*Corrected in v0.24.0, released 2026-08-23.*

**README and the site say what the built-in stats cannot do, instead of a claim that stopped
being true** (`B188`, `B189`). The opening said vendor dashboards count "spend, never output"; Claude Code
analytics has reported accepted lines and cost per commit, Copilot has reported code
generation and pull-request metrics, and Claude Code now ships `/insights` — a local 30-day
report — and `/usage` for plan consumption. The comparison is now specific and measured: a
bounded window against a whole store (22 of one machine's 52 days are older than the source's
own retention), a capped sample against every session, an LLM summary against a computed
figure, one vendor against five, a snapshot against a delta, and nothing that restates a past
report against a store where a parser fix reaches history. It also names the thing `/usage`
does better — plan and rate-limit consumption, which no local transcript carries, so assaio
does not claim it.

<a id="v1-freeze-policy-contradiction"></a>

### `docs/compatibility.md`: one answer to what v1.0 freezes

*Corrected in v0.24.0, released 2026-08-23.*

The roadmap said the SQLite schema would deliberately not be frozen in one section and listed
freezing it in another, the release guide promised a stable schema and SDK, and the extension
docs promised an in-process Go API "arriving toward v1.0". Every existing consistency check
passed the whole time, because each file was internally consistent. The policy now lives in one
published page the others link to: the exec protocols, the observation and signal contracts, the
recommendation record, the sync protocol and the machine-readable outputs freeze; the SQLite
schema stays an implementation detail with migration, export and backup guarantees; the
in-process Go API is deferred and is not a v1 contract. Two tests hold it — one rejects a
retired promise reappearing in any published file, one fails if a file stops linking to the
policy.

<a id="backfill-downward-restatement"></a>

### `backfill` reports how many stored rows a re-read moved *down*

*Corrected in v0.24.0, released 2026-08-23.*

Assigned columns exist so a corrected attribution rule can reach history, which leaves the store
unable to tell a fix landing from a parser regression erasing evidence. The count is printed
with that ambiguity stated. On its first run against the maintainer's store it found six rows
and a real defect behind them, now tracked as `B186`: a sub-agent transcript and its parent both
carry the same assistant message id and see different amounts of it, so the stored tool-call
count oscillates.

<a id="unsourced-verdicts-withdrawn"></a>

### Fourteen metrics stopped publishing a verdict they could not source

*Corrected in v0.24.0, released 2026-08-23.*

**Fourteen metrics stopped publishing a verdict they could not source** (`B176`, `B177`). `friction`,
`rework`, `turn-efficiency`, `context`, `explore-produce`, `cache-hygiene`, `model-fit`,
`model-right-sizing`, `concentration`, `skill-economics`, `recovery`, `rhythm`,
`reasoning-share` and `throughput` each decided good-versus-watch on a number picked once — a 15% error rate, a 20%
compaction rate, a 1.5× recovery multiple — with no published definition behind it and no
derivation from the window's own distribution. Rendered in colour, a number like that is read
afterwards as knowledge. No figure changed *because of this withdrawal* and every one is still printed (two moved for
their own reasons, both in `Fixed` below: `model-fit`'s denominator and `recovery`'s baseline);
what is gone is the colour, the promotion into "worth a week's attention", and a gauge that
moved -- it is pinned at half, carrying no verdict either way. Each now carries a caveat naming the authority it would need. `burn-anomaly` and `edit-loops`
are what a sourced line looks like here — both derive theirs from the window's own median and
MAD at the conventional 3.5 cutoff — and `adoption`, `coverage` and `subscription-fit` keep
verdicts resting on a definition, assaio's own confidence model, and your configured plan
price; `intent` keeps a favourable read about whether labels exist at all, which is a fact
about coverage rather than about the work. On the maintainer's corpus, 5 of the 21 reads on a 90-day window still carry a verdict.
`B185` tracks the line that would earn them back: your own earlier windows.

<a id="rhythm-off-hours-verdict"></a>

### `rhythm` no longer judges when the work happened, and prints the band it counts against

*Corrected in v0.24.0, released 2026-08-23.*

**`rhythm` no longer judges when the work happened, and prints the band it counts against**
(`B178`).
On a local store every session is one person's, so "off-hours: 31% [WATCH]" was a workload
judgement about an individual, promotable to the top of the dashboard — close enough to the
refusals that it should never have been a verdict. The shares are still reported, and the
8:00–18:00 weekday band they are measured against is now on the surface instead of in the
source, so a reader can disagree with it.

<a id="server-read-body-before-token"></a>

### The team server read the whole request body before checking the token

*Corrected in v0.24.0, released 2026-08-23.*

`handleUsage` confirmed only that an `Authorization` header existed, decoded up to 128 MiB of
JSON, and verified the secret afterwards -- so an unauthenticated caller could make the server
allocate a whole batch and then be told 401. The secret is verified before a byte of the body is
read.

<a id="rate-limiter-header-bypass"></a>

### The rate limiter could be bypassed by varying the header, and the bypass grew the map it walked

*Corrected in v0.24.0, released 2026-08-23.*

**The rate limiter could be bypassed by varying the header, and the bypass grew the map it
walked.** Keying on the presented string handed a caller a fresh budget per request and added
an entry per distinct header under the same mutex. Anything that is not a *known* secret now
shares one anonymous bucket, and `/healthz` is exempt so unrelated traffic cannot fail a
liveness probe.

<a id="digest-verdict-changes-after-upgrade"></a>

### `digest` would have reported fourteen verdict changes as findings after upgrading

*Corrected in v0.24.0, released 2026-08-23.*

The
stored snapshot records each metric's read key, and the comparability check compares the
*ingesting* build, which does not move when the analyzing binary does -- so the first digest
after this release would have listed every withdrawn verdict as something that moved in the
reader's work. `SnapshotVersion` is 2, which makes the first run after an upgrade report a
first run.

<a id="model-fit-two-denominators"></a>

### `model-fit`'s two figures and its sentence used different denominators

*Corrected in v0.24.0, released 2026-08-23.*

The shares divided by the window's whole token count while the takeaway divided by the tokens
that carry a tier, so a window with any unpriced usage read "60% premium" above a sentence
saying 80%. Both use the tiered tokens now, and the unpriced figure says it sits outside that
denominator rather than claiming to be excluded from a split it was inside.

<a id="lines-per-million-trivial-base"></a>

### A lines-per-million-tokens rate off a trivial token base

*Corrected in v0.24.0, released 2026-08-23.*

The maintainer's own dashboard printed 5,298,857 lines/1M tokens beside a real 6,386 -- an 830x
contrast off a 3,500-token base, under a line suggesting the cheaper model is the bargain. A
tier under 100,000 tokens now renders an em dash, the same refusal every other rate here already
makes.

<a id="session-step-ts-pinned"></a>

### `session_step.ts` stayed pinned while `usage_record.ts` became correctable

*Corrected in v0.24.0, released 2026-08-23.*

Both come from the same source line, so a timestamp fix would have moved a usage row into the
day it belongs to and left its steps in the old one -- and `PruneSteps` reads that column, so
the horizon would then have cut the wrong steps. `subpath` follows `project` for the same
reason: one `projectid.Resolve` sets both.

<a id="recovery-baseline-contained-the-aftermath"></a>

### `recovery` compared the aftermath of a failure against a baseline containing it

*Corrected in v0.24.0, released 2026-08-23.*

**`recovery` compared the aftermath of a failure against a baseline containing it** (`B175`). The
denominator counted every assistant turn in the window, including the turns following a
failure, so the ratio was compressed toward 1.0 — toward `CONTAINED`, a bias in the direction
that flatters, by exactly the share of the window that follows a failure. The baseline now
excludes the aftermath, and a window with no turns outside one reports no ratio rather than
comparing the aftermath with itself. Measured on the maintainer's corpus: 1.02× → 1.03×, where
the aftermath is 9,633 of 90,172 turns; the bias grows with that share. The per-step and
three-step figures quoted in the code and in `explain recovery` were re-measured against the
corrected baseline (1.06×, 1.37×, 1.04×).

<a id="throughput-volume-floor-one-side"></a>

### A collapse in AI-line output read as "too few lines to call"

*Corrected in v0.24.0, released 2026-08-23.*

The volume floor guarding the week-over-week trend applied to the recent side alone, so output
falling from 201,347 lines to zero was reported as unreadable rather than as the direction it
is. It now applies to the larger side.

<a id="identity-columns-uncorrectable"></a>

### A parser fix could not reach `ts`, `project`, `entrypoint` or `git_branch`

*Corrected in v0.24.0, released 2026-08-23.*

Those columns were stamped by the first read and correctable by nothing, so a fix to any of them
-- including the one that decides which day a record counts toward -- could not reach a single
stored row, and no canary looked at them (`B116`). A re-read of the same file now corrects all
four, on the rule that already decided `granularity`. The three text columns overwrite only when
the re-read has an answer, because clearing a correctly captured name is not correcting it.

<a id="toolchain-floor-reachable-advisories"></a>

### The supported build toolchain is Go 1.26.6 or newer, and `make vuln` is clean

*Corrected in v0.24.0, released 2026-08-23.*

The documented floor was 1.25, on which `govulncheck` reports six reachable standard-library
advisories; `go.mod` now pins the toolchain and the indirect `golang.org/x/text` is on v0.39.0,
which clears the last (unreachable) module advisory. The scan reports no vulnerabilities at all.

<a id="metric-plugin-raw-member-names"></a>

### A metric plugin was handed raw member names

*Corrected in v0.24.0, released 2026-08-23.*

The usage rows, session rows and step timelines sent to an out-of-tree subprocess carried the
name each member synced under. They now carry the same pseudonym every other surface uses, so a
plugin can still group by member without being told who that is. Regression tests search the
whole rendered payload for the name rather than checking a field, so a field added later cannot
reopen the hole.

## [0.23.1] - 2026-08-17

<a id="share-caveat-overflowed-the-footer"></a>

### `share`: the profile frame printed its own caveat over the card's footer

*Corrected in v0.23.1, released 2026-08-17.*

The block declared a fixed allowance for a line that wraps, so on a 1:1 reel the axis disclaimer
ran past the body band and landed on `assaio.dev`, the hashtag and the limit line — the three
things that exist to qualify everything above them. The scene now sizes itself to the space it
has, the way the poster already did, so an oversized stack shrinks instead of overflowing.

<a id="share-post-formatting"></a>

### `share`: the suggested post is formatted for where it gets pasted

*Corrected in v0.23.1, released 2026-08-17.*

It used padded columns and indented commands, which line up in a terminal and in nothing else;
composers that strip blank lines on paste then collapsed it into one wall of text. Every line
now stands on its own, the two commands are one unindented line, and **assaio.dev** is named in
the post rather than only inside the install command — a reader who wants to know what produced
the numbers before running anything needs somewhere to go that is not a shell.

## [0.22.0] - 2026-08-15

*Round two of the correctness lockdown: a whole-codebase review, then a four-reviewer pass over
the fixes, which overturned three of them. What that second pass changed is recorded in the
entries below rather than quietly corrected, because a fix written from a wrong cause is the
failure this release exists to stop. This note describes the release rather than one wrong
figure, which is why it carries no anchor of its own.*

<a id="metric-protocol-2-layer"></a>

### A metric-plugin result must declare its `layer`, and the protocol is now `2`

*Corrected in v0.22.0, released 2026-08-15.*

The four measurement layers — activity, output, outcome, impact — are a promise `ROADMAP.md` has
carried since the first commit ("a metric states which one it is") and nothing in the code said
which. Built-in validators are now forced to declare one by the `Validator` interface, which is
a compile error rather than a field one of them would forget; the exec metric protocol (ADR
0004) gets the equivalent at its boundary, rejecting a result whose `layer` is missing or
outside the vocabulary exactly as it already rejects an unknown `read.key`. An extension surface
weaker than the core it extends is the failure `B155` named. Every earlier addition to the
protocol was additive, so the version stayed at `1` through v0.17 and v0.21; a newly *required*
output field has to be a version a plugin can branch on, or its only signal is a contract
violation naming a field it has never heard of. Emit `{"assaio_metric": 2, …}` and add `"layer":
"activity"` (or `output`) to your result document — see [ADR
0013](adr/0013-measurement-layers.md) and the [metric-plugin guide](extending/metric-plugin.md).

<a id="report-by-member-refused"></a>

### `report --by member` and `effectiveness --by member` are refused, not caveated

*Corrected in v0.22.0, released 2026-08-15.*

Grouped by
member, `report` printed each person's tokens and spend and `effectiveness` printed their AI
lines, edits, rejections and cost-per-100-lines — under a caveat calling itself "a diagnostic
per project", a dimension the table was not grouped by, and with no caveat at all in `json` and
`csv`. That is the per-named-individual ranking `BACKLOG.md`'s Refusals rule out. The dashboard
removed the same figure in v0.14 (`B141`) and kept the reasoning; this is that decision reaching
the other two surfaces. The error says why, and is deliberately not phrased as an unknown
dimension: a refusal is a different answer from a typo. **What this does not do**, stated
because the first draft of this entry overclaimed it: `report --format csv|json` still carries a
`member` column on a central store, since the default `--by day` never reaches the grouping
check. Nothing is ranked, but the raw export is unchanged — see `B182`.

<a id="seven-day-window-divided-by-eight"></a>

### A "seven-day" window was divided by eight days, and reconciled against a whole vendor day it only half covered

*Corrected in v0.22.0, released 2026-08-15.*

**A "seven-day" window was divided by eight days, and reconciled against a whole vendor day it
only half covered.** `analyze.MonthlyRate` counted calendar day-buckets inclusive from a window
that opens at the current time of day, so a 168-hour window got a divisor of 8 — understating
`subscription-fit` by 12.5% at 7 days and 3.2% at the default 30 — while `reconcile` compared
that window's partial first day against the export's whole one and reported the difference as
`Unexplained`, the one number the command exists to produce. `B128` returning at two new call
sites. Both are fixed where they occur: the projection divides by the window's own length, and
the reconciliation's scope starts at the first date the window covers from midnight.
**A first attempt fixed it at the source instead, by making `--since Nd` mean N whole
day-buckets, and review found that costs more than it saves**: `sync --since 1d` on a daily cron
then covers `[today 00:00, now]`, so the hours between the previous run and midnight reach the
team server from no run at all; a `check --since 1d` pre-push hook near UTC midnight evaluates a
budget over minutes; and `0d` starts in the future. A window stays a duration, which is what a
recurring command needs, and the helpers that compare day-buckets keep aligning on their own —
which is what `compareWindow` and `recentCutoff` already did.

<a id="withheld-verdict-drew-an-empty-gauge"></a>

### A withheld verdict drew an empty gauge

*Corrected in v0.22.0, released 2026-08-15.*

`Result.noData` set the neutral read and the neutral takeaway and never touched `Purity`, which
stayed `0.0` and rendered as a bar at zero — a bad result, beside a verdict that was declining
to give one. Every `noData` call site was affected; a handful of others set the constant by
hand, and one more path in `intent` set a computed `0` and was found only by the test written
for this. Guaranteed on every served team dashboard, which never loads sequences, so
`edit-loops` and `recovery` always took that path. `B136` closed this in v0.14 for one
validator; it is now closed for the registry, by a test that runs every validator over two
empty-shaped windows.

<a id="rework-flagged-an-unmeasurable-window"></a>

### `rework` flagged a window it could not measure, and could render above 100%

*Corrected in v0.22.0, released 2026-08-15.*

Two defects in one verdict. A window whose start cuts between an addition and its removal holds
removals of lines it never counted as added — `ChurnStat`'s own doc has always said so and
nothing clamped or disclosed it, so `--since 1d` over a day that removed 400 of yesterday's
lines printed `rework: 8000%`. And the pair's verdict read `WATCH` whenever either half was
unmeasurable, which for a Codex store — it records an undone line and never a declined call —
meant "worth a closer look" on every window forever, behind a full gauge, with the ordering
promoting it. The rate is now withheld with its reason on both surfaces that print it, and the
verdict gives the three answers its rates support: a measured rate above its ceiling is a
finding and always shows, everything measured and low is a clean bill, and anything else is
withheld with the caveat naming the silent half. A pair cannot be certified low from one of its
halves, and it cannot be flagged from a silence either.

<a id="humanize-usd-unit-before-rounding"></a>

### `humanize.USD` chose its unit before rounding, could never reach "M", and rendered a real cost as `0.00`

*Corrected in v0.22.0, released 2026-08-15.*

**`humanize.USD` chose its unit before rounding, could never reach "M", and rendered a real cost
as `0.00`.** `USD(1,000,000)` rendered `1000.0K` and `USD(33,500,000)` rendered `33500.0K`, on
`subscription-fit` and the status line. `Count` and `Bytes` were moved onto `scaleTo` when
`B127` closed this in v0.13; `USD` was not, and had no such test. All four renderers now share
one rounding precision and one class test that walks every tier boundary of each, and `USD`
joins `USDCompact` in refusing to print a real amount below half a cent as nothing.

<a id="gemini-output-token-overflow"></a>

### Gemini CLI's output token total could overflow to zero, leaving reasoning above it

*Corrected in v0.22.0, released 2026-08-15.*

Three int64 fields summed with plain `+` where every other parser uses `parser.SumNonNeg`:
`{"output":MaxInt64,"thoughts":1}` stored `OutputTokens=0, ReasoningTokens=1`, violating
`usage.CheckCounts` — and the team server rejects a whole push on the first invalid record, so
one such row blocked a member's entire `sync`. **The checked-in fuzz seed already produced it**
and `make fuzz` passed, because the fuzzer asserted only non-negativity. `reasoning ⊆ output` is
now asserted in all five parser fuzzers — vacuously in Claude's, which reads no reasoning today,
which is exactly when a gap opens unnoticed — and clamped where two independent fields are read:
Copilot CLI and Codex had the same gap with no overflow. Gemini's cached tokens are clamped to
its input the way Codex already did, which stopped it storing more prompt tokens than the
vendor's own total.

<a id="billing-export-silently-truncated"></a>

### A billing export over 64 MiB was silently truncated and reconciled as complete

*Corrected in v0.22.0, released 2026-08-15.*

The `io.LimitReader(f, max+1)` overflow idiom was there and nothing ever compared against it:
`csv.Reader` saw the cut as a clean end of file, `Skipped` stayed 0, no limit was recorded, and
every row past the cut was reported as `Unexplained`. It is now an error naming the bound.

<a id="skipped-canary-lines-per-record"></a>

### The `skipped` drift canary divided lines by records and called the result a line share

*Corrected in v0.22.0, released 2026-08-15.*

`Records` counts emitted records — one per API response, several lines each — while `Skipped`
mixes unmarshal failures, undated records and steps refused at the vocabulary boundary. 2,000
lines yielding 500 records and 60 failures reported `60 of 560 line(s) skipped (10.7%)` against
a true 3%, clearing the 10% threshold on arithmetic alone. It is now a rate per file, in the
unit its numerator is counted in — this repo's own "a row count is not a size" rule, broken
inside the one component whose job is catching silent under-reporting.

<a id="drill-panel-lost-history-start"></a>

### The dashboard's project panel claimed the store's history could not be read

*Corrected in v0.22.0, released 2026-08-15.*

`buildDrill` copied `WindowStart`, `Ingested`, `ParsedBy` and `Trace` onto the scoped input and
not `HistoryStart`, so `adoption` and `throughput` inside the panel carried "how far back this
store's history goes could not be read" — on the page whose top-level ledger had just printed
it. One line. The panel surfaces caveats only as a `Prov.` badge, so the wrong sentence never
reached the rendered page; the verdict carrying it did.

<a id="subpath-table-one-row-per-member"></a>

### The dashboard's subpath table showed one row per member, with no member column

*Corrected in v0.22.0, released 2026-08-15.*

`Subpaths` grouped by `(subpath, member)` while the panel it feeds has only a subpath column, so
on a team store one subpath appeared once per person, each row holding that person's lines,
ordered by lines descending — a reader takes the top row for the subpath's total, and it is an
unlabelled per-person output ranking under a heading that says repository. One row per subpath
now.

<a id="metrics-verify-zero-plan-price"></a>

### `metrics verify` ran a plugin with a plan price of zero

*Corrected in v0.22.0, released 2026-08-15.*

Four of the five commands that build an `analyze.Input` set `PlanMonthlyCost` at their own call
site and `metrics verify` did not, so a plugin gating on `planMonthlyCost` — the pattern
`docs/recipes/extensions.md` teaches — verified `VALID` against a zero and behaved differently
under `analyze`. The shared builder now owns it: a field every caller has to remember is a field
one of them will forget.

<a id="edit-loops-bars-reordered"></a>

### `edit-loops` project bars reordered between identical runs

*Corrected in v0.22.0, released 2026-08-15.*

Sorted on rate alone over a slice built from map iteration, and `2/10` and `3/15` both give
`0.2` with a minimum of ten edits, so ties are ordinary — and the top-five cut then changed
between two runs over the same data. The name settles them, as it already does in
`copilot.dominantModel`, `dashboard.TopProject` and `cache.missCauseFigure`.

<a id="subscription-fit-projection-span"></a>

### `subscription-fit` described a projection method its divisor does not use

*Corrected in v0.22.0, released 2026-08-15.*

The caveat said the figure spans the window, and the divisor starts at the first day the window
carries usage — so a 30-day window whose usage began five days ago projected from five days, six
times what the sentence promised, and drove the pay-off verdict a reader keeps or cancels a plan
on. The divisor is right and the sentence was wrong; it now names the span, and a `projected
from` figure prints it, because a reader cannot otherwise tell a 30-day projection from a 5-day
one.

<a id="clear-labels-deleted-silently"></a>

### `clear --labels` with no other scope deleted every label with nothing said first

*Corrected in v0.22.0, released 2026-08-15.*

The pre-flight line named the record count and the label count was printed only afterwards,
while unscoped `--labels` runs a bare `DELETE FROM session_label`. Labels are the one thing in
the store no re-import can rebuild, because a person typed them; the count is now stated before
the deletion, and says when the scope is "all of them".

<a id="backfill-freed-bytes-silently"></a>

### `backfill` could free 70 MB and never mention `compact`

*Corrected in v0.22.0, released 2026-08-15.*

Tightening the trace horizon from 30 days to 7 on a copy of the maintainer's store moved 70.3 MB
onto the freelist and left the 170.1 MB file exactly the same size. `clear` has always said so;
`backfill` printed `steps-pruned=` and stopped.

<a id="structural-silence-read-as-zero-lines"></a>

### A structural silence no longer reads as an AI that produced nothing

*Corrected in v0.22.0, released 2026-08-15.*

ADR 0011 was applied throughout `internal/analyze` and to none of the surfaces that sum
`LinesAdded` directly. Gemini CLI and Cline answer no line signal, so their users read `0 AI
lines` on `status`, `+0 lines` on the status line every day forever, `AI lines total: 0` under
`throughput`'s newly-added *output* label, `0` in the digest mailed to an inbox, and `0 AI
LINES` beside a real cost in `effectiveness --by tool` — the last of which invites dropping the
tool that "produces nothing". All five are gated now: the figure is withheld where nothing
records it, and where only some sources do, the surface says what share of the window's tokens
the figure reaches. `status` also gained the `$`/100-lines denominator disclosure that
`effectiveness` and the dashboard colophon already carried and it did not — the ratio's two
sides count different populations.

<a id="barren-canary"></a>

### A new `barren` drift canary: files found, and no run on record has read a usage record out of them

*Corrected in v0.22.0, released 2026-08-15.*

**A new `barren` drift canary: files found, and no run on record has read a usage record out of
them.** It is the only canary judged on a condition rather than a rate, and deliberately carries
no sample floor. Every other one is a comparison, and a comparison cannot see this: a source
that has *always* yielded zero has a baseline of zero. **An A/B with all four sample floors set
to 1 fired nothing on either build** — which is why `B110`'s stated cause was wrong and a fix
written from it would have shipped green and changed nothing. It reads the whole history rather
than one run, because `Parsed` counts the files a run *attempted*: an ordinary incremental pass
whose single changed input yields nothing is not a barren source, and the first cut of this
canary would have failed `doctor --strict` on one. On the maintainer's machine `gemini-cli`
matches two files, both parse, neither carries a `tokens` key, and five surfaces read as
"gemini-cli is being counted" while none said otherwise; `doctor --strict` now fails on it, and
the warning says "nothing read from a detected source" rather than naming format drift as the
cause, which the evidence cannot support.

<a id="step-columns-kept-not-assigned"></a>

### A step's stored token total and outcome can be corrected downward

*Corrected in v0.22.0, released 2026-08-15.*

`session_step.tokens` took `MAX(stored, offered)` and `outcome` took a fill-only `CASE`, which
are the rules `insert_local.go` reserves for a vendor's own figure. Neither is one: a step's
total is assaio's sum of four chosen fields with reasoning deliberately excluded, and its
outcome is assaio's mapping of a stop reason and its attribution of a result to a call — a rule
whose own source comment records having been wrong once, putting 42 of 497 real denials on a
different call. Both are assigned now, so a corrected rule reaches every stored row. This is
`B116`'s second half, reopened in v0.20 in a brand-new table because nothing looked. A test now
reads the shipped SQL and fails when any assaio-derived column is kept rather than assigned — in
any of the four spellings SQLite accepts, not just the one that shipped.

<a id="migration-immutability-unenforced"></a>

### Migration immutability is enforced, in name and content

*Corrected in v0.22.0, released 2026-08-15.*

All twelve shipped migrations were verified byte-identical to the tag that first shipped them,
and `grep` found no test, no Make target, no CI step and no hook checking it — they held because
nobody had touched them. A digest per file now fails on an edit, a rename or a removal. The
rename half is the worse one and the easier to reach for while tidying: the runner keys on the
filename, so a rename re-executes a body it has already applied, and for `0008` that body moves
every `claude-code` row into an archive table and deletes the originals. `IF NOT EXISTS` guards
the DDL and nothing guards the DML. `RELEASING.md` now says "immutable in name and content".

<a id="trace-horizon-cost-undisclosed"></a>

### `doctor` states the timeline's own horizon, and says plainly when there is none

*Corrected in v0.22.0, released 2026-08-15.*

`trace.horizon_days: 0` is honoured on purpose — coercing it to 30 would delete history someone
asked to keep — but nothing in `doctor`, `backfill` or config validation said what it costs: a
measured 3.40 MB/day with no prune and no upper bound. A negative value, which `Validate`
rejects while a lenient load carries on and prunes at the default, is now reported as what it is
rather than as no horizon. The remedy names the horizon and nothing else: `clear --older-than`
deletes usage records first and steps second, so suggesting it would trade one table's growth
for another table's permanent history — the history the retention line two rows below declares
unrecoverable. `trace.horizon_days` also joins `config.example.yaml`, where it was the one
top-level key missing from the file `README.md` calls "a documented starting point", and a test
now fails when any key is absent from it.

<a id="step-table-size-bound-not-age-matched"></a>

### The step table's size bound is age-matched

*Corrected in v0.22.0, released 2026-08-15.*

`store.PruneSteps` carried a row multiplier and no bytes, which is the reading ADR 0012 says
"was wrong", so it now leads with 102.0 MB of table and indexes against `usage_record`'s 58.3
MB. **The multiplier beside it was also corrected in the opposite direction from the first
attempt**: `session_step` is pruned to 30 days and `usage_record` is not, so dividing their
totals compares a bounded table against an unbounded one and answers 1.88, where the figure over
the 30 days both tables cover is **2.19**. The un-age-matched comparison is the one this
project's honesty rules forbid for bug density, and it was about to be shipped as the
correction. ADR 0012 carries the correction; migration `0012`'s comment keeps the old figure,
because a shipped migration is immutable.

<a id="privacy-md-named-three-of-five-sources"></a>

### `PRIVACY.md` names all five parsed sources and the two settings files v0.21 began reading

*Corrected in v0.22.0, released 2026-08-15.*

GitHub Copilot CLI has been a source since v0.6 and appeared nowhere in the document a reader
uses to decide whether this is safe on a work machine. `claude.ReadRetention` reads
`managed-settings.json` and `~/.claude/settings.json` for one key each, stores neither, and said
so nowhere. The bookkeeping-table list also named two of the three that exist, and Cline's roots
named one editor of the four that are scanned.

<a id="readme-commands-and-counts"></a>

### `README.md`'s "Every command" includes `survival`

*Corrected in v0.22.0, released 2026-08-15.*

**`README.md`'s "Every command" includes `survival`**, shipped in v0.2, in `FEATURES.md` and on
the site; its validator count reads twenty-one rather than nineteen; the "what comes next" line
no longer promises two things documented as shipped further down the same file; the `backfill`
example carries the Copilot row and the step counts; and the roadmap tool list matches
`ROADMAP.md` rather than naming Cursor, whose own backlog entry records that local storage has
no token counts to parse.

<a id="site-contradicted-itself"></a>

### `site/index.html` no longer contradicts itself on the live page

*Corrected in v0.22.0, released 2026-08-15.*

`# the nineteen directional reads` sat 250 lines below a verified "Twenty-one", outside the
claim span the test reads, and the faceplate mock showed 19 cells under a heading saying
twenty-one. Both fixed, plus two `report --by` lists still offering the refused dimension, the
drift section's claim that every canary compares against a baseline, a "no per-person analytics
outside an opt-in" caveat for a refusal that now holds absolutely, and the order-of-work
dimension v0.21 added.

<a id="validator-examples-stopped-compiling"></a>

### Every published in-tree validator example compiles against the interface again

*Corrected in v0.22.0, released 2026-08-15.*

Adding `Layer()` broke all four, and the gate that exists to catch exactly that —
`recipes_go_test.go` — was green, because it hand-copied the interface into a `want` map. It now
reads the method set out of `internal/analyze/analyze.go`, so the copy cannot drift from the
source again.

<a id="guides-described-what-does-not-ship"></a>

### `docs/format-resilience.md`, `docs/automation.md` and `docs/extending/data-source.md` describe what ships

*Corrected in v0.22.0, released 2026-08-15.*

**`docs/format-resilience.md`, `docs/automation.md` and `docs/extending/data-source.md` describe
what ships.** The drift guide listed four canaries and a threshold that no longer exists and
said every canary has a sample floor; the automation guide said token counts take the maximum,
which is now true of a vendor's figure and false of a derived one; and the parser guide's
mandatory fuzz invariants omitted the subset rule whose absence is the Gemini overflow above.

<a id="query-your-data-described-a-different-store"></a>

### `docs/extending/query-your-data.md` describes the store that exists

*Corrected in v0.22.0, released 2026-08-15.*

It opened with "one table holds your data" while `session_step` is the larger of the two,
omitted eleven shipped columns, said "two bookkeeping tables" where there are three, sent
readers to the generated reference for column documentation it does not contain, and said
Copilot stores `0` lines when it has populated them since v0.6.

## [0.21.0] - 2026-08-13

<a id="target-ref-restated-with-max"></a>

### A re-read no longer keeps the higher of two parsers' target numbers

*Corrected in v0.21.0, released 2026-08-13.*

`target_ref` was restated with `MAX()`, which is harmless while numbering never changes and
wrong the moment it does: widening which calls register a target renumbers first-seen order, and
one sequence would have held a `7` beside a `3` standing for the same file with nothing able to
detect it. It now follows the current parse, exactly as `ordinal` already did for the same
reason.

## [0.19.0] - 2026-08-12

<a id="recipe-execution-claim-uncounted"></a>

### How each recipe is checked is classified in code, and the pages print the counts

*Corrected in v0.19.0, released 2026-08-12.*

Ten are executed, eleven have their command lines checked, two are loaded by the configuration
loader and two are shape-checked — a claim that was "every recipe is executed by the test suite"
until it was counted. A recipe missing from the classification, or a count that drifts from it,
fails the build.

<a id="ci-recipe-passed-at-any-spend"></a>

### A published CI recipe would have passed at any spend, forever

*Corrected in v0.19.0, released 2026-08-12.*

Its "fetch the team window" step set two environment variables and fetched nothing — and no
command could have: `sync` is push-only. A runner's store is empty, `check` finds no usage, and
the gate exits 0. The recipe now carries the store in as an artifact and says plainly that this
is the part every put-it-in-CI recipe gets wrong. The same block downloaded a release asset
filename goreleaser never produces, so the job would have died before reaching the gate.

<a id="cookbook-test-could-not-fail"></a>

### The test behind the cookbook's flagship claim could not fail

*Corrected in v0.19.0, released 2026-08-12.*

Its fixture gave the second tool zero lines and zero edits, so deleting the `answers` gate the
recipe exists to teach changed nothing: 30.0 either way, and the "would give 15.0" in the prose
was arithmetically impossible. The fixture now uses Copilot CLI's real depth — changed lines, no
edit count — so the gated figure is 30.0 and the ungated one 50.0, and removing the gate from
the document fails the build.

<a id="invocation-check-abandoned-a-line"></a>

### The invocation check abandoned any line whose command it did not recognize

*Corrected in v0.19.0, released 2026-08-12.*

**The invocation check abandoned any line whose command it did not recognize**, so a renamed
command took its flags out of scope with it and the whole line passed. It now reports the
command too, reading code rather than prose so English following the binary's name is not
mistaken for an invocation.

<a id="sa5011-in-ci-only"></a>

### A test tripped `SA5011` in CI and not locally

*Corrected in v0.19.0, released 2026-08-12.*

**A test tripped `SA5011` in CI and not locally**, on the same pinned linter version and the
same Go minor: staticcheck read the dereference after `t.Fatal` as a possible nil. An explicit
`return` satisfies both. The cause of the disagreement was not established, and the line worth
keeping is the one it proves — a green `make lint` on this machine is evidence about this
machine, not about the branch.

## [0.18.0] - 2026-08-11

<a id="site-never-named-v0170-commands"></a>

### `digest` and `mark --suggest` shipped in v0.17.0 and the website said neither

*Corrected in v0.18.0, released 2026-08-11.*

The Commands section listed 23 of 25 subcommands; `digest` appeared once, inside the roadmap
section, and the string `--suggest` appeared nowhere under `site/`. Both are on the page now,
and the gate above is what makes the third occurrence of this impossible rather than merely
unlikely. `site/llms.txt` did not contain the word `digest` either.

<a id="metric-input-fields-undocumented"></a>

### Six contract fields were absent from the documentation a metric author reads

*Corrected in v0.18.0, released 2026-08-11.*

The extension docs never described `WindowStart`, `CacheMisses`, `Ingested` or `ParsedBy` on the
in-tree `analyze.Input` — the other half of the gap `B155` closed on the wire — nor, on the
return path, that `lead` and `confidence.recorded` are decoded and then discarded because they
are answers about the window rather than about one metric. All three contracts now have a canary
that fails when a field is named nowhere in a code span or fenced block, so the English word
"skills" no longer counts as evidence that the `skills` field is documented.

<a id="site-guards-read-one-page"></a>

### `site.yml`'s guards cover every served page

*Corrected in v0.18.0, released 2026-08-11.*

**`site.yml`'s guards cover every served page**, not `index.html` alone — each page judged on
its own findings, so a pasted embed on one does not annotate the next — and the `noversion`
guard masks dotted quads before looking for a version, because `reference.html` publishes
`serve --addr`'s default of `127.0.0.1:8787` and an IPv4 literal contains a three-component
prefix. Masking what is structurally not a version is the same move the guard already made for
SVG geometry attributes.

<a id="site-deploy-resolved-wrangler-at-build-time"></a>

### The site deploy no longer resolves its own tooling at build time

*Corrected in v0.18.0, released 2026-08-11.*

`npx wrangler deploy` took whatever npm called `latest` when the build ran, and on 2026-08-11
that was a wrangler published 88 seconds earlier whose `miniflare` dependency was not resolvable
yet: `ETARGET`, three deploys skipped, the site serving an older commit while every other check
stayed green. `package.json` and its lockfile pin the tool — the version is now a line in a diff
rather than a race, and Cloudflare caches the dependency instead of reinstalling it each time.
Nothing in that file builds the page; `site/` is still copied, not compiled.

<a id="canonical-pointed-at-a-redirect"></a>

### Every internal link and canonical names a path the host actually serves

*Corrected in v0.18.0, released 2026-08-11.*

Workers Assets serves `site/reference.html` at `/reference` and `307`s the extension away, so
the first cut of the reference page pointed its own canonical at a redirect — correct in review
and in a browser opened on the file, wrong against the deployed site. A new `servedpaths` job
checks it, because that is a fact about the artifact and nothing was reading it.

## [0.17.0] - 2026-08-11

<a id="plugin-input-missing-six-fields"></a>

### The extension surface stops being weaker than the core it extends

*Corrected in v0.17.0, released 2026-08-11.*

**The extension surface stops being weaker than the core it extends (`B155`).** Six fields of
the prepared `Input` never crossed the metric-plugin boundary — `windowStart`,
`planMonthlyCost`, `skills`, `agents`, `turnSizing`, `cacheMisses` — while five shipped
validators read them (`cache-hygiene`, `model-right-sizing`, `monthrate`, `subscription-fit`,
`skill-economics`). An out-of-tree author could therefore not reproduce roughly a third of what
assaio itself publishes, which made ADR 0004's claim that a plugin receives "the same prepared
`Input` bundle every built-in validator reads" untrue in exactly six places. All six are on the
wire now, additively, so the envelope stays `assaio_metric_input: 1` and a plugin written
against the earlier shape keeps working. A reflective canary fails the build when a new `Input`
field reaches neither the envelope nor a listed exception with the reason a plugin cannot need
it; the only standing exceptions are `recent` (sent as `recentDays`) and `ingested`/`parsedBy`,
which the core stamps onto the plugin's own `Result`.

<a id="clear-did-not-name-the-store"></a>

### `clear` names the store it is about to empty

*Corrected in v0.17.0, released 2026-08-11.*

It has no `--db` and always acts on the default local store, so somebody who believes they are
clearing a copy gets exactly one chance to notice — and the deletion is not reversible. It now
prints the path and the record count before deleting anything, even under `--yes`. Found the
hard way: a tooling run set an environment variable that does not exist, ran `clear --all --yes`
expecting it to apply to a copy, and emptied a real 513,617-row store instead.

<a id="clear-dropped-the-digest-basis"></a>

### `clear` drops the digest's comparison basis with the records it described

*Corrected in v0.17.0, released 2026-08-11.*

Deleting rows is not a change in how the tools were used, and nothing downstream could tell the
difference: the next digest would have reported a pruned store as "tokens −62%, this model
gone", with none of its caveats applying. The snapshot table now goes with the usage rows, and
the next digest says it has no basis.

<a id="mark-accept-suggested-ignored-its-target"></a>

### `mark <id> --accept-suggested` no longer ignores the session it was given

*Corrected in v0.17.0, released 2026-08-11.*

The accept path returned before the target was resolved, so naming a session — or passing
`--last`, an axis flag, or `--unmark` — silently wrote derived labels across the entire
`--since` window. Labels are the one thing no re-import can rebuild. Every one of those
combinations is now a hard error naming what to run instead.

<a id="derived-labels-crossed-the-member-boundary"></a>

### Derived labels respect the member boundary

*Corrected in v0.17.0, released 2026-08-11.*

Signals were keyed on `session_id` alone, so on a store holding more than one machine's rows a
branch recorded by one member could derive the label written onto another's session — the leak
every other label path avoids by correlating on `(session_id, member)`.

<a id="digest-reported-what-did-not-move"></a>

### The digest stops reporting things that did not move

*Corrected in v0.17.0, released 2026-08-11.*

A verdict change now carries the confidence it rests on, so a read that moved on `insufficient`
data is marked as the noise it is instead of reading like a real change; a metric that did not
run (a failed metric plugin) is named as missing rather than rendered as a verdict that
vanished; and a move into or out of `neutral` is stated as the metric gaining or losing the data
it reads, not as an improvement.

<a id="digest-cost-had-no-error-bar"></a>

### The digest's cost figure carries the same error bar as every other cost surface

*Corrected in v0.17.0, released 2026-08-11.*

It stored no notion of what the price table could not cost, so a week where a new model went
unpriced rendered as a fall in spend. It now quotes the disclosure `report` prints, keyed on the
share of tokens missing a price rather than on the mere presence of a price-less row — and the
first run, which also prints a cost, no longer omits the caveats entirely.

<a id="digest-project-names-unpseudonymized"></a>

### Project names in a digest are pseudonymized by default

*Corrected in v0.17.0, released 2026-08-11.*

**Project names in a digest are pseudonymized by default**, like every other file assaio
writes to be shared (`privacy.anonymize`, default true). The pseudonym is applied when the
digest renders, never to the stored comparison key, so toggling the setting cannot make every
project read as one that appeared beside one that vanished.

## [0.16.0] - 2026-08-11

<a id="unpriced-share-was-one-asterisk"></a>

### A cost figure now says how much of itself is missing

*Corrected in v0.16.0, released 2026-08-11.*

**A cost figure now says how much of itself is missing (`B139`).** Every `$` assaio prints is
a token count times a vendored price table, and the `*` that disclosed a model the table does
not carry read identically at 0.1% and at the 45.5% that once left a window's estimate
$15,452.42 short for five weeks. The marker now carries the share, the token counts behind it,
and the set it was computed over — `report`, `effectiveness`, `report --compare`, `status`,
`check`, the statusline and the dashboard colophon all render the same sentence. `doctor`
gains an `unpriced:` line about *your* window rather than about the table, naming the models a
refresh has to cover, and `doctor --strict` fails above `pricing.max_unpriced_share` (default
5% of the reported window's tokens; `0` turns that half of the gate off). The ceiling is
measured, not chosen: on the maintainer's own corpus a newly adopted model reaches 12.0% of a
7-day window on its second day and 10.0% of a 30-day window on its third. A new calibration
test fails when a trace in this repository names a model the vendored table cannot cost, and
refreshing the table is now a named step in `RELEASING.md` rather than a chore with no owner.

<a id="insufficient-blamed-the-source"></a>

### `insufficient` stops blaming a source for history it does record

*Corrected in v0.16.0, released 2026-08-11.*

**`insufficient` stops blaming a source for history it does record (`B115`).** A verdict
resting on nothing now distinguishes a fourth cause: rows read by a build that could not
capture the field yet, which `backfill` repairs, from a subject no source records at all,
which nothing does. Only the first names a cure, and it is claimed only where the subject's
own denominator exists among rows that could have carried it — so a window whose tool calls
came from a source that records no failure is never sent after a re-import that cannot change
the figure.

<a id="counts-formatted-two-ways"></a>

### A count reads the same on every surface

*Corrected in v0.16.0, released 2026-08-11.*

**A count reads the same on every surface (`B146`).** Validator figures group thousands
through the shared formatter, so the dashboard no longer prints `16400 tokens` beside a report
table printing `16,400`. Judged per figure, not swept: counts that are small by construction —
active days, projects, task classes, a threshold inside a note — stay bare, and token counts
keep the compact `33.4B` form the dashboard already used.

## [0.15.0] - 2026-08-10

<a id="money-rendered-as-0-0000"></a>

### Money reads like money

*Corrected in v0.15.0, released 2026-08-10.*

Every cost cell in `report`, `effectiveness`, and `movers` was four fixed decimals, which is
unreadable at the top of the range (`107640.1234`) and a lie at the bottom (`0.0000` for a real
cost). Costs now group thousands and round to cents from a dollar up, keep two significant
digits below one (`0.0032`), and state a bound (`<0.0001`) rather than round a real amount into
zero. Token and line counts in the same tables group thousands, and numeric headers and totals
align with their digits. `--format json` and `--format csv` are untouched: both still carry full
precision.

<a id="dashboard-gauges-misread"></a>

### The Assay dashboard is easier to read

*Corrected in v0.15.0, released 2026-08-10.*

The verdict faceplate no longer leaves a coloured phantom tile when the number of reads does not
divide by the column count; each gauge takes its own verdict's colour, so a `WATCH` bar can no
longer look like a healthy one; the gauge ratio is captioned instead of floating as a bare
decimal. A ledger entry now lays its figures out on a grid rather than a ragged row, runs its
ranked bars the full width, and closes with the takeaway set as a banded line in the verdict's
colour. The per-entry confidence line is colour-coded only when the basis is weak, replacing a
`HIGH` chip that repeated on all nineteen reads.

## [0.14.0] - 2026-08-09

<a id="stored-activity-counts-rebuilt"></a>

### Your stored activity counts change, in both directions, and the upgrade rebuilds them for you

*Corrected in v0.14.0, released 2026-08-09.*

**Your stored activity counts change, in both directions, and the upgrade rebuilds them for
you.** This release corrects four things about *which* lines and events a Claude Code
transcript contributes at all, and the restate path could not have applied any of them: it
took `MAX(stored, offered)` on every column, so a fix that *reduces* a figure could never
reach an existing row — the record already exists, nothing new is inserted, and a maximum
never accepts a smaller number. The activity counts (lines, rework, edits, tool calls, the
purpose split, rejections, errors, compactions) are now assigned from the current parse; the
token counts stay on `MAX`, because those are the vendor's own figures rather than a rule
assaio wrote. Migration `0010` clears the `claude-code` ingest watermark so the next
`backfill` re-reads every transcript once — **the first import after upgrading is a slow
one**, and nothing is deleted by it.

On the maintainer's corpus of 5,586 real transcripts the net effect is:

| figure | before | after | why |
|---|---|---|---|
| lines added | 383,579 | **1,010,406** | a created file finally counts its body |
| rework lines | 32,550 | **46,948** | those lines are now a budget a removal can undo |
| compactions | 38 | **19** | one overflow was counted once per marker, and it writes two |
| added / removed | −460 / −656 | applied | a repeated transcript line went in twice |

Most stores will see AI lines rise sharply and compactions halve. Both are corrections, not
new activity.

<a id="team-member-row-showed-lines-and-cost"></a>

### The team dashboard's per-member row now shows sessions only

*Corrected in v0.14.0, released 2026-08-09.*

**The team dashboard's per-member row now shows sessions only** (`B141`). It also printed
each pseudonymous member's AI lines and spend, and pseudonymous is not anonymous to a
colleague who knows the roster — that is a productivity comparison however the list is
sorted. Lines and cost are now the team's total, shown once above the list. A self-hosted
team server that has been reading the old rows will see them disappear on upgrade; the
figures were never removable later without an argument, which is why it is done now.

<a id="per-month-was-thirty-active-days"></a>

### A projected "per month" figure is now a calendar month, not thirty *active* days

*Corrected in v0.14.0, released 2026-08-09.*

**A projected "per month" figure is now a calendar month, not thirty *active* days**
(`B142`). `subscription-fit` and `model-fit`'s savings estimate both divided the window's
cost by the days that carried usage and multiplied by 30, so somebody who codes five days a
week had a working week priced as a calendar month — and the error ran in the direction that
flatters the plan. The denominator is now the window's span in real days, clamped to the
first day carrying usage so a new user is not divided by a ninety-day window they were not
around for. **On the maintainer's own store** the 30-day API-equivalent moves $107.6K → 
$100.6K/mo (552× → 516× the plan) and the model-fit upper bound $84,856 → $79,382/mo. Every
existing user's headline moves down; nothing about their usage changed.

<a id="parser-plugin-unknown-field-stored-a-zero"></a>

### A parser plugin's record line is now rejected for a field the protocol does not define

*Corrected in v0.14.0, released 2026-08-09.*

**A parser plugin's record line is now rejected for a field the protocol does not define**
(`B143`), which the metric and rule protocols already did. A plugin writing `outputTokens`
where the protocol says `output_tokens` stored a zero and was counted as a valid record — a
wrong number arriving quietly rather than a protocol error arriving loudly. Emit exactly the
documented fields; a rejected line is skipped and counted like any other boundary failure.

<a id="every-compaction-counted-twice"></a>

### Every context compaction was counted twice

*Corrected in v0.14.0, released 2026-08-09.*

Claude Code writes one overflow as two adjacent lines — a `system` boundary and the user-side
summary that replaces the context — and the counter fired on each. **Measured on the
maintainer's corpus:** all 19 real compactions arrive as that pair and none arrives alone, so
the stored figure was 38. The parser now folds a run of markers into one event. The test that
covered this asserted the wrong number, which is the golden-file trap the calibration suite
exists to close.

<a id="cache-write-signal-overclaimed"></a>

### `codex` and `gemini-cli` claimed a cache-write signal neither parser can produce

*Corrected in v0.14.0, released 2026-08-09.*

Both were handed the shared cost bundle, which includes `ai.tokens.cache_write`; neither log
publishes a cache-write counter and neither parser ever sets the field, so `signals coverage`
promised a figure that could only ever be zero — a silence reported as a measurement, which is
what the depth matrix exists to prevent. Cache write is now declared per source, as reasoning
and the cache tiers already were, and both rows name the gap.

<a id="created-files-counted-zero-lines"></a>

### Every file Claude Code created counted as zero added lines

*Corrected in v0.14.0, released 2026-08-09.*

A creation arrives as the new file's whole body in `content` beside an *empty*
`structuredPatch`, and the counter walked only the patch — so a created file unmarshaled cleanly
and contributed nothing. This is the same defect Codex shipped as `B119`, in the flagship
parser, and it was found by the first calibration trace rather than by review. **Measured on
5,586 real transcripts:** added lines **383,579 → 1,010,406** — the corpus had been reporting
38% of what the tool wrote — and rework **32,550 → 46,948**, because a file born with no counted
additions gave a later removal nothing to undo. Migration `0010` rebuilds it on the next
`backfill`.

<a id="subagent-result-content-union"></a>

### A completed sub-agent's record depended on a field it does not own

*Corrected in v0.14.0, released 2026-08-09.*

Reading the created file's body meant declaring `toolUseResult.content`, which a sub-agent
result writes as an array rather than a string; typing it from one arm's evidence failed the
whole unmarshal for the other and dropped 477 sub-agent records with 21,369 of their lines.
Caught by the real-corpus A/B before it shipped, and the field is now read as raw JSON per arm.

<a id="repeated-transcript-line-counted-twice"></a>

### A repeated transcript line was counted twice, and every count that mattered lives on the repeated kind

*Corrected in v0.14.0, released 2026-08-09.*

**A repeated transcript line was counted twice, and every count that mattered lives on the
repeated kind.** Claude Code writes a streamed retry as the same line again, and the guard
against that sat inside the assistant branch of the parser — so assistant lines were
protected and user lines were not. An edit result's added, removed and rework lines, a tool
denial, a failed tool result and a compaction boundary all went in twice. **Measured on
5,597 real transcripts, not inferred:** 329 repeated edit results carrying **460** added and
**656** removed lines, plus 5 repeated denials — so the corpus reported 382,738 added lines
where the true figure is 382,278, and 32,767 rework lines against a true 32,387. The guard
now runs ahead of every line kind. A line carrying no uuid is still folded in, because there
is nothing to recognise it by.

<a id="stale-subagent-aggregate-survived"></a>

### A stored sub-agent aggregate outlived the transcript that superseded it

*Corrected in v0.14.0, released 2026-08-09.*

The parent transcript summarizes a completed sub-agent as one row; the sub-agent's own file
holds the same work per turn. Suppression at parse time only keeps a *new* aggregate out — a row
written before that file existed, or by a build that could not discover it, stayed beside the
detailed turns and double-counted its tokens and cost. `backfill` now drops the superseded ones.
This also removes the reason `Delegation` had to pick one definition of sub-agent work over the
other on a mixed store.

<a id="team-server-could-not-correct-a-partial-figure"></a>

### The team server could never correct a partial figure

*Corrected in v0.14.0, released 2026-08-09.*

A sync that runs mid-response pushes an output count that has not reached its true total yet,
and pushes went through first-write-wins `Insert`, whose repair never touches a token count — so
the undercount was permanent on the one surface a whole team reads. The endpoint prefixes every
dedupe key with its member and the member charset excludes the colon, so each row has exactly
one possible writer: restating one is that member correcting their own number, never overwriting
somebody else's.

<a id="quotepath-dropped-files-from-survival"></a>

### Non-ASCII filenames left the survival rate silently

*Corrected in v0.14.0, released 2026-08-09.*

git's `core.quotePath` is on by default, so `café.go` arrives as `"caf\303\251.go"`; that string
names no file, so `git blame` rejected it and its lines simply vanished from the rate, while
`path.Ext` read `.go"` and filed it under "other" instead of source. (`B124`)

<a id="failed-blame-counted-as-not-surviving"></a>

### A failed blame was reported as code that did not survive

*Corrected in v0.14.0, released 2026-08-09.*

Every non-context error in the survival walk hit a bare `continue` with no counter, so the rate
was printed as a confident percentage over an unknown fraction of the window's files. `survival`
now names how many files it could not read and keeps them outside the rate. (`B125`)

<a id="worktree-projects-named-dotdot"></a>

### Every worktree session in every repository resolved to a project named `..`

*Corrected in v0.14.0, released 2026-08-09.*

**Every worktree session in every repository resolved to a project named `..`** on git 2.48
and later, which writes the worktree pointer *relative* to the worktree. Read verbatim it
matched no `/.git/worktrees/` segment. Alongside it, a worktree checked out beside its main
repository produced `Subpath = "../tmp/feature"` — not a repository subpath, and host
directory names in a field PRIVACY.md promises holds none. A path that climbs out of the
root is now no subpath at all. (`B126`)

<a id="recent-window-covered-eight-buckets"></a>

### The "seven-day" recent window covered eight day-buckets

*Corrected in v0.14.0, released 2026-08-09.*

Subtracting the whole duration and then truncating to a date made the boundary date recent too,
so Hot / GoingStale / DormantTools and `adoption` compared eight days against six and read the
difference as a trend. (`B128`)

<a id="nameless-usage-counted-as-a-project"></a>

### Usage that names no project was counted as a project

*Corrected in v0.14.0, released 2026-08-09.*

A source that logs no working directory (Gemini CLI, Cline) leaves it empty on every row, and
pooling those under one nameless key made a single-repo user who also runs Gemini CLI read as
working across two — which is exactly what `adoption`'s breadth signal is computed from. The
spend still counts; only the project name is unknown. (`B129`)

<a id="label-dimension-csv-had-no-columns"></a>

### `report --by task|outcome|difficulty --format csv` emitted rows nothing could tell apart

*Corrected in v0.14.0, released 2026-08-09.*

**`report --by task|outcome|difficulty --format csv` emitted rows nothing could tell
apart.** Aggregating on a label dimension stamps the key into `Task`/`Outcome`/`Difficulty`
and leaves every other identity column empty, and the CSV header had none of the three. The
table and JSON forms were always correct. (`B130`)

<a id="plugin-record-with-no-timestamp-bound"></a>

### A metric plugin could inject a record with no timestamp bound

*Corrected in v0.14.0, released 2026-08-09.*

**A metric plugin could inject a record with no timestamp bound**, so a year-9999 row sat
inside every `--since` window forever. The range and magnitude rules the sync endpoint
applies are now one shared check both boundaries use, and it also holds
`reasoning_tokens <= output_tokens` — without which a record renders a reasoning share above
100%. (`B133`)

<a id="init-db-wrote-one-store-and-read-another"></a>

### `init --db` imported to one store and reported from another

*Corrected in v0.14.0, released 2026-08-09.*

**`init --db` imported to one store and reported from another**, so the command printed an
empty first run against the database it had just told the user about. `init` imports through
`backfill`, which only ever writes this machine's own store, so the flag is gone rather than
threaded through. (`B134`)

<a id="skill-economics-two-dimensions"></a>

### `skill-economics` totalled one dimension and ranked another

*Corrected in v0.14.0, released 2026-08-09.*

The "largest single share"
is taken within whichever of skills / sub-agents has two entries to compare, while the
"attributed tokens" figure above it — labelled "in the dimension below" — was read off the
*larger* one. An 80% share sat beside a total 80% was never taken from. (`B135`)

<a id="rework-full-gauge-beside-a-withheld-verdict"></a>

### `rework` drew a full gauge beside a withheld verdict

*Corrected in v0.14.0, released 2026-08-09.*

With neither churn nor rejection measurable, both structural silences entered the purity average
as zeros and the faceplate rendered 1.00 — the strongest possible "all clear" — next to a `—`. A
gauge with nothing behind it now sits at neutral. (`B136`)

<a id="codex-diff-lost-a-removed-line"></a>

### A Codex diff could lose a removed line to a comment marker

*Corrected in v0.14.0, released 2026-08-09.*

The unified-diff file headers were matched anywhere in the diff rather than at the position the
grammar puts them, so a removed line of SQL, Lua, Haskell or Ada — whose comments begin `-- ` —
was read as a file header and not counted. Unobserved on the audited corpus (0 of 349 real
diffs): this is the rule, not a correction to a figure.

<a id="codex-cached-input-exceeded-its-total"></a>

### Codex could store more prompt tokens than its own total gained

*Corrected in v0.14.0, released 2026-08-09.*

`input_tokens` is the whole prompt and `cached_input_tokens` the part of it served from cache,
but the two deltas were taken independently, so a turn whose cached counter advanced further
than its input counter stored the entire cached delta beside a clamped zero. The two classes now
add back up to the input delta. Unobserved on the audited corpus (0 of 1,686 events).

<a id="undated-record-stored-and-invisible"></a>

### A record with no timestamp was stored and then invisible

*Corrected in v0.14.0, released 2026-08-09.*

Every report, validator and dashboard window is bounded by `ts >= ?`, so such a row counted
toward the store's totals and appeared in no window. It is now counted as skipped — the honest
word for evidence that could not be read, and the number the drift canaries already watch.

<a id="late-model-name-never-applied"></a>

### A model name that arrived late could never be applied

*Corrected in v0.14.0, released 2026-08-09.*

Cline reads the model from a sidecar that may not exist when a task is first parsed, and the
restate path deliberately never touched an identity column — so those tokens stayed unpriceable
forever. A blank model is now filled by a later read; a stored name is still never overwritten.

## [0.13.0] - 2026-08-08

<a id="stored-numbers-rebuilt-v0130"></a>

### Your stored numbers change, and the upgrade rebuilds them for you

*Corrected in v0.13.0, released 2026-08-08.*

Migration `0009` clears the `claude-code` and `codex` ingest watermarks, so the next `backfill`
re-reads every transcript once — no flag, and nothing is deleted. Codex's added lines go up (a
created file finally counts), and rework goes down for both sources (a removal can no longer
claim the same added line twice). On the maintainer's corpus that is Codex 6,632 → 10,604 added
lines and Claude Code 37,885 → 33,752 rework lines. The first import after upgrading is
therefore a slow one.

<a id="codex-created-files-lost-every-line"></a>

### Codex dropped every line of every file it created, and about a third of its added lines with them

*Corrected in v0.13.0, released 2026-08-08.*

**Codex dropped every line of every file it created, and about a third of its added lines
with them.** A Codex `patch_apply_end` describes each changed file as a type-discriminated
union: an `update` carries a `unified_diff`, but a *creation* carries the file's whole body
as `{"type":"add","content":"…"}`. The counter read only the diff, so a created file
unmarshaled cleanly and contributed `(0, 0)`. **Measured on 61 real rollouts, not inferred:**
54 created files carried **3,972** added lines that were never counted, so the corpus
reported **6,632** added lines where the true figure is **10,604** — a 37% undercount, and
every file born that way also entered rework tracking with zero additions to be undone. The
parser now reads the discriminator: an `add` contributes its body's lines as added, a
`delete` its own as removed (unobserved on the audited corpus, but a union arm nobody counts
is exactly what this defect was). The `add` entry in the golden fixture was a hand-written
shape Codex does not emit — a diff on a creation — and has been replaced with a captured
one. (`B119`)

<a id="rework-cap-was-a-budget-nobody-spent"></a>

### The rework cap was a budget nobody spent, so churn could exceed the additions it undoes

*Corrected in v0.13.0, released 2026-08-08.*

`parser.Rework` clamped each removal at a file's *total* recorded additions rather than at the
additions not yet undone, so two removals could both claim the same lines: 3 added followed by
two 10-line deletions produced 6 rework lines on 3 added — a rate above 100%, which the
function's own doc comment said was impossible. The budget is now consumed as it is claimed and
refilled by later additions. **On the maintainer's corpus:** Claude Code rework **37,885 →
33,752** lines (a 13.3% rate becomes 11.9%), Codex **961 → 913**. (`B132`)

<a id="corrected-rework-rule-could-not-reach-a-row"></a>

### A corrected rework rule could not reach a single stored row

*Corrected in v0.13.0, released 2026-08-08.*

`restateActivitySQL` takes `MAX(stored, offered)` on every activity column, which repairs a
session ingested while it was still being written — and pins any figure a later build corrects
*downward*, forever. `rework_lines` is now assigned rather than maximised, joining `granularity`
as a value the current parse is the authority on: it is derived from the whole transcript by a
rule, not read from the log, and the rule is monotone in the prefix read, so a half-written
session still restates upward exactly as before. Migration `0009` clears the `claude-code` and
`codex` ingest watermarks so a plain `backfill` re-reads and rebuilds — no flag, nothing
deleted, one slower import. **Verified end to end:** a store carrying the old figures was
rebuilt by one `backfill` to exactly the totals a fresh store produces from the same logs.

<a id="clear-left-a-store-backfill-could-not-refill"></a>

### `clear` left a store that no `backfill` could refill

*Corrected in v0.13.0, released 2026-08-08.*

`Clear` emptied `usage_record` but never `ingest_file`, so every input still matched on
size/mtime/version and the next import reported `unchanged` and inserted nothing — while
`clear`'s own help implied usage records are re-importable. A clear that is not time-scoped now
drops the watermarks of the inputs it unread, so `backfill` rebuilds; `--older-than`
deliberately keeps them, because pruning history is a request to forget records rather than
re-read them, and the command now says which of the two just happened. (`B121`)

<a id="clear-labels-ignored-its-scope"></a>

### `clear --tool codex --labels` deleted every other tool's session labels

*Corrected in v0.13.0, released 2026-08-08.*

`DeleteLabels` was an unscoped `DELETE FROM session_label` that read none of the scope flags —
destroying the one thing in the store no re-import can rebuild, because a person typed it. It
now follows the same scope as the deletion beside it, and only ever takes the label of a session
the clear removes *entirely*: a session with records on both sides of a time cutoff survives, so
its annotation does too, and a label that cannot be tied to the scope at all is never guessed
at. Unscoped `--all --labels` still deletes everything. (`B122`)

<a id="doctor-strict-exited-zero-on-a-broken-store"></a>

### `doctor --strict` exited 0 on a store it could not open

*Corrected in v0.13.0, released 2026-08-08.*

Both store-failure paths printed `ERROR` and returned nil, short-circuiting before the strict
check — so a cron job with a corrupt database *and* a mistyped `sources:` path reported green,
and the drift canaries the whole `--strict` promise rests on never ran at all. A store failure
is now a strict failure like any other, the report continues to its caveats, and the drift line
says the canaries were not evaluated instead of "no canary fired". (`B123`)

<a id="half-the-tokens-had-no-price"></a>

### Nearly half the flagship setup's tokens had no price, so every `$` figure was a little over half the truth

*Corrected in v0.13.0, released 2026-08-08.*

**Nearly half the flagship setup's tokens had no price, so every `$` figure was a little
over half the truth.** The vendored LiteLLM snapshot dated 2026-07-11 and its newest Opus was
`claude-opus-4-8`; the model in heaviest real use, `claude-opus-5`, had no row, and usage on
an unpriced model is excluded from cost rather than estimated. **Measured on the maintainer's
store: 22.7B of 49.8B tokens — 45.5% — resolved to no price**, and the window's estimate rose
from `$23,750.98` to `$39,203.40` (+65%) once it did. The snapshot is refreshed to 2026-08-08
(2,949 → 2,988 models); four existing entries changed price and one was removed, none of them
a model this corpus uses. Not a matching bug — `NormalizeModel` resolved `claude-sonnet-5` and
`claude-fable-5` correctly all along — which is exactly why nothing caught it: a stale table
looks identical to a complete one from inside. Refreshing it is still a release chore with no
test behind it (`B139`).

<a id="check-max-cost-passed-on-an-unpriced-window"></a>

### `check --max-cost` reported OK on a window it could not price

*Corrected in v0.13.0, released 2026-08-08.*

Cost excludes usage on a
model the price table has no row for — the `*` in the output says so — but the gate compared
the budget against that partial figure anyway, so on the store above `--max-cost 100000`
passed against a figure covering a little over half the window. A cost budget over unpriced
*tokens* now fails, for the reason `check`'s own help already gives about rules — a gate that
could not be evaluated is not a gate that passed — and the cost line reads `UNPRICED` rather
than `OK`, so the printed verdict and the exit code cannot disagree. The token axis is
untouched: tokens are physical, and a missing price says nothing about them.

<a id="store-path-uri-escaping"></a>

### A `#` in the store's path opened a different database, and a `?` silently dropped the pragmas

*Corrected in v0.13.0, released 2026-08-08.*

**A `#` in the store's path opened a different database, and a `?` silently dropped the
pragmas.** The DSN was `"file:" + path` with nothing escaped, and everything after `file:` is
parsed as a URI: `#` starts a fragment and `?` starts the query, so either truncated the
filename while `Open` still returned success — the second taking WAL, `busy_timeout` and
`foreign_keys` with it. Reachable from any `XDG_DATA_HOME` or home directory containing one.
The three characters that change a URI's meaning (`%`, `?`, `#`) are now escaped; nothing
else is touched, so a Windows path or one containing spaces resolves exactly as before.
(`B120`)

<a id="compact-units-chosen-before-rounding"></a>

### Compact units were chosen before rounding, so a value printed its own ceiling

*Corrected in v0.13.0, released 2026-08-08.*

`humanize.Count(999,999,999)` rendered `1000.0M` instead of `1.0B`, `Count(999,950)` rendered
`1000.0K`, and `Bytes(1,048,575)` rendered `1024.0 KB`. Real cache-read totals sit in exactly
that band. The unit is now picked from the value that will actually be printed. A sibling of the
same class: an exact 0.5% printed `0%` — the precise rounding `humanize.Percent` exists to
refuse — because the small-share guard compared with `<` where the upper edge uses `>=`.
(`B127`)

<a id="dashboard-rendered-a-real-cost-as-zero"></a>

### The dashboard rendered a real cost as `$0`

*Corrected in v0.13.0, released 2026-08-08.*

The footnote's per-active-day figure dropped to whole dollars below $1,000, so $12 across 30
active days printed "**$0** per active day" — the fabricated zero `costDisplay`'s own doc
forbids. Cost rendering moved onto a single `humanize.USDCompact`, which keeps cents below a
dollar and says `<$0.01` rather than round a real amount to nothing; the dashboard's own copy of
the formatter is gone. (`B131`, part of `B75`)

## [0.12.0] - 2026-08-07

<a id="response-counted-once-per-content-block"></a>

### One Claude Code response was counted once per content block, and every token figure for the flagship source was roughly double

*Corrected in v0.12.0, released 2026-08-07.*

**One Claude Code response was counted once per content block, and every token figure for
the flagship source was roughly double.** Claude writes an API response as *one JSONL line
per content block*, and each of those lines repeats that response's `usage` verbatim — the
same `input_tokens`, the same `cache_creation_input_tokens`, the same
`cache_read_input_tokens` — while `output_tokens` stays partial until the last line. The
parser keyed a record on the line's `uuid`, so a three-block answer billed one request three
times. **Measured on 5,724 real transcripts, not inferred:** 354,904 assistant lines are
159,175 actual responses, and a full re-ingest moves the totals
input **64.7M → 22.3M**, output **282.7M → 143.4M**, cache-read **94.6B → 46.5B**,
cache-write **2.83B → 1.01B**, and the estimated cost **$53,208 → $24,339**. Records fall
353,847 → 158,776. A record is now keyed on the response id and carries that response's
tokens once, while the activity of each block — tool calls, edits, the purpose split — is
summed across the group, because that half was never duplicated. Sub-agent aggregates are
keyed by `agentId` and were never affected. A transcript predating the `message.id` field
still parses, one record per line, exactly as before.
**Your stored history is rebuilt for you, and nothing is destroyed to do it:** migration
`0008` moves the `claude-code` rows the old grain inflated into a
`usage_record_pre_response_grain` archive that no report reads, clears their `ingest_file`
watermarks, and lets the next `backfill` re-read every transcript unconditionally — no flag
needed. Two details are deliberate. Clearing the watermarks is what makes that promise true
rather than likely: `buildIdentity()` returns a constant `dev` for any source build, so a
version bump alone would have left a `go install`ed binary skipping every transcript as
unchanged and the history empty. And the rows are archived rather than deleted because
**the rebuild only reaches as far back as the transcripts you still have** — Claude Code
rotates its own logs (30 days by default) while the store is the durable record, and this
migration runs from `store.Open`, which the statusline calls, so a delete would have fired
on the first prompt after the binary was replaced. `assaio-agent doctor` reports the
archive's size and the statement that drops it; `assaio-agent compact` returns the pages
afterwards. A member's synced rows are archived too, for the same reason the local ones are:
nothing will ever collide with their old keys, so leaving them in place would have a team
server reporting the inflated total *plus* the rebuilt one. Members restore their share by
syncing again; widen `sync --since` if the window you care about is older than 30 days.

<a id="cache-write-priced-at-the-cheap-tier"></a>

### A cache write is now priced by the lifetime it actually bought

*Corrected in v0.12.0, released 2026-08-07.*

Claude splits every cache write into `cache_creation.ephemeral_5m_input_tokens` and
`ephemeral_1h_input_tokens`, and the vendor bills the 1-hour tier at 1.6× the 5-minute rate — a
distinction assaio read past, pricing every write at the cheap rate. It is not a rounding error:
**59.7%** of the audited corpus's cache-write tokens are 1-hour writes (84% on `claude-opus-5`,
68% on `claude-opus-4-8`, 0% on `claude-sonnet-5`), and pricing them correctly raises the
cache-write component by **35.8%** (**+$1,765** on that corpus). The rate was already in the
vendored price table as `cache_creation_input_token_cost_above_1hr`; nothing read it. A model
publishing a single cache-write rate bills both tiers at it, never zero, and a source that
reports no tier is priced exactly as before. New signal `ai.tokens.cache_write_1h`.

## [0.11.0] - 2026-08-06

<a id="rejection-rate-rounded-to-zero"></a>

### A real signal no longer rounds away to "0%"

*Corrected in v0.11.0, released 2026-08-06.*

`rework` printed its rejection rate at whole-number precision, so 102 recorded human refusals of
65,098 calls read as **`0%`** — while `friction`, one screen down, printed **`0.2%`** for the
same 102 refusals. One measurement, two answers, and the more prominent one said a signal that
exists is absent. Both now render through a single share formatter that refuses the two
dishonest roundings at any precision: a small but nonzero share reads `<1%` (or `<0.1%`), and a
share just under whole reads `>99%` rather than hiding a real remainder behind `100%`. The
formatter is `internal/humanize`'s, so `analyze`, `status` and the dashboard cannot drift apart
on it. The `status` session line is the one this mattered most on: it rounded with an integer
`+0.5` and could print *"100% produced code, 0% conversational"* for a 99.6% share — the
sentence shape [ADR 0011](adr/0011-capability-gated-metrics.md) exists to prevent.

<a id="rework-averaged-a-sources-silence"></a>

### `rework` averaged a source's silence into the churn rate

*Corrected in v0.11.0, released 2026-08-06.*

The rate summed added and reworked lines across *every* source, but `ai.rework.lines` is
answered by Claude Code and Codex only — a source recording changed lines and no undone one
(Copilot CLI) put its whole output in the denominator against a structural zero, lowering the
rate with code nobody watched being undone, and the verdict read `LOW` for it. It was the one
validator ADR 0011 did not reach. The gate now lives inside `report.BuildChurn`, so `analyze`
and `status` read one number; the figure prints `—` and the verdict is withheld where no source
records an undone line; and the reach is declared as `signalCoverage`. **Measured, not
assumed:** the audited store holds only Claude Code and Codex, both of which record rework, so
no shipped figure was wrong here — this closes the hole before a Copilot- or Cline-heavy window
meets it.

<a id="friction-and-explore-denominators"></a>

### Two more denominators counted work their source never recorded

*Corrected in v0.11.0, released 2026-08-06.*

`friction`'s "of N tool calls" and `explore-produce`'s coverage share both summed `ToolCalls`
across every source, including ones that name no tool call at all — so a source that records
none would have read as a gap in a capture that was never attempted. `explore-produce`'s own
caveat already said this ("a source that names no calls records none, so it neither raises nor
lowers this"); the arithmetic now agrees with it. Both were found by the widened invariant test
below, not by hand.

<a id="subagent-run-counted-as-one-turn"></a>

### A whole sub-agent run stopped counting as one turn

*Corrected in v0.11.0, released 2026-08-06.*

Migration `0006` relabelled stored Claude sub-agent aggregates `session`, and `TurnSizing` and
the granularity-coverage figure respected the label — but `store.Sessions`, which every
per-session turn figure reads, still counted every row with `COUNT(*)`. On the audited store 65
of 779 sessions carried 1,015 such rows, one of them inflated by **89 phantom turns**;
`turn-efficiency`'s median turns per code session moved 724 → 718 and the `status` p90 647 →
635. Turn count, peak context and the gaps behind focused minutes now read only `turn`-grain
rows; whole-session figures (timestamps, output tokens, edits, compactions) still read every
row, because those are honest at either grain. Peak context was checked before the claim: it was
**not** inflated in practice, since a sub-agent aggregate carries its last request's usage
rather than a run's sum.

<a id="context-median-read-an-unwritten-edit-count"></a>

### `context`'s code-session median read an edit count Cline never writes

*Corrected in v0.11.0, released 2026-08-06.*

The
"active work — code sessions: ~N min" contrast picked its subset with `Edits > 0` across
*every* session rather than the ones whose source records an edit, so a Cline or Gemini
window would have drawn that figure from sessions nobody counted an edit in. It now reads
the sessions answering both signals it needs — the edit count that selects the subset and
the focused minutes it takes the median of.

<a id="rhythm-confidence-contradicted-its-figures"></a>

### `rhythm`'s confidence line contradicted its own figures

*Corrected in v0.11.0, released 2026-08-06.*

With no source recording focused minutes it declared zero signal coverage, printing
*"insufficient — nothing in this window can answer it"* directly beneath an off-hours share
computed from 100% of the window. The reach is now declared only while there is a length half to
narrow; the withheld verdict and the takeaway carry that half's absence instead.

<a id="field-audit-overstated-its-own-consequence"></a>

### The field audit's Codex cache-write row overstated its own consequence

*Corrected in v0.11.0, released 2026-08-06.*

It said Codex cost is a floor because the cache-write count is unread. The count is unread, but
on the audited corpus it carried a value on 238 events and was **zero on every one**, so no
figure is currently wrong. The row and `B107` now say what was measured rather than what would
follow if it were non-zero — the same standard the audit applies to a vendor's fields applies to
its own claims.

<a id="adr0011-test-had-a-blind-spot"></a>

### The ADR 0011 invariant test now varies both row shapes and both shapes of silent source

*Corrected in v0.11.0, released 2026-08-06.*

It flipped only `SessionRow` fields on a single source, which is how two of the holes above
survived a release that was specifically about capability: a validator reading a `UsageRow`
column was invisible to it, and so was one reading an edit count, because its one silent source
records lines and masked it. Which fields to fill is now read from the depth matrix rather than
listed per tool, so a parser landing tomorrow is covered without touching the test. It caught
three live cases across its first two runs. `LinesAdded` is deliberately still out: whether a
cross-source line *rate* may keep its denominator is an open decision (`B118`), and a test
should not freeze it either way.

<a id="plugin-envelope-answers-field"></a>

### The exec metric envelope (`assaio_metric_input: 1`) gains an `answers` field

*Corrected in v0.11.0, released 2026-08-06.*

**The exec metric envelope (`assaio_metric_input: 1`) gains an `answers` field**, mapping each
tool in the window to the signal ids it can produce. Every activity count on the wire is zero
for a source that does not record it, and until now a plugin had no way to tell that from a
measurement — so every out-of-tree metric was structurally exposed to the bug ADR 0011 fixed
in-tree. This is an added field within version 1, not a reshape: a released plugin that
ignores it keeps working unchanged. [`docs/extending.md`](extending/metric-plugin.md#answers--which-zeros-are-measurements-and-which-are-silence)
documents the rule and a worked example.

## [0.10.0] - 2026-08-05

<a id="silence-read-as-a-zero"></a>

### A metric no longer reads a source's silence as a zero

*Corrected in v0.10.0, released 2026-08-05.*

Four validators and the `status` Sessions block read per-session fields — edits, turns, focused
minutes, compaction — that three of the five in-tree sources never record, and counted every one
of those sessions as a zero. A Gemini CLI, Cline or Copilot CLI window therefore read as **100%
conversational, 0% produced code, 0 marathons, 0% compaction** and carried a verdict on all
four; on a Claude window the two Copilot sessions did the same in miniature. Each figure is now
computed over the sessions whose source can answer it, declares that reach as its signal
coverage, prints `—` where nothing in the window records it, and withholds the verdict rather
than certifying a silence. Affects `session-taxonomy`, `context`, `rhythm` and
`turn-efficiency`; `status` names the narrower basis on its own line.

<a id="insufficient-had-one-sentence-for-three-causes"></a>

### `insufficient` now says which of the three ways a verdict rests on nothing

*Corrected in v0.10.0, released 2026-08-05.*

One sentence
covered all of them, so `explore-produce`, `friction` and `skill-economics` printed
"nothing to measure in this window" over a store holding 119,896 records — directly beneath
their own caveat saying the opposite. The three are different facts and now read as such:
*nothing in this window can answer it* (the metric declared zero reach), *0 tool calls* (it
counted none of its own unit), and *no stated basis* — which is what an exec metric plugin
omitting `confidence.samples` means.

<a id="completed-subagent-labelled-a-turn"></a>

### A completed Claude sub-agent is a session total, not a turn

*Corrected in v0.10.0, released 2026-08-05.*

One record summarizes a whole sub-agent run, and labelling it `turn` made every per-turn figure
count it as a single very large one — 1,015 such rows on the maintainer's machine, averaging
2.4× a real turn's output. It is now `session` grain, so `coverage` reports the mixed window it
always was and `model-right-sizing` stops counting an aggregate as a turn.

<a id="rejection-rate-had-the-wrong-denominator"></a>

### The rejection rate gets its own denominator

*Corrected in v0.10.0, released 2026-08-05.*

It divided by every tool call in the window, including calls from sources that record no refusal
at all, so a Codex-heavy window read as calmer than it was. `friction` and `rework` now divide
by the calls whose source records a refusal, and say so.

<a id="concentration-blamed-a-lineless-source"></a>

### `concentration` stops blaming a project for a source that writes no lines

*Corrected in v0.10.0, released 2026-08-05.*

A project running entirely on a cost-only source writes zero lines by construction, so its whole
token share read as the widest spend-versus-output gap in the window. Those projects are
excluded from the gap and counted in a caveat instead.

<a id="skill-economics-did-not-state-its-reach"></a>

### `skill-economics` states its reach, and a lone label no longer reads as zero tokens

*Corrected in v0.10.0, released 2026-08-05.*

The concentration share is of attributed tokens, which are a slice of the window — 18% of it on
the maintainer's store — and that is now the metric's declared signal coverage. A window with a
single skill prints `—` for the largest share rather than a share of nothing beside "attributed
tokens: 0".

## [0.9.0] - 2026-08-04

<a id="survival-counted-merge-lines"></a>

### `survival` counted merge lines as survivors that were never counted as added

*Corrected in v0.9.0, released 2026-08-04.*

`git blame` names a merge for every line of a hand-resolved conflict while `numstat` reports
none of them, so the rate divided a number by a total it was never part of — on a fixture with a
50-line resolution that printed `50 surviving of 3 added (100%)`. Both sides now count the same
commits, the merge is reported separately, and the clamp that hid the contradiction behind a
flat 100% is gone.

<a id="a-sliver-carried-a-confident-envelope"></a>

### A figure computed from a sliver of the window carried a confident envelope

*Corrected in v0.9.0, released 2026-08-04.*

On real data `reasoning-share` read a 20% share off under 1% of the output and reported `high`,
while `signals coverage` called the same signal partially supported — two surfaces disagreeing
about one number. `reasoning-share`, `friction` and `explore-produce` now declare their own
reach, and a window where no source can answer at all reads `insufficient` instead of `high`.

## [0.8.0] - 2026-08-03

<a id="copilot-cli-was-half-wired"></a>

### Copilot CLI was only half-wired, and three surfaces were the proof

*Corrected in v0.8.0, released 2026-08-03.*

The parser landed in v0.6.0, but three places still kept their own list of tool names instead of
reading the depth matrix: `sync` rejected every `copilot-cli` record as an unknown tool, so a
team member using it synced nothing and was told nothing; `clear --tool copilot-cli` failed the
same way, which meant its data could not be deleted per source; and `reasoning-share` skipped it
while printing "Only Codex and Gemini CLI report reasoning tokens today", though Copilot has
reported them all along and the store held them unread.

<a id="reasoning-signal-claimed-for-every-source"></a>

### The signal catalog claimed every source reports reasoning tokens

*Corrected in v0.8.0, released 2026-08-03.*

`ai.tokens.reasoning` was bundled into the cost signals every row inherits, so `signals
coverage` reported it as fully supported on a Claude-only machine — where the real answer is
*none*, since Claude Code never surfaces a thinking count. On the maintainer's own store the
figure moved from "100% of tokens" to "<1%, codex", which is the honest number. Reasoning is now
declared per source, like every other capability.

<a id="activity-coverage-was-one-bit"></a>

### A source that records lines and nothing else no longer passes for full activity coverage

*Corrected in v0.8.0, released 2026-08-03.*

The confidence envelope on every verdict was computed from the tier table's one-bit
`Activity` axis, which reads true for Copilot CLI — so a Copilot-only window reported
*activity coverage 100%* while `signals coverage` correctly said its edit, tool-call and
rework signals were unsupported. Two surfaces, two answers to one question. The envelope now
asks the per-signal question ([ADR 0008](adr/0008-signal-catalog.md)), and `coverage`
separates the two facts it used to conflate: "cost only" means a source contributes no
changed lines, and a source with lines but no edit counts is named as partial rather than
hidden behind either verdict.

<a id="label-filter-crossed-members"></a>

### A session annotation no longer selects another member's usage

*Corrected in v0.8.0, released 2026-08-03.*

On a central store the label filter matched on `session_id` alone while every other label query
joins on `(session_id, member)` — the label's own primary key — so two members whose locally
generated session ids collided could be filtered into each other's results. Local stores, where
`member` is always empty, were never affected.

<a id="copilot-session-with-no-id"></a>

### A Copilot session with no id or no timestamp is skipped instead of stored

*Corrected in v0.8.0, released 2026-08-03.*

Its dedupe key is `<session>:<model>`, so an unidentifiable session collapsed every other one
like it into a single row under `ON CONFLICT DO NOTHING`, and an undated one was stored at year
one. Both are now counted as skipped, the same skip-and-count policy every parser follows.

<a id="caveats-named-the-wrong-sources"></a>

### Caveats stopped naming sources

*Corrected in v0.8.0, released 2026-08-03.*

The line-coverage, tool-call, failure-capture, project-attribution and reasoning notes across
`analyze`, `effectiveness`, the dashboard, the `explain` pages and the binary's own `--help`
spelled out "Claude Code and Codex" or "Gemini CLI and Cline" — a sentence a new parser makes
wrong, and most of them already were. They now describe the property and point at `assaio-agent
signals coverage`, or name the window's own sources from the matrix. The `coverage` validator
also stops printing a cost-only caveat when the window contains no cost-only source, and no
longer blames cost-only sources for a window whose gap is a source recording lines and nothing
else.

## [0.7.0] - 2026-08-02

<a id="depth-matrix-overclaimed-per-axis"></a>

### The source-depth matrix now declares capability per signal, not per axis

*Corrected in v0.7.0, released 2026-08-02.*

`deep`, `standard` and `import-only` still summarise a source, but "has activity" turned out to
be one bit over a source that records changed lines and nothing else — which is what Copilot CLI
is. Its first run on real data reported sixteen of eighteen signals as fully supported when the
honest answer is ten, because Copilot totals a whole session and so carries no turn count, no
edit count, no tool calls and no rework. Each parser now lists the signals it actually answers,
a test asserts that list never contradicts the tier axes, and adding a parser means answering
"which of these can you produce" instead of one yes-or-no.

<a id="event-contract-clock-ordering"></a>

### Proving the event contract against 324,416 real records changed it twice

*Corrected in v0.7.0, released 2026-08-02.*

Proving it against 324,416 real records changed the design twice, which is the point of
proving it. Rejecting an event whose source timestamp is newer than the batch's reading time
dropped 51 real records from sessions still being written, so the two clocks are no longer
ordered against each other — a reading time is not a causality claim, and the evidence path
must not be stricter than the store it mirrors. The same run surfaced `B101`: 404 sub-agent
aggregates whose id collides across projects, which the store has always resolved by keeping
whichever file was ingested first.

## [0.6.0] - 2026-08-02

<a id="surfaces-counted-four-sources"></a>

### Four surfaces were counting four sources and eighteen validators

*Corrected in v0.6.0, released 2026-08-02.*

README, the website, `AGENTS.md` and `FEATURES.md` are corrected to five parsers and nineteen
validators, and Copilot CLI is listed as a source rather than as a roadmap item. `FEATURES.md`
had also dated the Copilot parser to v0.5, a release it missed by two commits. Miscounting your
own inventory is the cheapest possible way for an honesty-first tool to be wrong about itself.

## [0.5.0] - 2026-07-31

<a id="granularity-blended-silently"></a>

### Reports no longer blend per-turn and whole-session records silently

*Corrected in v0.5.0, released 2026-07-31.*

`granularity` travels from the store through every report row: sources that emit one record per
session (exec plugins today) are marked `‡` and footnoted, and a grouped total that merges both
units reads `mixed-granularity total` instead of quietly presenting session aggregates as
per-turn data. The `coverage` validator replaces its "not yet surfaced" caveat with a real
turn-level share, `report --format csv` gained a `granularity` column, and the metric-plugin
envelope carries the field so a plugin summing usage rows can see it too (`B69`).

<a id="readme-install-pointed-at-v010"></a>

### The README's manual install instructions pointed at v0.1.0

*Corrected in v0.5.0, released 2026-07-31.*

**The README's manual install instructions pointed at v0.1.0**, three releases behind, so
anyone following them got a binary without `statusline`, `explain`, incremental backfill or
the last two rounds of correctness fixes. Both the shell and PowerShell snippets now resolve
the latest tag before downloading.

## [0.4.0] - 2026-07-29

<a id="live-transcript-froze-a-half-attributed-turn"></a>

### A session ingested while it was still being written froze a half-attributed turn

*Corrected in v0.4.0, released 2026-07-29.*

A turn's failed tool calls, denials and edit counts are attributed by a *later* line in the log,
so ingesting a live transcript stored the turn with its calls but without its outcomes — and
nothing ever corrected it, because a re-read only ever repaired rows that carried no signals at
all. `friction` therefore reported an error rate it could prove was wrong. Re-reading a file the
store owns now restates that turn's activity columns, taking `MAX(stored, offered)` so a repair
can never lower a stored figure. Records pushed to a team server keep the opposite contract —
first-write-wins, so one member's push cannot restate another's row. A count that first read too
*high* is still not corrected; `doctor` says so. Closes B68, and with it the reason a per-turn
hook was not recommended.

<a id="doctor-under-reported-claude-code-files"></a>

### `doctor` under-reported Claude Code by thousands of files

*Corrected in v0.4.0, released 2026-07-29.*

It counted only top-level transcripts while `backfill` reads those *and* every sub-agent
transcript beneath them — on a real machine, 1993 files reported against 6398 actually read. The
diagnostic that answers "what will be read" now reports both counts separately.

<a id="golden-fixtures-built-in-utc"></a>

### The dashboard golden fixtures are built in `time.Local` rather than UTC

*Corrected in v0.4.0, released 2026-07-29.*

The dashboard golden fixtures are built in `time.Local` rather than UTC. `rhythm` reads
session starts in the machine's own zone, so a UTC fixture placed the same session in a
different time-of-day band depending on where the test ran -- the rendered goldens passed
on a CET laptop and failed in CI. Verified identical across five zones.

## [0.3.0] - 2026-07-26

<a id="legacy-bars-key-disarmed-pseudonymization"></a>

### A metric plugin written against the pre-rename `barsAreProjects` key lost its pseudonymization

*Corrected in v0.3.0, released 2026-07-26.*

**A metric plugin written against the pre-rename `barsAreProjects` key lost its
pseudonymization**, publishing real repository names into an anonymized dashboard. The
legacy key is accepted again and maps to `"project"`; every *other* unknown field is now
rejected outright, so a misspelled key can never quietly disarm a setting again.

<a id="drill-re-ran-window-scoped-metrics"></a>

### The project drill re-ran window-scoped metrics against one project's rows

*Corrected in v0.3.0, released 2026-07-26.*

Subscription Fit divided the whole plan price by a single project's spend and printed "API could
be cheaper" beside the window-level "your plan pays off" on the same page; Skill & Agent
Economics showed window-wide totals under a project heading. A validator can now declare itself
`WindowScoped` and the drill skips it — see `docs/extending.md`.

<a id="codex-produce-share-was-zero"></a>

### Codex's tool-purpose split reported 0% produced whatever the agent did

*Corrected in v0.3.0, released 2026-07-26.*

**Codex's tool-purpose split reported 0% produced whatever the agent did**, because Codex
applies file edits through a `patch_apply_end` event that no tool call names. Those edits
now count as write calls (and a failed one as an errored call), with a guard so a version
that *does* name its patch calls cannot double-count them.

<a id="friction-counted-unmarked-calls"></a>

### `friction` counted calls that cannot report a failure

*Corrected in v0.3.0, released 2026-07-26.*

Codex marks the outcome of file edits only — a shell command that exits non-zero looks identical
to one that succeeded — so its calls padded the denominator and diluted the error rate. Rates
are now taken over tools that mark every call, and the caveat says which.

<a id="subagent-tokens-double-counted"></a>

### Sub-agent tokens were double-counted

*Corrected in v0.3.0, released 2026-07-26.*

**Sub-agent tokens were double-counted** on a store holding both a pre-upgrade `agent:`
aggregate row and the per-turn rows later parsed from the same transcript. A window now
picks one definition: the `sidechain` marker when any row carries it, the older dedupe-key
shape otherwise.

<a id="concentration-passed-an-exam-it-never-ran"></a>

### `concentration` returned a green ALIGNED when no project was large enough to compute a gap

*Corrected in v0.3.0, released 2026-07-26.*

**`concentration` returned a green ALIGNED when no project was large enough to compute a
gap** — a passed check for an examination that never ran — and computed its headline shares
over the whole window while the Gini score used only named projects, so the two figures on
one line disagreed. Both now read as shares of attributable spend.

<a id="demo-left-three-panels-blank"></a>

### `assaio-agent demo` left three of eighteen panels blank

*Corrected in v0.3.0, released 2026-07-26.*

**`assaio-agent demo` left three of eighteen panels blank** — the tool's own showcase — for
want of the new signals on its bundled records. Its sample sessions are also pinned to a
working weekday morning now: they used to inherit the invocation clock, so the identical
fixture read as ordinary hours before lunch and as off-hours work after dinner, and its
rhythm verdict changed with the reader's timezone.

<a id="skill-names-reached-anonymized-reports"></a>

### Skill and sub-agent names reached anonymized reports verbatim

*Corrected in v0.3.0, released 2026-07-26.*

The team server's unauthenticated dashboard pooled them across every member beside pseudonymized
project and member names; skill names are user-authored and can name a client. They are now
pseudonymized like project names.

<a id="pushed-records-bypassed-the-numeric-check"></a>

### The nine columns migration 0002 adds bypassed the sync boundary's numeric check

*Corrected in v0.3.0, released 2026-07-26.*

**The nine columns migration 0002 adds bypassed the sync boundary's numeric check**, so a
pushed record could store a negative count (rendering impossible percentages) or an
overflow-magnitude one (breaking `SUM()` for the whole team). A reflection-based test now
fails the build if any numeric record field is left unbounded.

<a id="check-failed-open"></a>

### `check` failed open in two ways

*Corrected in v0.3.0, released 2026-07-26.*

A metric plugin that failed was dropped with a warning, so a rule gating on its verdict passed
on a verdict set it never saw; and a rule emitting `null` or `{}` — what a jq filter prints when
it stops matching — was read as "no alerts" instead of a rule that evaluated nothing. Both now
fail the gate, matching what the command already promised.

<a id="friction-fabricated-a-clean-zero"></a>

### `friction` reported a fabricated 0.0% and a green verdict

*Corrected in v0.3.0, released 2026-07-26.*

**`friction` reported a fabricated 0.0% and a green verdict** on windows where no call
could record a failure at all, and impossible percentages (`150%` errors, `-50%` clean)
when errors outnumbered counted calls. Rates are now taken over calls whose failure state
was actually recorded, coverage is stated, and incoherent counts render `—`.

## [0.2.0] - 2026-07-22

<a id="coverage-share-rounded-to-zero"></a>

### Coverage rounding

*Corrected in v0.2.0, released 2026-07-22.*

In the `coverage` validator, a small but nonzero token share (e.g. a few Codex sessions dwarfed
by Claude's cache-read volume) now reads `<1%` instead of `0%` (which looked absent), and a
share just under whole reads `>99%` instead of a gap-hiding `100%` — the honesty backbone must
not round either edge away.

<a id="gemini-session-ids-on-the-header-line"></a>

### Gemini session ids

*Corrected in v0.2.0, released 2026-07-22.*

The real Gemini CLI recording carries the session id only on the
file's header line, not on every message, so message records were left with an empty
session id. The parser now carries the header's id forward (an older per-line shape still
works), and skips `$set`/`$rewindTo` control records without miscounting them.
**Compatibility:** this changes the dedupe key for header-only Gemini logs, so if you
ingested Gemini usage under v0.1.x, run `assaio-agent clear --tool gemini-cli --yes` then
`assaio-agent backfill` once after upgrading to avoid double-counting those records.

## [0.1.1] - 2026-07-20

<a id="subagent-accounting-under-counted"></a>

### Claude Code sub-agent accounting

*Corrected in v0.1.1, released 2026-07-20.*

Background/async sub-agent (Task) token usage was not counted at all, and a completed
sub-agent's cost was read from a last-turn summary in the parent transcript rather than its full
per-turn record — a large under-count. assaio now reads each sub-agent's own transcript as the
source of truth and suppresses the redundant parent summary, so sub-agent usage is counted once
and in full. This is the accounting behavior the 0.1.0 notes claimed but did not fully deliver.
