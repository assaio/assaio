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
	if got.BarsPseudonym != "" {
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

// "Cost only" means a source contributes no changed lines. Copilot CLI does contribute them
// while answering no edit or tool-call signal, so it must lower activity coverage without
// being labelled cost-only -- two different questions the one Activity bit used to conflate.
func TestCoverageSeparatesCostOnlyFromPartialActivity(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", In: 500, Out: 100},
		{Day: "2026-07-10", Tool: "copilot-cli", Model: "claude-sonnet-4-5", Project: "web", In: 300, Out: 100, LinesAdded: 40},
		{Day: "2026-07-10", Tool: "gemini-cli", Model: "claude-sonnet-4-5", In: 150, Out: 50},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := mustGet(t, coverageName).Analyze(in)

	for _, b := range got.Bars {
		if strings.Contains(b.Label, "copilot-cli") && strings.Contains(b.Label, "cost only") {
			t.Errorf("copilot-cli contributes changed lines, so it is not cost-only: %q", b.Label)
		}
		if strings.Contains(b.Label, "gemini-cli") && !strings.Contains(b.Label, "cost only") {
			t.Errorf("gemini-cli contributes no lines, so it is cost-only: %q", b.Label)
		}
	}
	// activity (claude-code alone) 600/1200 -> 50%; cost-only (gemini-cli) 200/1200 -> 17%.
	joined := figureValues(got.Figures)
	if !strings.Contains(joined, "50%") || !strings.Contains(joined, "17%") {
		t.Fatalf("Figures = %q, want 50%% activity coverage and 17%% cost-only tokens", joined)
	}
	var named bool
	for _, c := range got.Caveats {
		if strings.Contains(c, "copilot-cli") {
			named = true
		}
	}
	if !named {
		t.Fatalf("Caveats = %q, want the lines-but-no-edits source named", got.Caveats)
	}
	if strings.Contains(got.Takeaway, "cost-only") {
		t.Fatalf("takeaway blames cost-only sources for a gap the caveat attributes to a "+
			"partial-activity one: %q", got.Takeaway)
	}
}

// TestCoverageReportsRecordGranularity closes the gap coverage used to disclose only in
// prose: a session-aggregate source contributes tokens that cannot be read per turn, and
// the share of the window that is per-turn is now a figure rather than a caveat.
func TestCoverageReportsRecordGranularity(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", Granularity: "turn", In: 600, Out: 150},
		{Day: "2026-07-10", Tool: "plugin:acme", Model: "claude-sonnet-4-5", Granularity: "session", In: 200, Out: 50},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := mustGet(t, coverageName).Analyze(in)

	joined := figureValues(got.Figures)
	// total 1000; per-turn tokens 750 -> 75%.
	if !strings.Contains(joined, "75%") {
		t.Fatalf("Figures = %q, want 75%% turn-level coverage (750/1000)", joined)
	}
	for _, c := range got.Caveats {
		if strings.Contains(c, "not yet surfaced") {
			t.Fatalf("granularity is surfaced now, so the caveat must be gone: %q", c)
		}
	}
}

// TestCoverageStaysSilentOnGranularityWhenEverythingIsPerTurn keeps the common case clean:
// a figure that always reads 100% is noise.
func TestCoverageStaysSilentOnGranularityWhenEverythingIsPerTurn(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", Granularity: "turn", In: 600, Out: 150},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := mustGet(t, coverageName).Analyze(in)
	for _, f := range got.Figures {
		if strings.Contains(f.Label, "turn-level") {
			t.Fatalf("all-per-turn data needs no granularity figure: %+v", got.Figures)
		}
	}
}
