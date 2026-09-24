# Write a metric plugin (any language)

*Part of [Extending assaio](../extending.md). In-tree equivalent: [Adding a metric
validator](metric-validator.md).*

**When to use this instead of an in-tree validator.** A metric plugin is your own **analyzer**
without an assaio fork. It can be an executable in any language. It reads the prepared `Input`
bundle used by built-in validators and returns one `Result`. It appears beside built-ins in
`assaio analyze`, `analyze --format json`, and the Assay dashboard, with the same faceplate cell,
ledger entry, and anonymization rules. Use it for a company-specific metric that does not belong
upstream. Use an [in-tree validator](metric-validator.md) if the metric belongs in every install or
needs domain data absent from the wire envelope.

Metric plugins are **opt-in only**. Declare them under `metrics:` in `~/.config/assaio/config.yaml`;
they are never discovered from `PATH` or downloaded. Entries have the same shape as `plugins:`. One
binary can serve both protocols (`scan` and `analyze` argv) by appearing in both lists:

```yaml
metrics:
  - name: weekend-usage      # required, [a-z0-9-]+; appears as "plugin:weekend-usage"
    command: /path/to/assaio-metric-weekend   # required; PATH lookup if not absolute
    timeout: 30s             # optional, default 60s
    needs: [usage]           # optional veto -- see "Declare what you read" below
```

**Start with a working skeleton.** `assaio-agent plugins init --kind metric --lang python` prints a
runnable metric plugin to stdout, with the correct handshake, both verbs, and one accepted result.
It prints next steps to stderr, so
`assaio-agent plugins init --kind metric --lang python > my-metric.py` writes only the program.
`--lang go|python|sh` and `--kind parser|metric|rule` cover every combination.

**Privacy note.** A metric plugin **receives your usage data on stdin**: project names, model names,
member pseudonyms, and token and line counts. This is the same aggregate metadata held by the store.
It never receives prompt, response, or repository content; none is stored (see `PRIVACY.md`). A
parser plugin instead reads a tool's own logs. Both are local programs you choose to run with your
privileges. Check what a binary receives before adding one you did not write to config.

## The protocol

`assaio` runs your plugin **twice per analysis**, each time with `ASSAIO_METRIC_PROTOCOL=1` in the
environment:

1. `<command> describe` — receives nothing on stdin. Write a handshake line and **one Declaration**
   stating what the metric reads.
2. `<command> analyze` — receives the window on stdin, limited to what you declared. Write a
   handshake line and **one `Result`**.

The extra process limits the envelope: on a real 30-day store, the full window is 53 MB; an ordinary
metric's projection is 43 KB (see below).

**stdin** — the prepared, versioned `Input` in camelCase, matching public `analyze --format json`
shapes. Only version keys use snake_case, matching the parser protocol's `assaio_plugin`. This
example declares everything; undeclared sections are **absent from the document**:

```json
{
  "assaio_metric_input": 4,
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
  "historyStart": "2026-07-13T08:11:04Z",
  "projection": {"needs":["usage","trace"],
                 "fields":{"usage":["day","in","out"],"trace.steps":["kind","outcome"]},
                 "where":{"trace.scope":["interactive"]},
                 "rows":{"usage":{"sent":522,"available":522},
                         "trace":{"sent":471,"available":4279}}}
}
```

### Declare what you read (since v0.25, protocol 4)

`describe` declares what the metric reads. assaio sends only that.

```json
{"assaio_metric": 4, "name": "weekend-usage"}
{"needs": ["usage"], "fields": {"usage": ["day", "tool", "in", "out"]}}
```

| Key | What it does |
|---|---|
| `needs` | **Required**, at least one. The capability vocabulary a built-in validator declares: `usage`, `sessions`, `trace`, `attribution`, `turn-sizing`, `cache-misses`, `prices`. An empty list is refused — a metric that reads nothing has nothing to report, and treating it as "everything" would restore the pre-4 default under a name saying the opposite |
| `fields` | Optional. Columns of a section, keyed by the section's JSON key. A section you do not name arrives whole. `trace.steps` is addressable on its own, which is where the bytes are |
| `where` | Optional. Rows, keyed `<section>.<column>` with the values that column may hold. **Grain is a column**: `usage.granularity` picks `turn` or `session` rows, `trace.scope` picks `interactive`, `sub-agent`, `programmatic` or `unstated`. Only string columns, and only top-level rows — a predicate inside a sequence would leave its ordinals describing a set nobody declared |

