# Extensions, written out in full

*Part of [Extending assaio](../extending.md). The contracts these implement: [in-tree validator](../extending/metric-validator.md) · [metric plugin](../extending/metric-plugin.md).*

The guides describe the contract; this page is the part people actually want first — complete
extensions, short enough to read in one go, each showing one thing worth copying.

**What is checked, stated plainly.** The Python plugins are *executed* by
`TestRecipeMetricPlugins` against a fixture window and asserted on — including the one below that
gates on `answers`, whose fixture is built so that deleting the gate changes the published figure
from 30.0 to 50.0 and fails the test. The Go validators are *shape-checked* by
`TestRecipeValidatorsMatchTheInterface`: parsed, and their method set held to the `Validator`
interface, because a Go file in a document cannot be compiled without a package around it.
Shape-checking catches a renamed method and a changed signature; it does not catch a wrong
number, which is what the honesty rules and a reviewer are for.

## A validator, as small as one gets

Everything a metric needs: read the prepared `Input`, return one `Result`, register from
`init()`. Nothing else is wired — it appears in `analyze`, in `analyze --format json` and in the
dashboard because the registry is walked, not enumerated.

```go #weekday-split
package analyze

func init() { Register(weekdayValidator{}) }

type weekdayValidator struct{}

func (weekdayValidator) Name() string  { return "weekday-split" }
func (weekdayValidator) Title() string { return "Weekday Split" }
func (weekdayValidator) Describe() string {
	return "Share of AI tokens spent Monday to Friday -- an out-of-hours proxy, not a verdict."
}

func (weekdayValidator) Analyze(in Input) Result {
	var weekday, total int64
	for _, u := range in.Usage {
		day, err := time.Parse("2006-01-02", u.Day)
		if err != nil {
			continue
		}
		tokens := u.In + u.Out + u.CacheRead + u.CacheWrite
		total += tokens
		if d := day.Weekday(); d != time.Saturday && d != time.Sunday {
			weekday += tokens
		}
	}
	r := Result{
		Name: "weekday-split", Title: "Weekday Split",
		Describe:  "Share of AI tokens spent Monday to Friday.",
		HowToRead: "A falling weekday share can mean crunch or flexible hours; read it beside how the team says it is working, never on its own.",
		Caveats:   []string{"Directional: a proxy for when work happened, not for how much."},
	}
	if total == 0 {
		r.Read = Read{Key: "neutral", Label: "—"}
		r.Takeaway = "No usage in this window."
		return r
	}
	share := float64(weekday) / float64(total)
	r.Purity = share
	r.Figures = []Figure{{Label: "weekday token share", Value: humanize.PercentOrDash(weekday, total, 1)}}
	r.Read = Read{Key: "good", Label: "STEADY"}
	r.Takeaway = "Most usage falls inside the working week."
	if share < 0.8 {
		r.Read = Read{Key: "watch", Label: "WATCH"}
		r.Takeaway = "A meaningful share of usage falls outside the working week."
	}
	return r
}
```

Note what the no-data path does: `neutral`, an em dash, and a takeaway that says so. A metric
that returns `0%` when it has no denominator is a lie dressed as a number, and it is the first
thing a review of a new validator looks for.

## A validator that refuses to read a silence as a zero

The rule that separates a metric from a mistake: before reading a column, keep only the rows
whose source can actually record it. A source that never writes a cache counter leaves the field
at zero, and averaging that in reports "the cache was never written" for tools that simply do not
say.

