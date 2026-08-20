package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// TestModelRightSizingNamesSmallPremiumTurns checks that a model whose premium turns are
// mostly tiny-output has that share reported -- and, since B177, reported without a verdict:
// a short answer can still need the strong model. It reads the per-turn TurnSizing counts
// (the daily Usage aggregate can't answer this), so the cheaper model's turns must be ignored.
func TestModelRightSizingNamesSmallPremiumTurns(t *testing.T) {
	in := BuildInput(nil, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	in.TurnSizing = []store.ModelTurns{
		{Model: "claude-opus-4-5", Turns: 25, SmallTurns: 20},    // premium: 80% tiny
		{Model: "claude-sonnet-4-5", Turns: 100, SmallTurns: 90}, // cheaper: ignored
	}
	got := mustGet(t, rightSizeName).Analyze(in)

	if got.Read != reportedRead {
		t.Fatalf("Read = %+v, want the share reported rather than graded", got.Read)
	}
	if !strings.Contains(figureValues(got.Figures), "80%") { // 20 of 25 premium small-output
		t.Fatalf("Figures = %q, want 80%% small-output premium", figureValues(got.Figures))
	}
	if !strings.Contains(got.Takeaway, "20 turns worth opening") {
		t.Fatalf("Takeaway = %q, want the candidate turns named", got.Takeaway)
	}
}