Capabilities carry these sections: `usage` carries `usage`, `byModel`, `byProject`, `totals`, and
`delegation`; `sessions` carries `sessions`; `prices` carries `prices` and `planMonthlyCost`;
`attribution` carries `skills` and `agents`; `turn-sizing`, `cache-misses`, and `trace` carry their
namesake sections. Prepared views accompany their rows because model and project counts bound them,
not observation counts, but only when those rows cover the whole window. A `where` predicate on
`usage`, `byModel`, or `byProject` **withholds `totals` and `delegation` entirely**: they span the
window, not your selected rows. Otherwise, `sum(usage.in) / totals.tokens` would compare different
populations. The window denominator remains available as `projection.rows[<section>].available`.
Presence depends on your declaration, never the outcome, so keys do not appear in one window and
vanish in the next. `now`, `recentDays`, `windowStart`, `historyStart`, `answers`, `projection`, and
`withheld` are always sent to show what you received.

**A column you did not project is absent, and absent is not zero.** `projection.fields` lists the
keys you can read. Decoding a missing key into a struct field can produce a `0` no source recorded.
For sections, check `projection.needs` before reading a key.

**A predicate changes your denominator.** `projection.rows[section]` reports `sent` rows received
and `available` rows in the window before your predicate. A share over filtered rows uses `sent` as
its denominator. State which denominator your figure uses in `confidence.signalCoverage`.

**Envelope sizes.** The same real 30-day window contains 522 usage rows, 3,376 sessions, 4,279 step
sequences, and 424,310 steps. Its serialized size under four declarations:

| What the plugin gets | Bytes |
|---|---|
| protocol 3, no `needs:` line (everything except `trace`) | 1,237,872 (1.18 MB) |
| protocol 3, `needs: [trace]` (everything) | 55,864,147 (53.28 MB) |
| protocol 4, `needs: [usage]` with four columns | 43,779 (0.04 MB) |
| protocol 4, `needs: [trace]`, three sequence and two step columns, `trace.scope = interactive` | 7,531,004 (7.18 MB) |

**Your config entry can veto the declaration.** Before v0.25, the reader set `needs:` in
`config.yaml` and had to know what your plugin read. Now your plugin declares it, and their `needs:`
only *narrows* it. No key means no constraint; allowing more grants nothing extra. Allowing less
omits the section, names the capability in top-level `withheld`, and adds a caveat to the rendered
verdict that it received less than requested.

### Migrating a protocol-3 plugin

A protocol-3 plugin fails the versioned handshake before its window is serialized. Make two edits:

1. Change the handshake integer from `3` to `4` in both verbs.
2. Add a `describe` branch that prints the handshake and one declaration. To reproduce protocol 3's
   payload exactly:
   `{"needs":["usage","sessions","trace","attribution","turn-sizing","cache-misses","prices"]}`.

You can narrow that declaration to get the smaller payloads in the table above. An existing `needs:`
line in someone's `config.yaml` still works, but now vetoes parts of your declaration instead of
extending it.

### `cacheWrite1h` — a subset, never a total (since v0.14)

`cacheWrite1h` on a usage row is the part of `cacheWrite` bought at the 1-hour cache lifetime.
`prices[model].cacheWrite1h` gives its higher rate. It is a **subset**: adding it to `cacheWrite`
double-counts tokens. Price it as the core does: charge `min(cacheWrite1h, cacheWrite)` at the
1-hour rate and the rest at `cacheWrite`. Both fields are `0` when a source does not report the
tier, indistinguishable from every write being 5-minute; declare coverage for figures based on them.
Before v0.14, neither field was on the wire, so plugin repricing billed all writes at the cheaper
rate and disagreed with core cost.

### `answers` — which zeros are measurements and which are silence

Every count on a `usage` or `sessions` row is **zero when the source does not record it**. That
differs from nothing happening. Averaging both states claims a measurement you lack; this made a
Cline-only window appear to have *100% conversational sessions* ([ADR
0011](../adr/0011-capability-gated-metrics.md)).

`answers` maps each tool in the window to the [signal](../adr/0008-signal-catalog.md) ids it can
produce (`assaio-agent signals list`). An out-of-tree parser gets the exec protocol's floor because
its author determines its capabilities. Follow the in-tree validator rule: **keep rows from tools
that answer the signal, compute from those rows, and declare the reach**. If none remain, print no
figure instead of zero and withhold the verdict.

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

