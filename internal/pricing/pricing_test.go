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
	got, ok := tbl.CostTokens("claude-opus-4-5", 100, 200, 50, 800)
	if !ok {
		t.Fatal("expected priced")
	}
	want := 0.0062125 // 100*5e-6 + 200*2.5e-5 + 50*6.25e-6 + 800*5e-7
	if diff := got - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("CostTokens = %.10f want %.10f", got, want)
	}
}
