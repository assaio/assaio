# Adding a metric validator

*Part of [Extending assaio](../extending.md). Next: [the worked example](metric-validator-example.md).*

Every block `assaio analyze` prints — adoption, model fit, context health, throughput,
rework — is a `Validator` under
[`internal/analyze/`](../../internal/analyze/analyze.go). This is the in-tree,
available-today realization of "one metric = one file" from
[`AGENTS.md`](../../AGENTS.md) and [`CONTRIBUTING.md`](../../CONTRIBUTING.md): a new metric is
one file, self-registering, with no central list to edit — and because the HTML
dashboard renders every registered validator's `Result` generically (see
[Verified](#verified-it-appears-in-the-cli-and-the-dashboard-automatically) below), it is
also a new dashboard section for free.

**How you actually ship one today — two ways.** An **in-tree validator** (this section)
is compiled into the `assaio-agent` binary: add your file under `internal/analyze/` in
your own fork or a private overlay of this repo, `make build`, and distribute the
resulting `bin/assaio-agent` (or upstream it via PR,
[`CONTRIBUTING.md`](../../CONTRIBUTING.md) — the intended path for a metric with broad
value). `internal/` is enforced by the Go compiler itself to be unimportable from
outside this module, so a validator can never be a separate Go dependency. The
**out-of-tree alternative needs no fork at all**: a [metric
plugin](metric-plugin.md) is a standalone executable in any language,
declared under `metrics:` in `config.yaml`, that reads the same prepared `Input` as JSON
and returns one `Result` — it appears in `analyze` and on the dashboard beside the
built-ins, namespaced `plugin:<name>`. Reach for in-tree when the metric belongs
upstream or needs data the wire envelope doesn't carry; reach for the plugin when the
metric is yours alone.

## What a validator reads: `Input`

```go
type Input struct {
	Usage           []store.UsageRow
	Sessions        []store.SessionRow
	Prices          pricing.Table
	Now             time.Time
	Recent          time.Duration
	Delegation      Delegation
	ByModel         []ModelStat
	ByProject       []ProjectStat
	Totals          Totals
	PlanMonthlyCost float64
	Skills          []store.AttributionRow
	Agents          []store.AttributionRow
	TurnSizing      []store.ModelTurns
	Trace           trace.Set
	HistoryStart    time.Time
}
```

`Input` is a read-only bundle; use only the fields your metric needs. `Analyze` must stay
a **pure function of `Input`** — no `time.Now()`, no file or network I/O, no reaching
into the store yourself — which is what makes it trivial to unit test and safe to run
identically from the CLI, the dashboard, and (see [The team
server](team-server.md)) the served endpoint.

| Field | Type | What it is |
|-------|------|------------|
| `Usage` | `[]store.UsageRow` | The window's usage, **pre-aggregated** by `(day, tool, model, project, entrypoint, member)` — one row per combination, tokens and activity counts summed (`internal/store/store.go`'s `Usage` query). Not one row per raw event: there is no per-record or per-file detail left at this point (see the [say-so-when-approximating](../extending.md#honesty-constraints-for-every-extension) rule). Each row carries `Day` (`"YYYY-MM-DD"`), `Tool`, `Model`, `Project`, `Entrypoint`, `Member`; token fields `In, Out, CacheRead, CacheWrite, Reasoning`; and activity fields `LinesAdded, LinesRemoved, Edits, ToolCalls, Rejected, Compactions, ReworkLines` (see the [`usage.Record` contract](data-source.md#the-usagerecord-contract) for what each counts). It also carries the tool-call purpose split `ToolReads, ToolSearches, ToolCommands, ToolWrites, ToolOther` — which sums to `ToolCalls` for tools that name their tool calls and is all-zero for tools that do not — and `ToolErrors`, calls that came back an error. **Every activity count here is zero for a source that does not record it**, exactly as on `Sessions`: keep the rows whose `Tool` answers the matching signal with `report.UsageAnswering` before computing a rate over one — see [Reading a field only the sources that record it carry](#reading-a-field-only-the-sources-that-record-it-carry). That applies to the tool-call purpose split too, which is populated by only some tools. |
| `Sessions` | `[]store.SessionRow` | One row per `(session_id, member)` in the window: `Project`, `Tool`, `Model`, `FirstTs`/`LastTs`, `Turns`, `OutputTokens`, `PeakContextTokens`, `Edits`, `Compactions`, and `ActiveMinutes` (focused time — inter-turn gaps over 30 minutes are excluded, so a resumed session's idle time never counts as work; `internal/store/sessions.go`). `Turns`, `PeakContextTokens` and the gaps behind `ActiveMinutes` read only `turn`-grain rows, so a whole sub-agent run stored as one session-grain record is never counted as a single very large turn. **Every one of those counts except the timestamps is zero for a source that does not record it**, so a metric reading one must first keep the sessions whose `Tool` answers the matching signal — see [Reading a field only the sources that record it carry](#reading-a-field-only-the-sources-that-record-it-carry). Also `Task`, `Outcome` and `Difficulty`: the closed-vocabulary labels a person attached with `assaio-agent mark`, `""` on every axis nobody set — which is most sessions. Treat unlabeled as "not stated", never as a failure or a zero, and report your own label coverage if your metric depends on them ([ADR 0006](../adr/0006-session-annotations.md)). |
| `Prices` | `pricing.Table` | `map[string]pricing.Price{Input, Output, CacheWrite, CacheRead float64}`, USD per token, the vendored LiteLLM snapshot. Indexing a model absent from the table returns a zero-value `Price` with **no error** — check the map's `ok` return if an unpriced model must be excluded from a cost figure rather than silently priced at $0. |
| `Now` | `time.Time` | Wall-clock at CLI invocation. Use this, never call `time.Now()` yourself. |
| `Recent` | `time.Duration` | The recent-vs-prior comparison window (7 days from the CLI today) — use it for trend/staleness splits the way `adoption` and `throughput` do. |
| `Delegation` | `Delegation{Sub, Total int64}` | Real sub-agent token-delegation share for the window: `Sub` is tokens on records whose `dedupe_key` marks a Task sub-agent turn, `Total` is every token in the same window. Computed once by the CLI via `store.Store.Delegation` so validators never reach into the store themselves. |
| `ByModel` | `[]ModelStat` | `Usage` aggregated per model, already tier-classified and priced. **Read this instead of grouping `Usage` by model yourself.** See the table below. |
| `ByProject` | `[]ProjectStat` | `Usage` aggregated per project. **Read this instead of grouping `Usage` by project yourself.** See the table below. |
| `Totals` | `Totals` | `Usage`'s grand totals across every model and project. See the table below. |
| `PlanMonthlyCost` | `float64` | The user's configured flat monthly plan price (`pricing.monthly_subscription_cost`); `0` means unset — prompt to configure it rather than comparing against nothing. |
| `Skills` / `Agents` | `[]store.AttributionRow` | The window's per-skill and per-sub-agent totals (`Name`, `Tokens`, `Lines`, `Records`, `Sessions`), each sorted by `Tokens` descending. `Name` is a category label the tool assigned — never a prompt or any content. Empty when no tool in the window reports attribution (only Claude Code does today), so a metric over them must state its own coverage. |
| `TurnSizing` | `[]store.ModelTurns` | Per-model raw turn counts (`Turns`, `SmallTurns`) for metrics that need the per-turn grain the daily `Usage` aggregate hides. Empty in the drill and in tests that do not set it. |
| `Trace` | `trace.Set` | The window's step sequences (ADR 0012). Never read the whole set: call `Trace.Scope(...)` with the one population your metric declares, and render the `View.Caveat()` it hands back. Declaring the scope is not politeness — 89% of the sequences on the audited store are one-shot SDK calls holding 5.7% of its steps — and a validator that reads sequences declares it structurally by implementing `analyze.TraceReader`. Empty on a store with no step history and for every source with no step reading. |
| `HistoryStart` | `time.Time` | The earliest observation the store holds, ignoring the window, so a figure comparing one span against an earlier one can say whether the earlier one existed. A validator that renders such a comparison implements `analyze.Trending`, and the horizon line is stamped onto its Result automatically. Zero means unknown. |
| `CacheMisses` | `[]store.CacheMissRow` | The window's stated cache-miss reasons per tool (`Tool`, `Reason`, `Turns`), from the vendor's own closed vocabulary — a category, never content. A turn that hit cache states no reason and is absent, as is every turn from a source that reports none, so the absence of a row is not evidence the cache was hit. |
| `WindowStart` | `time.Time` | The `--since` boundary usage was queried with. The zero time means the caller scoped no window, and a rate then spans the usage itself rather than the window. Divide a projection by real days from here, because a day inside the window carrying no usage is still a day a flat plan was paid for. |
| `Ingested` / `ParsedBy` | `time.Time` / `string` | When the newest data in the store was read, and the build that read it. `analyze.Stamp` copies both onto every `Result.Confidence`, so a validator does not set them — the zero time and `""` mean unknown, which is what a caller that cannot answer leaves them as rather than guessing. `digest` reads `ParsedBy` to declare when two runs are not comparable because the parser changed between them. |

### Reading a field only the sources that record it carry

Most fields on `Usage` and `Sessions` are optional at the parser (see [Activity fields are
optional](data-source.md#activity-fields-are-optional-honesty-note)), which means a zero has two meanings:
nothing happened, or nothing was written down. Averaging the two produces a number nobody
measured — the mistake that made a Cline-only window read as *100% conversational sessions*,
because Cline records no edit count and every session sat at zero edits.

The rule for any metric over such a field: **keep the rows whose source answers the signal,
compute over those, and declare the reach.**

```go
// sessionsAnswering keeps the sessions whose source can answer the signal, and the share of
// the window's sessions they are.
edited, coverage := sessionsAnswering(in.Sessions, parser.SignalEditsCount)
r.restsOn(len(edited), "sessions with edit capture")
r.covering(coverage)
if len(edited) == 0 {
	r.Read = noDataRead // no source here records it: withhold the verdict, do not certify a silence
	...
}
```

Three consequences worth stating: a figure whose subset is empty prints `—` rather than `0`;
the verdict is withheld rather than earned from a silence; and `covering()` makes the
confidence envelope say how much of the window the figure describes. Spell the signal with
the `parser.Signal*` constant, and if the field you need has no signal id yet, add one — a
catalog entry plus the matrix rows that answer it ([ADR 0008](../adr/0008-signal-catalog.md)) —
rather than testing the tool name.

**Both row shapes need it.** `report.UsageAnswering` is `SessionsAnswering` for
`[]store.UsageRow`, and the same rule applies to a rate over a stored column: a source
recording changed lines but no rework contributed its whole output to the churn denominator
against a structural zero, which lowered the rate with code nobody watched being undone. The
generic test in `internal/analyze/capability_test.go` varies both shapes for exactly that
reason — a hole on either grain is the same bug.

### Opting a metric out of the project drill

The dashboard re-runs every validator over the top project's rows alone. A metric whose
answer belongs to the whole window — a flat plan price, attribution pooled across projects,
per-model turn counts — cannot honestly be narrowed that way: re-run against a slice it
compares a window-wide constant with part of the usage and prints a verdict that contradicts
the window-level one on the same page. Declare it window-scoped and the drill skips it:

```go
// WindowScoped: the plan price covers the whole window, not one project's share of it.
func (myValidator) WindowScoped() {}
```

Nothing else changes — the metric still renders normally at window level.

### Read these first: `ByModel`, `ByProject`, `Totals`

`BuildInput` (`internal/analyze/prepared_build.go`) computes these three once, before any
validator runs, from the same `Usage` rows above. **Most validators — built-in or
custom — should read one of these instead of re-grouping `Usage` by hand or importing
`internal/report`.** `model-fit` is the reference: it used to call
`report.BuildEffectiveness(in.Usage, in.Prices, "model")` and then loop over the result to
classify each model's tier by price; it now reads `in.ByModel` directly and imports
neither `internal/report` nor `internal/pricing` at all (compare
[`internal/analyze/model_fit.go`](../../internal/analyze/model_fit.go) to the description
above — the whole re-derivation is gone).

**`ModelStat`** (`in.ByModel`, sorted by `Tokens` descending):

| Field | Type | What it is |
|-------|------|------------|
| `Model` | `string` | The model name, as `store.UsageRow.Model`. |
| `Tier` | `string` | `"premium"`, `"cheaper"`, or `"unknown"` — already classified from `Model`'s real price. Never re-derive tier from the model's name yourself. |
| `Tokens` | `int64` | `In+Output+CacheRead+CacheWrite+Reasoning`, summed across this model's usage. |
| `Input`, `Output`, `CacheRead`, `CacheWrite` | `int64` | The same usage, summed per token type. |
| `Lines` | `int64` | AI-added code lines summed across this model's usage. |
| `Cost` | `*float64` | USD cost priced from `Prices`; `nil` when `Priced` is `false` — check the pointer, never compare a bare `0`. |
| `Priced` | `bool` | `false` when `Model` has no known price — `Cost` is then unknown, not a real zero. |
| `TokenShare` | `float64` | This model's share of every `ModelStat`'s `Tokens` in `ByModel`, `0..1`. |

**`ProjectStat`** (`in.ByProject`, sorted by `Lines` descending):

| Field | Type | What it is |
|-------|------|------------|
| `Project` | `string` | The project name; `""` for unattributed usage. |
| `Lines` | `int64` | AI-added code lines summed across this project's usage. |
| `Cost` | `*float64` | USD cost priced from `Prices`, summed from this project's priced usage only; `nil` when none of it priced. |
| `Priced` | `bool` | `false` when at least one contributing row's model has no known price — `Cost` then undercounts this project's real spend (but is still non-`nil` as long as at least one row priced). |
| `TokenShare` | `float64` | This project's share of `Totals.Tokens`, `0..1`. |

**`Totals`** (`in.Totals`, the window's grand totals):

| Field | Type | What it is |
|-------|------|------------|
| `Tokens` | `int64` | `In+Output+CacheRead+CacheWrite+Reasoning`, summed across all `Usage`. |
| `Input`, `Output`, `CacheRead`, `CacheWrite` | `int64` | The same usage, summed per token type. |
| `Lines` | `int64` | AI-added code lines summed across all `Usage`. |
| `Cost` | `*float64` | USD cost priced from `Prices`, summed from priced usage only; `nil` when nothing priced. |
| `Priced` | `bool` | `false` when at least one usage row's model has no known price — `Cost` then undercounts real spend (but is still non-`nil` as long as at least one row priced). |
| `CacheEfficiency` | `float64` | `CacheRead / (CacheRead + Input)`, `0` when that sum is zero. |

A custom "which model is eating the budget" metric needs no grouping helpers at all —
just the prepared fields. `Cost` is `*float64` precisely so this comparison cannot
silently prefer an unpriced model: `top` only advances to a model that is `Priced`, so a
big, unpriced model never loses to a smaller priced one just because its zero-value `Cost`
would otherwise compare as "less" — and never panics dereferencing a `nil` `Cost` either:

```go
top := in.ByModel[0] // ByModel is already sorted by Tokens descending
for _, m := range in.ByModel[1:] {
	if m.Priced && (!top.Priced || *m.Cost > *top.Cost) {
		top = m
	}
}
var share float64
if top.Priced && in.Totals.Cost != nil && *in.Totals.Cost > 0 {
	share = *top.Cost / *in.Totals.Cost
}
r.Figures = []Figure{
	{Label: "top model by cost", Value: top.Model, Note: top.Tier},
	{Label: "its share of window cost", Value: formatPercent(share, 1)},
}
```

Run for real against a seeded store, that prints real figures like `top model by cost:
claude-opus-4-8 (premium)` and `its share of window cost: 74.7%` — no dimension grouping,
no `pricing.Table` handling, no `internal/report` import.

`Usage` and `Sessions` remain available for signals the prepared views don't cover — a
day-level split (`weekend-usage` below), a session-grain signal (`context`'s compaction
rate), or a friction count the prepared views deliberately leave out (`rework`'s rejection
rate). Reach for them, or `internal/report`'s own aggregations (`BuildInsights`,
`BuildSessionStats`, `BuildChurn`), only when `ByModel`/`ByProject`/`Totals` above don't
already have what you need.

A metric that needs domain data `Input` doesn't carry yet (per-file paths, for instance —
deliberately never persisted; see [Parsers stay
hermetic](data-source.md#parsers-stay-hermetic--project-resolution-is-ingests-job)) can't be built from
stored data today. Open an issue describing the signal; that is exactly the kind of
request that shapes `Input` before an out-of-tree interface is ever frozen.

## What a validator returns: `Result`

```go
type Result struct {
	Name, Title, Describe string
	Read            Read
	Purity          float64
	HowToRead       string
	Figures         []Figure
	Bars            []Bar
	BarsPseudonym   string
	Takeaway        string
	Caveats         []string
}
```

One `Result` value feeds every surface — the CLI text report
(`analyze.RenderResultText`, `internal/analyze/format.go`), JSON
(`analyze --format json`), and the HTML dashboard (`dashboard.html.tmpl`'s
`faceplateCell`/`ledgerEntry` templates). The table below is field-by-field, including
exactly how each one renders on each surface — the mechanics behind the "new dashboard
section for free" claim.

| Field | Meaning | CLI text | HTML dashboard |
|-------|---------|----------|-----------------|
| `Name` | Stable kebab-case slug, e.g. `"weekend-usage"`. | The `analyze <name>` argument; first column of `analyze --list`; upper-cased in the header line. | Not shown as text — used only to look the `Result` up (e.g. `findVerdict` in tests). |
| `Title` | Human label. | Header line: `"WEEKEND-USAGE · Weekend Usage  [WATCH]  (activity)"`. | Faceplate cell label (`"06 · Weekend Usage"`) and the ledger entry's label. |
| `Describe` | One-line summary. | `analyze --list`'s third column only — **not** printed by `RenderResultText`. | Not rendered. |
| `Layer` | One of `activity`, `output`, `outcome`, `impact` — which of the four measurement layers the **verdict** rests on ([ADR 0013](../adr/0013-measurement-layers.md)). A figure inside the same `Result` may sit on another layer as context; the label answers what the verdict is a claim about. In-tree it is the `Layer()` method, and `Evaluate` stamps it onto the `Result`; over the plugin wire it is a required `layer` key and a result without it is rejected whole. | Header line, after the read: `"[WATCH]  (activity)"`. | The ledger entry's layer line, with the four-layer explanation as its tooltip. |
| `Read.Key` | `"good"`, `"watch"`, or `"neutral"` (no data). | Drives no CLI styling (plain text). | Drives the dashboard's color via CSS classes `cell__read--{key}` / `entry__read--{key}` (verdigris/oxide/muted) — an unrecognized key renders unstyled, so stick to these three. |
| `Read.Label` | Short upper-cased word, e.g. `"WATCH"`, `"STRONG"`, `"—"`. | Printed in `[brackets]` on the header line. | Printed as the faceplate/ledger read text. |
| `Purity` | `0..1`, how "well-used" this dimension reads. It is your own quantity, not an index shared with other validators, so it is never rendered as a bare number anybody could compare across cells. | **Not rendered.** | The faceplate gauge's fill width, unlabelled — clamp to `[0,1]` yourself (`clamp01`) or the CSS width silently over/underflows. |
| `HowToRead` | One-sentence explainer of what the metric means and what to do about it. Must be non-empty on **every** code path, including the no-data one. | The `"  ? …"` line under the header. | The ledger entry's muted "How to read — …" line. |
| `Figures` | The headline numbers: `{Label, Value, Note}`. | One `"  label: value (note)"` line each. | One stat tile per figure in `.entry__stats`; **the first figure gets an accent color** — order your most important number first. Not shown in the faceplate. |
| `Bars` | Optional ranked list: `{Label, Value, Frac 0..1}`. Three states matter: `nil` → no Bars section anywhere; non-nil empty → an honest "none in this window" line; non-nil non-empty → a ranked bar list. | ASCII bar `label: value  [####----]` (20 chars wide, scaled by `Frac`). | `.projectbars` list, bar width from `Frac`. Scale `Frac` against that list's own max (`fracOf`), not a global scale. |
| `BarsPseudonym` | What kind of user-authored name `Bars`' labels carry: `"project"`, `"skill"`, or `""` for none. | Not rendered. | Tells `--anonymize` whether to pseudonymize `Bars` labels, and under which prefix — see [Honesty constraints](../extending.md#honesty-constraints-for-every-extension). Set it for anything a person named (a repository, a skill, a sub-agent); leave it empty for a fixed vocabulary the tool defines (models, tools, time bands). Set it wrongly and you either scramble a label that was never PII, or leak a real name into a report meant to be shared. |
| `Takeaway` | One-line plain-language conclusion. Always populated, even on "no data". | Last line: `"  Takeaway: …"`. | `.entry__takeaway`, prefixed with an em dash. |
| `Caveats` | Honesty notes (directional, contested, approximate, …). | Each on its own `"  …"` line. | Each becomes a muted "Note — …" paragraph, **and** any non-empty `Caveats` adds a small "Prov." (provenance) badge next to the Read label on both the faceplate cell and the ledger entry. |

## Register it: the `Validator` interface

```go
type Validator interface {
	Name() string         // kebab-case slug, e.g. "model-fit" -- the CLI arg and JSON key
	Title() string        // human label for the report header
	Describe() string     // one line, shown by `assaio analyze --list`
	Layer() layer.Layer   // which of the four measurement layers the verdict rests on (ADR 0013)
	Analyze(Input) Result // pure: reads only what it needs from Input, returns a Result
}
```

Add a file under `internal/analyze/`, implement `Validator`, and register it from that
file's own `init()`:

```go
func init() { Register(myMetricValidator{}) }
```

Nothing else to wire up. `assaio analyze --list` and a bare `assaio analyze` (no
arguments) both call `analyze.Validators()`, which returns every self-registered
validator, name-sorted — your new metric appears in both automatically, and so does the
dashboard, per below.

## Verified: it appears in the CLI and the dashboard automatically

This is not a claim taken on faith — it was verified end to end while writing this
document, using a throwaway `weekend-usage` validator (the same metric turned into the
[worked example](metric-validator-example.md) below), then deleted so the tree stays
clean. With the validator registered and `assaio-agent` rebuilt (the list is elided here — the
shipped set is in the [generated reference](https://assaio.dev/docs/reference#validators), which
cannot fall behind the way a pasted transcript does):

```console
$ assaio-agent analyze --list
adoption       Adoption & Usage Breadth         Sessions, active days, and project/tool breadth: how broad AI usage is, and whether it's growing.
…
weekend-usage  Weekend Usage                    Share of AI tokens run on Saturday/Sunday -- an out-of-hours usage signal.

$ assaio-agent analyze weekend-usage
WEEKEND-USAGE · Weekend Usage  [WATCH]  (activity)
  ? A rising weekend share can mean crunch time or just flexible hours -- read it next to team sentiment, not as a verdict on its own.
  weekend token share: 80.4%
  weekend AI lines: 240
  Directional: a proxy for out-of-hours work, not a burnout measurement.
  Takeaway: A meaningful share of usage falls on weekends -- worth checking in on workload.

$ assaio-agent analyze --format json weekend-usage
[
  {
    "name": "weekend-usage",
    "title": "Weekend Usage",
    "describe": "Share of AI tokens run on Saturday/Sunday -- an out-of-hours usage signal.",
    "read": { "key": "watch", "label": "WATCH" },
    "purity": 0.1964285714285714,
    "howToRead": "A rising weekend share can mean crunch time or just flexible hours -- read it next to team sentiment, not as a verdict on its own.",
    "figures": [
      { "label": "weekend token share", "value": "80.4%" },
      { "label": "weekend AI lines", "value": "240" }
    ],
    "takeaway": "A meaningful share of usage falls on weekends -- worth checking in on workload.",
    "caveats": ["Directional: a proxy for out-of-hours work, not a burnout measurement."]
  }
]

$ assaio-agent dashboard --output assaio-dashboard.html
Wrote dashboard to assaio-dashboard.html (window: last 30 days, project/member names pseudonymized).
```

And the generated HTML — **with zero edits to `internal/dashboard/dashboard.go`,
`render.go`, or `dashboard.html.tmpl`** — contains a new faceplate cell and a full ledger
entry:

```html
<div class="cell">
  <span class="cell__label">06 · Weekend Usage</span>
  <div class="cell__read-row">
    <span class="cell__read cell__read--watch">WATCH</span>
    <span class="prov">Prov.</span>
  </div>
  <div class="gauge"><span class="gauge__fill" style="--fill:19.6%"></span></div>
</div>

<article class="entry">
  <div class="entry__gutter">
    <span class="entry__num">06</span>
    <span class="entry__label">Weekend Usage</span>
    <span class="entry__read entry__read--watch">WATCH</span>
    <span class="prov">Prov.</span>
  </div>
  <div>
    <p class="entry__howto">A rising weekend share can mean crunch time or just flexible hours -- read it next to team sentiment, not as a verdict on its own.</p>
    <div class="entry__stats">
      <div class="stat"><span class="stat__value stat__value--accent">80.4%</span><span class="stat__label">weekend token share</span></div>
      <div class="stat"><span class="stat__value">240</span><span class="stat__label">weekend AI lines</span></div>
    </div>
    <p class="entry__caveat">Directional: a proxy for out-of-hours work, not a burnout measurement.</p>
    <p class="entry__takeaway">A meaningful share of usage falls on weekends -- worth checking in on workload.</p>
  </div>
</article>
```

This generic rendering is why `internal/dashboard/dashboard.go` and `.html.tmpl` both
carry an `EXTENSIBILITY SEAM` comment at the exact loop that walks `Data.Verdicts`: it is
generic over `analyze.Validators()`'s registration order, on purpose.

One gap *was* found and fixed while verifying this: `Bars` pseudonymization used to be
hardcoded to the validator named `"throughput"`, which meant a **custom** validator
ranking `Bars` by project would leak real project names under `--anonymize`. It is now
driven by the `Result.BarsPseudonym` field described above, applied generically by
`internal/dashboard.anonymizeVerdicts` to any validator — see [Honesty
constraints](../extending.md#honesty-constraints-for-every-extension).

## Conventions and lint

- File name: snake_case matching the metric, e.g. `weekend_usage.go` for `Name()`
  `"weekend-usage"` (mirrors `model_fit.go` → `"model-fit"`). Test file alongside it:
  `weekend_usage_test.go`.
- `Analyze(Input) Result` will trip `golangci-lint`'s `gocritic` performance check for
  passing/returning a non-trivial struct by value; every built-in validator silences it
  the same way — copy the comment verbatim, it is what `nolintlint`'s
  `require-explanation`/`require-specific` settings expect:

  ```go
  //nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
  ```

- Reuse the shared helpers in `internal/analyze/format.go` rather than re-deriving them:
  `readFor(ok, favorableLabel)` for `Read`, `humanize.PercentOrDash`/`perActiveDay`
  for `—`-safe ratios, `clamp01` for `Purity`, `fracOf` for `Bar.Frac`, `groupLabel` for
  an empty dimension value.
- Render a number through `internal/humanize` so it reads the same here as in the report
  table beside it: `humanize.Count` for tokens (`33.4B`), `humanize.Int` for a count of
  things that can pass a thousand (`329,612` calls, lines, sessions, turns). A count that
  is small by construction — active days, projects, task classes, a threshold in a note —
  stays bare; grouping it is noise.
- Reach for `in.ByModel`/`in.ByProject`/`in.Totals` before `in.Usage` — see [Read these
  first](#read-these-first-bymodel-byproject-totals) above. If your metric ends up
  grouping `Usage` by model or project itself, that is usually a sign the prepared views
  already have what you need.
- Give the file a test that seeds a small `Input`, calls `Analyze(...)`, renders it with
  `RenderResultText`, and asserts the figures/read you expect — plus a zero-value
  `Input{}` case: no panic, the honest "no data" block, never a favorable read computed
  from nothing (see `TestValidatorsEmptyInputSafe` in `internal/analyze/validators_test.go`
  for the pattern every built-in validator is held to).