```go #answers-gated
package analyze

func init() { Register(editSizeValidator{}) }

type editSizeValidator struct{}

func (editSizeValidator) Name() string     { return "edit-size" }
func (editSizeValidator) Title() string    { return "Edit Size" }
func (editSizeValidator) Describe() string { return "AI lines per edit, over the sources that count edits." }

func (editSizeValidator) Analyze(in Input) Result {
	// Only rows whose tool records both signals. Everything else is a silence, not a zero.
	rows := report.UsageAnswering(in.Usage, parser.SignalEditsCount)
	rows = report.UsageAnswering(rows, parser.SignalLinesAdded)

	var lines, edits int64
	for _, u := range rows {
		lines += u.LinesAdded
		edits += u.Edits
	}
	r := Result{
		Name: "edit-size", Title: "Edit Size",
		Describe:  "AI lines per edit, over the sources that count edits.",
		HowToRead: "Large edits are not worse than small ones; a sharp change in either direction is the signal.",
		Caveats: []string{fmt.Sprintf("Covers %s of the window's tokens: the rest comes from sources that record no edit count.",
			humanize.PercentOrDash(report.TokensIn(rows), report.TokensIn(in.Usage), 0))},
	}
	if edits == 0 {
		r.Read = Read{Key: "neutral", Label: "—"}
		r.Takeaway = "No source in this window records an edit count."
		return r
	}
	r.Read = Read{Key: "good", Label: "STEADY"}
	r.Figures = []Figure{{Label: "AI lines per edit", Value: fmt.Sprintf("%.1f", float64(lines)/float64(edits))}}
	r.Takeaway = "Edit size is measurable in this window."
	return r
}
```

The caveat is not decoration. It states the share of the window the figure actually describes,
which is the difference between a number and a number you can act on.

## A metric plugin, any language

The same metric as an executable: no fork, no Go, declared in config under `metrics:`. It reads
one JSON document on stdin and writes a handshake plus one `Result` on stdout.

```python #plugin-weekday
#!/usr/bin/env python3
"""Weekday token share as an out-of-tree metric plugin."""
import datetime as dt
import json
import sys

inp = json.load(sys.stdin)
weekday = total = 0
for row in inp.get("usage", []):
    tokens = row["in"] + row["out"] + row["cacheRead"] + row["cacheWrite"]
    total += tokens
    if dt.date.fromisoformat(row["day"]).weekday() < 5:
        weekday += tokens

result = {
    "title": "Weekday Split",
    "describe": "Share of AI tokens spent Monday to Friday.",
    "howToRead": "A falling weekday share can mean crunch or flexible hours; read it beside how the team says it is working.",
    "caveats": ["Directional: a proxy for when work happened, not for how much."],
}
if total == 0:
    result |= {"read": {"key": "neutral", "label": "—"}, "takeaway": "No usage in this window."}
else:
    share = weekday / total
    result |= {
        "read": {"key": "good" if share >= 0.8 else "watch",
                 "label": "STEADY" if share >= 0.8 else "WATCH"},
        "purity": share,
        "figures": [{"label": "weekday token share", "value": f"{share:.1%}"}],
        "takeaway": ("Most usage falls inside the working week." if share >= 0.8
                     else "A meaningful share of usage falls outside the working week."),
    }

print(json.dumps({"assaio_metric": 1, "name": "weekday-split"}))
print(json.dumps(result))
```

`name` is stamped by the core as `plugin:<name>` on arrival, so a plugin cannot shadow a
built-in validator; setting it in the result is not an error, it is simply overwritten.

## A metric plugin that checks what the window can answer

The wire carries `answers`: a map from each tool present in the window to the signal ids it can
produce. A plugin that ignores it is exposed to exactly the bug the in-tree gate exists to
prevent — and unlike a validator, it cannot call into the capability matrix, which is why the
envelope hands it over.

```python #plugin-answers
#!/usr/bin/env python3
"""AI lines per edit, counting only the sources that record edits."""
import json
import sys

NEEDED = ("ai.edits.count", "ai.lines.added")

inp = json.load(sys.stdin)
answers = inp.get("answers", {})
capable = {tool for tool, ids in answers.items() if all(n in ids for n in NEEDED)}

lines = edits = counted = total = 0
for row in inp.get("usage", []):
    tokens = row["in"] + row["out"] + row["cacheRead"] + row["cacheWrite"]
    total += tokens
    if row["tool"] not in capable:
        continue
    counted += tokens
    lines += row["linesAdded"]
    edits += row["edits"]

covered = (counted / total) if total else 0
result = {
    "title": "Edit Size",
    "describe": "AI lines per edit, over the sources that count edits.",
    "howToRead": "A sharp change in either direction is the signal; the level itself is a house style.",
    "caveats": [f"Covers {covered:.0%} of the window's tokens: the rest comes from sources that record no edit count."],
}
if edits == 0:
    result |= {"read": {"key": "neutral", "label": "—"},
               "takeaway": "No source in this window records an edit count."}
else:
    result |= {
        "read": {"key": "good", "label": "STEADY"},
        "figures": [{"label": "AI lines per edit", "value": f"{lines / edits:.1f}"}],
        "takeaway": "Edit size is measurable in this window.",
    }

print(json.dumps({"assaio_metric": 1, "name": "edit-size"}))
print(json.dumps(result))
```

