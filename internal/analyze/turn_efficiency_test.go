package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// TestTurnEfficiencyOneShotRate checks the one-shot rate over code-producing sessions:
// conversational sessions are excluded, and only Turns <= 2 count as one-shot.
func TestTurnEfficiencyOneShotRate(t *testing.T) {
	sessions := []store.SessionRow{
		{Tool: "claude-code", SessionID: "a", Edits: 2, Turns: 1, OutputTokens: 500},
		{Tool: "claude-code", SessionID: "b", Edits: 1, Turns: 2, OutputTokens: 800},
		{Tool: "claude-code", SessionID: "c", Edits: 5, Turns: 10, OutputTokens: 3000},
		{Tool: "claude-code", SessionID: "d", Edits: 3, Turns: 8, OutputTokens: 1600},
		{Tool: "claude-code", SessionID: "e", Edits: 4, Turns: 6, OutputTokens: 1200},
		{Tool: "claude-code", SessionID: "f", Edits: 0, Turns: 3, OutputTokens: 100}, // conversational: ignored
	}
	in := BuildInput(nil, sessions, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := mustGet(t, turnEffName).Analyze(in)

	// 5 code sessions, 2 one-shot -> 40%.
	if !strings.Contains(figureValues(got.Figures), "40%") {
		t.Fatalf("Figures = %q, want a 40%% one-shot rate", figureValues(got.Figures))
	}
}
