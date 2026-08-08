package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

func TestEvaluateBudget(t *testing.T) {
	tests := []struct {
		name       string
		totals     checkTotals
		b          budget
		wantBreach int
	}{
		{"no budget", checkTotals{Tokens: 100, Cost: 5}, budget{}, 0},
		{"tokens within", checkTotals{Tokens: 100}, budget{MaxTokens: 200}, 0},
		{"tokens at limit ok", checkTotals{Tokens: 200}, budget{MaxTokens: 200}, 0},
		{"tokens over", checkTotals{Tokens: 300}, budget{MaxTokens: 200}, 1},
		{"cost over", checkTotals{Cost: 12.5}, budget{MaxCost: 10}, 1},
		{"both over", checkTotals{Tokens: 300, Cost: 12.5}, budget{MaxTokens: 200, MaxCost: 10}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := len(evaluateBudget(tt.totals, tt.b)); got != tt.wantBreach {
				t.Fatalf("evaluateBudget breaches = %d, want %d", got, tt.wantBreach)
			}
		})
	}
}

func TestSumCheckTotals(t *testing.T) {
	c := 1.25
	rows := []report.Row{
		{In: 100, Out: 200, CacheRead: 10, CacheWrite: 5, Reasoning: 1, Cost: &c, Priced: true},
		{In: 50, Out: 60, HasUnpriced: true},
	}
	got := sumCheckTotals(rows)
	if got.Tokens != 425 {
		t.Fatalf("Tokens = %d, want 425 (input, output, cache read and cache write summed)", got.Tokens)
	}
	if got.Cost != 1.25 {
		t.Fatalf("Cost = %v, want 1.25 (only priced rows)", got.Cost)
	}
	if !got.HasUnpriced {
		t.Fatal("HasUnpriced = false, want true")
	}
}

// TestSumCheckTotalsExcludesReasoning locks the gate to the same token definition the
// reports use. Reasoning tokens are already inside output (usage.Record), so re-adding them
// would fail a CI build on a window report and effectiveness both show under budget.
func TestSumCheckTotalsExcludesReasoning(t *testing.T) {
	usageRows := []store.UsageRow{{
		Day: "2026-07-20", Tool: "codex", Model: "claude-sonnet-4-5", Project: "web",
		In: 1000, Out: 2000, CacheRead: 300, CacheWrite: 100, Reasoning: 1500,
	}}
	table, err := pricing.Load()
	if err != nil {
		t.Fatal(err)
	}
	eff, err := report.BuildEffectiveness(usageRows, table, "project")
	if err != nil {
		t.Fatal(err)
	}
	var reported int64
	for i := range eff {
		reported += eff[i].TokensTotal
	}

	got := sumCheckTotals(report.Build(usageRows, table))
	if got.Tokens != reported {
		t.Fatalf("check tokens = %d, effectiveness tokens = %d -- the gate must count the window the reports show", got.Tokens, reported)
	}
	if got.Tokens != 3400 {
		t.Fatalf("Tokens = %d, want 3400 (in+out+cache read+cache write; reasoning is a subset of out)", got.Tokens)
	}
}

func TestEffectiveBasisLine(t *testing.T) {
	if _, ok := effectiveBasisLine(config.Pricing{}, 1000); ok {
		t.Fatal("unconfigured pricing must not render a basis line")
	}
	line, ok := effectiveBasisLine(config.Pricing{Mode: "subscription", EffectivePerToken: 1e-6}, 1_000_000)
	if !ok || !strings.Contains(line, "effective ~$1.00") {
		t.Fatalf("effective-rate line = %q, ok=%v", line, ok)
	}
	line, ok = effectiveBasisLine(config.Pricing{Mode: "subscription", MonthlySubscriptionCost: 200}, 0)
	if !ok || !strings.Contains(line, "$200.00/mo") {
		t.Fatalf("monthly line = %q, ok=%v", line, ok)
	}
}

