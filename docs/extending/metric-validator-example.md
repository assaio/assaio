# Worked example: Weekend Usage

*Part of [Extending assaio](../extending.md). The contract this follows: [Adding a metric validator](metric-validator.md).*

A realistic company-specific metric: what share of AI token usage falls on a Saturday or
Sunday — an out-of-hours/DevEx signal a security or engineering-management team might
want that has no reason to be a built-in. (A per-**file** metric like "share of edits
touching test files" is *not* possible from stored data today — file paths are read
transiently during ingest and never persisted, only aggregate counts are, per [Parsers
stay hermetic](data-source.md#parsers-stay-hermetic--project-resolution-is-ingests-job). This example
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
		{Label: "weekend token share", Value: humanize.PercentOrDash(weekendTokens, total, 1)},
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
