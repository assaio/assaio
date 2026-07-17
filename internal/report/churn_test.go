package report

import (
	"testing"

	"github.com/assaio/assaio/internal/store"
)

func TestBuildChurnAggregatesAcrossRows(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "2026-07-01", Project: "web", LinesAdded: 100, ReworkLines: 20},
		{Day: "2026-07-02", Project: "web", LinesAdded: 50, ReworkLines: 5},
	}
	got := BuildChurn(rows)
	if got.LinesAdded != 150 || got.ReworkLines != 25 {
		t.Fatalf("got = %+v, want LinesAdded=150 ReworkLines=25", got)
	}
	wantRate := 25.0 / 150.0
	if diff := got.ReworkRate - wantRate; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("ReworkRate = %v, want %v", got.ReworkRate, wantRate)
	}
}

func TestBuildChurnRateZeroWhenNoLinesAdded(t *testing.T) {
	got := BuildChurn([]store.UsageRow{{Day: "d", ReworkLines: 0, LinesAdded: 0}})
	if got.ReworkRate != 0 {
		t.Fatalf("ReworkRate = %v, want 0 (never a divide-by-zero)", got.ReworkRate)
	}
}

func TestBuildChurnEmptyInputIsZeroValue(t *testing.T) {
	got := BuildChurn(nil)
	if got != (ChurnStat{}) {
		t.Fatalf("got = %+v, want zero value", got)
	}
}
