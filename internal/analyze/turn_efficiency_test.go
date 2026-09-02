package analyze

import (
	"fmt"
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

// TestTurnEfficiencyWithholdsOutputPerTurnWithoutATokenCounter: a source can record every turn
// and every edit and still publish no token at all. Dividing its zero output by its real turns
// would put a structural zero into the median of a figure denominated in tokens, while the two
// figures beside it stay legitimately readable -- which is why the gate is per figure.
func TestTurnEfficiencyWithholdsOutputPerTurnWithoutATokenCounter(t *testing.T) {
	sessions := make([]store.SessionRow, 0, 8)
	for i := range 8 {
		sessions = append(sessions, store.SessionRow{
			SessionID: fmt.Sprintf("s%d", i), Tool: "agy", Turns: 3, Edits: 2,
		})
	}
	got := mustGet(t, turnEffName).Analyze(Input{Sessions: sessions, Now: validatorsTestNow})
	byLabel := map[string]Figure{}
	for _, f := range got.Figures {
		byLabel[f.Label] = f
	}
	if byLabel["output/turn"].Value != "—" {
		t.Errorf("output/turn = %q, want — from a source with no token counter", byLabel["output/turn"].Value)
	}
	// The reach is stated on the figure, not left to the result's one coverage number: that
	// envelope reports the edits-and-turns basis the other two figures rest on, and a bare
	// dash under it reads as an empty window rather than as a capability this one lacks.
	if byLabel["output/turn"].Note != "no source here records it" {
		t.Errorf("output/turn note = %q, want the figure to state its own basis", byLabel["output/turn"].Note)
	}
	if byLabel["median turns"].Value != "3" {
		t.Errorf("median turns = %q, want the turn figure to stay readable", byLabel["median turns"].Value)
	}
}
