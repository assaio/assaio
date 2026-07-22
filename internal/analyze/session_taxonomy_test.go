package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// TestSessionTaxonomyBucketsByEdits checks the conversational/light/heavy split and that a
// conversational-majority window reads as chat-led (design/debug work), not waste.
func TestSessionTaxonomyBucketsByEdits(t *testing.T) {
	sessions := []store.SessionRow{
		{SessionID: "a", Edits: 0, Turns: 3},
		{SessionID: "b", Edits: 0, Turns: 2},
		{SessionID: "c", Edits: 0, Turns: 1},
		{SessionID: "d", Edits: 3, Turns: 5},
		{SessionID: "e", Edits: 25, Turns: 12},
	}
	in := BuildInput(nil, sessions, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := mustGet(t, taxonomyName).Analyze(in)

	if !strings.Contains(figureValues(got.Figures), "60%") { // 3 of 5 conversational
		t.Fatalf("Figures = %q, want 60%% conversational", figureValues(got.Figures))
	}
	if !strings.Contains(got.Takeaway, "conversational") {
		t.Fatalf("Takeaway = %q, want the chat-led message", got.Takeaway)
	}
}