Only tools in the window are sent. The plugin needs their capabilities, not the whole matrix, which
would publish it again in the envelope.

**Session labels are absent from this document.** In-tree validators can read task, outcome, and
difficulty labels added with `assaio-agent mark`. The plugin wire does not carry them or split usage
rows by them; its shape is unchanged. Labels are local, and sending them across the process boundary
requires a separate decision ([ADR 0006](../adr/0006-session-annotations.md)). For a metric by kind
of work, run the plugin with `assaio-agent analyze --task <kind>`. The filter runs before input is
built, so the plugin receives only matching sessions without knowing why.

The envelope follows [`Input`'s](metric-validator.md#what-a-validator-reads-input) semantics:
`usage` is pre-aggregated by `(day, tool, model, project, entrypoint, member)`; unpriced `cost`
fields are `null`, never fabricated `0`; read the prepared `byModel`/`byProject`/`totals` views
first; and `prices` includes only models used in the window. Like `Input`, it is **versioned but
pre-1.0 unstable**. Releases that reshape it say so explicitly (see `RELEASING.md`).

**stdout** — one handshake line, then exactly **one** JSON `Result` document. Pretty printing is
fine; anything after it violates the protocol:

1. `{"assaio_metric": 4, "name": "<name>"}` — version must be `4` (`3` before v0.25; see
   [CHANGELOG.md](../../CHANGELOG.md)), and `name` must match the configured name. This line also
   opens `describe`.
2. One `Result` in the shape emitted by `analyze --format json` (see [What a validator returns:
   Result](metric-validator.md#what-a-validator-returns-result)). assaio ignores the wire `name` and
   stamps `plugin:<name>`, so a plugin cannot shadow a built-in validator.

Anything written to stderr passes through with the prefix `[metric/<name>] `.

**Declare what supports your verdict** (v0.5). Every result has a confidence envelope. assaio fills
every part except the observation count, which only your plugin knows.

```json
"confidence": {"samples": 12, "samplesUnit": "sessions"}
```

assaio stamps coverage, freshness, and parsing build from the same window used by built-in metrics.
If your plugin omits the field, its verdict is `insufficient` because it states no basis. Declare it
even for large counts. Count observations, not reported buckets: "3 models" describes a shape; "31
active days" is evidence.

**Declare how much of the window your figure covers** (v0.9) if it covers less than all of it:

```json
"confidence": {"samples": 12, "samplesUnit": "sessions", "signalCoverage": 0.05}
```

The three stamped axes describe the *window*; this axis describes your *question*, which only your
metric can assess. A figure from one source in a five-source window covers a small share that
window-level measures miss. `reasoning-share` reported a 20% share from under 1% of a store's output
and kept a `high` label until it declared this share. Omitting the key claims the whole window, as
plugins released before this axis did; that remains correct for most metrics. Declaring `0` yields
`insufficient`: nothing in the window answered, rather than a thin answer. This share helps set the
label, and `assaio analyze` names it when it is the weakest axis:

```
Confidence: low · 43 active days · signal coverage <1%
```

**An unsupported verdict says why** (v0.10). `insufficient` has four distinct causes. The summary
line names the one that applies:

| Line | What you declared |
|---|---|
| `insufficient — a source here records it, but your stored rows predate that capture` | `signalCoverage: 0` on a subject whose own denominator exists and whose capture a source in the window does have — the one cause with a cure (`backfill`) |
| `insufficient — nothing in this window can answer it` | `signalCoverage: 0` — the window may be full of usage, none of which reaches your question |
| `insufficient — 0 sessions` | `samples: 0` with a unit — you counted none of your own observations |
| `insufficient — no stated basis` | no `samples` at all — the honest reading of a plugin that did not say what it rests on |

The first row does not apply to an exec metric plugin: it requires the validator to declare which
missing capture a zero represents, and the plugin protocol has no field for that. A plugin covering
none of the window uses the second row.

Declare `samples: 0` with a unit instead of omitting the field: "zero sessions" is a measurement;
"no stated basis" is not.

### The window, not just its rows (since v0.17)

Six fields answer **window** questions that sums over `usage` cannot. Before v0.17, none crossed
this boundary, so out-of-tree authors lacked data used by five shipped validators. They are now in
the envelope. A test enforces parity: every `analyze.Input` field must appear here or be listed as a
deliberate exception with a reason.

| Field | What it is, and the trap in it |
|---|---|
| `windowStart` | the `--since` boundary the usage was queried with. The zero time means the caller scoped no window. A monthly projection divides by *real days*, including the ones inside the window that carry no usage — those are still days a flat plan was paid for |
| `planMonthlyCost` | the configured flat subscription price. `0` means nobody configured one, **never** a plan that costs nothing — comparing against it as if it were free reports a saving that does not exist |
| `skills`, `agents` | per-skill and per-sub-agent totals. A row carrying no attribution is absent rather than bucketed under `""`, and both lists are empty when no tool in the window records attribution at all. That emptiness is a coverage fact to state, not a zero to publish |
| `turnSizing` | per-model turn counts at the raw per-turn grain the daily `usage` rows aggregate away. `smallTurns` is a **subset** of `turns`, not a separate population |
| `cacheMisses` | turns that stated a cache-miss reason, per tool. A turn that hit cache states nothing and is absent, as is every turn from a source that reports no reason — so this is never a denominator |
| `withheld` | the capabilities you declared that this install's `needs:` entry refused. Present only when something was refused, and it is the *only* absence assaio decided — a section you never declared is simply not on the document, and `projection.needs` is what says so. Read both: an absent section and a window that genuinely holds none of that evidence look identical otherwise, and dividing by one you were never sent is the fabricated zero this protocol exists to refuse |
| `projection` | what this envelope carries and why: `needs` (the capabilities granted), `fields` and `where` (your own narrowing, echoed back after the config's veto), and `rows` — per section, how many rows you were `sent` out of how many were `available` before your predicate ran. It is what makes a projected document self-describing; nothing else on the wire says whether an absence was chosen or measured |
| `trace` | the window's step sequences: what each session did, in what order (ADR 0012). One entry per sequence — a session's main loop, or one sub-agent inside it — carrying the `scope` the core classified it as (`interactive`, `sub-agent`, `programmatic`, `unstated`). **Read one scope, not the set:** 89% of the sequences on the audited store are one-shot SDK calls holding 5.7% of its steps, so a rate spanning two scopes describes neither, and the share you excluded belongs beside your figure — declare `where: {"trace.scope": ["interactive"]}` and `projection.rows.trace` tells you what you left out. `outcome` is `""` when the source said nothing, which is never `ok`; `targetRef` stands for the file a step named and is comparable **only inside its own sequence**, never across sequences and never a path. By far the largest section — 424,000 steps encode to 53 MB — which is what projecting `trace.steps` down to the two columns you read is for |
| `historyStart` | the earliest observation the store holds, **ignoring the window**. Compare it against the span your figure leans on: a trend against "the prior week" means nothing if the store's history began inside that week, and several tools delete their own transcripts (Claude Code after 30 days by default), which makes that the ordinary case rather than the odd one. The zero time means the core could not answer |

`skills`, `agents`, and `cacheMisses` share a rule: **present means recorded; absent means not
recorded; neither means zero.** Before publishing a figure based on them, check `answers` to see
whether the window's tools record them, and declare actual coverage.

These additions kept the envelope at `assaio_metric_input: 1`: older plugins ignored the new fields
and still worked. Protocol 4 first changed the request instead of the answer, so it could not be
additive.

## What the boundary enforces

assaio enforces the honesty rules. A result failing **any** check is rejected whole; no fabricated
or partly sanitized verdict is rendered. On a bare `analyze`/`dashboard` run, assaio skips a failing
plugin, prints one `warning:` line, and still renders built-ins. An explicitly selected plugin
(`analyze plugin:<name>`) fails with a hard error.

- The **declaration** is rejected whole, with every reason, before any window is serialized. `needs`
  must be present, non-empty, contain only known capabilities, and have no duplicates. Each `fields`
  key must name a section whose capability you declared; each listed column must exist in that
  section. Each `where` key must address `<section>.<column>` on a declared top-level section, use a
  string column, and allow at least one value. Nothing is repaired: a silently dropped projected
  column would read as absent.
- `layer` is required and must be `activity`, `output`, `outcome`, or `impact`: the verdict's
  measurement layer (ADR 0013). Built-in metrics must state it to compile, so extensions must state
  it too. A figure within the result may provide context from a lower layer. `activity` covers what
  happened (tokens, turns, tool calls, edits, and cost); `output` covers what was produced (changed
  lines, commits, tests); `outcome` covers whether it held (merged, survived, passed CI); `impact`
  covers value delivered. Never relabel one as another.
- `read.key` must be `good`, `watch`, or `neutral`; `read.label` must be non-empty and ≤ 16 chars.
- `title` (≤ 80), `howToRead`, and `takeaway` (≤ 400) are required; `describe` (≤ 200) and `caveats`
  (≤ 400 each, max 8) are optional.
- At most 12 figures and 30 bars; each `label`/`value`/`note` is ≤ 120 chars.
- No control characters anywhere (a terminal-escape guard; dashboard HTML escaping is separate and
  automatic).
- `purity` and every `bars[].frac` are clamped to `[0,1]`.
- Stdout is capped at 1 MiB; the run is killed after `timeout` (default 60s).
- Two keys are **decoded and then discarded** because they describe the window, not your metric:
  `lead` (`lead.rank`, `lead.reasons`) ranks findings, but a metric cannot see the others;
  `confidence.recorded` determines whether assaio prints the insufficient-data reason that names a
  cure. Setting either is allowed, but the core clears and computes both itself.
- `barsPseudonym` works as it does for built-ins: set it when your `Bars` rank by a person-chosen
  name. The older `barsAreProjects: true` is accepted and maps to `"project"`, so old plugins still
  pseudonymize project names. Any *other* unknown field is rejected: a typo must not disable a
  setting. The [honesty constraints](../extending.md#honesty-constraints-for-every-extension) apply
  as they do to in-tree validators.

## A complete example (Python)

The same weekend-usage metric as the [in-tree worked example](metric-validator-example.md), shown
through both extension paths:

```python
#!/usr/bin/env python3
"""assaio-metric-weekend: share of AI tokens used on Saturday/Sunday."""
import json, sys
from datetime import date

HANDSHAKE = {"assaio_metric": 4, "name": "weekend-usage"}

if sys.argv[1:2] == ["describe"]:
    # Four columns of one section. Everything else -- sessions, the step timeline, the
    # price table -- is never serialized for this plugin at all.
    print(json.dumps(HANDSHAKE))
    print(json.dumps({"needs": ["usage"], "fields": {"usage": ["day", "in", "out"]}}))
    sys.exit(0)

inp = json.load(sys.stdin)
weekend = total = 0
for row in inp["usage"]:
    tokens = row["in"] + row["out"]
    total += tokens
    if date.fromisoformat(row["day"]).weekday() >= 5:
        weekend += tokens

print(json.dumps(HANDSHAKE))

if total == 0:
    print(json.dumps({
        "title": "Weekend Usage",
        "layer": "activity",
        "read": {"key": "neutral", "label": "\u2014"},
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

Make the plugin executable and declare it under `metrics:` as above. Check conformance with
`metrics verify`: it runs the plugin on your real store's window **without storing anything** and
prints any violations plus the rendered result:

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

A bare `assaio-agent analyze` then prints it after the built-ins.
`assaio-agent analyze plugin:weekend-usage` runs it alone; `analyze --list` lists it without
executing it; and `assaio-agent dashboard` gives it a faceplate cell and ledger entry like a
built-in.

## Conformance vectors — your CI without this binary

`docs/conformance/` contains every document this boundary accepts or rejects, with the verdict and
reason:

| File | The document it judges |
|---|---|
| [`metric-declaration.json`](../conformance/metric-declaration.json) | what `describe` writes |
| [`metric-result.json`](../conformance/metric-result.json) | what `analyze` writes |
| [`rule-alerts.json`](../conformance/rule-alerts.json) | what a [rule plugin](rule-plugin.md) writes |
| [`parser-record.json`](../conformance/parser-record.json) | one record from a [parser plugin](parser-plugin.md) |

Each file is `{about, contract, protocol, vectors: [{id, doc, accept, expect, why}]}`. `doc`
is the document as a **string**, so a malformed one — trailing data, a control character, text
that is not JSON at all — is representable. Feed each `doc` to your decoder and assert
`accept`; when it is `false`, your rejection reason should mention `expect`.

The same files drive assaio's tests and seed its fuzzers. If a vector stops describing the boundary,
assaio's build fails instead of yours.

**Where it does not run:** the [team server](team-server.md)'s served dashboard (`GET /` rebuilds on
each request, so spawning configured subprocesses could enable denial of service), the dashboard's
per-project drill-down (built-ins only), and `demo` (a deterministic sample). It *does* run in
`assaio-agent check` when [rule plugins](rule-plugin.md) are configured, so a rule can gate on your
metric. See [ADR 0004](../adr/0004-exec-metric-plugin-protocol.md) for the full rationale.

---
