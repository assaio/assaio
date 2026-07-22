package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// TestModelRightSizingFlagsSmallPremiumTurns checks that a model whose premium turns are
// mostly tiny-output reads as watch, with the small-output share reported. It reads the
// per-turn TurnSizing counts (the daily Usage aggregate can't answer this), so the cheaper
// model's turns must be ignored.
func TestModelRightSizingFlagsSmallPremiumTurns(t *testing.T) {
	in := BuildInput(nil, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	in.TurnSizing = []store.ModelTurns{
		{Model: "claude-opus-4-5", Turns: 25, SmallTurns: 20},    // premium: 80% tiny
		{Model: "claude-sonnet-4-5", Turns: 100, SmallTurns: 90}, // cheaper: ignored
	}
	got := mustGet(t, rightSizeName).Analyze(in)

	if got.Read.Key != "watch" {
		t.Fatalf("Read = %+v, want watch (most premium turns are tiny)", got.Read)
	}
	if !strings.Contains(figureValues(got.Figures), "80%") { // 20 of 25 premium small-output
		t.Fatalf("Figures = %q, want 80%% small-output premium", figureValues(got.Figures))
	}
}
