# Extending assaio

Adapt `assaio` through documented, working extension points: add a **metric and dashboard section**
(one in-tree file or **out-of-tree in any language** without a fork), change a **log-source
location** in config, add a **tool as an out-of-tree plugin** (any language, no Go), define a **CI
gate** with a rule plugin (any language), or use **direct SQL** on your data.

This page maps the extension points and their rules. Each has its own guide. The binary generates
lists of every signal, source, validator, command, setting, and protocol field from its registries:

- **[`assaio-agent docs export`](#the-generated-reference)** — a machine-readable reference
  published at [assaio.dev/docs/reference](https://assaio.dev/docs/reference).

**The headline mechanism, in one paragraph.** Put a metric in one Go file under `internal/analyze/`.
It reads the same `Input` bundle as built-in metrics and returns one `Result`. Register it in that
file's `init()`, and it appears in `assaio analyze`, `assaio analyze --format json`, **and** the
HTML dashboard as a faceplate cell and ledger section with the same layout, colors, and captions as
built-ins. No other code or template edits are needed. [The worked
example](extending/metric-validator-example.md) verifies this end to end, including anonymization
when the metric's `Bars` rank by project name.

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

For background, read [what each source's log carries](extending/source-fields.md), the
field-by-field audit behind every depth row, and [the worked
example](extending/metric-validator-example.md).

## Recipes

The guides define the contracts; these are working examples to copy. **Recipes have different levels
of verification.** Code classifies each recipe, and the build fails if one is missing:

| how | what it means | how many |
|---|---|---|
| executed | run, and the output asserted — the label rules through the rule engine, the plugins against a fixture window with their output held to the protocol | 10 |
| commands-checked | every `assaio-agent` invocation in it names a real command and real flags; nothing runs the surrounding shell | 11 |
| loaded | parsed by assaio's own configuration loader and validated | 2 |
| shape-checked | parsed, and the method set held to the `Validator` interface: a renamed method fails, a wrong number does not | 3 |

Shell recipes have the weakest verification: the flags are real, but a reviewer judges whether the
surrounding pipeline works as described.

- [Extensions, written out in full](recipes/extensions.md) — complete validators and metric plugins,
  including how to gate on what the window can answer.
- [Label rules you can paste in](recipes/label-rules.md) — branch, skill, sub-agent, and entrypoint
  conventions for `mark --suggest`, which intentionally ships almost no defaults.
- [Rule plugins you can run today](recipes/rule-plugins.md) — three complete gates, starting with
  one that catches a verdict withheld for lack of data.
- [Gating CI on what a window cost](recipes/ci-gates.md) — `check` as a pre-push hook and scheduled
  job, with the meaning of each non-zero exit.
- [Running it without being asked](recipes/automation.md) — a weekly loop, digest delivery, and what
  to avoid automating.

## The generated reference

`assaio-agent docs export --format json` prints the binary's enumerable surfaces: the signal
catalog, source-depth matrix, validators and their scope, command tree with flags and defaults,
configuration keys and environment variables, and metric-plugin protocol fields. The same document
appears at [assaio.dev/docs/reference](https://assaio.dev/docs/reference).

The binary generates enumerable references, and `make test` fails when a published page disagrees
with it. Published files declare which claims can be checked. A hand-copied website list fell three
releases behind, and a shipped command went unpublished for another release. These pages still
explain what figures *mean* by hand: reflection can read names and types, but not intent.

If you are building on assaio, read the reference directly:

```console
$ assaio-agent docs export | jq '.validators[] | select(.scope=="window") | .name'
$ assaio-agent docs export | jq '.sources[] | {tool, tier, signals: (.answers | length)}'
```

## Honesty constraints for every extension

`assaio` promises to **measure value, not people; honest statistics or nothing** (`AGENTS.md`,
`CONTRIBUTING.md`). This applies to every extension whose metric or dashboard output people read:
in-tree validators, community PRs, and validator files in private forks. In practice:

- **Directional, not authoritative.** A `Read` (`Strong`/`Watch`/`Healthy`/…) is a diagnostic
  signal, not a verdict. If your metric relies on contested or incomplete evidence, or measures a
  proxy, explain that in `HowToRead` or a `Caveat`. Put the word "directional" in the rendered text,
  not only here.
- **`—` for an undefined ratio, never a fabricated one.** Show a dash for division by zero, never
  zero or 100%. Use `humanize.PercentOrDash` (`internal/humanize/percent.go`), `perActiveDay`
  (`internal/analyze/format.go`), or the same pattern. "0%" without a denominator is false. Even if
  an aggregate defaults to `0` when its denominator is zero (for example,
  `report.ChurnStat.ReworkRate`), a `Figure` must check the raw denominator instead of formatting
  that default. See the "rework" figure in `internal/analyze/rework.go`, which passes
  `ReworkLines`/`LinesAdded` to `humanize.PercentOrDash` instead of formatting `ReworkRate`.
- **A silence is not a zero.** Before reading a column, check whether the row's source can produce
  it: `answers` on the wire or `parser.Answers` in-tree (ADR 0011). A source that never records
  cache writes leaves the counter at zero; that does not mean the cache was never written.
- **Aggregate and pseudonymized by default; per-person only as a governed opt-in.** `Input` has no
  user identity today. It groups by project, tool, model, and entrypoint, never person. Rank those
  dimensions as `throughput` ranks projects, never individuals. If your `Bars` rank names chosen by
  people, set `Result.BarsPseudonym` to `"project"` or `"skill"`. The dashboard's `--anonymize`, on
  by default, then pseudonymizes those labels as it does for built-in `throughput`.
  `internal/dashboard.anonymizeVerdicts` applies this to every validator, including yours. Leave the
  field empty for other dimensions, such as models and tools; they must never be pseudonymized. Any
  future per-member breakdown requires a deliberate, consented team-mode opt-in. It must never be
  silent, a leaderboard, or a tool for individual performance evaluation.
- **Say so when you approximate.** If stored aggregates cannot show a value precisely, label the
  rendered figure approximate. `Input.Usage` is already grouped, so per-record detail is gone.
- **Never a per-person scoreboard.** Even in team mode, an extension must not present ranked, named
  individual usage as a performance signal. See `PRIVACY.md`.

The built-in validators in `internal/analyze` follow these rules, with tests
(`TestValidatorsEmptyInputSafe`, `TestReworkDashOnZeroToolCalls`,
`TestBuildNeverAnonymizesModelNames`). Code review holds new validators to the same standard.

## Custom metrics (what's shipped vs. roadmap)

Custom metrics ship **two ways today**: an in-tree, one-file-per-metric
[validator](extending/metric-validator.md), compiled in and available everywhere, including [the
team server](extending/team-server.md); or an out-of-tree [metric
plugin](extending/metric-plugin.md), written in any language without a fork, declared in config, and
run by `analyze` and the local dashboard. Set thresholds for those metrics with out-of-tree [rule
plugins](extending/rule-plugin.md).

A *dynamically loaded, in-process Go API*—the `plugin/metric/` and `plugin/rule/` tree proposed in
[`CONTRIBUTING.md`](../CONTRIBUTING.md)—is **deferred and is not a v1 contract**.
[compatibility.md](compatibility.md) explains why and what could change that. Exec protocols define
the extension boundary. Their wire envelope is versioned but unstable before 1.0. If your metric
needs domain data absent from the envelope or `Input`, open an issue describing it so the contract
can account for it before it freezes.