func TestCheckExitsNonZeroOnTokenBreach(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	db := filepath.Join(t.TempDir(), "u.db")
	seedStoreAt(t, db, []usage.Record{
		{Tool: "claude-code", SessionID: "s1", Timestamp: time.Now().UTC(), Model: "claude-opus-4-5", InputTokens: 1000, OutputTokens: 2000, DedupeKey: "1"},
	})
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"check", "--db", db, "--max-tokens", "1000"})
	if err := root.Execute(); err == nil {
		t.Fatal("check must exit non-zero when the token budget is exceeded")
	}
	if !strings.Contains(out.String(), "EXCEEDED") {
		t.Fatalf("check output must mark the breach EXCEEDED: %q", out.String())
	}
}

func TestCheckExitsZeroWithinBudget(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	db := filepath.Join(t.TempDir(), "u.db")
	seedStoreAt(t, db, []usage.Record{
		{Tool: "claude-code", SessionID: "s1", Timestamp: time.Now().UTC(), Model: "claude-opus-4-5", InputTokens: 1000, OutputTokens: 2000, DedupeKey: "1"},
	})
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"check", "--db", db, "--max-tokens", "100000"})
	if err := root.Execute(); err != nil {
		t.Fatalf("check must exit zero within budget: %v", err)
	}
	if !strings.Contains(out.String(), "OK") {
		t.Fatalf("check output must mark the axis OK: %q", out.String())
	}
}

// TestCheckCostGateFailsOnUnpricedUsage: a --max-cost gate that compares a budget against a
// cost missing the window's unpriced models reports OK on a window it cannot price. On the
// maintainer's own store that is 45% of the tokens, because the newest model in use has no
// row in the vendored price table -- so the gate would pass a budget exceeded ~2x.
func TestCheckCostGateFailsOnUnpricedUsage(t *testing.T) {
	totals := checkTotals{Tokens: 1000, Cost: 10, HasUnpriced: true, UnpricedTokens: 450}
	breaches := evaluateBudget(totals, budget{MaxCost: 100})
	if len(breaches) != 1 {
		t.Fatalf("breaches = %v, want one: a cost budget cannot be evaluated over unpriced usage", breaches)
	}
	if !strings.Contains(breaches[0], "450") {
		t.Fatalf("breach %q must say how much of the window is unpriced", breaches[0])
	}
}

// TestCheckCostGateStillPassesWhenEverythingIsPriced keeps the gate usable: a fully priced
// window under budget is not a failure.
func TestCheckCostGateStillPassesWhenEverythingIsPriced(t *testing.T) {
	if b := evaluateBudget(checkTotals{Tokens: 1000, Cost: 10}, budget{MaxCost: 100}); len(b) != 0 {
		t.Fatalf("breaches = %v, want none", b)
	}
}

// TestCheckTokenGateIgnoresPricing: tokens are physical, so an unpriced model says nothing
// about a token budget.
func TestCheckTokenGateIgnoresPricing(t *testing.T) {
	totals := checkTotals{Tokens: 10, HasUnpriced: true, UnpricedTokens: 10}
	if b := evaluateBudget(totals, budget{MaxTokens: 100}); len(b) != 0 {
		t.Fatalf("breaches = %v, want none: a token budget does not depend on a price", b)
	}
}

// TestCheckCostGateIgnoresAZeroTokenUnpricedModel is what a real store caught: Claude's
// locally-generated "<synthetic>" turns are an unpriced row with no tokens, so gating on the
// HasUnpriced flag failed every run on a window that was in fact fully priced.
func TestCheckCostGateIgnoresAZeroTokenUnpricedModel(t *testing.T) {
	totals := checkTotals{Tokens: 1000, Cost: 10, HasUnpriced: true, UnpricedTokens: 0}
	if b := evaluateBudget(totals, budget{MaxCost: 100}); len(b) != 0 {
		t.Fatalf("breaches = %v, want none: an unpriced model with no tokens hides no spend", b)
	}
	if v := costVerdict(totals, budget{MaxCost: 100}); v != "OK" {
		t.Fatalf("costVerdict = %q, want OK", v)
	}
}
