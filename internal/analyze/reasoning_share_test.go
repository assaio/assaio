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
