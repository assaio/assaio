package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// TestReasoningShareUsesReportingToolsOnly checks that the reasoning share is taken over
// output from tools that actually report reasoning (Codex), not diluted by Claude output
// that never carries a reasoning count.
func TestReasoningShareUsesReportingToolsOnly(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-10", Tool: "codex", Model: "gpt-x", Project: "p", In: 100, Out: 1000, Reasoning: 400},
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "p", In: 100, Out: 9000},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := mustGet(t, reasoningName).Analyze(in)

	// 400 reasoning of 1000 Codex output -> 40%, not 400/10000 diluted by Claude.
	if !strings.Contains(figureValues(got.Figures), "40%") {
		t.Fatalf("Figures = %q, want 40%% reasoning share of reporting output", figureValues(got.Figures))
	}
}

// Which sources report reasoning is declared by the depth matrix, not by a list written into
// this validator: Copilot CLI has reported it since v0.6.0 and was silently excluded from
// both the share and its own coverage figure.
func TestReasoningShareCountsEverySourceThatReportsIt(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-10", Tool: "copilot-cli", Model: "gpt-x", Project: "p", In: 100, Out: 1000, Reasoning: 300},
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "p", In: 100, Out: 1000},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := mustGet(t, reasoningName).Analyze(in)

	// 300 reasoning of 1000 reporting output -> 30%, covering half the window's output.
	joined := figureValues(got.Figures)
	if !strings.Contains(joined, "30%") || !strings.Contains(joined, "50%") {
		t.Fatalf("Figures = %q, want a 30%% share at 50%% reporting coverage", joined)
	}
	for _, c := range got.Caveats {
		if strings.Contains(c, "Only Codex and Gemini CLI") {
			t.Fatalf("the caveat names a fixed pair of sources and is already wrong: %q", c)
		}
	}
}
