package dashboard

import (
	"testing"

	"github.com/assaio/assaio/internal/report"
)

func TestFormatCompactUSD(t *testing.T) {
	tests := []struct {
		v    float64
		want string
	}{
		{0, "$0"},
		{17, "$17"},
		{999, "$999"},
		{1000, "$1.0K"},
		{31500, "$31.5K"},
		{1_500_000, "$1.5M"},
	}
	for _, tt := range tests {
		if got := formatCompactUSD(tt.v); got != tt.want {
			t.Fatalf("formatCompactUSD(%v) = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestCostBasisDashesWhenCostUnknown(t *testing.T) {
	got := costBasis(report.Inventory{}, "last 30 days")
	want := "— / last 30 days · — per active day"
	if got != want {
		t.Fatalf("costBasis(zero Inventory) = %q, want %q", got, want)
	}
}

func TestCostBasisRendersCompactTotals(t *testing.T) {
	cost := 31500.0
	inv := report.Inventory{TotalCost: &cost, Days: 30}
	got := costBasis(inv, "last 30 days")
	want := "$31.5K / last 30 days · $1.1K per active day"
	if got != want {
		t.Fatalf("costBasis = %q, want %q", got, want)
	}
}

// TestCostBasisPerDayDashedWhenNoActiveDays covers cost known but Days == 0: never a
// divide-by-zero, an honest dash for the per-active-day half only.
func TestCostBasisPerDayDashedWhenNoActiveDays(t *testing.T) {
	cost := 100.0
	got := costBasis(report.Inventory{TotalCost: &cost}, "last 7 days")
	want := "$100 / last 7 days · — per active day"
	if got != want {
		t.Fatalf("costBasis = %q, want %q", got, want)
	}
}
