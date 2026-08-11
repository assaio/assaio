# Extending assaio

`assaio` v0.1 is intentionally small, but every axis a company needs to adapt for its own
use is a documented, working extension point today: your own **metric and dashboard
section** (in-tree, one file — or **out-of-tree in any language**, no fork), your own
**log-source location** (a config change, no code), an entirely new **tool as an
out-of-tree plugin** (any language, no Go), your own **CI gate** (a rule plugin, any
language), and **direct SQL** against your own data.
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
- [Write a rule plugin (any language)](#write-a-rule-plugin-any-language) — an
  out-of-tree **gate**: read the window's verdicts, emit alerts, fail `check` in CI.
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
| Out-of-tree exec **rule** plugin (any language) | today | An executable speaking the [rule plugin protocol](#write-a-rule-plugin-any-language), declared under `rules:` in `config.yaml` — your own thresholds gating `assaio-agent check` in CI. |
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
  `Bars` rank by a name a person chose, set `Result.BarsPseudonym` (`"project"` or
  `"skill"`) so the dashboard's
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
}
```

`Input` is a read-only bundle; use only the fields your metric needs. `Analyze` must stay
a **pure function of `Input`** — no `time.Now()`, no file or network I/O, no reaching
into the store yourself — which is what makes it trivial to unit test and safe to run
identically from the CLI, the dashboard, and (see [The team
server](#the-team-server)) the served endpoint.

| Field | Type | What it is |
|-------|------|------------|
| `Usage` | `[]store.UsageRow` | The window's usage, **pre-aggregated** by `(day, tool, model, project, entrypoint, member)` — one row per combination, tokens and activity counts summed (`internal/store/store.go`'s `Usage` query). Not one row per raw event: there is no per-record or per-file detail left at this point (see the [say-so-when-approximating](#honesty-constraints-for-every-extension) rule). Each row carries `Day` (`"YYYY-MM-DD"`), `Tool`, `Model`, `Project`, `Entrypoint`, `Member`; token fields `In, Out, CacheRead, CacheWrite, Reasoning`; and activity fields `LinesAdded, LinesRemoved, Edits, ToolCalls, Rejected, Compactions, ReworkLines` (see the [`usage.Record` contract](#the-usagerecord-contract) for what each counts). It also carries the tool-call purpose split `ToolReads, ToolSearches, ToolCommands, ToolWrites, ToolOther` — which sums to `ToolCalls` for tools that name their tool calls and is all-zero for tools that do not — and `ToolErrors`, calls that came back an error. **Every activity count here is zero for a source that does not record it**, exactly as on `Sessions`: keep the rows whose `Tool` answers the matching signal with `report.UsageAnswering` before computing a rate over one — see [Reading a field only the sources that record it carry](#reading-a-field-only-the-sources-that-record-it-carry). That applies to the tool-call purpose split too, which is populated by only some tools. |
| `Sessions` | `[]store.SessionRow` | One row per `(session_id, member)` in the window: `Project`, `Tool`, `Model`, `FirstTs`/`LastTs`, `Turns`, `OutputTokens`, `PeakContextTokens`, `Edits`, `Compactions`, and `ActiveMinutes` (focused time — inter-turn gaps over 30 minutes are excluded, so a resumed session's idle time never counts as work; `internal/store/sessions.go`). `Turns`, `PeakContextTokens` and the gaps behind `ActiveMinutes` read only `turn`-grain rows, so a whole sub-agent run stored as one session-grain record is never counted as a single very large turn. **Every one of those counts except the timestamps is zero for a source that does not record it**, so a metric reading one must first keep the sessions whose `Tool` answers the matching signal — see [Reading a field only the sources that record it carry](#reading-a-field-only-the-sources-that-record-it-carry). Also `Task`, `Outcome` and `Difficulty`: the closed-vocabulary labels a person attached with `assaio-agent mark`, `""` on every axis nobody set — which is most sessions. Treat unlabeled as "not stated", never as a failure or a zero, and report your own label coverage if your metric depends on them ([ADR 0006](adr/0006-session-annotations.md)). |
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

#### Reading a field only the sources that record it carry

Most fields on `Usage` and `Sessions` are optional at the parser (see [Activity fields are
optional](#activity-fields-are-optional-honesty-note)), which means a zero has two meanings:
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
catalog entry plus the matrix rows that answer it ([ADR 0008](adr/0008-signal-catalog.md)) —
rather than testing the tool name.

**Both row shapes need it.** `report.UsageAnswering` is `SessionsAnswering` for
`[]store.UsageRow`, and the same rule applies to a rate over a stored column: a source
recording changed lines but no rework contributed its whole output to the churn denominator
against a structural zero, which lowered the rate with code nobody watched being undone. The
generic test in `internal/analyze/capability_test.go` varies both shapes for exactly that
reason — a hole on either grain is the same bug.

#### Opting a metric out of the project drill

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
| `Title` | Human label. | Header line: `"WEEKEND-USAGE · Weekend Usage  [WATCH]"`. | Faceplate cell label (`"06 · Weekend Usage"`) and the ledger entry's label. |
| `Describe` | One-line summary. | `analyze --list`'s third column only — **not** printed by `RenderResultText`. | Not rendered. |
| `Read.Key` | `"good"`, `"watch"`, or `"neutral"` (no data). | Drives no CLI styling (plain text). | Drives the dashboard's color via CSS classes `cell__read--{key}` / `entry__read--{key}` (verdigris/oxide/muted) — an unrecognized key renders unstyled, so stick to these three. |
| `Read.Label` | Short upper-cased word, e.g. `"WATCH"`, `"STRONG"`, `"—"`. | Printed in `[brackets]` on the header line. | Printed as the faceplate/ledger read text. |
| `Purity` | `0..1`, how "well-used" this dimension reads. It is your own quantity, not an index shared with other validators, so it is never rendered as a bare number anybody could compare across cells. | **Not rendered.** | The faceplate gauge's fill width, unlabelled — clamp to `[0,1]` yourself (`clamp01`) or the CSS width silently over/underflows. |
| `HowToRead` | One-sentence explainer of what the metric means and what to do about it. Must be non-empty on **every** code path, including the no-data one. | The `"  ? …"` line under the header. | The ledger entry's muted "How to read — …" line. |
| `Figures` | The headline numbers: `{Label, Value, Note}`. | One `"  label: value (note)"` line each. | One stat tile per figure in `.entry__stats`; **the first figure gets an accent color** — order your most important number first. Not shown in the faceplate. |
| `Bars` | Optional ranked list: `{Label, Value, Frac 0..1}`. Three states matter: `nil` → no Bars section anywhere; non-nil empty → an honest "none in this window" line; non-nil non-empty → a ranked bar list. | ASCII bar `label: value  [####----]` (20 chars wide, scaled by `Frac`). | `.projectbars` list, bar width from `Frac`. Scale `Frac` against that list's own max (`fracOf`), not a global scale. |
| `BarsPseudonym` | What kind of user-authored name `Bars`' labels carry: `"project"`, `"skill"`, or `""` for none. | Not rendered. | Tells `--anonymize` whether to pseudonymize `Bars` labels, and under which prefix — see [Honesty constraints](#honesty-constraints-for-every-extension). Set it for anything a person named (a repository, a skill, a sub-agent); leave it empty for a fixed vocabulary the tool defines (models, tools, time bands). Set it wrongly and you either scramble a label that was never PII, or leak a real name into a report meant to be shared. |
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
does not set `BarsPseudonym` (it has no `Bars` at all) and does not need `Delegation`,
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
`project`, `git_branch`, and `entrypoint` are optional. **A field the protocol does not
define is rejected**, as it already was for the metric and rule protocols: a plugin writing
`outputTokens` where the protocol says `output_tokens` would otherwise store a zero and be
counted as a valid record, which is a wrong number arriving quietly instead of a protocol
error arriving loudly. Emit exactly the fields above. The same
[`usage.Record` contract](#the-usagerecord-contract) rules apply: `project` is a
directory **basename**, never a full path, and `dedupe_key` must be
[deterministic](#dedupekey-determinism-hard-rule) so re-runs never double-count.

Anything the plugin writes to stderr passes through to `assaio`'s stderr prefixed with
`[plugin/<name>] `, so diagnostics stay attributable.

### What the boundary enforces

`assaio` validates every record line and **skips** (and counts) any line that breaks one of
the boundary invariants — the same skip-and-count policy in-tree parsers apply to corrupt
log lines:

| Rejected | Why |
|---|---|
| empty `session_id` or `dedupe_key` | `dedupe_key` is half the store's uniqueness constraint; a blank one collapses rows onto each other. |
| unparseable `timestamp` | a record that cannot be placed in time can appear in no window. |
| a field the protocol does not define | a misspelled field is a silent zero; naming it is the only way the plugin author finds out. |
| `timestamp` before 2020-01-01 or more than 48h in the future | since v0.14. Every query is `ts >= ?` with no ceiling, so a year-9999 record sits inside every `--since` window forever. Identical to what the sync endpoint enforces on the same shape — the two are one shared check (`internal/usage`). |
| invalid `granularity` | see the [granularity honesty rule](#granularity-honesty-hard-rule). |
| a negative count, or one above 1,000,000,000 | a negative renders impossible percentages; an overflow-magnitude one distorts every `SUM()` it lands in. |
| `reasoning_tokens` above `output_tokens` | since v0.14. Reasoning is a *subset* of output, and a record claiming more renders a reasoning share above 100%. |
| a string field over 512 bytes | these are identities and labels, not free text. |

Stored records get the tool label `plugin:<name>`, so a plugin can never impersonate a
built-in source and its dedupe keyspace `(tool, dedupe_key)` never collides with anyone
else's. A plugin that exits non-zero, times out, or fails the handshake is reported as
failed for that run; the rest of the backfill continues. Stdout is capped at 64 MiB per run.

Unknown fields are currently ignored rather than rejected, unlike the metric and rule
protocols — so a misspelled key stores a zero instead of raising a violation. That
inconsistency is tracked as `B143` and will change behind a handshake version bump, not
silently.

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
                "entrypoint":"","member":"","granularity":"turn","in":100,"out":200,
                "cacheRead":0,"cacheWrite":0,"cacheWrite1h":0,"reasoning":0,
                "linesAdded":40,"linesRemoved":5,
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
                "cacheRead":0.0000015,"cacheWrite":0.00001875,
                "cacheWrite1h":0.00003}},
  "answers":   {"claude-code":["ai.compactions.count","ai.edits.count","..."],
                "copilot-cli":["ai.cost.estimated","ai.lines.added","..."]}
}
```

#### `cacheWrite1h` — a subset, never a total (since v0.14)

`cacheWrite1h` on a usage row is the portion of `cacheWrite` that bought a 1-hour cache
lifetime, and `prices[model].cacheWrite1h` is that portion's own higher rate. It is a
**subset**: adding it to `cacheWrite` double-counts those tokens. Price a row the way the
core does — `min(cacheWrite1h, cacheWrite)` at the 1-hour rate, the remainder at
`cacheWrite`. Both fields are `0` for a source that does not report the tier, which reads the
same as "every write was 5-minute", so a plugin publishing a figure over them should declare
its own coverage. Before v0.14 neither field was on the wire, so a plugin re-pricing what it
was handed necessarily billed every write at the cheaper rate and reported a cost the core
disagreed with.

#### `answers` — which zeros are measurements and which are silence

Every count on a `usage` or `sessions` row is **zero for a source that does not record it**,
and "nothing happened" and "nothing was written down" are different facts. A metric that
averages the two states what it did not measure — the mistake that made a Cline-only window
read as *100% conversational sessions* ([ADR 0011](adr/0011-capability-gated-metrics.md)).

`answers` maps every tool present in this window to the
[signal](adr/0008-signal-catalog.md) ids it can produce
(`assaio-agent signals list`; an out-of-tree parser gets the exec protocol's floor, since its
capability is whatever its author implemented). The rule is the same one an in-tree validator
follows: **keep the rows whose tool answers the signal, compute over those, and declare the
reach** — a figure with no rows left prints nothing rather than a zero, and the verdict is
withheld rather than earned from a silence.

```python
capable = [r for r in inp["usage"] if "ai.rework.lines" in inp["answers"].get(r["tool"], [])]
if not capable:
    result["read"] = {"key": "neutral", "label": "—"}     # withhold, never certify a silence
    result["takeaway"] = "No source in this window records an undone line."
else:
    reach = sum(tokens(r) for r in capable) / max(inp["totals"]["tokens"], 1)
    result["confidence"] = {"signalCoverage": reach, "samples": len(capable),
                            "samplesUnit": "usage rows"}
```

Only the tools actually in the window are sent: a plugin needs the capability of the data it
was handed, and shipping the whole matrix would make the envelope a second publication of it.

**Session labels are deliberately absent from this document.** In-tree validators can read
the task/outcome/difficulty a person attached with `assaio-agent mark`, but the plugin wire
does not carry them and the usage rows above are never split by them: a plugin receives
exactly the same shape it received before labels existed. That is a decision, not an
oversight — labels are local, and widening what leaves the machine is its own choice rather
than a side effect ([ADR 0006](adr/0006-session-annotations.md)). If you need a metric per
kind of work today, run your plugin under `assaio-agent analyze --task <kind>`: the filter
is applied to the window before the input document is built, so your plugin computes over
exactly those sessions without needing to know why.

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

**Declare what your verdict rests on** (v0.5). Every result carries a confidence envelope,
and assaio fills all of it except the one part only your plugin knows: how many
observations you counted.

```json
"confidence": {"samples": 12, "samplesUnit": "sessions"}
```

Coverage, freshness and the parsing build are stamped for you from the same window the
built-in metrics use, so your verdict is judged on the same footing as theirs. A plugin
that omits the field reads as `insufficient` — the honest label for "did not say what it
rests on" — so declare it even when the number is large. Count observations, not the
buckets you report: "3 models" is a shape, "31 active days" is evidence.

**Declare how much of the window your figure describes** (v0.9), if it is not all of it:

```json
"confidence": {"samples": 12, "samplesUnit": "sessions", "signalCoverage": 0.05}
```

The three stamped axes describe the *window*; this one describes your *question*, and only
your metric can answer it. A figure computed from one source's records in a five-source
window is real and describes a sliver, and nothing at window level can see the difference —
`reasoning-share` read a 20% share off under 1% of a store's output and carried `high` until
it started declaring this. Omit the key and you claim the whole window, which is what every
plugin released before the axis existed meant and stays true for most metrics. Declare `0`
and the verdict reads `insufficient`: nothing in this window could answer, which is a
different fact from a thin answer. The share sets the label alongside the other axes, and
`assaio analyze` names it when it is the weakest one:

```
Confidence: low · 43 active days · signal coverage <1%
```

**A verdict resting on nothing says which nothing it is** (v0.10). `insufficient` has four
causes and they are different facts about different things, so the summary line names the one
that applies:

| Line | What you declared |
|---|---|
| `insufficient — a source here records it, but your stored rows predate that capture` | `signalCoverage: 0` on a subject whose own denominator exists and whose capture a source in the window does have — the one cause with a cure (`backfill`) |
| `insufficient — nothing in this window can answer it` | `signalCoverage: 0` — the window may be full of usage, none of which reaches your question |
| `insufficient — 0 sessions` | `samples: 0` with a unit — you counted none of your own observations |
| `insufficient — no stated basis` | no `samples` at all — the honest reading of a plugin that did not say what it rests on |

The first row is not available to an exec metric plugin: it rests on the validator declaring
which capture a zero would be missing, which the plugin protocol has no field for. A plugin
covering none of the window reads as the second row.

Declaring `samples: 0` with a unit is therefore better than omitting the field: "zero
sessions" is a measurement, "no stated basis" is an absence of one.

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
- `barsPseudonym` works exactly as for built-ins: set it when your `Bars` rank
  by a name a person chose. The pre-rename key `barsAreProjects: true` is still accepted
  and maps to `"project"`, so a plugin released against it keeps being pseudonymized
  rather than silently publishing real project names. Any *other* unknown field is
  rejected outright — a misspelled key must not quietly disarm a setting. The
  [honesty constraints](#honesty-constraints-for-every-extension) bind a metric plugin the
  same as any in-tree validator.

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
It *does* run in `assaio-agent check` when you have configured [rule
plugins](#write-a-rule-plugin-any-language), so a rule can gate on your own metric.
See [ADR 0004](adr/0004-exec-metric-plugin-protocol.md) for the full rationale.

---

## Write a rule plugin (any language)

**When to reach for this instead of a metric.** A rule plugin is your own **gate**: an
executable that reads the verdicts assaio just computed and answers one question — is
this window acceptable? It runs inside `assaio-agent check`, so an `error` alert exits
non-zero and reddens CI or blocks a push. Reach for a [metric
plugin](#write-a-metric-plugin-any-language) when you want to *measure* something new;
reach for a rule when the measurement exists and what you need is *your* threshold on it.
Thresholds are exactly what assaio refuses to ship built-in — the right number is
organizational, and a number we picked would be a claim about your team we cannot honestly
make.

Rule plugins are **opt-in only**, declared under `rules:` in
`~/.config/assaio/config.yaml` — never discovered from `PATH`, never downloaded. The
entry shape is the same as `plugins:` and `metrics:`, and one binary may appear in all
three lists, serving all three protocols (`scan`, `analyze`, and `evaluate` argv):

```yaml
rules:
  - name: budget-drift       # required, [a-z0-9-]+; stamped on every alert it raises
    command: /path/to/assaio-rule-budget      # required; PATH lookup if not absolute
    timeout: 15s             # optional, default 60s
```

A rule sees **less than a metric does**: no usage rows, no sessions, no prices — only the
verdict array `analyze --format json` already prints. Nothing crosses the process boundary
that you could not print yourself with one command.

### The protocol

`assaio` invokes `<command> evaluate` with `ASSAIO_RULE_PROTOCOL=1` in the environment,
writes one JSON envelope to the plugin's **stdin**, closes it, and reads stdout.

**stdin** — this window's verdicts, versioned:

```json
{
  "assaio_rule_input": 1,
  "verdicts": [
    {"name":"adoption","title":"Adoption","describe":"...","read":{"key":"watch","label":"WATCH"},
     "purity":0.42,"howToRead":"...","figures":[{"label":"AI lines","value":"1,204"}],
     "bars":[{"label":"web","value":"800","frac":1}],"takeaway":"...","caveats":["..."]}
  ]
}
```

Each entry is one [`Result`](#what-a-validator-returns-result) — every registered
validator, plus every configured [metric plugin](#write-a-metric-plugin-any-language)
(named `plugin:<name>`), in the order `analyze` renders them. Like the metric envelope, it
is **versioned but pre-1.0 unstable** — a release that reshapes it says so explicitly (see
`RELEASING.md`).

**stdout** — a one-line handshake, then exactly **one** JSON alerts document
(pretty-printed is fine; anything after it is a violation):

1. `{"assaio_rule": 1, "name": "<name>"}` — version must be `1`, `name` must equal the
   configured name.
2. `{"alerts": [...]}`, each alert:

| Field | Required | Meaning |
|-------|----------|---------|
| `rule` | yes | Stable id of the check that fired, e.g. `premium-share`. |
| `severity` | yes | `info`, `warn`, or `error`. **Only `error` fails the gate.** |
| `message` | yes | One line a human reads on a red build. |
| `validator` | no | The verdict this alert is about, echoed back for the reader. |

An empty `alerts` array is a normal, passing answer. Anything written to stderr passes
through prefixed `[rule/<name>] `.

### What the boundary enforces

A document that fails **any** check is rejected whole — assaio never applies a
partially-sanitized alert set, because a silently dropped alert would weaken a gate you
believe is running.

- `severity` must be exactly `info`, `warn`, or `error` (lower-case).
- `rule` (≤ 64 chars) and `message` (≤ 400) are required; `validator` (≤ 64) is optional.
- Max 50 alerts per plugin; no control characters anywhere (terminal-escape guard).
- Unknown JSON fields are rejected: a misspelled `alerts` or `severity` key must fail
  loudly rather than quietly disarm the gate.
- Stdout is capped at 1 MiB; the run is killed after `timeout` (default 60s).
- The emitting plugin's name is stamped onto every alert at the boundary, so an alert is
  always attributable and a plugin cannot claim to be another.

**Failure is fail-closed.** A rule that could not be evaluated — bad handshake, timeout,
non-zero exit, contract violation — is reported on stderr *and* fails `check`, while the
remaining rules still run. A gate that did not run is not a gate that passed.

### A complete example (Python)

```python
#!/usr/bin/env python3
"""assaio-rule-premium: gate on how much of the window runs on premium models."""
import json, sys

verdicts = {v["name"]: v for v in json.load(sys.stdin)["verdicts"]}
print(json.dumps({"assaio_rule": 1, "name": "premium"}))

fit = verdicts.get("model-fit")
if fit is None:
    print(json.dumps({"alerts": [{"rule": "model-fit-missing", "severity": "warn",
                                  "message": "model-fit did not report this window."}]}))
    sys.exit(0)

alerts = []
if fit["read"]["key"] == "watch":
    alerts.append({"rule": "premium-share", "severity": "error", "validator": "model-fit",
                   "message": fit["takeaway"][:400]})
print(json.dumps({"alerts": alerts}))
```

Make it executable, declare it under `rules:` as above, and run the gate:

```console
$ assaio-agent check --since 30d
budget check · last 30d
  total tokens: 4812004
  total cost:   $61.20 (API-equivalent estimate)

  no budget set -- pass --max-tokens or --max-cost to gate.

rules
  [error] premium/premium-share: Nearly all tokens run on premium models -- consider
  delegating routine work to cheaper models or sub-agents. (model-fit)

Cost is an estimate at public pay-as-you-go API prices -- not your actual spend; ...
error: rule gate failed: premium/premium-share
$ echo $?
1
```

**Where it deliberately does not run:** `analyze` (a per-validator report, and its
`--format json` array is the metric-plugin contract — alerts would either reshape that
public surface or print in text what JSON omits), the dashboard, and the team server
(unauthenticated `GET /`, same reasoning as metric plugins). `check` is the gate, and the
gate is where rules live. See [ADR 0005](adr/0005-exec-rule-plugin-protocol.md) for the
full rationale.

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
entry, same anonymization rules, with nothing to configure on the server side. The
deliberate exception is **exec plugins**: `serve` executes neither [metric
plugins](#write-a-metric-plugin-any-language) nor [rule
plugins](#write-a-rule-plugin-any-language), because its dashboard endpoint is
unauthenticated and rebuilt per request — they are local-CLI surfaces (`analyze`,
`dashboard`, `metrics verify`, and `check` for rules; see [ADR
0004](adr/0004-exec-metric-plugin-protocol.md) and [ADR
0005](adr/0005-exec-rule-plugin-protocol.md)). The one
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

One table holds your data, `usage_record`
([`internal/store/migrations/0001_init.sql`](../internal/store/migrations/0001_init.sql)):

| Column | Type | Notes |
|--------|------|-------|
| `id` | `INTEGER PRIMARY KEY` | Row id. |
| `tool` | `TEXT` | Source, e.g. `claude-code`, `codex`, `gemini-cli`, `copilot-cli`, `cline`. |
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

**Two bookkeeping tables sit beside it**, neither holding usage: `ingest_file` (one row per
input already parsed — path, size, mtime, parsing build) makes a repeat `backfill` nearly
free, and `ingest_source` (one row per source per run — files found, files read, records,
skipped lines, zero-token records) is the baseline the [format-drift
canaries](format-resilience.md) compare against. Both are caches: dropping them costs one
slow re-parse and a reset drift baseline, nothing more. `ingest_file` is pruned to what is
actually on disk after each pass, and `ingest_source` keeps only the newest runs per tool,
so neither grows with how long assaio has been installed. Use `assaio-agent compact` to
return freed pages to the filesystem — SQLite does not do that on its own.

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

**While you work on a parser, run `assaio-agent backfill --full`.** Ingest skips inputs it
has already parsed unchanged, and the stored state is keyed on the build's identity — which
stays constant for a local build, precisely so a rebuild does not force a re-parse of every
file. A released binary invalidates the state automatically; your development build does
not, so `--full` is how you see a parser change take effect.

Where `Discover`'s root itself comes from — the built-in default, or a team's own
override — is a separate, non-code concern; see [Custom log-source
paths](#custom-log-source-paths).

#### Declare what your source can answer

A parser also adds one row to the depth matrix in
[`internal/parser/depth.go`](../internal/parser/depth.go), and the part that matters most is
`Answers`: the ids of the signals your source can actually produce (`assaio-agent signals
list` prints them all). Do not reach for the nearest tier and move on — the tier's three
axes are a summary, and `Activity: true` says nothing about *which* activity. Copilot CLI
records changed lines and no edit count, no tool calls, no turns and no rework, so it lists
two activity signals and not the other four; declaring the axis alone made `signals coverage`
report sixteen of eighteen signals as fully supported when the truth was ten.

What your source writes but you choose *not* to read belongs in the audit below — [What each
source's log carries](#what-each-sources-log-carries-and-what-assaio-reads) — with the reason,
so the next reader can tell a deliberate omission from an oversight.

The rule is one question per signal: **would a figure computed from my records be right, or
merely non-empty?** If the log does not carry it, leave it out — an absent signal is reported
as "this source cannot answer it", which is a useful fact, while a claimed one becomes a
number someone trusts. That applies inside the token group too: `ai.tokens.reasoning` is
declared per source rather than inherited, because Claude Code and Cline never surface a
thinking count and claiming it for them reported full support for a figure their records can
only leave at zero. A test asserts the ids you list are real and that they do not contradict
the tier axes ([ADR 0008](adr/0008-signal-catalog.md)).

**A metric reads this row before it reads your records.** A validator that reports a
per-session figure — the session mix, context health, how long sessions run, the rejection
rate — first asks `parser.Answers(tool, id)` and keeps only the sessions your source can
answer for, because a field you never write is not a zero: it is a silence, and averaging it
in reports a fact about someone's work that came from your parser's gap. Leaving a signal out
therefore *removes* your sessions from that figure rather than dragging it down. Spell the id
with the `parser.Signal*` constant, never as a literal — a typo answers false for every tool
and empties a metric instead of failing a build.

The row does more than describe your source. `parser.Tools()` and `parser.Answers()` are how
everything downstream asks what exists and what it can do — sync validation, `clear --tool`,
the confidence envelope on every verdict, and every caveat that used to spell out tool names.
Wiring your parser into `internal/ingest` and `doctor`'s scan is still a separate step, and
two tests bind the three together: the set ingest reads, the set doctor scans, and the set the
matrix publishes must be identical. A parser that ships without its row is not merely
undocumented — its records get rejected by the team server and its data cannot be deleted per
source, which is exactly what happened to Copilot CLI between v0.6.0 and v0.8.0.

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

**Today the Claude Code and Codex parsers are the two that populate the full set —
`LinesAdded`, `LinesRemoved`, `Edits`, `ToolCalls`, `Compactions`, and `ReworkLines`**
(Copilot CLI carries the two line counts per session and none of the rest) (Claude Code from structured edit
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

The rule bites inside a source, not only between them: a Claude Code transcript is per-turn
throughout except for the one record summarizing a completed sub-agent, which totals a whole
run and is therefore `"session"`. It was labelled `"turn"` until v0.10, which let every
per-turn figure count it as a single very large turn. If one shape in your log aggregates,
label that shape — the field is per record, not per parser.

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

## What each source's log carries, and what assaio reads

Every parser turns a rich log into one fixed record and drops the rest. This is the inventory
of that drop, source by source: each field a source writes ends in exactly one of two states —
**extracted**, meaning a signal in the catalog is computed from it and a golden covers it, or
**skipped**, with the reason written down. A field whose meaning the vendor does not document
is skipped *with that stated*; it is never guessed at from its name.

**How it was produced.** Key paths were inventoried from real logs on one machine plus each
parser's synthetic fixture — names and counts only, never a value, except for discriminator
keys (`type`, `say`, `role`, `status`, `stop_reason`, …) whose values are the format's own
vocabulary. The corpus is stated per source below, because it is the honest limit of this
table: **a field that does not appear in the corpus is exactly the one most likely to be
missing from it.** Re-run the audit when a vendor ships a major version.

None of these formats is documented as an interface. Every "meaning" below is either stated by
the vendor or inferred from a name that leaves no room — and where it does leave room, the
field is skipped for that reason.

### Claude Code

*Corpus: 5,602 transcripts · 703,320 lines · 2,284 distinct key paths, plus
`testdata/session.jsonl`.*

| Field | State | Notes |
|---|---|---|
| `uuid`, `sessionId`, `timestamp`, `cwd`, `gitBranch`, `entrypoint` | extracted | Record identity, dedupe key and dimensions. |
| `message.model`, `message.usage.{input_tokens,output_tokens,cache_read_input_tokens,cache_creation_input_tokens}` | extracted | Every token signal and the cost estimate. |
| `message.id` | extracted (v0.12) | The API response a line belongs to. Claude writes one line per content block and repeats that response's usage on each, so keying a record on the line `uuid` counted one request once per block — 354,904 lines were 159,175 responses on the audited corpus, inflating output tokens 1.97x and cache-write 2.81x. A record is now keyed on this. |
| `message.content[].type` = `tool_use` / `tool_result` + `.name` / `.is_error` | extracted | Tool-call count, the purpose split, and `ai.tool_errors.count`. |
| `toolUseResult.structuredPatch[].lines` | extracted | `ai.lines.added` / `.removed` and, via the shared helper, `ai.rework.lines`. |
| `toolUseResult.{agentId,agentType,resolvedModel,usage,toolStats.linesAdded,toolStats.linesRemoved}` | extracted | The completed sub-agent's own record. |
| `toolDenialKind` | extracted | `ai.rejected.count`. |
| `isCompactSummary`, `subtype` = `compact_boundary` | extracted | `ai.compactions.count`. |
| `isSidechain`, `attributionSkill`, `attributionAgent` | extracted | Delegation share and the skill / sub-agent split. |
| `message.usage.cache_creation.ephemeral_5m_input_tokens` / `ephemeral_1h_input_tokens` | extracted (v0.12) | The cache TTL tier, on every assistant turn. 59.7% of the audited corpus's cache-write tokens bought the 1-hour lifetime, which bills at 1.6x the 5-minute rate, so reading it raised the cache-write cost component by 35.8%. Signal `ai.tokens.cache_write_1h`. |
| `message.diagnostics.cache_miss_reason.type` | extracted (v0.12) | Six documented-by-vocabulary reasons a prompt cache missed (`messages_changed`, `model_changed`, `previous_message_not_found`, `system_changed`, `tools_changed`, `unavailable`). `cache-hygiene` names the top one, turning a cache rate into a cause. Signal `ai.cache.miss_reason`. |
| `toolUseResult.userModified` | **skipped — worth extracting** | The human edited what the AI wrote. A direct correction signal where `rework` has only a proxy (`B111`). |
| `toolUseResult.toolStats.{readCount,searchCount,bashCount,editFileCount,otherToolCount}` | **skipped — accounting undecided** | The sub-agent's own purpose split, already in the log. Populating it means also setting `ToolCalls`, which would double-count against the parent turn — the open half of `B78`. |
| `attributionPlugin`, `attributionMcpServer`, `attributionMcpTool` | **skipped — worth extracting** | Two more attribution dimensions beside skill and sub-agent, on a third of all turns (`B112`). |
| `version` | **skipped — worth extracting** | The Claude Code build that wrote the line, on every turn. The harness-version cohort input (`B96`) is already on disk (`B112`). |
| `message.stop_reason` | **skipped — worth extracting** | Why generation stopped; `max_tokens` marks a truncated answer (`B113`). |
| `toolUseResult.interrupted` | **skipped — worth extracting** | A command the human cut short (`B113`). |
| `message.content[].thinking` | skipped — no count exists | Thinking blocks are present, but Claude publishes no thinking **token** count, and inferring one from text length would be a fabricated number. What is honestly derivable is a share of turns that carried one — a different signal from `ai.tokens.reasoning`, and not one the catalog declares today. |
| `effort` | skipped — undocumented | Appears on most turns; the vendor documents neither its vocabulary nor whether it is a request or an outcome. Named in `B113` as research, not as an extraction. |
| `message.usage.server_tool_use.{web_search_requests,web_fetch_requests}` | skipped — undocumented billing | Server-side tool calls the vendor bills on its own terms; folding them into any token figure would state a price we cannot compute. |
| `message.usage.{speed,inference_geo,iterations,service_tier}` | skipped — undocumented | No published meaning. `service_tier` reads `standard` on every line here, which is a constant, not a signal. |
| `slug`, `promptId`, `requestId`, `messageId`, `leafUuid`, `parentUuid`, `sourceToolAssistantUUID` | skipped — identity, no measure | Threading and request identifiers. `parentUuid` would let a fork be detected; nothing measures forks yet, so it stays out rather than being stored speculatively. |
| `attachment.*`, `snapshot.*`, `backup.*`, `trackingPath`, `lastPrompt`, `aiTitle`, `content`, `message.content[].text`, `toolUseResult.{file,content,oldString,newString,originalFile,stdout,stderr}` | skipped — content, by construction | Prompts, code, file contents, command output, editor backups and the file-history snapshots. These are what PRIVACY.md promises are never collected; they are read transiently at most (diff markers, a path used only to key rework in memory) and never stored. |
| `attachment.type` = `hook_*`, `skill_listing`, `invoked_skills`, `mcp_instructions_delta`, `queued_command`, `max_turns_reached`, `plan_mode*` | skipped — harness inventory, not usage | Hook outcomes, skill and MCP availability, mode changes. These belong to the harness inventory (`B95`), which stores an artifact's *shape*, not a usage record. |

### Codex CLI

*Corpus: 21 rollouts · 3,599 lines · 273 distinct key paths, plus `testdata/rollout.jsonl`.*

| Field | State | Notes |
|---|---|---|
| `timestamp`, `type`, `payload.type` | extracted | Line routing and every record's own timestamp. |
| `payload` of `session_meta`: `id`, `cwd`, `model`, `timestamp` | extracted | Session identity and dimensions. |
| `turn_context.model` | extracted | Model carried forward across a mid-session switch. |
| `token_count` → `payload.info.total_token_usage.{input_tokens,cached_input_tokens,output_tokens,reasoning_output_tokens}` | extracted | Cumulative totals, differenced into per-turn records. |
| `patch_apply_end` → `success`, `changes.<path>.{type,unified_diff,content}` | extracted | Lines, edits, rework, and the tool error on a failed apply. `changes` is a type-discriminated union: an `update` carries `unified_diff`, a creation carries the whole file as `content`. Reading only the diff dropped every created file's lines — 37% of Codex's added lines on the audited corpus (`B119`, fixed in v0.13). |
| `response_item` → `type` = `function_call` / `custom_tool_call`, `name`, `status` | extracted | Tool-call count, purpose split and call failures. |
| `compacted` | extracted | `ai.compactions.count`. |
| `payload.info.total_token_usage.cache_write_input_tokens` | **skipped — worth extracting** | Codex reports a cache-write count and the parser has no field for it. On the audited corpus it was present on 238 `token_count` events and **zero on every one of them**, so nothing is currently mis-stated here — but a zero nobody read and a zero the vendor reported are different facts, and another plan or model may not share it (`B107`). |
| `payload.info.model_context_window` | **skipped — worth extracting** | The model's own context limit, on every `token_count`. `B16` proposes vendoring a context-window table; for Codex the log states it (`B114`). |
| `payload.rate_limits.{plan_type,primary.used_percent,primary.window_minutes,primary.resets_at,credits.*}` | **skipped — worth extracting** | How close the session ran to the plan's limit, and on which plan. The only place any source states a subscription's real constraint (`B114`). |
| `payload.info.last_token_usage.*` | skipped — redundant, kept as a check | The vendor's own per-turn figure. assaio differences the cumulative totals instead, which survives a missed line; this field is the cross-check that would prove that arithmetic on real data. |
| `payload.changes.<path>.move_path` | **skipped — worth extracting** | A rename is still an undifferentiated edit (`B113`). Its sibling `type` was on this row too, filed as a nice-to-have — and it was the field whose absence dropped a third of Codex's added lines. An audit that asks "is this field read?" cannot see that; that is what the calibration suite (`B137`) is for. |
| `turn_aborted` → `payload.reason` | **skipped — worth extracting** | A turn the human interrupted (`B113`). |
| `payload.thread_settings.reasoning_effort` | skipped — setting, not usage | A configuration value; it belongs to the harness inventory (`B95`) rather than to a usage record. |
| `world_state` → `payload.state.{model,permissions,personality,multi_agent_mode,host_skills,collaboration_mode,git_attribution}` | skipped — harness inventory | The agent's configuration at a point in time. Same reason (`B95`). |
| `payload.{content,input,output,arguments,text,summary,message,encrypted_content}`, `payload.action.{queries,url}`, `base_instructions.text`, `state.agents_md.text` | skipped — content, by construction | Prompts, completions, tool inputs and outputs, search queries, instruction files. |
| `payload.{id,call_id,turn_id,window_id,internal_chat_message_metadata_passthrough.turn_id}` | skipped — identity, no measure | Codex assigns its own turn id; assaio keys on a file fingerprint plus a positional counter, which is stable across a resumed session. Adopting the vendor's id would change a shipped dedupe contract for no new figure. |
| `payload.{approval_policy,approvals_reviewer,sandbox_policy.writable_roots,phase}` | skipped — undocumented | No published meaning, and each reads as a policy setting rather than an observation. |

### GitHub Copilot CLI

*Corpus: 3 sessions · 43 lines · 145 distinct key paths, plus `testdata/session.jsonl`. This is
the thinnest corpus of the five and the table should be read as provisional.*

| Field | State | Notes |
|---|---|---|
| `session.start` → `data.sessionId`, `data.context.cwd`, `timestamp` | extracted | Session identity, project and date. |
| `session.shutdown` → `data.modelMetrics.<model>.tokenDetails.{input,cache_read,cache_write,output}.tokenCount`, `.usage.reasoningTokens` | extracted | Per-model tokens and the cost estimate. |
| `session.shutdown` → `data.codeChanges.{linesAdded,linesRemoved}` | extracted | Session line counts, credited whole to the model with the most requests. |
| `data.toolRequests[].name`, `data.toolName` | **skipped — the depth row understates the source** | Copilot names its tool calls. Its matrix row says it carries no tool-call count, which was true of the parser and not of the log (`B109`). |
| `data.toolTelemetry.metrics.{linesAdded,linesRemoved}` | **skipped — the depth row understates the source** | Per-tool-call line counts. Today only the session total is read, which is why the whole session's changes are credited to one model (`B109`). |
| `data.modelMetrics.<model>.requests.count` | **skipped — worth extracting** | Requests per model — a turn count for a source the matrix says has none (`B109`). |
| `data.context.{branch,gitRoot,baseCommit,headCommit}` | **skipped — deliberate, pending a privacy decision** | The only source that records the commit its session started and ended at. That is attribution evidence of a quality no heuristic reaches (`B85`), and also exactly what the correlation threat model (`B100`) exists to decide about. It is not extracted ahead of that decision. |
| `data.modelCacheState[].{cacheTtlSeconds,cacheExpiresAt}` | **skipped — worth extracting** | A published cache TTL, which `cache-hygiene` states it cannot see (`B109`). |
| `data.parentAgentTaskId` | **skipped — worth extracting** | Sub-agent parentage, the delegation signal Claude Code has and Copilot's matrix row does not claim (`B109`). |
| `data.modelMetrics.<model>.{requests.cost,totalNanoAiu}` | skipped — a different unit | Copilot's own billing units. assaio recomputes cost from tokens for cross-tool consistency (the same decision as Cline's `cost`); this is useful only as an external check on the price table. |
| `data.{currentTokens,conversationTokens,systemTokens,toolDefinitionsTokens}` | skipped — undocumented composition | Context composition at a moment; how these overlap the billed counts is not documented, and adding them to a token total would double-count. |
| `data.{content,arguments,result,attachments,reasoningText,reasoningOpaque}`, `toolRequests[].{arguments,intentionSummary}`, `shellToolInfo.{displayCommand,possiblePaths}`, `codeChanges.filesModified` | skipped — content, by construction | Prompts, completions, tool arguments, command lines and file paths. |
| `data.{allowAllPermissions,previousAllowAllPermissions,remoteSteerable,copilotVersion,reasoningEffort,contextTier}` | skipped — harness inventory | Permission and version state (`B95`). |
| `data.{apiCallId,clientRequestId,requestId,serviceRequestId,interactionId,messageId,toolCallId}` | skipped — identity, no measure | Request correlation identifiers. |

### Gemini CLI

*Corpus: 355 files under `~/.gemini` · 2,045 lines · 27 distinct key paths — of which only **2
files** match the discovery glob, and neither contains a token field.*

| Field | State | Notes |
|---|---|---|
| `sessionId`, `timestamp`, `model`, `tokens.{input,output,cached,thoughts,tool,total}` | extracted | Every token signal, per the shape `testdata/session.jsonl` captures. |
| `type`, `source`, `status`, `step_index`, `content`, `thinking`, `error`, `error_code`, `truncated_fields`, `projectHash`, `startTime`, `workspace`, `$set.*` | **not classified — the corpus does not contain the parsed shape** | The chat files this install writes carry none of the token fields above. Either the recording moved, or these files were never the token source. Until that is settled the fields are not classified, because guessing which of two formats is current is exactly what this audit refuses to do (`B110`). |

The honest reading of that second row is a warning about assaio, not about Gemini: this source
produces **2 discovered files and 0 records** here, and no drift canary fires, because every
canary needs a sample floor (20 files) a two-file source can never reach. `B110` covers both
halves.

### Cline

*Corpus: no Cline install on the audited machine — `testdata/ui_messages.json` and
`testdata/task_metadata.json` only. This table is the weakest of the five and says so.*

| Field | State | Notes |
|---|---|---|
| `ui_messages[].{ts,type,say}` and the `api_req_started` payload's `{tokensIn,tokensOut,cacheReads,cacheWrites}` | extracted | Every token signal, per request. |
| `task_metadata.model_usage[].{ts,model_id}` | extracted | The model in force at each request, carried forward across a mid-task switch. |
| `api_req_started` payload `cost` | skipped — recomputed | Cline's own per-request cost. Record has no cost field by design; cost is computed from tokens for cross-tool consistency. Useful only as an external check on the price table. |
| `api_req_started` payload `request` | skipped — content, by construction | The prompt. |
| `task_metadata.model_usage[].{mode,model_provider_id}` | skipped — unverified against a real install | `mode` looks like Cline's plan/act distinction, which would be a genuine work-kind signal, and `model_provider_id` would separate the same model served by two providers. Neither is confirmed against a real corpus, so both stay skipped rather than being read from a fixture the project wrote itself. |
| `task_metadata.{files_in_context,environment_history}` | skipped — content and paths | File paths and environment snapshots. |
| `ui_messages[]` `say` = `tool` payloads | **skipped — the known activity gap** | The diffs that would give Cline line counts (`B39`). |

### What this audit is not

It does not claim the five formats have no other fields — only that these are the fields the
corpus above contains. Two of the five tables rest on a corpus too thin to be conclusive and
say so in their own heading. Widening that is the same work as widening the golden corpus
(`B20`), and the two should be re-run together.

---

## Custom metrics (what's shipped vs. roadmap)

Custom metrics ship **two ways today**: the in-tree, one-file-per-metric validator
([Adding a metric validator](#adding-a-metric-validator) — compiled in, runs everywhere
including [the team server](#the-team-server)), and the out-of-tree [metric
plugin](#write-a-metric-plugin-any-language) — any language, no fork, declared in
config, running in `analyze` and the local dashboard. Thresholds *on* those metrics ship
as [rule plugins](#write-a-rule-plugin-any-language), out-of-tree for the same reason.

What remains roadmap is a *dynamically loaded, in-process Go API* — the `plugin/metric/`
and `plugin/rule/` tree sketched in [`CONTRIBUTING.md`](../CONTRIBUTING.md): a metric or
rule as a linked Go unit that a running `assaio-agent` (or the team server) picks up
without a rebuild and without a subprocess, arriving toward v1.0 (see the
[roadmap](../ROADMAP.md)). The exec protocol's wire envelope is versioned but pre-1.0
unstable; if you have a metric in mind that needs domain data the envelope (or `Input`)
doesn't yet carry, open an issue and describe it — that is exactly what shapes both
before the interfaces are frozen.
