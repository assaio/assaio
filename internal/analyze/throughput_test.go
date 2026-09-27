package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// TestThroughputTrendDirectionIsAShareNotAPercentage is a unit regression: changePct is the
// fraction (recent-prior)/prior despite its name, so dividing it again renders a collapse to
// zero as "down 1%".
func TestThroughputTrendDirectionIsAShareNotAPercentage(t *testing.T) {
	for _, tc := range []struct {
		change float64
		want   string
	}{
		{-1, "down 100%"},
		{0.35, "up 35%"},
		{0, "unchanged"},
	} {
		if got := trendDirection(tc.change); got != tc.want {
			t.Errorf("trendDirection(%v) = %q, want %q", tc.change, got, tc.want)
		}
	}
}

// TestThroughputReadsACollapseAsADirection guards the floor's meaning: it exists to stop a
// 1-to-2-line swing reading as +100%, not to hide output falling from thousands of lines to
// none, which is exactly the direction worth reading. It drives Analyze rather than computing
// the floor in the test body -- the first cut of this test did the latter, which made it pass
// against the very bug it names.
func TestThroughputReadsACollapseAsADirection(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	rows := []store.UsageRow{
		// The source was already running before the earlier week, so that week is whole.
		{Day: "2026-08-01", Tool: "claude-code", Model: "m", Project: "p", In: 100},
		{Day: "2026-08-10", Tool: "claude-code", Model: "m", Project: "p", In: 100, LinesAdded: 201347},
	}
	in := BuildInput(rows, nil, testPrices(), now, 7*24*time.Hour, Delegation{})
	got := mustGet(t, throughputName).Analyze(in)

	if !strings.Contains(got.Takeaway, "down 100%") {
		t.Fatalf("Takeaway = %q, want the collapse to zero stated as a direction", got.Takeaway)
	}
}

// TestThroughputSaysWhyThereIsNoDirection separates a source with no earlier week from a trivial
// comparison: "too few lines" over a window holding plenty of them sends a reader after the
// wrong thing.
func TestThroughputSaysWhyThereIsNoDirection(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	rows := []store.UsageRow{
		{Day: "2026-08-19", Tool: "claude-code", Model: "m", Project: "p", In: 100, LinesAdded: 50000},
	}
	in := BuildInput(rows, nil, testPrices(), now, 7*24*time.Hour, Delegation{})
	got := mustGet(t, throughputName).Analyze(in)

	if !strings.Contains(got.Takeaway, "has a row by") {
		t.Fatalf("Takeaway = %q, want the missing prior span named rather than a line shortage", got.Takeaway)
	}
}

// TestThroughputDividesOnlyDaysALineCouldBeRecordedOn: a day on which only a source that records
// no lines ran is not a day of zero output, so lines per active day leaves it out and says so.
func TestThroughputDividesOnlyDaysALineCouldBeRecordedOn(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	rows := []store.UsageRow{
		{Day: "2026-08-08", Tool: "claude-code", Model: "claude-opus-4-5", In: 100, LinesAdded: 30},
		{Day: "2026-08-09", Tool: "claude-code", Model: "claude-opus-4-5", In: 100, LinesAdded: 10},
		{Day: "2026-08-10", Tool: "agy"},
	}
	res := throughputValidator{}.Analyze(BuildInput(rows, nil, testPrices(), now, 7*24*time.Hour, Delegation{}))
	for _, f := range res.Figures {
		if f.Label != "lines/active-day" {
			continue
		}
		if f.Value != "20.0" || f.Note != "1 of 3 active days had only sources that record no lines" {
			t.Fatalf("lines/active-day = %q (%q), want 20.0 over the two line-recording days", f.Value, f.Note)
		}
		if res.Confidence.Samples != 2 {
			t.Fatalf("the read rests on %d days, want the 2 a line could be recorded on", res.Confidence.Samples)
		}
		return
	}
	t.Fatal("no lines/active-day figure")
}
