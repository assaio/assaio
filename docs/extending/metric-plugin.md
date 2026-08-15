# Write a metric plugin (any language)

*Part of [Extending assaio](../extending.md). The in-tree equivalent: [Adding a metric validator](metric-validator.md).*

**When to reach for this instead of an in-tree validator.** A metric plugin is your own
**analyzer** without forking assaio: an executable in any language that reads the same
prepared `Input` bundle every built-in validator reads and returns one `Result`. It
renders in `assaio analyze`, `analyze --format json`, and the Assay dashboard beside the
built-ins — same faceplate cell, same ledger entry, same anonymization rules. Reach for
it when the metric is company-specific and has no reason to be upstreamed; reach for an
[in-tree validator](metric-validator.md) when it belongs in every install or
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

## The protocol

`assaio` invokes `<command> analyze` with `ASSAIO_METRIC_PROTOCOL=1` in the environment,
writes one JSON envelope to the plugin's **stdin**, closes it, and reads stdout.

**stdin** — the prepared `Input`, versioned, camelCase (mirroring the public
`analyze --format json` shapes; only the version keys stay snake_case, matching the
parser protocol's `assaio_plugin`):

```json
{
  "assaio_metric_input": 2,
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
                "copilot-cli":["ai.cost.estimated","ai.lines.added","..."]},
  "windowStart": "2026-07-10T00:00:00Z",
  "planMonthlyCost": 200,
  "skills":     [{"name":"brainstorming","tokens":0,"lines":0,"records":0,"sessions":0}],
  "agents":     [{"name":"reviewer","tokens":0,"lines":0,"records":0,"sessions":0}],
  "turnSizing": [{"model":"claude-opus-5","turns":0,"smallTurns":0}],
  "cacheMisses":[{"tool":"claude-code","reason":"ttl_expired","turns":0}],
  "trace": [{"tool":"claude-code","sessionId":"…","member":"","timeline":"","entrypoint":"cli",
             "project":"…","scope":"interactive",
             "steps":[{"ordinal":1,"at":"2026-08-12T09:00:00Z","kind":"edit","outcome":"ok",
                       "model":"claude-opus-5","tokens":0,"targetRef":1}]}],
  "historyStart": "2026-07-13T08:11:04Z"
}
```

### `cacheWrite1h` — a subset, never a total (since v0.14)

`cacheWrite1h` on a usage row is the portion of `cacheWrite` that bought a 1-hour cache
lifetime, and `prices[model].cacheWrite1h` is that portion's own higher rate. It is a
**subset**: adding it to `cacheWrite` double-counts those tokens. Price a row the way the
core does — `min(cacheWrite1h, cacheWrite)` at the 1-hour rate, the remainder at
`cacheWrite`. Both fields are `0` for a source that does not report the tier, which reads the
same as "every write was 5-minute", so a plugin publishing a figure over them should declare
its own coverage. Before v0.14 neither field was on the wire, so a plugin re-pricing what it
was handed necessarily billed every write at the cheaper rate and reported a cost the core
disagreed with.

### `answers` — which zeros are measurements and which are silence

Every count on a `usage` or `sessions` row is **zero for a source that does not record it**,
and "nothing happened" and "nothing was written down" are different facts. A metric that
averages the two states what it did not measure — the mistake that made a Cline-only window
read as *100% conversational sessions* ([ADR 0011](../adr/0011-capability-gated-metrics.md)).

`answers` maps every tool present in this window to the
[signal](../adr/0008-signal-catalog.md) ids it can produce
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
than a side effect ([ADR 0006](../adr/0006-session-annotations.md)). If you need a metric per
kind of work today, run your plugin under `assaio-agent analyze --task <kind>`: the filter
is applied to the window before the input document is built, so your plugin computes over
exactly those sessions without needing to know why.

The semantics are exactly [`Input`'s](metric-validator.md#what-a-validator-reads-input): `usage` is
pre-aggregated by `(day, tool, model, project, entrypoint, member)`; `cost` fields are
`null` when unpriced, never a fabricated `0`; `byModel`/`byProject`/`totals` are the
prepared views to read first; `prices` carries only models present in the window's
usage. Like `Input`, the envelope is **versioned but pre-1.0 unstable** — a release that
reshapes it says so explicitly (see `RELEASING.md`).

**stdout** — a one-line handshake, then exactly **one** JSON `Result` document
(pretty-printed is fine; anything after it is a violation):

1. `{"assaio_metric": 2, "name": "<name>"}` — version must be `1`, `name` must equal
   the configured name.
2. One `Result` in the same shape `analyze --format json` emits — see [What a validator
   returns: Result](metric-validator.md#what-a-validator-returns-result). The wire `name` field is ignored:
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

### The window, not just its rows (since v0.17)

Six fields answer questions about the **window** that no sum over `usage` can reach, and until
v0.17 none of them crossed this boundary — which meant five shipped validators read data an
out-of-tree author could not. They are on the envelope now, and the parity is enforced by a
test: an `analyze.Input` field must either appear here or be listed as a deliberate exception
with its reason.

| Field | What it is, and the trap in it |
|---|---|
| `windowStart` | the `--since` boundary the usage was queried with. The zero time means the caller scoped no window. A monthly projection divides by *real days*, including the ones inside the window that carry no usage — those are still days a flat plan was paid for |
| `planMonthlyCost` | the configured flat subscription price. `0` means nobody configured one, **never** a plan that costs nothing — comparing against it as if it were free reports a saving that does not exist |
| `skills`, `agents` | per-skill and per-sub-agent totals. A row carrying no attribution is absent rather than bucketed under `""`, and both lists are empty when no tool in the window records attribution at all. That emptiness is a coverage fact to state, not a zero to publish |
| `turnSizing` | per-model turn counts at the raw per-turn grain the daily `usage` rows aggregate away. `smallTurns` is a **subset** of `turns`, not a separate population |
| `cacheMisses` | turns that stated a cache-miss reason, per tool. A turn that hit cache states nothing and is absent, as is every turn from a source that reports no reason — so this is never a denominator |
| `trace` | the window's step sequences: what each session did, in what order (ADR 0012). One entry per sequence — a session's main loop, or one sub-agent inside it — carrying the `scope` the core classified it as (`interactive`, `sub-agent`, `programmatic`, `unstated`). **Read one scope, not the set:** 89% of the sequences on the audited store are one-shot SDK calls holding 5.7% of its steps, so a rate spanning two scopes describes neither, and the share you excluded belongs beside your figure. `outcome` is `""` when the source said nothing, which is never `ok`; `targetRef` stands for the file a step named and is comparable **only inside its own sequence**, never across sequences and never a path. This is by far the largest thing on the wire — 339,000 steps encode to about 44 MB — and it is sent whether or not you read it, because the protocol has no way for you to say (`B168`) |
| `historyStart` | the earliest observation the store holds, **ignoring the window**. Compare it against the span your figure leans on: a trend against "the prior week" means nothing if the store's history began inside that week, and several tools delete their own transcripts (Claude Code after 30 days by default), which makes that the ordinary case rather than the odd one. The zero time means the core could not answer |

`skills`, `agents` and `cacheMisses` are all shaped the same way: **present means recorded,
absent means not recorded, and neither means zero.** Before publishing a figure over any of
them, check `answers` for whether the tools in this window record it at all, and declare the
coverage you actually had.

Those additions kept the envelope at `assaio_metric_input: 1`, because a plugin written
against the earlier shape keeps working and simply ignores them.

## What the boundary enforces

The honesty rules are enforced, not requested. A result that fails **any** check is
rejected whole — assaio never renders a fabricated or partially-sanitized verdict. On a
bare `analyze`/`dashboard` run a failing plugin is skipped with one `warning:` line and
the built-ins still render; an explicitly selected one (`analyze plugin:<name>`) is a
hard error.

- `layer` must be `activity`, `output`, `outcome` or `impact` — which of the four measurement
  layers your verdict rests on (ADR 0013). Required, because a built-in metric cannot compile
  without stating it and an extension surface weaker than the core it extends is not a surface.
  Declare what the **verdict** is a claim about: a figure inside the result may sit on a lower
  layer as context. `activity` is what happened (tokens, turns, tool calls, edits — cost too);
  `output` is what was produced (changed lines, commits, tests); `outcome` is whether it held
  (merged, survived, passed CI); `impact` is value delivered. Never relabel one as another.
- `read.key` must be `good`, `watch`, or `neutral`; `read.label` non-empty, ≤ 16 chars.
- `title` (≤ 80), `howToRead` and `takeaway` (≤ 400) are required; `describe` (≤ 200)
  and `caveats` (≤ 400 each, max 8) optional.
- Figures max 12, bars max 30; their `label`/`value`/`note` ≤ 120 chars each.
- No control characters anywhere (terminal-escape guard; the dashboard's HTML escaping
  is separate and automatic).
- `purity` and every `bars[].frac` are clamped to `[0,1]`.
- Stdout is capped at 1 MiB; the run is killed after `timeout` (default 60s).
- Two keys are **decoded and then discarded**, because they are answers about the window
  rather than about your metric: `lead` (`lead.rank`, `lead.reasons`) is which findings the
  window promotes, and a metric cannot see the others to rank itself among them; and
  `confidence.recorded`, which decides whether assaio prints the one insufficient-data reason
  that names a cure. Setting either is not an error and costs you nothing but the work — the
  core clears both on arrival and computes them itself.
- `barsPseudonym` works exactly as for built-ins: set it when your `Bars` rank
  by a name a person chose. The pre-rename key `barsAreProjects: true` is still accepted
  and maps to `"project"`, so a plugin released against it keeps being pseudonymized
  rather than silently publishing real project names. Any *other* unknown field is
  rejected outright — a misspelled key must not quietly disarm a setting. The
  [honesty constraints](../extending.md#honesty-constraints-for-every-extension) bind a metric plugin the
  same as any in-tree validator.

## A complete example (Python)

The same weekend-usage metric as the [in-tree worked
example](metric-validator-example.md) — one metric, both extension paths:

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

print(json.dumps({"assaio_metric": 2, "name": "weekend-usage"}))

if total == 0:
    print(json.dumps({
        "title": "Weekend Usage",
        "layer": "activity",
        "read": {"key": "neutral", "label": "—"},
        "howToRead": "A rising weekend share can mean crunch time or just flexible hours -- read it next to team sentiment, not as a verdict on its own.",
        "takeaway": "No usage in this window.",
    }))
    sys.exit(0)

share = weekend / total
watch = share > 0.2
print(json.dumps({
    "title": "Weekend Usage",
    "layer": "activity",
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
PLUGIN:WEEKEND-USAGE · Weekend Usage  [WATCH]  (activity)
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

**Where it deliberately does not run:** the [team server](team-server.md)'s served
dashboard (`GET /` is unauthenticated and rebuilds per request — spawning
config-declared subprocesses per request would be a denial-of-service vector), the
dashboard's per-project drill-down (built-ins only), and `demo` (deterministic sample).
It *does* run in `assaio-agent check` when you have configured [rule
plugins](rule-plugin.md), so a rule can gate on your own metric.
See [ADR 0004](../adr/0004-exec-metric-plugin-protocol.md) for the full rationale.

---
