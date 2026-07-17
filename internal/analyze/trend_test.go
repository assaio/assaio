package analyze

import (
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

var trendTestNow = time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)

func trendRows() []store.UsageRow {
	return []store.UsageRow{
		{Day: "2026-07-10", LinesAdded: 200}, // recent (within last 7d)
		{Day: "2026-07-08", LinesAdded: 50},  // recent
		{Day: "2026-07-01", LinesAdded: 100}, // prior (7-14d ago)
	}
}

// TestWeekOverWeekMatchesRecentOverPriorMinusOne verifies the fix's math requirement:
// changePct must equal recent/prior - 1, not some other formula.
func TestWeekOverWeekMatchesRecentOverPriorMinusOne(t *testing.T) {
	recent, prior, changePct, ok := weekOverWeek(trendRows(), trendTestNow, 7*24*time.Hour)
	if !ok {
		t.Fatal("weekOverWeek ok = false, want true")
	}
	if recent != 250 || prior != 100 {
		t.Fatalf("recent=%d prior=%d, want 250/100", recent, prior)
	}
	want := float64(recent)/float64(prior) - 1
	if changePct != want {
		t.Fatalf("changePct = %v, want recent/prior-1 = %v", changePct, want)
	}
	if want != 1.5 {
		t.Fatalf("sanity: 250/100-1 should be 1.5, got %v", want)
	}
}

func TestWeekOverWeekZeroPriorIsUndefined(t *testing.T) {
	rows := []store.UsageRow{{Day: "2026-07-10", LinesAdded: 200}}
	_, prior, _, ok := weekOverWeek(rows, trendTestNow, 7*24*time.Hour)
	if ok {
		t.Fatal("ok = true with a zero prior base, want false")
	}
	if prior != 0 {
		t.Fatalf("prior = %d, want 0", prior)
	}
}

func TestTrendLabelFormatsSignAndDash(t *testing.T) {
	cases := []struct {
		pct  float64
		ok   bool
		want string
	}{
		{1.5, true, "+150%"},
		{-0.5, true, "-50%"},
		{0, true, "+0%"},
		{0, false, "—"},
	}
	for _, c := range cases {
		if got := trendLabel(c.pct, c.ok); got != c.want {
			t.Fatalf("trendLabel(%v, %v) = %q, want %q", c.pct, c.ok, got, c.want)
		}
	}
}

// TestTrendFigureSharedLabel asserts adoption and throughput render the trend under the
// identical label, so the same signal reads the same way in both reports.
func TestTrendFigureSharedLabel(t *testing.T) {
	f := trendFigure(250, 100, 1.5, true)
	if f.Label != "week-over-week AI lines" {
		t.Fatalf("Label = %q, want the shared week-over-week AI lines label", f.Label)
	}
	if f.Value != "+150%" {
		t.Fatalf("Value = %q, want +150%%", f.Value)
	}
	if f.Note != "250 recent vs 100 prior" {
		t.Fatalf("Note = %q, want the recent/prior counts", f.Note)
	}
}

// TestAdoptionAndThroughputShareTrendComputation is fix #5's cross-validator guarantee:
// given the same Usage/Now/Recent, adoption's and throughput's week-over-week Figures
// must agree exactly (same helper, same numbers).
func TestAdoptionAndThroughputShareTrendComputation(t *testing.T) {
	in := BuildInput(trendRows(), nil, testPrices(), trendTestNow, 7*24*time.Hour, Delegation{})
	adoptionResult := mustGet(t, "adoption").Analyze(in)
	throughputResult := mustGet(t, "throughput").Analyze(in)

	adoptionTrend := findFigure(t, adoptionResult.Figures, weekOverWeekLabel)
	throughputTrend := findFigure(t, throughputResult.Figures, weekOverWeekLabel)
	if adoptionTrend != throughputTrend {
		t.Fatalf("adoption trend figure %+v != throughput trend figure %+v", adoptionTrend, throughputTrend)
	}
}

func TestTrendPurityNeutralWhenUnknown(t *testing.T) {
	if got := trendPurity(0, false); got != 0.5 {
		t.Fatalf("trendPurity(_, false) = %v, want 0.5 (neutral)", got)
	}
}

func TestTrendPuritySaturatesAtExtremes(t *testing.T) {
	if got := trendPurity(5, true); got != 1 {
		t.Fatalf("trendPurity(+500%%) = %v, want 1 (clamped)", got)
	}
	if got := trendPurity(-5, true); got != 0 {
		t.Fatalf("trendPurity(-500%%) = %v, want 0 (clamped)", got)
	}
}

func mustGet(t *testing.T, name string) Validator {
	t.Helper()
	v, ok := Get(name)
	if !ok {
		t.Fatalf("validator %q not registered", name)
	}
	return v
}

func findFigure(t *testing.T, figures []Figure, label string) Figure {
	t.Helper()
	for _, f := range figures {
		if f.Label == label {
			return f
		}
	}
	t.Fatalf("no figure labeled %q in %+v", label, figures)
	return Figure{}
}
