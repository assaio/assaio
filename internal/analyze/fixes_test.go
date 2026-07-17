// Tests for the C1 real-data metric fixes: throughput's top-projects ordering,
// model-fit's real delegation share and one-decimal percentages, and context's bimodal
// active-work honesty. The shared trend fix has its own trend_test.go.
package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// TestThroughputBarsSortedByLinesDescending is the throughput top-projects ordering fix:
// a project with a small line count but high cost must not outrank a project with far
// more lines but lower cost. Bars must be ordered, and labeled, by AI lines descending.
func TestThroughputBarsSortedByLinesDescending(t *testing.T) {
	usage := []store.UsageRow{
		// costly-low-lines: expensive premium model, few lines.
		{
			Day: "2026-07-12", Tool: "claude-code", Model: "claude-opus-4-5", Project: "costly-low-lines",
			In: 100000, Out: 200000, LinesAdded: 1916,
		},
		// cheap-high-lines: cheap model, far more lines, far less cost.
		{
			Day: "2026-07-12", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "cheap-high-lines",
			In: 1000, Out: 2000, LinesAdded: 18496,
		},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	v, _ := Get("throughput")
	got := v.Analyze(in)
	if len(got.Bars) != 2 {
		t.Fatalf("Bars = %+v, want 2 entries", got.Bars)
	}
	if got.Bars[0].Label != "cheap-high-lines" {
		t.Fatalf("Bars[0] = %+v, want the higher-lines project (cheap-high-lines) first", got.Bars[0])
	}
	if got.Bars[1].Label != "costly-low-lines" {
		t.Fatalf("Bars[1] = %+v, want the lower-lines project (costly-low-lines) second", got.Bars[1])
	}
	if !strings.Contains(got.Bars[0].Value, "18496") {
		t.Fatalf("Bars[0].Value = %q, want it to show 18496 lines", got.Bars[0].Value)
	}
	if got.Bars[0].Frac != 1 {
		t.Fatalf("Bars[0].Frac = %v, want 1 (the max)", got.Bars[0].Frac)
	}
}

// TestModelFitPercentagesUseOneDecimalNeverRoundToZero is the model-fit precision fix: a
// genuinely small nonzero share must render with one decimal (e.g. "0.1%"), never a bare
// "0%" that reads as nothing happened. The token counts mirror a real captured split.
func TestModelFitPercentagesUseOneDecimalNeverRoundToZero(t *testing.T) {
	prices := pricing.Table{
		"claude-opus-4-5":   {Output: 25e-6},
		"claude-sonnet-4-5": {Output: 15e-6},
	}
	usage := []store.UsageRow{
		{Day: "2026-07-12", Tool: "claude-code", Model: "claude-opus-4-5", Project: "p", In: 40000000000, Out: 3368558199},
		{Day: "2026-07-12", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "p", In: 40000000, Out: 11733256},
	}
	in := BuildInput(usage, nil, prices, validatorsTestNow, 7*24*time.Hour, Delegation{})
	v, _ := Get("model-fit")
	got := v.Analyze(in)
	joined := figureValues(got.Figures)
	if !strings.Contains(joined, "99.9%") {
		t.Fatalf("Figures = %q, want a one-decimal 99.9%% premium share", joined)
	}
	if !strings.Contains(joined, "0.1%") {
		t.Fatalf("Figures = %q, want a one-decimal 0.1%% cheaper share, not a rounded-away 0%%", joined)
	}
	if strings.Contains(joined, " 0%") || strings.Contains(joined, ": 0%") {
		t.Fatalf("Figures = %q, a nonzero share must never render as bare 0%%", joined)
	}
}

// TestModelFitDelegationUsesRealInputNotCheaperShare is the delegation-honesty fix: the
// sub-agent delegation figure must come from in.Delegation (the real dedupe_key-derived
// count), not be approximated from the cheaper-model token share, which this scenario
// deliberately sets to a different value than Delegation's ratio.
func TestModelFitDelegationUsesRealInputNotCheaperShare(t *testing.T) {
	prices := pricing.Table{"claude-sonnet-4-5": {Output: 15e-6}}
	usage := []store.UsageRow{
		// 100% of tokens on a cheaper model -- the old approximation would have read
		// sub-agent delegation as ~100%.
		{Day: "2026-07-12", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "p", In: 1000, Out: 1000},
	}
	in := BuildInput(usage, nil, prices, validatorsTestNow, 7*24*time.Hour, Delegation{Sub: 25, Total: 1000})
	v, _ := Get("model-fit")
	got := v.Analyze(in)
	joined := figureValues(got.Figures)
	if !strings.Contains(joined, "2.5%") {
		t.Fatalf("Figures = %q, want the real delegation share 2.5%% (25/1000), not the ~100%% cheaper-token approximation", joined)
	}
}

// TestContextActiveWorkReportsCodeSessionMedianSeparately is the bimodal-honesty fix:
// most sessions are quick one-offs pulling the overall median near zero, so the figure
// must also surface the median for sessions that actually produced code.
func TestContextActiveWorkReportsCodeSessionMedianSeparately(t *testing.T) {
	sessions := []store.SessionRow{
		{SessionID: "quick1", FirstTs: validatorsTestNow, Turns: 1, ActiveMinutes: 0, Edits: 0},
		{SessionID: "quick2", FirstTs: validatorsTestNow, Turns: 1, ActiveMinutes: 1, Edits: 0},
		{SessionID: "quick3", FirstTs: validatorsTestNow, Turns: 1, ActiveMinutes: 0, Edits: 0},
		{SessionID: "code1", FirstTs: validatorsTestNow, Turns: 10, ActiveMinutes: 14, Edits: 3},
		{SessionID: "code2", FirstTs: validatorsTestNow, Turns: 12, ActiveMinutes: 18, Edits: 5},
	}
	in := BuildInput(nil, sessions, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	v, _ := Get("context")
	got := v.Analyze(in)

	var activeWork *Figure
	for i := range got.Figures {
		if got.Figures[i].Label == "active work" {
			activeWork = &got.Figures[i]
		}
	}
	if activeWork == nil {
		t.Fatalf("Figures = %+v, missing the \"active work\" figure", got.Figures)
	}
	if !strings.Contains(activeWork.Value, "1 min") {
		t.Fatalf("active work Value = %q, want the bare overall median (~1 min)", activeWork.Value)
	}
	if !strings.Contains(activeWork.Note, "code sessions") || !strings.Contains(activeWork.Note, "16 min") {
		t.Fatalf("active work Note = %q, want the code-session median (~16 min) called out separately", activeWork.Note)
	}
}

// figureValues joins every Figure's Label/Value/Note into one searchable string.
func figureValues(figures []Figure) string {
	var b strings.Builder
	for _, f := range figures {
		b.WriteString(f.Label)
		b.WriteString(": ")
		b.WriteString(f.Value)
		b.WriteString(" (")
		b.WriteString(f.Note)
		b.WriteString(")\n")
	}
	return b.String()
}
