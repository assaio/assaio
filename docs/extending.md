# Extending assaio

`assaio` v0.1 is intentionally small, but every axis a company needs to adapt for its own
use is a documented, working extension point today: your own **metric and dashboard
section** (in-tree, one file — or **out-of-tree in any language**, no fork), your own
**log-source location** (a config change, no code), an entirely new **tool as an
out-of-tree plugin** (any language, no Go), and **direct SQL** against your own data.
This document is the contract for all of them — what's available, how to use it, and a
complete worked example for the one most contributors reach for first: a custom metric.

**The headline mechanism, in one paragraph.** A metric is one Go file under
`internal/analyze/` that reads the same `Input` bundle every built-in metric reads and
returns one `Result` value. Register it from that file's own `init()`, and it appears in
`assaio analyze`, `assaio analyze --format json`, **and** the HTML dashboard — a new
faceplate cell and a new ledger section, laid out, colored, and captioned like every
built-in one — with no other code to write and no template to touch. That claim is not
aspirational: it is verified end to end in [Adding a metric
validator](#adding-a-metric-validator) below, including what happens if your metric's
`Bars` rank by project name and the report is anonymized.

- [Extension surfaces today vs. planned](#extension-surfaces-today-vs-planned)
- [Honesty constraints for every extension](#honesty-constraints-for-every-extension)
- [Adding a metric validator](#adding-a-metric-validator) — the main path: your own
  metric *and* dashboard section, one file.
- [Custom log-source paths](#custom-log-source-paths) — point `assaio` at non-default
  log locations, no code.
- [Write a plugin (any language)](#write-a-plugin-any-language) — an out-of-tree parser
  for an entirely new tool, no Go required.
- [Write a metric plugin (any language)](#write-a-metric-plugin-any-language) — an
  out-of-tree **metric**: your own analyzer on `analyze` and the dashboard, no fork, no
  Go required.
- [The team server](#the-team-server) — your compiled-in validators run there too,
  automatically.
- [Query your own data](#query-your-own-data) — the SQLite store as a documented surface.
- [Add a data source](#add-a-data-source) — teach `assaio` to read a new tool's logs,
  in-tree.
- [Custom metrics (what's shipped vs. roadmap)](#custom-metrics-whats-shipped-vs-roadmap)
  — the two shipped paths, and where the dynamically loaded in-process API is headed.

## Extension surfaces today vs. planned

| Surface | Status | How |
|---------|--------|-----|
| In-tree metric validator (`assaio analyze` + dashboard) | today | One file under `internal/analyze/` implementing `Validator`, self-registered via `init()` — appears in `analyze`, `analyze --format json`, and the HTML dashboard automatically. See [Adding a metric validator](#adding-a-metric-validator). |
| Custom log-source paths | today | `sources.<tool>` in `config.yaml`, no code. See [Custom log-source paths](#custom-log-source-paths). |
| Out-of-tree exec plugin (any language) | today | An executable speaking the [plugin protocol](#write-a-plugin-any-language), declared in `config.yaml`. |
| Out-of-tree exec **metric** plugin (any language) | today | An executable speaking the [metric plugin protocol](#write-a-metric-plugin-any-language), declared under `metrics:` in `config.yaml` — your own analyzer in `analyze` and the dashboard without forking. |
| Team server | today (MVP) | `assaio-agent serve` + `sync`; the served dashboard runs the same validator registry as the local CLI. See [The team server](#the-team-server). |
| SQL queries against the schema | today | Point any SQLite client at the documented `usage_record` table. |
| JSON/CSV pipes | today | `report --format json\|csv` into your own tooling or BI. |
| In-tree parser (new data source) | today | Add one Go package under `internal/parser/`, with golden and fuzz tests; merge via PR. |
| Out-of-tree Go plugin API (library import, dynamically loaded) | planned | A public API for connectors, metrics, and rules, loaded without a rebuild; see [Custom metrics](#custom-metrics-whats-shipped-vs-roadmap) and [`ROADMAP.md`](../ROADMAP.md). |

---

## Honesty constraints for every extension

`assaio`'s product promise is **measure value, not people; honest statistics or
nothing** (`AGENTS.md`, `CONTRIBUTING.md`). That promise is not a built-in-only
courtesy — it binds every extension whose output a person reads as a metric or a
dashboard section: an in-tree validator, a community PR, or a private fork's own
validator file. Concretely:

- **Directional, not authoritative.** A `Read` (`Strong`/`Watch`/`Healthy`/…) is a
  diagnostic signal, not a verdict. If the evidence behind your metric is contested,
  incomplete, or a proxy for the thing you actually care about, say so in `HowToRead` or
  a `Caveat` — the word "directional" belongs in your rendered text, not just in this
  document.
- **`—` for an undefined ratio, never a fabricated one.** Divide-by-zero is a dash, not a
  zero or a 100% — use `shareOrDash`/`perActiveDay` (`internal/analyze/format.go`) or the
  same pattern by hand. A metric that reports "0%" when it actually has no denominator to
  divide by is a lie dressed as a number. This holds even when an underlying aggregate's
  own zero-denominator default is `0` (e.g. `report.ChurnStat.ReworkRate`) — a `Figure`
  must still check the raw denominator itself rather than formatting that default
  directly (see `internal/analyze/rework.go`'s "rework" figure, which reads
  `ReworkLines`/`LinesAdded` via `shareOrDash` instead of formatting `ReworkRate`).
- **Aggregate and pseudonymized by default; per-person only as a governed opt-in.**
  `Input` carries no user identity today — it groups by project, tool, model, and
  entrypoint, never by person, so a validator that ranks something ranks *those*
  dimensions, the same way `throughput` ranks projects, never individuals. If your
  `Bars` rank by **project name**, set `Result.BarsAreProjects = true` so the dashboard's
  `--anonymize` (on by default) pseudonymizes those labels exactly like it does for the
  built-in `throughput` validator — this is enforced generically by
  `internal/dashboard.anonymizeVerdicts`, not hardcoded to any one validator's name, so it
  applies to your validator too. Leave it `false` for any other dimension (models, tools,
  …); those must never be pseudonymized. A future per-member breakdown is only ever a
  deliberate, consented, team-mode opt-in — never silent, never a leaderboard, never
  built for individual performance evaluation.
- **Say so when you approximate.** If your metric can't observe something precisely from
  the stored aggregate — `Input.Usage` is already grouped (see [What a validator reads:
  Input](#what-a-validator-reads-input)), so per-record detail is gone — label the figure
  as approximate in the rendered text rather than presenting it as exact.
- **Never a per-person scoreboard.** Even in team mode, an extension must not turn
  individual usage into a ranked, named list presented as a performance signal. See
  `PRIVACY.md`.

These are the same rules `internal/analyze`'s built-in validators are held to and tested
against (`TestValidatorsEmptyInputSafe`, `TestReworkDashOnZeroToolCalls`,
`TestBuildNeverAnonymizesModelNames` in the test suite) — a code review of a new
validator should hold it to the same bar.

---

## Adding a metric validator

Every block `assaio analyze` prints — adoption, model fit, context health, throughput,
rework — is a `Validator` under
[`internal/analyze/`](../internal/analyze/analyze.go). This is the in-tree,
available-today realization of "one metric = one file" from
[`AGENTS.md`](../AGENTS.md) and [`CONTRIBUTING.md`](../CONTRIBUTING.md): a new metric is
one file, self-registering, with no central list to edit — and because the HTML
dashboard renders every registered validator's `Result` generically (see
[Verified](#verified-it-appears-in-the-cli-and-the-dashboard-automatically) below), it is
also a new dashboard section for free.

**How you actually ship one today — two ways.** An **in-tree validator** (this section)
is compiled into the `assaio-agent` binary: add your file under `internal/analyze/` in
your own fork or a private overlay of this repo, `make build`, and distribute the
resulting `bin/assaio-agent` (or upstream it via PR,
[`CONTRIBUTING.md`](../CONTRIBUTING.md) — the intended path for a metric with broad
value). `internal/` is enforced by the Go compiler itself to be unimportable from
outside this module, so a validator can never be a separate Go dependency. The
**out-of-tree alternative needs no fork at all**: a [metric
plugin](#write-a-metric-plugin-any-language) is a standalone executable in any language,
declared under `metrics:` in `config.yaml`, that reads the same prepared `Input` as JSON
and returns one `Result` — it appears in `analyze` and on the dashboard beside the
built-ins, namespaced `plugin:<name>`. Reach for in-tree when the metric belongs
upstream or needs data the wire envelope doesn't carry; reach for the plugin when the
metric is yours alone.

### What a validator reads: `Input`

```go
type Input struct {
	Usage      []store.UsageRow
	Sessions   []store.SessionRow
	Prices     pricing.Table
	Now        time.Time
	Recent     time.Duration
	Delegation Delegation
	ByModel    []ModelStat
	ByProject  []ProjectStat
	Totals     Totals
}
```

`Input` is a read-only bundle; use only the fields your metric needs. `Analyze` must stay
a **pure function of `Input`** — no `time.Now()`, no file or network I/O, no reaching
into the store yourself — which is what makes it trivial to unit test and safe to run
identically from the CLI, the dashboard, and (see [The team
server](#the-team-server)) the served endpoint.

| Field | Type | What it is |
|-------|------|------------|
| `Usage` | `[]store.UsageRow` | The window's usage, **pre-aggregated** by `(day, tool, model, project, entrypoint, member)` — one row per combination, tokens and activity counts summed (`internal/store/store.go`'s `Usage` query). Not one row per raw event: there is no per-record or per-file detail left at this point (see the [say-so-when-approximating](#honesty-constraints-for-every-extension) rule). Each row carries `Day` (`"YYYY-MM-DD"`), `Tool`, `Model`, `Project`, `Entrypoint`, `Member`; token fields `In, Out, CacheRead, CacheWrite, Reasoning`; and activity fields `LinesAdded, LinesRemoved, Edits, ToolCalls, Rejected, Compactions, ReworkLines` (see the [`usage.Record` contract](#the-usagerecord-contract) for what each counts). |
| `Sessions` | `[]store.SessionRow` | One row per `(session_id, member)` in the window: `Project`, `Tool`, `Model`, `FirstTs`/`LastTs`, `Turns`, `OutputTokens`, `PeakContextTokens`, `Edits`, `Compactions`, and `ActiveMinutes` (focused time — inter-turn gaps over 30 minutes are excluded, so a resumed session's idle time never counts as work; `internal/store/sessions.go`). |
| `Prices` | `pricing.Table` | `map[string]pricing.Price{Input, Output, CacheWrite, CacheRead float64}`, USD per token, the vendored LiteLLM snapshot. Indexing a model absent from the table returns a zero-value `Price` with **no error** — check the map's `ok` return if an unpriced model must be excluded from a cost figure rather than silently priced at $0. |
| `Now` | `time.Time` | Wall-clock at CLI invocation. Use this, never call `time.Now()` yourself. |
| `Recent` | `time.Duration` | The recent-vs-prior comparison window (7 days from the CLI today) — use it for trend/staleness splits the way `adoption` and `throughput` do. |
| `Delegation` | `Delegation{Sub, Total int64}` | Real sub-agent token-delegation share for the window: `Sub` is tokens on records whose `dedupe_key` marks a Task sub-agent turn, `Total` is every token in the same window. Computed once by the CLI via `store.Store.Delegation` so validators never reach into the store themselves. |
| `ByModel` | `[]ModelStat` | `Usage` aggregated per model, already tier-classified and priced. **Read this instead of grouping `Usage` by model yourself.** See the table below. |
| `ByProject` | `[]ProjectStat` | `Usage` aggregated per project. **Read this instead of grouping `Usage` by project yourself.** See the table below. |
| `Totals` | `Totals` | `Usage`'s grand totals across every model and project. See the table below. |

#### Read these first: `ByModel`, `ByProject`, `Totals`

`BuildInput` (`internal/analyze/prepared_build.go`) computes these three once, before any
validator runs, from the same `Usage` rows above. **Most validators — built-in or
custom — should read one of these instead of re-grouping `Usage` by hand or importing
`internal/report`.** `model-fit` is the reference: it used to call
`report.BuildEffectiveness(in.Usage, in.Prices, "model")` and then loop over the result to
classify each model's tier by price; it now reads `in.ByModel` directly and imports
neither `internal/report` nor `internal/pricing` at all (compare
[`internal/analyze/model_fit.go`](../internal/analyze/model_fit.go) to the description
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
hermetic](#parsers-stay-hermetic-project-resolution-is-ingests-job)) can't be built from
stored data today. Open an issue describing the signal; that is exactly the kind of
request that shapes `Input` before an out-of-tree interface is ever frozen.

### What a validator returns: `Result`

```go
type Result struct {
	Name, Title, Describe string
	Read            Read
	Purity          float64
	HowToRead       string
	Figures         []Figure
	Bars            []Bar
	BarsAreProjects bool
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
| `Title` | Human label. | Header line: `"WEEKEND-USAGE · Weekend Usage  [WATCH]"`. | Faceplate cell label (`"06 · Weekend Usage"`) and the ledger entry's label. |
| `Describe` | One-line summary. | `analyze --list`'s third column only — **not** printed by `RenderResultText`. | Not rendered. |
| `Read.Key` | `"good"`, `"watch"`, or `"neutral"` (no data). | Drives no CLI styling (plain text). | Drives the dashboard's color via CSS classes `cell__read--{key}` / `entry__read--{key}` (verdigris/oxide/muted) — an unrecognized key renders unstyled, so stick to these three. |
| `Read.Label` | Short upper-cased word, e.g. `"WATCH"`, `"STRONG"`, `"—"`. | Printed in `[brackets]` on the header line. | Printed as the faceplate/ledger read text. |
| `Purity` | `0..1`, how "well-used" this dimension reads. | **Not rendered.** | The faceplate gauge's fill width and the small `0.XX` ratio underneath — clamp to `[0,1]` yourself (`clamp01`) or the CSS width silently over/underflows. |
| `HowToRead` | One-sentence explainer of what the metric means and what to do about it. Must be non-empty on **every** code path, including the no-data one. | The `"  ? …"` line under the header. | The ledger entry's muted "How to read — …" line. |
| `Figures` | The headline numbers: `{Label, Value, Note}`. | One `"  label: value (note)"` line each. | One stat tile per figure in `.entry__stats`; **the first figure gets an accent color** — order your most important number first. Not shown in the faceplate. |
| `Bars` | Optional ranked list: `{Label, Value, Frac 0..1}`. Three states matter: `nil` → no Bars section anywhere; non-nil empty → an honest "none in this window" line; non-nil non-empty → a ranked bar list. | ASCII bar `label: value  [####----]` (20 chars wide, scaled by `Frac`). | `.projectbars` list, bar width from `Frac`. Scale `Frac` against that list's own max (`fracOf`), not a global scale. |
| `BarsAreProjects` | Whether `Bars`' labels are **project names**. | Not rendered. | Tells `--anonymize` whether to pseudonymize `Bars` labels — see [Honesty constraints](#honesty-constraints-for-every-extension). Get this wrong in the "true" direction and you scramble a label that was never PII (e.g. a model name); get it wrong in the "false" direction and a real project name leaks into a report meant to be shared. |
| `Takeaway` | One-line plain-language conclusion. Always populated, even on "no data". | Last line: `"  Takeaway: …"`. | `.entry__takeaway`, prefixed with an em dash. |
| `Caveats` | Honesty notes (directional, contested, approximate, …). | Each on its own `"  …"` line. | Each becomes a muted "Note — …" paragraph, **and** any non-empty `Caveats` adds a small "Prov." (provenance) badge next to the Read label on both the faceplate cell and the ledger entry. |

### Register it: the `Validator` interface

```go
type Validator interface {
	Name() string         // kebab-case slug, e.g. "model-fit" -- the CLI arg and JSON key
	Title() string        // human label for the report header
	Describe() string     // one line, shown by `assaio analyze --list`
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

### Verified: it appears in the CLI and the dashboard automatically

This is not a claim taken on faith — it was verified end to end while writing this
document, using a throwaway `weekend-usage` validator (the same metric turned into the
[worked example](#worked-example-weekend-usage) below), then deleted so the tree stays
clean. With the validator registered and `assaio-agent` rebuilt:

```console
$ assaio-agent analyze --list
adoption       Adoption & Usage Breadth         Sessions, active days, and project/tool breadth: how broad AI usage is, and whether it's growing.
context        Context Health                   Conversation depth, peak context size, active time, and how often sessions hit compaction.
model-fit      Model Fit                        Premium vs. cheaper model token share, lines-per-token contrast, and real sub-agent delegation share.
rework         Rework & Rejection               Within-session code churn and human tool-call rejections -- a directional friction proxy.
throughput     Throughput                       Total AI-added lines, lines per active day, top projects by lines, and the week-over-week trend.
weekend-usage  Weekend Usage                    Share of AI tokens run on Saturday/Sunday -- an out-of-hours usage signal.

$ assaio-agent analyze weekend-usage
WEEKEND-USAGE · Weekend Usage  [WATCH]
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
  <div class="cell__foot">
    <div class="gauge"><span class="gauge__fill" style="--fill:19.6%"></span></div>
    <span class="cell__ratio tnum">0.20</span>
  </div>
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
driven by the `Result.BarsAreProjects` field described above, applied generically by
`internal/dashboard.anonymizeVerdicts` to any validator — see [Honesty
constraints](#honesty-constraints-for-every-extension).

### Conventions and lint

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
  `readFor(ok, favorableLabel)` / `noDataRead` for `Read`, `shareOrDash`/`perActiveDay`
  for `—`-safe ratios, `clamp01` for `Purity`, `fracOf` for `Bar.Frac`, `groupLabel` for
  an empty dimension value.
- Reach for `in.ByModel`/`in.ByProject`/`in.Totals` before `in.Usage` — see [Read these
  first](#read-these-first-bymodel-byproject-totals) above. If your metric ends up
  grouping `Usage` by model or project itself, that is usually a sign the prepared views
  already have what you need.
- Give the file a test that seeds a small `Input`, calls `Analyze(...)`, renders it with
  `RenderResultText`, and asserts the figures/read you expect — plus a zero-value
  `Input{}` case: no panic, the honest "no data" block, never a favorable read computed
  from nothing (see `TestValidatorsEmptyInputSafe` in `internal/analyze/validators_test.go`
  for the pattern every built-in validator is held to).

### Worked example: Weekend Usage

A realistic company-specific metric: what share of AI token usage falls on a Saturday or
Sunday — an out-of-hours/DevEx signal a security or engineering-management team might
want that has no reason to be a built-in. (A per-**file** metric like "share of edits
touching test files" is *not* possible from stored data today — file paths are read
transiently during ingest and never persisted, only aggregate counts are, per [Parsers
stay hermetic](#parsers-stay-hermetic-project-resolution-is-ingests-job). This example
uses a day-level signal instead, which `Input.Usage`'s `Day` field already supports.)

`internal/analyze/weekend_usage.go`:

```go
package analyze

import (
	"strconv"
	"time"

	"github.com/assaio/assaio/internal/store"
)

const (
	weekendUsageName     = "weekend-usage"
	weekendUsageTitle    = "Weekend Usage"
	weekendUsageDescribe = "Share of AI tokens run on Saturday/Sunday -- an out-of-hours usage signal."
	// weekendUsageHowToRead is Result.HowToRead for this validator -- see its doc comment.
	weekendUsageHowToRead = "A rising weekend share can mean crunch time or just flexible hours -- read it next to team sentiment, not as a verdict on its own."
	// weekendUsageWatchCeiling is the weekend-token-share threshold above which usage is
	// flagged for a closer look.
	weekendUsageWatchCeiling = 0.2
)

func init() { Register(weekendUsageValidator{}) }

// weekendUsageValidator reads what share of AI token usage falls on a Saturday or
// Sunday -- a company-specific out-of-hours/DevEx signal, not a built-in metric.
type weekendUsageValidator struct{}

func (weekendUsageValidator) Name() string     { return weekendUsageName }
func (weekendUsageValidator) Title() string    { return weekendUsageTitle }
func (weekendUsageValidator) Describe() string { return weekendUsageDescribe }

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (weekendUsageValidator) Analyze(in Input) Result {
	r := Result{Name: weekendUsageName, Title: weekendUsageTitle, Describe: weekendUsageDescribe, HowToRead: weekendUsageHowToRead}
	if len(in.Usage) == 0 {
		r.Read = noDataRead
		r.Takeaway = "No usage in this window."
		return r
	}

	weekendTokens, weekdayTokens, weekendLines := weekendTotals(in.Usage)
	total := weekendTokens + weekdayTokens
	var weekendShare float64
	if total > 0 {
		weekendShare = float64(weekendTokens) / float64(total)
	}
	watch := weekendShare > weekendUsageWatchCeiling

	r.Read = readFor(!watch, "Low")
	r.Purity = clamp01(1 - weekendShare)
	r.Figures = []Figure{
		{Label: "weekend token share", Value: shareOrDash(weekendTokens, total, 1)},
		{Label: "weekend AI lines", Value: strconv.FormatInt(weekendLines, 10)},
	}
	r.Caveats = []string{"Directional: a proxy for out-of-hours work, not a burnout measurement."}
	r.Takeaway = weekendUsageTakeaway(watch)
	return r
}

func weekendUsageTakeaway(watch bool) string {
	if watch {
		return "A meaningful share of usage falls on weekends -- worth checking in on workload."
	}
	return "Weekend usage is a small share of the total."
}

// weekendTotals sums token/line totals split by whether UsageRow.Day falls on a Saturday
// or Sunday.
func weekendTotals(usage []store.UsageRow) (weekendTokens, weekdayTokens, weekendLines int64) {
	for i := range usage {
		u := &usage[i]
		tokens := u.In + u.Out
		if isWeekend(u.Day) {
			weekendTokens += tokens
			weekendLines += u.LinesAdded
			continue
		}
		weekdayTokens += tokens
	}
	return weekendTokens, weekdayTokens, weekendLines
}

// isWeekend reports whether day (YYYY-MM-DD) is a Saturday or Sunday. An unparseable day
// (should not happen -- Day is stamped by the store) is treated as a weekday rather than
// silently inflating the weekend share.
func isWeekend(day string) bool {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return false
	}
	wd := t.Weekday()
	return wd == time.Saturday || wd == time.Sunday
}
```

`internal/analyze/weekend_usage_test.go`:

```go
package analyze

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

func TestWeekendUsageWatchOnHighWeekendShare(t *testing.T) {
	in := Input{
		Now: validatorsTestNow, Recent: 7 * 24 * time.Hour, Prices: testPrices(),
		Usage: []store.UsageRow{
			{Day: "2026-07-11", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", In: 900, Out: 900, LinesAdded: 40}, // Saturday
			{Day: "2026-07-13", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", In: 100, Out: 100, LinesAdded: 5},  // Monday
		},
	}
	v, ok := Get("weekend-usage")
	if !ok {
		t.Fatal(`validator "weekend-usage" not registered`)
	}
	var buf bytes.Buffer
	if err := RenderResultText(&buf, v.Analyze(in)); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "[WATCH]") {
		t.Fatalf("high weekend share output missing [WATCH]:\n%s", out)
	}
	if !strings.Contains(out, "weekend token share: 90.0%") {
		t.Fatalf("weekend token share figure wrong:\n%s", out)
	}
}

func TestWeekendUsageLowShareIsFavorable(t *testing.T) {
	in := Input{
		Now: validatorsTestNow, Recent: 7 * 24 * time.Hour, Prices: testPrices(),
		Usage: []store.UsageRow{
			{Day: "2026-07-13", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", In: 900, Out: 900, LinesAdded: 40}, // Monday
			{Day: "2026-07-11", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", In: 100, Out: 100, LinesAdded: 5},  // Saturday
		},
	}
	v, _ := Get("weekend-usage")
	var buf bytes.Buffer
	if err := RenderResultText(&buf, v.Analyze(in)); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "[LOW]") {
		t.Fatalf("low weekend share output missing [LOW]:\n%s", out)
	}
}

func TestWeekendUsageEmptyInputSafe(t *testing.T) {
	v, _ := Get("weekend-usage")
	var buf bytes.Buffer
	if err := RenderResultText(&buf, v.Analyze(Input{})); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No usage in this window.") {
		t.Fatalf("empty input must render the no-data hint, got %q", buf.String())
	}
}
```

`go test ./internal/analyze/... -run Weekend -v` passes all three cases. This validator
does not set `BarsAreProjects` (it has no `Bars` at all) and does not need `Delegation`,
`Sessions`, or `Prices` — a validator only touches the `Input` fields its metric needs.

---

## Custom log-source paths

For a team whose logs don't live at the built-in default path — a custom install
location, an OS-variant path the defaults don't cover, a synced or mounted home
directory, an external volume, or a CI runner with a non-standard `HOME` — the fix is a
config change, not code. `internal/paths.Resolve` (see
[`internal/paths/resolve.go`](../internal/paths/resolve.go)) backs every tool's root
resolution: a non-empty `sources.<tool>` list in `config.yaml` **replaces** the built-in
default roots entirely for that tool (never merged with them), so the result is always
exactly what you configured; an empty or omitted list keeps the default.

```yaml
# ~/.config/assaio/config.yaml (honors XDG_CONFIG_HOME)
sources:
  claude:
    - /Volumes/work/.claude/projects   # e.g. Claude Code logging to an external volume
  codex: []                            # default: ~/.codex/sessions, ~/.codex/archived_sessions
  gemini: []                           # default: ~/.gemini
  cline: []                            # default: VS Code global storage, and ~/.cline/data
```

Each tool accepts a **list** of roots — set more than one when a team has usage spread
across two locations (e.g. a laptop's default path plus an old profile directory that
hasn't been cleaned up yet):

```yaml
sources:
  claude:
    - ~/.claude/projects
    - /Volumes/archive/old-laptop/.claude/projects
```

Override per-tool from the environment instead of a file with `ASSAIO_SOURCES_<TOOL>`
(one root per variable — use the YAML list form above for more than one root), e.g.
`ASSAIO_SOURCES_CLAUDE=/Volumes/work/.claude/projects`. Environment variables win over
the config file, which wins over the built-in default (see `internal/config`'s
precedence: defaults < file < `ASSAIO_*` env < flags).

Verify what's actually in effect with `assaio-agent doctor`: it reports every tool's
resolved roots, whether each is the built-in default or config-overridden, and flags a
configured root that doesn't exist on disk — so a typo'd path fails loudly instead of
silently importing nothing.

This surface changes *where* the existing parsers look; it does not change what they
parse. To make `assaio` understand a log format it doesn't already know, see [Add a data
source](#add-a-data-source) (in-tree) or [Write a plugin](#write-a-plugin-any-language)
(out-of-tree).

---

## Write a plugin (any language)

**When to reach for this instead of a validator.** A [metric
validator](#adding-a-metric-validator) only *reads* usage that is already in the store —
it cannot manufacture tokens, lines, or sessions that were never ingested. Reach for a
plugin when the gap is upstream of that: an entirely new **tool** `assaio` has no parser
for yet (an internal AI tool, a vendor not covered by a built-in parser). A plugin's job
is narrow and specific — discover that tool's logs and emit normalized `usage.Record`
rows into the store — after which every existing surface (`report`, `effectiveness`,
`analyze`, `dashboard`, and any validator you've added) sees its data like any other
source. If the tool is one your organization alone uses, a plugin is almost always the
right call over an in-tree parser PR, since it needs no review from this project and no
release wait.

An exec plugin is an executable that discovers and parses one tool's usage data itself
and emits normalized records to stdout. `assaio` runs it as a subprocess during
`backfill`, validates every line, and stores what passes. The contract is the data
format below — there is no Go to link against and no library version to track. The core
lives under Go's `internal/`, which the compiler forbids any external module from
importing, because freezing a public Go API before v1.0 would bind us to its shape under
semver while the data model is still moving (see [ADR 0003](adr/0003-exec-plugin-protocol.md)).
An exec plugin's contract is the **data format** instead — a handshake line and JSONL
records over stdout — so you can write one in Python, Rust, or a shell script, and
nothing you depend on breaks when the core refactors.

Plugins are **opt-in only**: they run exclusively when declared in
`~/.config/assaio/config.yaml`. `assaio` never scans `PATH`, never auto-discovers, and
never downloads plugins.

```yaml
plugins:
  - name: mytool            # required, [a-z0-9-]+; records are stored as tool "plugin:mytool"
    command: /path/to/assaio-parser-mytool   # required; resolved via PATH lookup if not absolute
    timeout: 60s            # optional, default 60s
```

### The protocol

`assaio` invokes `<command> scan` with `ASSAIO_PLUGIN_PROTOCOL=1` in the environment.
The plugin writes to stdout:

1. **Handshake** (line 1): `{"assaio_plugin": 1, "tool": "<name>"}`. The protocol
   version must be `1` and `tool` must equal the configured `name`; any mismatch fails
   the run.
2. **Records** (every following line): one JSON object per line, snake_case:

```json
{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"some-model","input_tokens":100,"output_tokens":200,"cache_read_tokens":0,"cache_write_tokens":0,"reasoning_tokens":0,"dedupe_key":"s1:0","project":"myrepo","git_branch":"main","entrypoint":"cli","granularity":"turn"}
```

Required: `session_id`, `timestamp` (RFC3339), `model`, `dedupe_key`, and `granularity`
(`turn` or `session` — the [granularity honesty rule](#granularity-honesty-hard-rule)
applies to plugins exactly as it does to in-tree parsers). Token fields default to 0;
`project`, `git_branch`, and `entrypoint` are optional. The same
[`usage.Record` contract](#the-usagerecord-contract) rules apply: `project` is a
directory **basename**, never a full path, and `dedupe_key` must be
[deterministic](#dedupekey-determinism-hard-rule) so re-runs never double-count.

Anything the plugin writes to stderr passes through to `assaio`'s stderr prefixed with
`[plugin/<name>] `, so diagnostics stay attributable.

### What the boundary enforces

`assaio` validates every record line and **skips** (and counts) any line with a negative
token field, an empty `dedupe_key`, an unparseable timestamp, or an invalid
`granularity` — the same skip-and-count policy in-tree parsers apply to corrupt log
lines. Stored records get the tool label `plugin:<name>`, so a plugin can never
impersonate a built-in source and its dedupe keyspace `(tool, dedupe_key)` never
collides with anyone else's. A plugin that exits non-zero, times out, or fails the
handshake is reported as failed for that run; the rest of the backfill continues.
Stdout is capped at 64 MiB per run.

### A complete example (Python)

```python
#!/usr/bin/env python3
"""assaio-parser-mytool: emit usage records for the fictional mytool CLI."""
import json, sys
from pathlib import Path

print(json.dumps({"assaio_plugin": 1, "tool": "mytool"}))

for log in sorted(Path.home().glob(".mytool/sessions/*.jsonl")):
    for i, line in enumerate(log.read_text().splitlines()):
        entry = json.loads(line)
        print(json.dumps({
            "session_id": entry["session"],
            "timestamp": entry["ts"],            # RFC3339
            "model": entry["model"],
            "input_tokens": entry["in_tokens"],
            "output_tokens": entry["out_tokens"],
            "dedupe_key": f'{entry["session"]}:{i}',
            "granularity": "turn",
        }))
```

Make it executable, add it to `config.yaml` as shown above, and check conformance —
`plugins verify` runs the plugin and validates the full stream **without storing
anything**:

```console
$ assaio-agent plugins verify mytool
mytool: handshake OK
records ok: 42
skipped:    1
violations:
  line 17: empty dedupe_key
$ assaio-agent plugins list
mytool            /path/to/assaio-parser-mytool  (timeout 1m0s)
```

Once `verify` is clean, `assaio-agent backfill` ingests the plugin after the built-in
sources and reports a `plugin:mytool` line alongside them.

---

## Write a metric plugin (any language)

**When to reach for this instead of an in-tree validator.** A metric plugin is your own
**analyzer** without forking assaio: an executable in any language that reads the same
prepared `Input` bundle every built-in validator reads and returns one `Result`. It
renders in `assaio analyze`, `analyze --format json`, and the Assay dashboard beside the
built-ins — same faceplate cell, same ledger entry, same anonymization rules. Reach for
it when the metric is company-specific and has no reason to be upstreamed; reach for an
[in-tree validator](#adding-a-metric-validator) when it belongs in every install or
needs domain data the wire envelope doesn't carry yet.

Metric plugins are **opt-in only**, declared under `metrics:` in
`~/.config/assaio/config.yaml` — never discovered from `PATH`, never downloaded. The
entry shape is the same as `plugins:`, and one binary may appear in both lists, serving
both protocols (`scan` and `analyze` argv):

```yaml
metrics:
  - name: weekend-usage      # required, [a-z0-9-]+; appears as "plugin:weekend-usage"
    command: /path/to/assaio-metric-weekend   # required; PATH lookup if not absolute
    timeout: 30s             # optional, default 60s
```

**One privacy note before the protocol.** Unlike a parser plugin (which reads a tool's
own logs), a metric plugin **receives your usage data on stdin**: project names, model
names, member pseudonyms, and token/line counts — the same aggregate metadata the store
holds, never prompts or code (those are never collected at all, see `PRIVACY.md`). The
trust model is unchanged — a plugin is a local program you chose to run, with your own
privileges — but know what crosses the process boundary before pointing config at a
binary you didn't write.

### The protocol

`assaio` invokes `<command> analyze` with `ASSAIO_METRIC_PROTOCOL=1` in the environment,
writes one JSON envelope to the plugin's **stdin**, closes it, and reads stdout.

**stdin** — the prepared `Input`, versioned, camelCase (mirroring the public
`analyze --format json` shapes; only the version keys stay snake_case, matching the
parser protocol's `assaio_plugin`):

```json
{
  "assaio_metric_input": 1,
  "now": "2026-07-17T10:00:00Z",
  "recentDays": 7,
  "usage":    [{"day":"2026-07-16","tool":"claude-code","model":"...","project":"...",
                "entrypoint":"","member":"","in":100,"out":200,"cacheRead":0,
                "cacheWrite":0,"reasoning":0,"linesAdded":40,"linesRemoved":5,
                "edits":3,"toolCalls":7,"rejected":1,"compactions":1,"reworkLines":2}],
  "sessions": [{"sessionId":"...","project":"...","tool":"...","model":"...",
                "member":"","firstTs":"...","lastTs":"...","turns":4,
                "outputTokens":200,"peakContextTokens":1100,"edits":3,
                "compactions":1,"activeMinutes":42.5}],
  "delegation": {"sub":0,"total":0},
  "byModel":   [{"model":"...","tier":"premium","tokens":0,"input":0,"output":0,
                 "cacheRead":0,"cacheWrite":0,"lines":0,"cost":1.23,"priced":true,
                 "tokenShare":0.5}],
  "byProject": [{"project":"...","lines":0,"cost":null,"priced":false,"tokenShare":0.5}],
  "totals":    {"tokens":0,"input":0,"output":0,"cacheRead":0,"cacheWrite":0,"lines":0,
                "cost":null,"priced":false,"cacheEfficiency":0.9},
  "prices":    {"claude-opus-4-8":{"input":0.000015,"output":0.000075,
                "cacheRead":0.0000015,"cacheWrite":0.00001875}}
}
```

The semantics are exactly [`Input`'s](#what-a-validator-reads-input): `usage` is
pre-aggregated by `(day, tool, model, project, entrypoint, member)`; `cost` fields are
`null` when unpriced, never a fabricated `0`; `byModel`/`byProject`/`totals` are the
prepared views to read first; `prices` carries only models present in the window's
usage. Like `Input`, the envelope is **versioned but pre-1.0 unstable** — a release that
reshapes it says so explicitly (see `RELEASING.md`).

**stdout** — a one-line handshake, then exactly **one** JSON `Result` document
(pretty-printed is fine; anything after it is a violation):

1. `{"assaio_metric": 1, "name": "<name>"}` — version must be `1`, `name` must equal
   the configured name.
2. One `Result` in the same shape `analyze --format json` emits — see [What a validator
   returns: Result](#what-a-validator-returns-result). The wire `name` field is ignored:
   assaio always stamps `plugin:<name>`, so a plugin can never shadow a built-in
   validator.

Anything written to stderr passes through prefixed `[metric/<name>] `.

### What the boundary enforces

The honesty rules are enforced, not requested. A result that fails **any** check is
rejected whole — assaio never renders a fabricated or partially-sanitized verdict. On a
bare `analyze`/`dashboard` run a failing plugin is skipped with one `warning:` line and
the built-ins still render; an explicitly selected one (`analyze plugin:<name>`) is a
hard error.

- `read.key` must be `good`, `watch`, or `neutral`; `read.label` non-empty, ≤ 16 chars.
- `title` (≤ 80), `howToRead` and `takeaway` (≤ 400) are required; `describe` (≤ 200)
  and `caveats` (≤ 400 each, max 8) optional.
- Figures max 12, bars max 30; their `label`/`value`/`note` ≤ 120 chars each.
- No control characters anywhere (terminal-escape guard; the dashboard's HTML escaping
  is separate and automatic).
- `purity` and every `bars[].frac` are clamped to `[0,1]`.
- Stdout is capped at 1 MiB; the run is killed after `timeout` (default 60s).
- `barsAreProjects` works exactly as for built-ins: set it `true` when your `Bars` rank
  project names and the dashboard's `--anonymize` pseudonymizes them; the [honesty
  constraints](#honesty-constraints-for-every-extension) bind a metric plugin the same
  as any in-tree validator.

### A complete example (Python)

The same weekend-usage metric as the [in-tree worked
example](#worked-example-weekend-usage) — one metric, both extension paths:

```python
#!/usr/bin/env python3
"""assaio-metric-weekend: share of AI tokens used on Saturday/Sunday."""
import json, sys
from datetime import date

inp = json.load(sys.stdin)
weekend = total = 0
for row in inp["usage"]:
    tokens = row["in"] + row["out"]
    total += tokens
    if date.fromisoformat(row["day"]).weekday() >= 5:
        weekend += tokens

print(json.dumps({"assaio_metric": 1, "name": "weekend-usage"}))

if total == 0:
    print(json.dumps({
        "title": "Weekend Usage",
        "read": {"key": "neutral", "label": "—"},
        "howToRead": "A rising weekend share can mean crunch time or just flexible hours -- read it next to team sentiment, not as a verdict on its own.",
        "takeaway": "No usage in this window.",
    }))
    sys.exit(0)

share = weekend / total
watch = share > 0.2
print(json.dumps({
    "title": "Weekend Usage",
    "read": {"key": "watch" if watch else "good", "label": "WATCH" if watch else "LOW"},
    "purity": 1 - share,
    "howToRead": "A rising weekend share can mean crunch time or just flexible hours -- read it next to team sentiment, not as a verdict on its own.",
    "figures": [{"label": "weekend token share", "value": f"{share:.1%}"}],
    "takeaway": "A meaningful share of usage falls on weekends -- worth checking in on workload."
                if watch else "Weekend usage is a small share of the total.",
    "caveats": ["Directional: a proxy for out-of-hours work, not a burnout measurement."],
}))
```

Make it executable, declare it under `metrics:` as above, and check conformance —
`metrics verify` runs the plugin on your real store's window **without storing
anything** and prints the violations, if any, plus the rendered result:

```console
$ assaio-agent metrics verify weekend-usage
weekend-usage: handshake OK
result: VALID
PLUGIN:WEEKEND-USAGE · Weekend Usage  [WATCH]
  ? A rising weekend share can mean crunch time or just flexible hours -- read it next to team sentiment, not as a verdict on its own.
  weekend token share: 34.2%
  Directional: a proxy for out-of-hours work, not a burnout measurement.
  Takeaway: A meaningful share of usage falls on weekends -- worth checking in on workload.

$ assaio-agent metrics list
weekend-usage     /path/to/assaio-metric-weekend  (timeout 30s)
```

From then on a bare `assaio-agent analyze` prints it after the built-ins,
`assaio-agent analyze plugin:weekend-usage` runs it alone, `analyze --list` shows it
(without executing it), and `assaio-agent dashboard` gives it a faceplate cell and
ledger entry like any built-in's.

**Where it deliberately does not run:** the [team server](#the-team-server)'s served
dashboard (`GET /` is unauthenticated and rebuilds per request — spawning
config-declared subprocesses per request would be a denial-of-service vector), the
dashboard's per-project drill-down (built-ins only), and `demo` (deterministic sample).
See [ADR 0004](adr/0004-exec-metric-plugin-protocol.md) for the full rationale.

---

## The team server

`assaio-agent serve` runs a self-hosted team server: teammates' `assaio-agent sync` runs
push their local usage to it over HTTP, and it serves back one aggregated,
pseudonymized-by-default Assay dashboard for the whole team at `GET /`
(`internal/server`). This is an MVP — a single shared bearer token gates the *write*
endpoint (`POST /v1/usage`) only, there is no TLS, and **the served dashboard itself has
no auth at all** in this version — anyone who can reach the port sees it; run `serve`
behind a reverse proxy on a trusted network, not exposed to the open internet (see
`internal/server`'s package doc and the security note `serve` prints on startup).

The extension mechanism does not change at that boundary. `server.BuildDashboard`
(`internal/server/dashboard.go`) calls the exact same `dashboard.Build` the local
`assaio-agent dashboard` command calls, over the exact same process-wide
`analyze.Validators()` registry every validator self-registers into — there is no
separate server-side validator list. That means a custom validator compiled into your
team's `assaio-agent` build (see [Adding a metric validator](#adding-a-metric-validator))
shows up on the team server's dashboard automatically: same faceplate cell, same ledger
entry, same anonymization rules, with nothing to configure on the server side. The one
deliberate exception is [exec metric plugins](#write-a-metric-plugin-any-language):
`serve` never executes them, because its dashboard endpoint is unauthenticated and
rebuilt per request — they are a local-CLI surface (`analyze`, `dashboard`,
`metrics verify`; see [ADR 0004](adr/0004-exec-metric-plugin-protocol.md)). The one
difference from the local CLI is that the served dashboard's anonymization is not
optional — `BuildDashboard` hardcodes `anonymize = true`, so a real-name view is only
ever available locally, as an explicit `--no-anonymize` run against a copy of the store
(`assaio-agent dashboard --db <path-to-central-db> --no-anonymize`), never as the
served default.

```yaml
# on the server
server:
  addr: 127.0.0.1:8787    # loopback by default; widen deliberately
  token: ""    # required; override with ASSAIO_SERVER_TOKEN, do not commit a real one

# on each teammate's machine
sync:
  server: "http://assaio.internal:8787"
  token: ""    # override with ASSAIO_SYNC_TOKEN
  member: ""   # opt-in self-identification; default: an auto pseudonym from hostname+OS-user
```

---

## Query your own data

Everything `assaio` collects lives in one SQLite file:

```
~/.local/share/assaio/assaio.db
```

The location honors `XDG_DATA_HOME`. It is an ordinary SQLite database — point `sqlite3`,
DB Browser, or any client at it and query directly. `assaio` never phones home, so this
file is the whole of your data.

### Schema

There is one table, `usage_record`
([`internal/store/migrations/0001_init.sql`](../internal/store/migrations/0001_init.sql)):

| Column | Type | Notes |
|--------|------|-------|
| `id` | `INTEGER PRIMARY KEY` | Row id. |
| `tool` | `TEXT` | Source, e.g. `claude-code`, `codex`, `gemini-cli`, `cline`. |
| `session_id` | `TEXT` | The tool's session/conversation ID. |
| `ts` | `TEXT` | UTC RFC3339 timestamp. Day is `substr(ts,1,10)`. |
| `model` | `TEXT` | Model name as recorded by the tool. |
| `input_tokens` | `INTEGER` | Non-cached input tokens. |
| `output_tokens` | `INTEGER` | Output tokens. |
| `cache_read_tokens` | `INTEGER` | Tokens served from cache. |
| `cache_write_tokens` | `INTEGER` | Tokens written to cache. |
| `reasoning_tokens` | `INTEGER` | Reasoning tokens, when reported. |
| `dedupe_key` | `TEXT` | Unique with `tool` (`UNIQUE(tool, dedupe_key)`). |
| `project` | `TEXT` | Basename of the resolved git repository root, or `''`. Monorepo subdirectories share one value here. |
| `subpath` | `TEXT` | Working directory relative to that repository root (e.g. `apps/mobile`), or `''` at the root. |
| `git_branch` | `TEXT` | Branch name, or `''`. |
| `entrypoint` | `TEXT` | Invocation label, or `''`. |
| `granularity` | `TEXT` | `turn` or `session`. |
| `lines_added` | `INTEGER` | AI-added lines (from diff `+` markers), or `0`. |
| `lines_removed` | `INTEGER` | AI-removed lines (from diff `-` markers), or `0`. |
| `edits` | `INTEGER` | File-editing tool calls, or `0`. |
| `tool_calls` | `INTEGER` | All tool-use calls, or `0`. |
| `rejected` | `INTEGER` | Tool proposals the human declined, or `0`. |
| `compactions` | `INTEGER` | Context-compaction events attributed to the record, or `0`. |
| `rework_lines` | `INTEGER` | AI-added lines later undone within the same transcript+file, or `0`. |
| `member` | `TEXT` | `''` for purely local usage; non-empty only on a central store synced from a team member (see [The team server](#the-team-server)). |

The activity columns (`lines_added` … `rework_lines`) are populated by the Claude Code
and Codex parsers today, except `rejected`, which is Claude-Code-only; Gemini and Cline
store `0` throughout. They hold **counts only** — never the code content of the lines
they count. `report --format csv` covers tokens and cost; `effectiveness --format csv`
adds the activity and `$`/100-lines columns.

**Cost is not stored.** The database holds tokens only; dollar cost is computed at report
time against the embedded price table, because prices change and unpriced models must stay
honestly blank. For cost figures, use `assaio-agent report --format csv` (which carries a
`cost` column) rather than SQL.

**Stability.** The schema may still evolve before v1.0. Changes will be additive where
possible — new nullable columns rather than renames — but treat direct queries as coupled
to a version you have pinned, not a frozen contract. The report/JSON/CSV output is the
more stable surface.

### Ready-made queries

```sh
DB=~/.local/share/assaio/assaio.db
```

**Token spend per project, last 30 days** (the dimension behind `report --by project`;
join to your own price sheet for dollars, or use the CSV report):

```sh
sqlite3 -header -column "$DB" "
  SELECT project,
         SUM(input_tokens)      AS in_tok,
         SUM(output_tokens)     AS out_tok,
         SUM(cache_read_tokens) AS cache_read
  FROM usage_record
  WHERE ts >= date('now','-30 days')
  GROUP BY project
  ORDER BY out_tok DESC;"
```

**Total tokens per model:**

```sh
sqlite3 -header -column "$DB" "
  SELECT model, SUM(input_tokens + output_tokens) AS total_tok
  FROM usage_record
  GROUP BY model
  ORDER BY total_tok DESC;"
```

**Cache efficiency per project** — cache reads as a share of input + cache reads, the same
ratio the `Cache%` column shows:

```sh
sqlite3 -header -column "$DB" "
  SELECT project,
         ROUND(100.0 * SUM(cache_read_tokens)
               / NULLIF(SUM(input_tokens + cache_read_tokens), 0), 1) AS cache_pct
  FROM usage_record
  GROUP BY project
  ORDER BY cache_pct DESC;"
```

**Busiest days:**

```sh
sqlite3 -header -column "$DB" "
  SELECT substr(ts,1,10) AS day,
         SUM(input_tokens + output_tokens) AS total_tok,
         COUNT(*)                          AS records
  FROM usage_record
  GROUP BY day
  ORDER BY total_tok DESC
  LIMIT 10;"
```

---

## Add a data source

A data source is one Go package under `internal/parser/<tool>/`. It turns a tool's
on-disk session logs into a slice of normalized `usage.Record` values. That is the
entire job — pricing, aggregation, storage, and rendering are the core's responsibility,
not the parser's.

A parser exposes exactly two functions:

```go
// Discover returns the log files (or task directories) under one root this tool has
// written. Sources with more than one root (Codex, Cline) are looped by the caller —
// one Discover call per root, never a []string of roots.
func Discover(root string) ([]string, error)

// Parse reads one log and returns its normalized usage records, plus the count of
// lines that failed to unmarshal as JSON.
func Parse(r io.Reader) ([]usage.Record, int, error)
```

`Discover` is a filesystem glob rooted at a path the core resolves for you (see
[`internal/paths`](../internal/paths/paths.go)). Keep the glob narrow — `~/.gemini`, for
example, is shared with other tools, so the Gemini discoverer only matches
`tmp/*/chats/session-*.jsonl`. `Parse` takes an `io.Reader` (not a path) so it is trivial
to test against a fixture. A source whose unit of work is a directory rather than a single
file — Cline reads `ui_messages.json` alongside `task_metadata.json` — may expose a
`ParseDir(dir string) ([]usage.Record, int, error)` helper instead, but the file-oriented
`Parse(io.Reader)` shape is the default and the one to reach for first.

Where `Discover`'s root itself comes from — the built-in default, or a team's own
override — is a separate, non-code concern; see [Custom log-source
paths](#custom-log-source-paths).

#### Corrupt-line policy: skip and count

Session logs are live, append-only files a tool can be writing to while `assaio` reads
them — a truncated final line or one bad byte is expected, not exceptional. `Parse`
therefore never aborts a file over one malformed line: a line that fails `json.Unmarshal`
is counted in the returned `skipped` int and parsing continues, so the records on either
side of it are never lost to one corrupt entry. A log line that unmarshals fine but
carries no usage is simply *filtered*, not counted as skipped — only unmarshal failures
count. The scanner itself can still fail (e.g. `bufio.ErrTooLong` past `parser.MaxLineBytes`);
that is a structural problem with the whole file, not one line, so it is returned as an
error, wrapped with context. `internal/ingest.Run` mirrors this at the file level: a file
that cannot be opened or parsed at all is counted as `Failed` and the run continues with
the remaining files, so one corrupt log never blocks a `backfill` of the rest.

### The `usage.Record` contract

Every record you emit is one normalized usage event. The struct lives in
[`internal/usage/record.go`](../internal/usage/record.go); fill in what the log gives you
and leave the rest at its zero value.

| Field | Type | Meaning | Rules |
|-------|------|---------|-------|
| `Tool` | `string` | Stable identifier for the source, e.g. `"claude-code"`. | Constant per package. Becomes the `tool` column and pairs with `DedupeKey` for uniqueness. |
| `SessionID` | `string` | The tool's own session/conversation ID. | Pass through verbatim; do not synthesize. |
| `Timestamp` | `time.Time` | When the usage occurred. | Stored as UTC RFC3339. Parse the log's timestamp; do not use "now". |
| `Model` | `string` | Model name as the tool records it. | Pass through verbatim — normalization to the price table happens in the core. |
| `InputTokens` | `int64` | Non-cached input tokens. | If the log's input count **includes** cached tokens (Codex, Gemini do this), subtract them so input and cache-read never double-count. |
| `OutputTokens` | `int64` | Generated output tokens. | Fold tool-use tokens here only if the vendor bills them as output (Gemini); document the choice in a one-line comment and a `doctor` caveat. |
| `CacheReadTokens` | `int64` | Tokens served from prompt cache. | Feeds the `Cache%` column and cache-read pricing. |
| `CacheWriteTokens` | `int64` | Tokens written to prompt cache. | |
| `ReasoningTokens` | `int64` | Reasoning/thinking tokens, when reported separately. | Recorded for transparency; whether they are billed separately is model-dependent. |
| `DedupeKey` | `string` | Stable per-record identity within a `Tool`. | **Must be deterministic** — see below. |
| `Cwd` | `string` | The session's full working-directory path, exactly as the log reports it. | **Never persisted.** `internal/ingest` reads it only to resolve `Project`/`Subpath` (`internal/projectid`) and then discards it. Leave `""` if the log has no cwd — never fabricate one. |
| `Project` | `string` | The **basename of the git repository root** containing the session's working directory. | Set it as a **fallback only** — `filepath.Base(cwd)` — for when ingest cannot resolve a repository root (e.g. `Cwd` left `""`). Whenever `Cwd` is set, ingest overwrites this with the resolved repo-root basename, so a monorepo's subdirectories roll up to one project. |
| `Subpath` | `string` | `Cwd` relative to the resolved repository root (e.g. `apps/mobile`), or `""` at the root. | Set by ingest, not by parsers — leave it at its zero value. Always relative; never an absolute path. |
| `GitBranch` | `string` | Branch name, when the log carries it. | Else `""`. |
| `Entrypoint` | `string` | How the tool was invoked, e.g. `"cli"`, `"sdk-py"`. | Else `""`. |
| `Granularity` | `string` | `"turn"` for per-request records, `"session"` for session aggregates. | **Honesty rule** — see below. |
| `LinesAdded` | `int64` | AI-added code lines for this record — the primary effect proxy. | Count only the `+`-prefixed lines of the edit's diff hunks (or a sub-agent's reported added lines). **The code on the line is never stored — only the count.** `0` if the source exposes no diff. |
| `LinesRemoved` | `int64` | AI-removed code lines. | The `-`-prefixed diff lines, same rule. `0` if unknown. |
| `Edits` | `int64` | File-editing tool calls (`Edit`/`Write`/`NotebookEdit`/`MultiEdit`). | A subset of `ToolCalls`. `0` if the log does not expose tool calls. |
| `ToolCalls` | `int64` | All tool-use calls in the turn, edits included. | `0` if unknown. |
| `Rejected` | `int64` | Tool proposals the human declined — a friction signal. | `0` if unknown. |
| `Compactions` | `int64` | Context-compaction events attributed to this record — a context-strain signal. | `0` if the source exposes no compaction/summarization marker. |
| `ReworkLines` | `int64` | AI-added lines later removed by a subsequent edit to the same file, within one transcript — a rework/thrash proxy. | Computed via the shared [`internal/parser.Rework`](../internal/parser/rework.go) helper. The file path used to detect it is read transiently and **never** copied onto the record. `0` if unknown. |

Records with no token usage should be skipped, not emitted with zeros.

#### Parsers stay hermetic — project resolution is ingest's job

A parser's only filesystem access is the `io.Reader` `Parse` was handed. It must never
open, stat, or walk anything else — in particular, it must not import
`internal/projectid` or otherwise try to resolve `Cwd` to a repository root itself. Emit
`Cwd` verbatim from the log and, as a fallback `Project`, your own best guess (typically
`filepath.Base(cwd)`); `internal/ingest` re-resolves `Project` (and fills `Subpath`) for
every record after `Parse` returns, by walking the real filesystem via
`internal/projectid`. This split keeps parsers trivially testable against a fixture
reader — no temp directories, no `.git` scaffolding — and keeps the one place that
touches the filesystem for identity resolution auditable in one file
([`internal/ingest/project.go`](../internal/ingest/project.go)). It is also why a
per-file metric can't be built from stored data: the file path itself never survives
past this step (see [What a validator reads: Input](#what-a-validator-reads-input)).

#### Activity fields are optional (honesty note)

`LinesAdded`, `LinesRemoved`, `Edits`, `ToolCalls`, `Rejected`, `Compactions`, and
`ReworkLines` are session-level activity signals that power the `effectiveness` report
(AI output vs. cost) and the `analyze` validators. A new parser **MAY** populate them
where its source genuinely exposes edit/diff data, and **MUST** leave them at `0` where
it does not — an honest zero, never a guess. When you do count lines, count only the
`+`/`-` diff markers; the content of the line is never stored.

**Today the Claude Code and Codex parsers both populate `LinesAdded`, `LinesRemoved`,
`Edits`, `ToolCalls`, `Compactions`, and `ReworkLines`** (Claude Code from structured edit
results, sub-agent tool stats, and compaction-boundary lines; Codex from
`patch_apply_end` diffs, function/custom tool-call events, and `compacted` events — both
share the [`internal/parser.Rework`](../internal/parser/rework.go) helper for rework
detection). `Rejected` is Claude-Code-only: Codex's rollout logs don't surface tool-use
denials the way Claude Code's do. Gemini and Cline report token usage but leave every
activity field at `0`, so they contribute cost but not line counts — which is exactly what
the `effectiveness` view discloses.

#### DedupeKey determinism (hard rule)

Inserts are idempotent: the store's uniqueness constraint is `(tool, dedupe_key)`, so
`backfill` is safe to run repeatedly. That guarantee only holds if **re-parsing the same
file always produces the same keys**. A `DedupeKey` must therefore be a pure function of
the log's content, never of wall-clock time, iteration randomness, or map ordering.

- When the log gives you a stable per-record UUID, use it directly — Claude Code keys on
  the message `uuid`.
- Otherwise, derive a positional key like `fmt.Sprintf("%s:%d", sessionID, index)` where
  `index` counts emitted records in file order — Codex, Gemini, and Cline do this.

If two parses of one unchanged file disagree on keys, you will silently double-count on
the next `backfill`. The golden test below is your guard against exactly that.

#### Granularity honesty (hard rule)

`assaio` will not let session-level data masquerade as per-turn data. If your source only
reports totals for a whole session (a daily vendor aggregate, a single end-of-session
summary), you **must** set `Granularity: "session"`. Emit `"turn"` only when each record
genuinely corresponds to one request/response. When in doubt, choose `"session"` — an
honest coarse label beats a precise-looking lie.

### Golden-file testing

Parsers are tested against captured fixtures under the package's `testdata/` directory,
compared to a checked-in `.golden` snapshot of the parsed records. The convention (see
[`internal/parser/claude/claude_test.go`](../internal/parser/claude/claude_test.go)):

- A fixture (`testdata/session.jsonl`) and its golden output (`testdata/session.golden`,
  the records marshaled as indented JSON).
- An `-update` flag that regenerates the golden file:

  ```sh
  go test ./internal/parser/<tool>/ -run TestParseGolden -update
  ```

  Run it once you have eyeballed the parse, then commit the `.golden` file. Review it in
  the diff on every future change — a golden mismatch is how you catch a vendor changing
  their format out from under you.

- **Fixtures must be synthetic.** Never commit a real transcript. Fabricate a minimal log
  that exercises the fields and edge cases you care about (dedupe, model switches mid-
  session, cache tokens, missing cwd). Synthetic fixtures keep prompts and code out of the
  repo and make the test's intent legible.

Add a second, assertion-style test for behavior the golden file cannot make obvious —
that duplicates collapse, that non-usage lines are filtered, that dimensions land on every
record.

### Fuzzing

Every parser must ship a native Go fuzz test — `FuzzParse` (Cline, whose unit is a task
directory, uses `FuzzParseTask`). It seeds `f.Add` with the package's `testdata/` fixture
plus a few hand-written edge seeds (empty input, `{}`, a truncated JSON line, int64-max
token values, invalid UTF-8), and asserts the parser's invariants on every returned
record: `Parse` never panics (a non-nil error returns early, which is fine), `skipped >= 0`,
no token field is negative, `Tool` equals the package constant, and `DedupeKey` is
non-empty. `make fuzz` runs each fuzzer for `FUZZTIME` (default `20s`); a discovered
crasher is committed as a corpus file under `testdata/fuzz/` so it becomes a permanent
regression seed.

### Wire it in

Two touch points connect a finished parser to the CLI.

1. **Ingest** — [`internal/ingest/ingest.go`](../internal/ingest/ingest.go). Add your
   discovery call and append a `source` (tool name + discovered files + `Parse` function)
   to the `sources` slice. Directory-oriented sources follow the Cline branch instead.
   Add the root resolver to [`internal/paths`](../internal/paths/paths.go). If your
   source populates `Cwd`, project/subpath resolution happens automatically — every
   `source` and the Cline branch already run through
   [`internal/ingest/project.go`](../internal/ingest/project.go) before `Insert`;
   nothing more to wire up.

2. **Doctor** — [`internal/cli/doctor.go`](../internal/cli/doctor.go). Print a discovery
   line so `assaio-agent doctor` reports how many files were found, and add a one-line
   caveat for any modeling assumption your parser makes (folded token classes, recomputed
   cost, shared directories). Every honesty compromise the parser makes belongs in
   `doctor` output.

### The intake path: open a connector issue first

Before writing code, open a **Connector request** issue
([`.github/ISSUE_TEMPLATE/connector.yml`](../.github/ISSUE_TEMPLATE/connector.yml)). It
captures the tool, which channels its data is available through (local logs, vendor API,
OTLP, editor/CLI hooks), and — most importantly — a redacted sample of the log format.
That sample becomes the synthetic fixture, and the discussion settles the token-mapping
questions (does input include cache? how is reasoning billed?) before they turn into
wrong numbers. A connector is a well-scoped first contribution; the issue is where it
starts.

---

## Custom metrics (what's shipped vs. roadmap)

Custom metrics ship **two ways today**: the in-tree, one-file-per-metric validator
([Adding a metric validator](#adding-a-metric-validator) — compiled in, runs everywhere
including [the team server](#the-team-server)), and the out-of-tree [metric
plugin](#write-a-metric-plugin-any-language) — any language, no fork, declared in
config, running in `analyze` and the local dashboard.

What remains roadmap is a *dynamically loaded, in-process Go API* — the `plugin/metric/`
and `plugin/rule/` tree sketched in [`CONTRIBUTING.md`](../CONTRIBUTING.md): a metric or
rule as a linked Go unit that a running `assaio-agent` (or the team server) picks up
without a rebuild and without a subprocess, arriving toward v1.0 (see the
[roadmap](../ROADMAP.md)). The exec protocol's wire envelope is versioned but pre-1.0
unstable; if you have a metric in mind that needs domain data the envelope (or `Input`)
doesn't yet carry, open an issue and describe it — that is exactly what shapes both
before the interfaces are frozen.
