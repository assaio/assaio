package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// TestCoverageReportsActivityAndPricedShares checks the provenance meter: tokens from a
// cost-only tool lower activity coverage, and tokens on an unpriced model lower priced
// coverage, each reported as its own share -- never silently folded into the others.
func TestCoverageReportsActivityAndPricedShares(t *testing.T) {
	usage := []store.UsageRow{
		// Activity-capable + priced.
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", In: 600, Out: 200},
		// Cost-only tool, priced model.
		{Day: "2026-07-10", Tool: "gemini-cli", Model: "claude-sonnet-4-5", In: 150, Out: 50},
		// Activity-capable tool, unpriced model.
		{Day: "2026-07-10", Tool: "codex", Model: "unknown-model", Project: "api", In: 800, Out: 200},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := mustGet(t, coverageName).Analyze(in)

	joined := figureValues(got.Figures)
	// total 2000; activity-capable (claude-code+codex) = 1800 -> 90%.
	if !strings.Contains(joined, "90%") {
		t.Fatalf("Figures = %q, want 90%% activity coverage (1800/2000)", joined)
	}
	// priced (models in testPrices) = the two claude-sonnet rows, 1000 -> 50%.
	if !strings.Contains(joined, "50%") {
		t.Fatalf("Figures = %q, want 50%% priced coverage (1000/2000)", joined)
	}
	if got.BarsAreProjects {
		t.Fatal("coverage bars label tools, so they must never be pseudonymized as projects")
	}
	var sawCostOnly bool
	for _, b := range got.Bars {
		if strings.Contains(b.Label, "gemini-cli") && strings.Contains(b.Label, "cost only") {
			sawCostOnly = true
		}
	}
	if !sawCostOnly {
		t.Fatalf("Bars = %+v, want gemini-cli marked '(cost only)'", got.Bars)
	}
}
