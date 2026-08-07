package pricing

import (
	"os"
	"testing"

	"github.com/assaio/assaio/internal/usage"
)

func loadTestTable(t *testing.T) Table {
	t.Helper()
	f, err := os.Open("testdata/prices.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	tbl, err := LoadReader(f)
	if err != nil {
		t.Fatal(err)
	}
	return tbl
}

func TestCostClaude(t *testing.T) {
	tbl := loadTestTable(t)
	r := &usage.Record{
		Model: "claude-opus-4-5", InputTokens: 100, OutputTokens: 200,
		CacheWriteTokens: 50, CacheReadTokens: 800,
	}
	got, ok := tbl.Cost(r)
	if !ok {
		t.Fatal("expected priced")
	}
	want := 0.0062125 // 100*5e-6 + 200*2.5e-5 + 50*6.25e-6 + 800*5e-7
	if diff := got - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("Cost = %.10f want %.10f", got, want)
	}
}

func TestCostCodexNoCacheWrite(t *testing.T) {
	tbl := loadTestTable(t)
	// Codex: cache-write price is null; non-cached input handled by caller (input already excludes cached here).
	r := &usage.Record{Model: "gpt-5.1", InputTokens: 800, OutputTokens: 300, CacheReadTokens: 200}
	got, ok := tbl.Cost(r)
	if !ok {
		t.Fatal("expected priced")
	}
	want := 800*1.25e-6 + 300*1e-5 + 200*1.25e-7 // 0.001 + 0.003 + 0.000025
	if diff := got - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("Cost = %.10f want %.10f", got, want)
	}
}

func TestCostUnknownModel(t *testing.T) {
	tbl := loadTestTable(t)
	if _, ok := tbl.Cost(&usage.Record{Model: "nope"}); ok {
		t.Fatal("unknown model must return ok=false")
	}
}

func TestCostTokensMatchesCost(t *testing.T) {
	tbl := loadTestTable(t)
	got, ok := tbl.CostTokens("claude-opus-4-5", Tokens{In: 100, Out: 200, CacheWrite: 50, CacheRead: 800})
	if !ok {
		t.Fatal("expected priced")
	}
	want := 0.0062125 // 100*5e-6 + 200*2.5e-5 + 50*6.25e-6 + 800*5e-7
	if diff := got - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("CostTokens = %.10f want %.10f", got, want)
	}
}

// A cache write is billed by the lifetime it bought: the 1-hour tier costs 1.6x the
// 5-minute one, and 59.7% of the audited corpus's cache-write tokens were 1-hour writes.
func TestCostSplitsACacheWriteByTheLifetimeItBought(t *testing.T) {
	tbl := loadTestTable(t)
	base := 100*5e-6 + 200*2.5e-5 + 800*5e-7
	tests := []struct {
		name string
		tk   Tokens
		want float64
	}{
		{"no tier reported prices exactly as before", Tokens{In: 100, Out: 200, CacheWrite: 50, CacheRead: 800}, base + 50*6.25e-6},
		{"all five-minute", Tokens{In: 100, Out: 200, CacheWrite: 50, CacheRead: 800, CacheWrite1h: 0}, base + 50*6.25e-6},
		{"all one-hour", Tokens{In: 100, Out: 200, CacheWrite: 50, CacheRead: 800, CacheWrite1h: 50}, base + 50*1e-5},
		{"split across both tiers", Tokens{In: 100, Out: 200, CacheWrite: 50, CacheRead: 800, CacheWrite1h: 20}, base + 30*6.25e-6 + 20*1e-5},
		{"portion larger than its write is clamped", Tokens{In: 100, Out: 200, CacheWrite: 50, CacheRead: 800, CacheWrite1h: 999}, base + 50*1e-5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tbl.CostTokens("claude-opus-4-5", tt.tk)
			if !ok {
				t.Fatal("expected priced")
			}
			if diff := got - tt.want; diff > 1e-12 || diff < -1e-12 {
				t.Fatalf("CostTokens = %.12f want %.12f", got, tt.want)
			}
		})
	}
}

// A model whose entry publishes one cache-write rate bills both tiers at it. Falling back
// to zero would price a 1-hour write as free.
func TestModelWithoutALongTierBillsBothAtOneRate(t *testing.T) {
	tbl := loadTestTable(t)
	if p := tbl["claude-haiku-4-5"]; p.CacheWrite1h != p.CacheWrite {
		t.Fatalf("CacheWrite1h = %v, want the single published rate %v", p.CacheWrite1h, p.CacheWrite)
	}
	long, ok := tbl.CostTokens("claude-haiku-4-5", Tokens{CacheWrite: 100, CacheWrite1h: 100})
	if !ok {
		t.Fatal("expected priced")
	}
	if want := 100 * 1.25e-6; long != want {
		t.Fatalf("cost = %.12f, want %.12f", long, want)
	}
}
