# Extensions, written out in full

*Part of [Extending assaio](../extending.md). Contracts: [in-tree
validator](../extending/metric-validator.md) · [metric plugin](../extending/metric-plugin.md).*

These complete, short extensions show one reusable pattern each. The linked guides define their
contracts.

**What the tests check.** `TestRecipeMetricPlugins` **executes** the Python plugins on a fixture
window and checks their results. For the plugin gated on `answers`, removing the gate changes the
published figure from 30.0 to 50.0 and fails the test. assaio's config loader reads the config
block. `TestRecipeValidatorsMatchTheInterface` parses the Go validators and checks their methods
against `Validator`; a Go file in a doc cannot compile without a package. This catches renamed
methods and changed signatures, but not wrong numbers. Honesty rules and review cover those.

## A validator, as small as one gets

A metric reads prepared `Input`, returns one `Result`, and registers from `init()`. The registry
automatically includes it in `analyze`, `analyze --format json`, and the dashboard.

```go #weekday-split
package analyze

func init() { Register(weekdayValidator{}) }

type weekdayValidator struct{}

func (weekdayValidator) Name() string  { return "weekday-split" }
func (weekdayValidator) Title() string { return "Weekday Split" }
func (weekdayValidator) Layer() layer.Layer { return layer.Activity } // a token share is what happened

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

With no data, return `neutral`, an em dash, and a takeaway explaining why. Reporting `0%` without a
denominator gives a false number; check this first when reviewing a new validator.

## A validator that refuses to read a silence as a zero

Before reading a column, keep only rows from sources that can record it. A source without a cache
counter leaves zero in that field; averaging it would falsely say the cache was never written.

```go #answers-gated
package analyze

func init() { Register(editSizeValidator{}) }

type editSizeValidator struct{}

func (editSizeValidator) Name() string     { return "edit-size" }
func (editSizeValidator) Title() string    { return "Edit Size" }
func (editSizeValidator) Describe() string { return "AI lines per edit, over the sources that count edits." }
func (editSizeValidator) Layer() layer.Layer { return layer.Output } // a line count is what was produced

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

The caveat gives the share of the window covered by the figure, so readers can judge whether to act
on it.

## A metric plugin, any language

This executable implements the same metric without a fork or Go. Declare it under `metrics:` in
config. It answers `describe` with the fields it reads and `analyze` with one `Result` over those
fields. `assaio-agent plugins init --kind metric --lang python` prints the skeleton.

```python #plugin-weekday
#!/usr/bin/env python3
"""Weekday token share as an out-of-tree metric plugin."""
import datetime as dt
import json
import sys

HANDSHAKE = {"assaio_metric": 4, "name": "weekday-split"}

if sys.argv[1:2] == ["describe"]:
    print(json.dumps(HANDSHAKE))
    print(json.dumps({"needs": ["usage"],
                      "fields": {"usage": ["day", "in", "out", "cacheRead", "cacheWrite"]}}))
    sys.exit(0)

inp = json.load(sys.stdin)
weekday = total = 0
for row in inp.get("usage", []):
    tokens = row["in"] + row["out"] + row["cacheRead"] + row["cacheWrite"]
    total += tokens
    if dt.date.fromisoformat(row["day"]).weekday() < 5:
        weekday += tokens

result = {
    "title": "Weekday Split",
    "layer": "activity",
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

print(json.dumps(HANDSHAKE))
print(json.dumps(result))
```

Core stamps `name` as `plugin:<name>` on arrival, so a plugin cannot shadow a built-in validator. A
result may set it, but core overwrites it.

## A metric plugin that checks what the window can answer

The wire includes `answers`, mapping each tool in the window to signal ids it can produce. A plugin
that ignores it risks the same error the in-tree gate prevents. Unlike a validator, it cannot read
the capability matrix, so the envelope supplies it.

```python #plugin-answers
#!/usr/bin/env python3
"""AI lines per edit, counting only the sources that record edits."""
import json
import sys

NEEDED = ("ai.edits.count", "ai.lines.added")
HANDSHAKE = {"assaio_metric": 4, "name": "edit-size"}

if sys.argv[1:2] == ["describe"]:
    print(json.dumps(HANDSHAKE))
    print(json.dumps({"needs": ["usage"],
                      "fields": {"usage": ["tool", "in", "out", "cacheRead", "cacheWrite",
                                           "linesAdded", "edits"]}}))
    sys.exit(0)

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
    "layer": "output",
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

print(json.dumps(HANDSHAKE))
print(json.dumps(result))
```

Wire both plugins the same way:

```yaml #metric-config
metrics:
  - name: weekday-split
    command: ./scripts/weekday-split.py
    timeout: 30s
```

Check both before trusting them:

```sh #verify-metric
assaio-agent metrics verify weekday-split --since 30d
```

`metrics verify` runs the plugin on your real window and prints contract violations and the rendered
result. It stores nothing.

## A detector: reading the sequence, not the total

The step timeline (ADR 0012) is the only input that is not aggregated: it records session actions in
order. A detector reads one *scope*, never the whole set, because the populations differ. On the
audited store, one-shot SDK calls make up 89% of sequences but only 5.7% of steps, so a rate across
scopes describes neither. `TraceReader` declares the scope. The caveat must also say what the
pattern cannot distinguish: a hard bug and a loop look identical on a timeline.

This counts when a sequence reads the same file again.

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
func (reReadsValidator) Layer() layer.Layer { return layer.Activity } // repeated reads are what happened

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

A target is comparable **only inside its own sequence**. Its integer is assigned in first-seen
order; it is neither a path nor a path digest. You can count nine reads of the same file but cannot
identify the file.

## Where to go next

- Full `Input` and `Result` field tables: [Adding a metric
  validator](../extending/metric-validator.md).
- Every plugin wire field, generated from the types: [the
  reference](https://assaio.dev/docs/reference#metric-plugin).
- To add a tool assaio does not yet read: [write a parser plugin](../extending/parser-plugin.md) in
  any language.
- To set thresholds on these: [rule plugins](rule-plugins.md).
