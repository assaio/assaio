package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// labeledSessions builds n sessions per task class, plus unlabeled ones.
func labeledSessions(unlabeled int, byTask map[string]int) []store.SessionRow {
	var out []store.SessionRow
	for i := range unlabeled {
		out = append(out, store.SessionRow{SessionID: "u" + string(rune('a'+i)), Project: "p"})
	}
	for task, n := range byTask {
		for i := range n {
			out = append(out, store.SessionRow{
				SessionID: task + string(rune('a'+i)), Project: "p", Task: task, Outcome: "done",
			})
		}
	}
	return out
}

func intentResult(t *testing.T, sessions []store.SessionRow) Result {
	t.Helper()
	in := BuildInput(nil, sessions, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	return mustGet(t, intentName).Analyze(in)
}

func TestIntentReads(t *testing.T) {
	for _, tc := range []struct {
		name      string
		sessions  []store.SessionRow
		wantKey   string
		wantLabel string
	}{
		{"no sessions at all", nil, "neutral", "—"},
		{"nothing labeled", labeledSessions(5, nil), "neutral", "—"},
		{
			"one class only, nothing to compare against",
			labeledSessions(5, map[string]int{"refactor": 4}),
			"neutral", "BUILDING",
		},
		{
			"a class too small to compare",
			labeledSessions(5, map[string]int{"refactor": 4, "docs": 1}),
			"neutral", "BUILDING",
		},
		{
			"two comparable classes",
			labeledSessions(5, map[string]int{"refactor": 4, "bugfix": 3}),
			"good", "STRATIFIABLE",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := intentResult(t, tc.sessions)
			if got.Read.Key != tc.wantKey || got.Read.Label != tc.wantLabel {
				t.Fatalf("Read = %+v want {%s %s}", got.Read, tc.wantKey, tc.wantLabel)
			}
		})
	}
}

// TestIntentNeverWatches is the property that keeps this metric from scoring a person:
// not labeling, and doing work that is genuinely all one kind, are both fine.
func TestIntentNeverWatches(t *testing.T) {
	for _, sessions := range [][]store.SessionRow{
		labeledSessions(500, nil),
		labeledSessions(0, map[string]int{"feature": 50}),
		labeledSessions(200, map[string]int{"feature": 1}),
	} {
		if got := intentResult(t, sessions).Read; got.Key == "watch" {
			t.Fatalf("Read = %+v; intent must never report an unfavorable verdict", got)
		}
	}
}

// TestIntentRestsOnEverySessionExamined guards the confidence basis: "none of your
// sessions is labeled" is a confident finding, not an under-sampled one.
func TestIntentRestsOnEverySessionExamined(t *testing.T) {
	got := intentResult(t, labeledSessions(9, nil))
	if got.Confidence.Samples != 9 || got.Confidence.Unit != "sessions" {
		t.Fatalf("Confidence = %d %q want 9 \"sessions\"", got.Confidence.Samples, got.Confidence.Unit)
	}
}

func TestIntentCountsCoverageAndClasses(t *testing.T) {
	got := intentResult(t, labeledSessions(6, map[string]int{"refactor": 3, "docs": 1}))

	figures := figureValues(got.Figures)
	if !strings.Contains(figures, "4 of 10") {
		t.Fatalf("Figures = %q, want the labeled share as 4 of 10", figures)
	}
	// Two classes exist but only one clears the comparison floor.
	if !strings.Contains(figures, "2") || !strings.Contains(figures, "1") {
		t.Fatalf("Figures = %q, want 2 classes used and 1 comparable", figures)
	}
	if len(got.Bars) != 2 || got.Bars[0].Label != "refactor" || got.Bars[0].Value != "3" {
		t.Fatalf("Bars = %+v, want refactor ranked first with 3", got.Bars)
	}
	// The bar labels are a vocabulary assaio defines, never a name a person chose, so an
	// anonymized dashboard must not pseudonymize them.
	if got.BarsPseudonym != "" {
		t.Fatalf("BarsPseudonym = %q, want empty for a fixed vocabulary", got.BarsPseudonym)
	}
}

// TestIntentSessionLabelledOnAnyAxisCounts covers marking difficulty or outcome without a
// task: the session is labeled, even though it contributes to no task class.
func TestIntentSessionLabelledOnAnyAxisCounts(t *testing.T) {
	sessions := []store.SessionRow{
		{SessionID: "a", Project: "p"},
		{SessionID: "b", Project: "p", Difficulty: "high"},
	}
	got := intentResult(t, sessions)
	if !strings.Contains(figureValues(got.Figures), "1 of 2") {
		t.Fatalf("Figures = %q, want 1 of 2 labeled", figureValues(got.Figures))
	}
}
