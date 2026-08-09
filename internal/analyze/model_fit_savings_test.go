package analyze

import (
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// oneDay is a window one calendar day long, so the monthly projection is exactly 30x the
// window figure and the arithmetic under test is visible.
var oneDay = &Input{
	Usage:       []store.UsageRow{{Day: "2026-03-02"}},
	WindowStart: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
	Now:         time.Date(2026, 3, 2, 18, 0, 0, 0, time.UTC),
}

// TestComputeModelSavingsUpperBound checks the counterfactual: premium bundle (1000 in,
// 2000 out) costs $0.165 on opus and $0.033 repriced on sonnet, a $0.132 window saving that
// projects to ~$3.96/mo over a one-day window.
func TestComputeModelSavingsUpperBound(t *testing.T) {
	cost := 0.165
	premium := ModelStat{Model: "claude-opus-4-5", Tier: tierPremium, Input: 1000, Output: 2000, Cost: &cost, Priced: true}
	cheaper := ModelStat{Model: "claude-sonnet-4-5", Tier: tierCheaper, Input: 10, Output: 20, Priced: true}

	s, ok := computeModelSavings([]ModelStat{premium, cheaper}, testPrices(), oneDay)
	if !ok {
		t.Fatal("want a savings estimate when premium is priced and a cheaper model is in use")
	}
	if s.TargetModel != "claude-sonnet-4-5" {
		t.Fatalf("TargetModel = %q, want claude-sonnet-4-5 (cheapest cheaper model in use)", s.TargetModel)
	}
	if s.MonthlyUpper < 3.9 || s.MonthlyUpper > 4.0 {
		t.Fatalf("MonthlyUpper = %v, want ~3.96 ($0.132 over a one-day span, projected to 30)", s.MonthlyUpper)
	}
}

func TestComputeModelSavingsNoCheaperModel(t *testing.T) {
	cost := 0.165
	premium := ModelStat{Model: "claude-opus-4-5", Tier: tierPremium, Input: 1000, Output: 2000, Cost: &cost, Priced: true}
	if _, ok := computeModelSavings([]ModelStat{premium}, testPrices(), oneDay); ok {
		t.Fatal("no cheaper model in use -> no honest target to reprice onto -> no estimate")
	}
}

func TestComputeModelSavingsUnpricedPremium(t *testing.T) {
	premium := ModelStat{Model: "mystery", Tier: tierPremium, Input: 1000, Output: 2000, Priced: false}
	cheaper := ModelStat{Model: "claude-sonnet-4-5", Tier: tierCheaper, Priced: true}
	if _, ok := computeModelSavings([]ModelStat{premium, cheaper}, testPrices(), oneDay); ok {
		t.Fatal("premium cost unknown -> cannot compute a real saving")
	}
}

func TestDistinctDays(t *testing.T) {
	rows := []store.UsageRow{{Day: "2026-07-10"}, {Day: "2026-07-10"}, {Day: "2026-07-11"}}
	if got := distinctDays(rows); got != 2 {
		t.Fatalf("distinctDays = %d, want 2", got)
	}
}
