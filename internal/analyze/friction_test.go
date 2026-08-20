package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// frictionInput seeds one usage row whose failure state was recorded: the tool-call purpose
// split is populated, which is what marks a row as coming from a build that also captures
// errors. calls are split arbitrarily across reads and writes so the taxonomy sums to them.
func frictionInput(calls, errs, rejected int64) Input {
	row := store.UsageRow{
		Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web",
		In: 1000, Out: 500, ToolCalls: calls, ToolErrors: errs, Rejected: rejected,
		ToolReads: calls / 2, ToolWrites: calls - calls/2,
	}
	return BuildInput([]store.UsageRow{row}, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
}

// The error rate divides by the calls whose failure state was recorded, which can be a small
// part of the window's calls. The figure already says so; the envelope has to as well, or a
// rate read off a tenth of the calls travels as a fully covered verdict.
func TestFrictionCarriesItsFailureCaptureCoverage(t *testing.T) {
	rows := []store.UsageRow{
		{
			Day: "2026-07-08", Tool: "claude-code", Model: "claude-sonnet-4-5", Granularity: "turn", Project: "web",
			In: 1000, Out: 500, ToolCalls: 40, ToolErrors: 2, ToolReads: 20, ToolWrites: 20,
		},
		{
			Day: "2026-07-09", Tool: "codex", Model: "claude-sonnet-4-5", Granularity: "turn", Project: "web",
			In: 1000, Out: 500, ToolCalls: 360,
		},
		{
			Day: "2026-07-10", Tool: "codex", Model: "claude-sonnet-4-5", Granularity: "turn", Project: "web",
			In: 1000, Out: 500, ToolCalls: 360,
		},
	}
	in := BuildInput(rows, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := Evaluate(mustGet(t, frictionName), &in)

	if got.Confidence.signalShare() >= 1 {
		t.Fatalf("Signal = %v, want the failure-capture share of the window's calls", got.Confidence.signalShare())
	}
	if got.Confidence.Label == ConfidenceHigh {
		t.Errorf("Label = %q for a rate read off %.0f%% of the calls, want it held down",
			got.Confidence.Label, got.Confidence.signalShare()*100)
	}
}

// TestFrictionReportsTheRateWithoutGradingIt is B177's regression: an 8% error rate and a 40%
// one both get the same read, because nothing published says which of them is a problem. What
// separates them is the figure, which is what a reader acts on.
func TestFrictionReportsTheRateWithoutGradingIt(t *testing.T) {
	for _, tc := range []struct {
		name string
		errs int64
		rate string
	}{
		{"ordinary probing", 8, "8.0%"},
		{"most calls failing", 40, "40.0%"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := mustGet(t, frictionName).Analyze(frictionInput(100, tc.errs, 2))

			if got.Read != reportedRead {
				t.Fatalf("Read = %+v, want the descriptive read at a %s error rate", got.Read, tc.rate)
			}
			if got.Purity != neutralPurity {
				t.Fatalf("Purity = %v, want the neutral gauge: a gauge is a verdict rendering", got.Purity)
			}
			if !strings.Contains(figureValues(got.Figures), tc.rate) {
				t.Fatalf("Figures = %q, want the %s error rate", figureValues(got.Figures), tc.rate)
			}
			if !strings.Contains(strings.Join(got.Caveats, " "), "No verdict on purpose") {
				t.Fatalf("Caveats = %q, want the missing-line caveat", got.Caveats)
			}
		})
	}
}

// TestFrictionSeparatesErrorsFromRejections asserts a human declining a call is reported
// apart from a call that failed: they are different facts with different fixes.
func TestFrictionSeparatesErrorsFromRejections(t *testing.T) {
	got := mustGet(t, frictionName).Analyze(frictionInput(100, 5, 25))

	values := figureValues(got.Figures)
	if !strings.Contains(values, "5.0%") || !strings.Contains(values, "25.0%") {
		t.Fatalf("Figures = %q, want the error and rejection rates reported separately", values)
	}
}

// TestFrictionNeverPassesOnUnrecordedFailures is the fabricated-zero regression: a window
// whose rows predate failure capture has tool calls but no way to report a failure. Reading
// that as "0.0% errors, SMOOTH" is a green verdict earned from data that never existed.
func TestFrictionNeverPassesOnUnrecordedFailures(t *testing.T) {
	row := store.UsageRow{
		Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web",
		In: 1000, Out: 500, ToolCalls: 100, // no taxonomy: an un-restated pre-0002 row
	}
	in := BuildInput([]store.UsageRow{row}, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := mustGet(t, frictionName).Analyze(in)

	if got.Read != noDataRead {
		t.Fatalf("Read = %+v, want the neutral read when no call could report a failure", got.Read)
	}
	// The rejection rate legitimately reads 0.0%: rejections were captured on these rows.
	// The error rate is the one that has no denominator, and must say so.
	for _, f := range got.Figures {
		if f.Label == "error rate" && f.Value != "—" {
			t.Fatalf("error rate = %q, want a dash over calls whose failure state was never recorded", f.Value)
		}
	}
	// The row made 100 calls, so the missing failure state is history this build can restate
	// -- naming the source as the thing that records none of it would blame it for the wrong fact.
	if !strings.Contains(got.Takeaway, "backfill") {
		t.Fatalf("Takeaway = %q, want the stale-capture explanation", got.Takeaway)
	}
}

// TestFrictionRefusesIncoherentCounts covers a row whose recorded failures outnumber the
// calls they were counted over -- reachable from a corrupt or plugin-pushed record. No
// percentage over the two means anything, so the figures must read "—" rather than 150% and
// a negative share.
func TestFrictionRefusesIncoherentCounts(t *testing.T) {
	row := store.UsageRow{
		Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web",
		In: 1000, ToolCalls: 40, ToolWrites: 40, ToolErrors: 60,
	}
	in := BuildInput([]store.UsageRow{row}, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := mustGet(t, frictionName).Analyze(in)

	values := figureValues(got.Figures)
	if strings.Contains(values, "150") || strings.Contains(values, "-") && strings.Contains(values, "%") && strings.Contains(values, "-5") {
		t.Fatalf("Figures = %q must not render an impossible percentage", values)
	}
	if !strings.Contains(got.Takeaway, "outnumber the calls") {
		t.Fatalf("Takeaway = %q, want the incoherent-counts explanation", got.Takeaway)
	}
}

// TestFrictionThinSampleIsUndefined asserts too few calls yields the neutral read.
func TestFrictionThinSampleIsUndefined(t *testing.T) {
	got := mustGet(t, frictionName).Analyze(frictionInput(6, 4, 0))

	if got.Read != noDataRead {
		t.Fatalf("Read = %+v, want the neutral read below the %d-call floor", got.Read, frictionMinCalls)
	}
}

// TestFrictionAlwaysStatesCoverage asserts the coverage caveat is always present, so a zero
// error count is never mistaken for "nothing failed" when it means "nothing recorded".
func TestFrictionAlwaysStatesCoverage(t *testing.T) {
	got := mustGet(t, frictionName).Analyze(frictionInput(100, 0, 0))

	if len(got.Caveats) == 0 || !strings.Contains(got.Caveats[0], "record whether they failed") {
		t.Fatalf("Caveats = %q, want the failure-capture coverage disclosure", got.Caveats)
	}
	if !strings.Contains(got.Caveats[0], "100%") {
		t.Fatalf("Caveats = %q, want the coverage share stated", got.Caveats)
	}
}

// TestFrictionEmptyInputSafe asserts the zero-value Input renders the no-data block.
func TestFrictionEmptyInputSafe(t *testing.T) {
	got := mustGet(t, frictionName).Analyze(Input{})
	if got.Read != noDataRead || got.Takeaway != "No usage in this window." {
		t.Fatalf("empty Input = %+v, want the no-data read and takeaway", got)
	}
}

// The `backfill` cure is the one insufficient reason that recommends an action, so it must
// never reach a window a backfill cannot change. Codex records tool calls and deliberately
// records no failure for them: its calls are not history missing a field, they are work no
// re-read will ever cover. Asking the depth matrix whether *some* tool in the window records
// failures answered about the wrong source and sent this window after a pointless re-import.
func TestFrictionNeverBlamesStaleHistoryForASourceThatRecordsNoFailure(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "2026-07-10", Tool: "codex", Model: "claude-sonnet-4-5", In: 1000, ToolCalls: 360, ToolReads: 200, ToolWrites: 160},
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", In: 500}, // chat turns, no tool call
	}
	in := BuildInput(rows, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := Evaluate(mustGet(t, frictionName), &in)

	if summary := ConfidenceSummary(&got.Confidence); strings.Contains(summary, "backfill") {
		t.Fatalf("confidence = %q, want no backfill cure for calls from a source that records no failure", summary)
	}
	if strings.Contains(got.Takeaway, "backfill") {
		t.Fatalf("Takeaway = %q, want no backfill cure a re-read could not act on", got.Takeaway)
	}
}
