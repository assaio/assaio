package dashboard

import (
	"testing"

	"github.com/assaio/assaio/internal/report"
)

// TestCostBasisNeverRendersARealCostAsZero is B131: the per-active-day figure rounded to
// whole dollars, so $12 across 30 days printed "$0 per active day" -- exactly the
// fabricated zero costDisplay's own doc forbids.
func TestCostBasisNeverRendersARealCostAsZero(t *testing.T) {
	cost := 12.0
	got := costBasis(&report.Inventory{TotalCost: &cost, Days: 30}, "last 30 days")
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
	inv := report.Inventory{TotalCost: &cost, Days: 30}
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
