# Extending assaio

Every axis a company needs to adapt `assaio` for its own use is a documented, working
extension point: your own **metric and dashboard section** (in-tree, one file — or
**out-of-tree in any language**, no fork), your own **log-source location** (a config change,
no code), an entirely new **tool as an out-of-tree plugin** (any language, no Go), your own
**CI gate** (a rule plugin, any language), and **direct SQL** against your own data.

This page is the map and the rules that bind all of them. Each surface has its own page, and
what can be *listed* rather than explained — every signal, source, validator, command, setting
and protocol field — is generated from the binary's own registries:

- **[`assaio-agent docs export`](#the-generated-reference)** — the machine-readable reference,
  published as [assaio.dev/docs/reference](https://assaio.dev/docs/reference).

**The headline mechanism, in one paragraph.** A metric is one Go file under
`internal/analyze/` that reads the same `Input` bundle every built-in metric reads and returns
one `Result` value. Register it from that file's own `init()`, and it appears in `assaio
analyze`, `assaio analyze --format json`, **and** the HTML dashboard — a new faceplate cell and
a new ledger section, laid out, colored and captioned like every built-in one — with no other
code to write and no template to touch. That claim is verified end to end in
[the worked example](extending/metric-validator-example.md), including what happens when your
metric's `Bars` rank by project name and the report is anonymized.

## The surfaces

| Surface | Status | How |
|---------|--------|-----|
| [In-tree metric validator](extending/metric-validator.md) | today | One file under `internal/analyze/` implementing `Validator`, self-registered via `init()` — appears in `analyze`, `analyze --format json` and the HTML dashboard automatically. |
| [Custom log-source paths](extending/parser-plugin.md#custom-log-source-paths) | today | `sources.<tool>` in `config.yaml`, no code. |
| [Out-of-tree exec parser plugin](extending/parser-plugin.md#write-a-plugin-any-language) | today | An executable speaking the parser protocol, declared in `config.yaml`. |
| [Out-of-tree exec **metric** plugin](extending/metric-plugin.md) | today | An executable declared under `metrics:` — your own analyzer in `analyze` and the dashboard without forking. |
| [Out-of-tree exec **rule** plugin](extending/rule-plugin.md) | today | An executable declared under `rules:` — your own thresholds gating `assaio-agent check` in CI. |
| [Team server](extending/team-server.md) | today (MVP) | `serve` + `sync`; the served dashboard runs the same validator registry as the local CLI. |
| [SQL against the schema](extending/query-your-data.md) | today | Any SQLite client against the documented `usage_record` table. |
| JSON/CSV pipes | today | `report --format json\|csv` into your own tooling or BI. |
| [In-tree parser (new data source)](extending/data-source.md) | today | One Go package under `internal/parser/`, with golden and fuzz tests; merge via PR. |
| Out-of-tree Go plugin API (library import, dynamically loaded) | deferred | Not a v1 contract and not scheduled: the exec protocols are the extension boundary. See [compatibility.md](compatibility.md). |

Two more pages exist for reading rather than writing:
[what each source's log carries](extending/source-fields.md), the field-by-field audit behind
every depth row, and [the worked example](extending/metric-validator-example.md).

## Recipes

The guides describe the contracts; these are working examples to copy. **Every recipe is held to
still working, and not all of them equally** — the classification is code, and a recipe missing
from it fails the build:

| how | what it means | how many |
|---|---|---|
| executed | run, and the output asserted — the label rules through the rule engine, the plugins against a fixture window with their output held to the protocol | 10 |
| commands-checked | every `assaio-agent` invocation in it names a real command and real flags; nothing runs the surrounding shell | 11 |
| loaded | parsed by assaio's own configuration loader and validated | 2 |
| shape-checked | parsed, and the method set held to the `Validator` interface: a renamed method fails, a wrong number does not | 3 |

A shell recipe is the weak one, and it is weak in a specific way worth knowing: the flags are
real, and whether the pipeline around them does what the prose says is a reviewer's judgement.

- [Extensions, written out in full](recipes/extensions.md) — complete validators and metric
  plugins, including the one thing that separates a metric from a mistake: gating on what the
  window can actually answer.
- [Label rules you can paste in](recipes/label-rules.md) — branch, skill, sub-agent and
  entrypoint conventions for `mark --suggest`, which ships defaults for almost nothing on purpose.
- [Rule plugins you can run today](recipes/rule-plugins.md) — three complete gates, starting with
  the one that catches a verdict withheld for want of data.
- [Gating CI on what a window cost](recipes/ci-gates.md) — `check` as a pre-push hook and a
  scheduled job, and what each non-zero exit actually means.
- [Running it without being asked](recipes/automation.md) — the weekly loop, delivering a digest,
  and what not to automate.

## The generated reference

`assaio-agent docs export --format json` prints everything about the binary that can be
enumerated: the signal catalog, the source-depth matrix, the validators with their scope, the
whole command tree with flags and defaults, every configuration key with its environment
variable, and the metric-plugin protocol's fields. The same document renders as
[assaio.dev/docs/reference](https://assaio.dev/docs/reference).

It exists because a hand-copied list is a list that falls behind: the website went three
releases stale before anyone noticed, and a shipped command sat unpublished for a whole release
after that. So the enumerable surfaces are generated, the published files declare which of
their claims are checkable, and `make test` fails when a page and the binary disagree. What a
figure *means* stays prose — reflection reads names and types, never intent — which is why
these pages are still written by hand.

Read it directly if you are building on assaio:

```console
$ assaio-agent docs export | jq '.validators[] | select(.scope=="window") | .name'
$ assaio-agent docs export | jq '.sources[] | {tool, tier, signals: (.answers | length)}'
```

## Honesty constraints for every extension

`assaio`'s product promise is **measure value, not people; honest statistics or nothing**
(`AGENTS.md`, `CONTRIBUTING.md`). That promise is not a built-in-only courtesy — it binds every
extension whose output a person reads as a metric or a dashboard section: an in-tree validator,
a community PR, or a private fork's own validator file. Concretely:

- **Directional, not authoritative.** A `Read` (`Strong`/`Watch`/`Healthy`/…) is a diagnostic
  signal, not a verdict. If the evidence behind your metric is contested, incomplete, or a proxy
  for the thing you actually care about, say so in `HowToRead` or a `Caveat` — the word
  "directional" belongs in your rendered text, not just in this document.
- **`—` for an undefined ratio, never a fabricated one.** Divide-by-zero is a dash, not a zero
  or a 100% — use `humanize.PercentOrDash` (`internal/humanize/percent.go`) or `perActiveDay`
  (`internal/analyze/format.go`), or the same
  pattern by hand. A metric that reports "0%" when it has no denominator to divide by is a lie
  dressed as a number. This holds even when an underlying aggregate's own zero-denominator
  default is `0` (e.g. `report.ChurnStat.ReworkRate`) — a `Figure` must still check the raw
  denominator itself rather than formatting that default directly (see
  `internal/analyze/rework.go`'s "rework" figure, which reads `ReworkLines`/`LinesAdded` via
  `humanize.PercentOrDash` instead of formatting `ReworkRate`).
- **A silence is not a zero.** Before reading a column, check that the source recording the row
  can produce it at all — `answers` on the wire, `parser.Answers` in-tree (ADR 0011). A source
  that never writes a cache-write counter leaves the field at zero, and a metric that reads that
  as "the cache was never written" is reporting a silence as a measurement.
- **Aggregate and pseudonymized by default; per-person only as a governed opt-in.** `Input`
  carries no user identity today — it groups by project, tool, model and entrypoint, never by
  person, so a validator that ranks something ranks *those* dimensions, the same way
  `throughput` ranks projects, never individuals. If your `Bars` rank by a name a person chose,
  set `Result.BarsPseudonym` (`"project"` or `"skill"`) so the dashboard's `--anonymize` (on by
  default) pseudonymizes those labels exactly as it does for the built-in `throughput`
  validator — enforced generically by `internal/dashboard.anonymizeVerdicts`, not hardcoded to
  any one validator's name, so it applies to yours too. Leave it empty for any other dimension
  (models, tools, …); those must never be pseudonymized. A future per-member breakdown is only
  ever a deliberate, consented, team-mode opt-in — never silent, never a leaderboard, never
  built for individual performance evaluation.
- **Say so when you approximate.** If your metric cannot observe something precisely from the
  stored aggregate — `Input.Usage` is already grouped, so per-record detail is gone — label the
  figure as approximate in the rendered text rather than presenting it as exact.
- **Never a per-person scoreboard.** Even in team mode, an extension must not turn individual
  usage into a ranked, named list presented as a performance signal. See `PRIVACY.md`.

These are the rules `internal/analyze`'s built-in validators are held to and tested against
(`TestValidatorsEmptyInputSafe`, `TestReworkDashOnZeroToolCalls`,
`TestBuildNeverAnonymizesModelNames`) — a code review of a new validator holds it to the same
bar.

## Custom metrics (what's shipped vs. roadmap)

Custom metrics ship **two ways today**: the in-tree, one-file-per-metric
[validator](extending/metric-validator.md) — compiled in, runs everywhere including
[the team server](extending/team-server.md) — and the out-of-tree
[metric plugin](extending/metric-plugin.md), any language, no fork, declared in config, running
in `analyze` and the local dashboard. Thresholds *on* those metrics ship as
[rule plugins](extending/rule-plugin.md), out-of-tree for the same reason.

A *dynamically loaded, in-process Go API* — the `plugin/metric/` and `plugin/rule/` tree
sketched in [`CONTRIBUTING.md`](../CONTRIBUTING.md) — is **deferred and is not a v1 contract**;
see [compatibility.md](compatibility.md) for why and for what would change that. The exec
protocols are the extension boundary. Their wire envelope is versioned but pre-1.0 unstable; if
you have a metric in mind that needs domain data the envelope (or `Input`) does not yet carry,
open an issue and describe it — that is exactly what shapes it before the contract freezes.
