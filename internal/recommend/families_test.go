package recommend

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// resultWith builds the shape a family reads: a named verdict carrying a confidence envelope.
func resultWith(name, label string) analyze.Result {
	r := analyze.Result{Name: name}
	r.Confidence.Label = label
	return r
}

// TestPricingEvidenceNeverPredictsACostSize: the unpriced share is a share of tokens, and an
// unpriced model's rate is exactly what nobody knows.
func TestPricingEvidenceNeverPredictsACostSize(t *testing.T) {
	in := analyze.BuildInput([]store.UsageRow{
		{Day: "2026-07-01", Tool: "claude-code", Model: "unpriced-local", In: 1_000_000, Out: 1000},
		{Day: "2026-07-01", Tool: "claude-code", Model: "priced", In: 1000, Out: 100},
	}, nil, pricing.Table{"priced": {Input: 1e-6, Output: 2e-6}}, time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC), 0, analyze.Delegation{})

	got := (pricingCoverage{}).Propose(&Evidence{Input: &in})
	if len(got) != 1 {
		t.Fatalf("proposed %d records for a mostly unpriced window, want 1", len(got))
	}
	if strings.Contains(got[0].Effect, "exactly that share") {
		t.Fatalf("Effect = %q, still equates a token share with a cost error", got[0].Effect)
	}
	if !strings.Contains(got[0].Effect, "share of tokens") {
		t.Fatalf("Effect = %q, want the share named as tokens", got[0].Effect)
	}
}

// TestRightSizingNamesTheFlatPlanCase: the verdict it reads from says a subscription user gets
// speed rather than money, and the record has a field built for exactly that.
func TestRightSizingNamesTheFlatPlanCase(t *testing.T) {
	verdict := resultWith("model-right-sizing", analyze.ConfidenceHigh)
	verdict.Confidence.Samples, verdict.Confidence.Unit = 200, "premium-model turns"
	verdict.Figures = []analyze.Figure{
		{Label: "downgrade candidates", Value: "120"},
		{Label: "small-output premium", Value: "60%"},
	}
	got := (premiumSmallTurns{}).Propose(&Evidence{Results: []analyze.Result{verdict}})
	if len(got) != 1 {
		t.Fatalf("proposed %d records, want 1", len(got))
	}
	if !strings.Contains(strings.Join(got[0].Prerequisites, " "), "flat subscription") {
		t.Fatalf("Prerequisites = %q, want the flat-plan case named", got[0].Prerequisites)
	}
}

// TestPricingComesBeforeWorkflowAdvice is the ordering ADR 0015 argues for and the alphabetical
// sort contradicted: a reader acts on what is at the top, and advice about assaio's own cost
// basis comes before advice about how somebody works.
func TestPricingComesBeforeWorkflowAdvice(t *testing.T) {
	in := analyze.BuildInput([]store.UsageRow{
		{Day: "2026-07-01", Tool: "claude-code", Model: "unpriced-local", In: 1_000_000, Out: 1000},
		{Day: "2026-07-01", Tool: "claude-code", Model: "priced", In: 1000, Out: 100},
	}, nil, pricing.Table{"priced": {Input: 1e-6, Output: 2e-6}},
		time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC), 0, analyze.Delegation{})

	verdict := resultWith("model-right-sizing", analyze.ConfidenceHigh)
	verdict.Confidence.Samples, verdict.Confidence.Unit = 200, "premium-model turns"
	verdict.Figures = []analyze.Figure{
		{Label: "downgrade candidates", Value: "120"},
		{Label: "small-output premium", Value: "60%"},
	}

	got := From(&Evidence{Input: &in, Results: []analyze.Result{verdict}})
	if len(got) != 2 {
		t.Fatalf("got %d records, want both families to fire", len(got))
	}
	if got[0].Family != "pricing-coverage" {
		t.Fatalf("first record is %q; the cost basis comes before workflow advice", got[0].Family)
	}
}
