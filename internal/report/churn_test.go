package report

import (
	"testing"

	"github.com/assaio/assaio/internal/store"
)

func TestBuildChurnAggregatesAcrossRows(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "2026-07-01", Tool: "claude-code", Project: "web", LinesAdded: 100, ReworkLines: 20},
		{Day: "2026-07-02", Tool: "claude-code", Project: "web", LinesAdded: 50, ReworkLines: 5},
	}
	got := BuildChurn(rows)
	if got.LinesAdded != 150 || got.ReworkLines != 25 {
		t.Fatalf("got = %+v, want LinesAdded=150 ReworkLines=25", got)
	}
	if got.Rows != 2 {
		t.Fatalf("Rows = %d, want both rows counted as the basis", got.Rows)
	}
	wantRate := 25.0 / 150.0
	if diff := got.ReworkRate - wantRate; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("ReworkRate = %v, want %v", got.ReworkRate, wantRate)
	}
}

func TestBuildChurnRateZeroWhenNoLinesAdded(t *testing.T) {
	got := BuildChurn([]store.UsageRow{{Day: "d", Tool: "claude-code", ReworkLines: 0, LinesAdded: 0}})
	if got.ReworkRate != 0 {
		t.Fatalf("ReworkRate = %v, want 0 (never a divide-by-zero)", got.ReworkRate)
	}
}

// A source recording changed lines but never an undone one would otherwise contribute its
// whole output to the denominator against a structural zero, reporting less churn than the
// sources that actually measured it found.
func TestBuildChurnExcludesSourcesThatRecordNoUndoneLine(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "2026-07-01", Tool: "claude-code", LinesAdded: 100, ReworkLines: 25},
		{Day: "2026-07-01", Tool: "copilot-cli", LinesAdded: 900},
	}
	got := BuildChurn(rows)
	if got.LinesAdded != 100 || got.Rows != 1 {
		t.Fatalf("got = %+v, want only the rework-capable row's 100 lines", got)
	}
	if got.ReworkRate != 0.25 {
		t.Fatalf("ReworkRate = %v, want 0.25 rather than the 0.025 the silent source would produce", got.ReworkRate)
	}
}

// No capable source at all is a window that cannot answer the question, and Rows says so
// rather than the rate reading as a clean 0%.
func TestBuildChurnReportsNoBasisWhenNoSourceRecordsRework(t *testing.T) {
	got := BuildChurn([]store.UsageRow{{Day: "d", Tool: "copilot-cli", LinesAdded: 400}})
	if got.Rows != 0 || got.LinesAdded != 0 {
		t.Fatalf("got = %+v, want an empty basis", got)
	}
}

func TestBuildChurnEmptyInputIsZeroValue(t *testing.T) {
	got := BuildChurn(nil)
	if got != (ChurnStat{}) {
		t.Fatalf("got = %+v, want zero value", got)
	}
}
