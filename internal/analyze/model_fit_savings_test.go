package analyze

import (
	"testing"

	"github.com/assaio/assaio/internal/store"
)

// TestComputeModelSavingsUpperBound checks the counterfactual: premium bundle (1000 in,
// 2000 out) costs $0.165 on opus and $0.033 repriced on sonnet, a $0.132 window saving that
// projects to ~$3.96/mo at one active day.
func TestComputeModelSavingsUpperBound(t *testing.T) {
	cost := 0.165
	premium := ModelStat{Model: "claude-opus-4-5", Tier: tierPremium, Input: 1000, Output: 2000, Cost: &cost, Priced: true}
	cheaper := ModelStat{Model: "claude-sonnet-4-5", Tier: tierCheaper, Input: 10, Output: 20, Priced: true}

	s, ok := computeModelSavings([]ModelStat{premium, cheaper}, testPrices(), 1)
	if !ok {
		t.Fatal("want a savings estimate when premium is priced and a cheaper model is in use")
	}
	if s.TargetModel != "claude-sonnet-4-5" {
		t.Fatalf("TargetModel = %q, want claude-sonnet-4-5 (cheapest cheaper model in use)", s.TargetModel)
	}
	if s.MonthlyUpper < 3.9 || s.MonthlyUpper > 4.0 {
		t.Fatalf("MonthlyUpper = %v, want ~3.96 ($0.132 window saving x 30 active-day projection)", s.MonthlyUpper)
	}
}

func TestComputeModelSavingsNoCheaperModel(t *testing.T) {
	cost := 0.165
	premium := ModelStat{Model: "claude-opus-4-5", Tier: tierPremium, Input: 1000, Output: 2000, Cost: &cost, Priced: true}
	if _, ok := computeModelSavings([]ModelStat{premium}, testPrices(), 1); ok {
		t.Fatal("no cheaper model in use -> no honest target to reprice onto -> no estimate")
	}
}

func TestComputeModelSavingsUnpricedPremium(t *testing.T) {
	premium := ModelStat{Model: "mystery", Tier: tierPremium, Input: 1000, Output: 2000, Priced: false}
	cheaper := ModelStat{Model: "claude-sonnet-4-5", Tier: tierCheaper, Priced: true}
	if _, ok := computeModelSavings([]ModelStat{premium, cheaper}, testPrices(), 1); ok {
		t.Fatal("premium cost unknown -> cannot compute a real saving")
	}
}

func TestDistinctDays(t *testing.T) {
	rows := []store.UsageRow{{Day: "2026-07-10"}, {Day: "2026-07-10"}, {Day: "2026-07-11"}}
	if got := distinctDays(rows); got != 2 {
		t.Fatalf("distinctDays = %d, want 2", got)
	}
}
