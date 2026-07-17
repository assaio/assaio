package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/report"
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
	if got.Tokens != 426 {
		t.Fatalf("Tokens = %d, want 426 (all token types summed)", got.Tokens)
	}
	if got.Cost != 1.25 {
		t.Fatalf("Cost = %v, want 1.25 (only priced rows)", got.Cost)
	}
	if !got.HasUnpriced {
		t.Fatal("HasUnpriced = false, want true")
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
