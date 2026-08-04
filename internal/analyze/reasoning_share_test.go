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

// The share is real but describes a sliver: on live data a 20% reasoning share computed off
// under 1% of the window's output still carried "high", while `signals coverage` called the
// same signal partially supported. The verdict has to carry its own reach.
func TestReasoningShareCarriesItsReportingCoverageIntoTheEnvelope(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-08", Tool: "codex", Model: "claude-sonnet-4-5", Granularity: "turn", Project: "p", In: 10, Out: 10, Reasoning: 2},
		{Day: "2026-07-09", Tool: "claude-code", Model: "claude-sonnet-4-5", Granularity: "turn", Project: "p", In: 10, Out: 5000},
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Granularity: "turn", Project: "p", In: 10, Out: 4990},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := Evaluate(mustGet(t, reasoningName), &in)

	if got.Confidence.Samples < confidenceSampleFloor {
		t.Fatalf("Samples = %d, too few for this test to be about coverage at all", got.Confidence.Samples)
	}
	if got.Confidence.Label == ConfidenceHigh {
		t.Fatalf("Label = %q for a share read off 0.1%% of the output, want it held down (envelope %+v)",
			got.Confidence.Label, got.Confidence)
	}
	if got.Confidence.signalShare() > 0.01 {
		t.Errorf("Signal = %v, want the reporting coverage the validator already prints", got.Confidence.signalShare())
	}
}

// No source in the window reports reasoning at all: there is no thin answer here, there is
// no question. The verdict said so in its takeaway while its envelope read "high · 43 active
// days" on live data.
func TestReasoningShareWithoutAReportingSourceReadsInsufficient(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-08", Tool: "claude-code", Model: "claude-sonnet-4-5", Granularity: "turn", Project: "p", In: 100, Out: 1000},
		{Day: "2026-07-09", Tool: "claude-code", Model: "claude-sonnet-4-5", Granularity: "turn", Project: "p", In: 100, Out: 1000},
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Granularity: "turn", Project: "p", In: 100, Out: 1000},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := Evaluate(mustGet(t, reasoningName), &in)

	if got.Confidence.Samples < confidenceSampleFloor {
		t.Fatalf("Samples = %d, so this would read insufficient for want of observations, not reach",
			got.Confidence.Samples)
	}
	if got.Confidence.Label != ConfidenceInsufficient {
		t.Fatalf("Label = %q, want %q when nothing in the window can report reasoning",
			got.Confidence.Label, ConfidenceInsufficient)
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
