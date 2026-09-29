package dashboard

import (
	"testing"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

// TestCostBasisNeverRendersARealCostAsZero is B131: the per-active-day figure rounded to
// whole dollars, so $12 across 30 days printed "$0 per active day" -- exactly the
// fabricated zero costDisplay's own doc forbids.
func TestCostBasisNeverRendersARealCostAsZero(t *testing.T) {
	cost := 12.0
	got := costBasis(&report.Inventory{TotalCost: &cost, Days: 30, TokenDays: 30}, "last 30 days")
	want := "$12 / last 30 days · $0.40 per active day"
	if got != want {
		t.Fatalf("costBasis = %q, want %q", got, want)
	}
}

func TestCostBasisDashesWhenCostUnknown(t *testing.T) {
	got := costBasis(&report.Inventory{}, "last 30 days")
	want := "— / last 30 days · — per active day"
	if got != want {
		t.Fatalf("costBasis(zero Inventory) = %q, want %q", got, want)
	}
}

func TestCostBasisRendersCompactTotals(t *testing.T) {
	cost := 31500.0
	inv := report.Inventory{TotalCost: &cost, Days: 30, TokenDays: 30}
	got := costBasis(&inv, "last 30 days")
	want := "$31.5K / last 30 days · $1.1K per active day"
	if got != want {
		t.Fatalf("costBasis = %q, want %q", got, want)
	}
}

// TestCostBasisPerDayDashedWhenNoActiveDays covers cost known but Days == 0: never a
// divide-by-zero, an honest dash for the per-active-day half only.
func TestCostBasisPerDayDashedWhenNoActiveDays(t *testing.T) {
	cost := 100.0
	got := costBasis(&report.Inventory{TotalCost: &cost}, "last 7 days")
	want := "$100 / last 7 days · — per active day"
	if got != want {
		t.Fatalf("costBasis = %q, want %q", got, want)
	}
}

// TestCostBasisDividesOnlyDaysThatCouldCarryACost: a day on which only Antigravity CLI ran has
// no token counter and so no cost, and counting it as an active day shrank the per-day figure.
func TestCostBasisDividesOnlyDaysThatCouldCarryACost(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "2026-09-01", Tool: "claude-code", Model: "claude-opus-4-5", In: 1_000_000},
		{Day: "2026-09-02", Tool: "claude-code", Model: "claude-opus-4-5", In: 1_000_000},
		{Day: "2026-09-03", Tool: "agy"},
		{Day: "2026-09-04", Tool: "agy"},
	}
	inv := report.BuildInventory(rows, pricing.Table{"claude-opus-4-5": {Input: 5e-6}})
	if inv.Days != 4 || inv.TokenDays != 2 {
		t.Fatalf("Days/TokenDays = %d/%d, want 4/2", inv.Days, inv.TokenDays)
	}
	want := "$10* / last 7 days · $5* per active day (2 day(s) with no token counter left out)"
	if got := costBasis(&inv, "last 7 days"); got != want {
		t.Fatalf("costBasis = %q, want %q", got, want)
	}
}

// TestCostBasisExplainsNoDenominatorWithoutAFigure: with no known cost there is no per-day
// figure, so there is no denominator to explain either.
func TestCostBasisExplainsNoDenominatorWithoutAFigure(t *testing.T) {
	got := costBasis(&report.Inventory{Days: 3, TokenDays: 1}, "last 7 days")
	if want := "— / last 7 days · — per active day"; got != want {
		t.Fatalf("costBasis = %q, want %q", got, want)
	}
}