Both plugins are wired the same way:

```yaml #metric-config
metrics:
  - name: weekday-split
    command: ./scripts/weekday-split.py
    timeout: 30s
```

and checked before you trust them:

```sh #verify-metric
assaio-agent metrics verify weekday-split --since 30d
```

`metrics verify` runs the plugin on your real window and prints both the contract violations and
the rendered result, storing nothing.

## A detector: reading the sequence, not the total

The step timeline (ADR 0012) is the one input that is not an aggregate: it holds what a session
did, in what order. A detector reads a *scope* of it — never the whole set — because the
populations are not comparable: 89% of the sequences on the audited store are one-shot SDK calls
holding 5.7% of its steps, so a rate spanning two scopes describes neither. Declaring the scope is
what `TraceReader` is for, and the caveat naming what the pattern cannot be told apart from is not
optional decoration: a hard bug and a loop look identical on a timeline.

This one counts a file read again inside the same sequence — the cheapest form of "it looked at the
same thing twice".

```go #read-repeats
package analyze

import (
	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/trace"
	"github.com/assaio/assaio/internal/usage"
)

func init() { Register(reReadsValidator{}) }

type reReadsValidator struct{}

func (reReadsValidator) Name() string  { return "re-reads" }
func (reReadsValidator) Title() string { return "Re-reads" }
func (reReadsValidator) Describe() string {
	return "How often a session reads a file it has already read in the same sequence."
}

// TraceScope is the population this answers for. Implementing it is what makes the validator a
// detector: the core reads it to skip the sequence query when nothing wants it, and a scope
// outside internal/trace's vocabulary yields an empty view rather than a silently wider one.
func (reReadsValidator) TraceScope() string { return trace.Interactive }

func (v reReadsValidator) Analyze(in Input) Result {
	r := Result{Name: "re-reads", Title: "Re-reads", Describe: v.Describe(),
		HowToRead: "Re-reading a file is normal -- context is lost, a file changes. A session doing it far more than the rest is worth opening."}
	view := in.Trace.Scope(v.TraceScope())
	if view.Empty() {
		r.noData("sessions", "No sequence in this window is a session someone ran from a terminal.")
		return r
	}

	var reads, repeats int64
	for i := range view.Sequences {
		seen := map[int64]bool{}
		for _, step := range view.Sequences[i].Steps {
			if step.Kind != usage.StepRead || step.TargetRef == 0 {
				continue
			}
			reads++
			if seen[step.TargetRef] {
				repeats++
			}
			seen[step.TargetRef] = true
		}
	}

	r.restsOn(len(view.Sequences), "sessions")
	r.covering(1 - view.ExcludedStepShare())
	r.Figures = []Figure{{
		Label: "re-read rate",
		Value: humanize.PercentOrDash(repeats, reads, 1),
		Note:  humanize.Int(repeats) + " of " + humanize.Int(reads) + " reads had seen the file already",
	}}
	r.Takeaway = "Directional: a re-read is how an agent recovers context, not a fault."
	// Both sentences are the contract. The first states the population and what asking for it left
	// out; the second states what this pattern cannot be told apart from.
	r.Caveats = append(r.Caveats, view.Caveat(),
		"Cannot distinguish: a file legitimately re-read after it changed from one re-read because "+
			"the agent lost track of it, and a read whose call named no file is absent from the "+
			"denominator rather than counted as a first read.")
	return r
}
```

A target is comparable **only inside its own sequence**: it is an integer assigned in first-seen
order, never a path and never a digest of one, so "the same file nine times" stays answerable while
"which file" stays permanently unanswerable.

## Where to go next

- The full `Input` and `Result` field tables: [Adding a metric validator](../extending/metric-validator.md).
- Every field of the plugin wire, generated from the types: [the reference](https://assaio.dev/docs/reference#metric-plugin).
- A whole tool assaio does not read yet: [write a parser plugin](../extending/parser-plugin.md), any language.
- Thresholds on top of these: [rule plugins](rule-plugins.md).
