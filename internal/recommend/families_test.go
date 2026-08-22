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
func resultWith(name, label string, signal *float64, recorded *bool) analyze.Result {
	r := analyze.Result{Name: name}
	r.Confidence.Label = label
	r.Confidence.Signal = signal
	r.Confidence.Recorded = recorded
	return r
}

func ptrF(v float64) *float64 { return &v }
func ptrB(v bool) *bool       { return &v }

// TestCaptureGapFiresOnlyOnARepairableGap is the polarity this family shipped inverted: it
// must fire where a backfill can help (a source that records the field ran here) and stay
// silent where the source structurally records nothing.
func TestCaptureGapFiresOnlyOnARepairableGap(t *testing.T) {
	repairable := &Evidence{Results: []analyze.Result{
		resultWith("friction", analyze.ConfidenceLow, ptrF(0.3), ptrB(true)),
	}}
	if got := (captureGap{}).Propose(repairable); len(got) != 1 {
		t.Fatalf("proposed %d records for a repairable gap, want 1", len(got))
	}

	structural := &Evidence{Results: []analyze.Result{
		resultWith("friction", analyze.ConfidenceLow, ptrF(0.3), ptrB(false)),
	}}
	if got := (captureGap{}).Propose(structural); len(got) != 0 {
		t.Fatalf("proposed %d records for a gap no backfill can close, want 0", len(got))
	}

	silent := &Evidence{Results: []analyze.Result{
		resultWith("friction", analyze.ConfidenceHigh, ptrF(1), ptrB(true)),
	}}
	if got := (captureGap{}).Propose(silent); len(got) != 0 {
		t.Fatalf("proposed %d records for a fully covered window, want 0", len(got))
	}
}

// TestCaptureGapReadsAVerdictThatActuallySetsRecorded is the second half of the same defect:
// the family originally asked `coverage`, which never calls missingCaptureWhen, so Recorded
// was permanently nil and the family could not fire at all.
func TestCaptureGapReadsAVerdictThatActuallySetsRecorded(t *testing.T) {
	in := analyze.BuildInput(
		[]store.UsageRow{{Day: "2026-07-01", Tool: "claude-code", Model: "m", In: 10, ToolCalls: 40}},
		nil, pricing.Table{}, time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC), 0, analyze.Delegation{},
	)
	for _, name := range captureRepairable {
		v, ok := analyze.Get(name)
		if !ok {
			t.Fatalf("%s is not a registered validator", name)
		}
		if got := analyze.Evaluate(v, &in); got.Confidence.Recorded == nil {
			t.Fatalf("%s never sets Recorded, so this family can never fire on it", name)
		}
	}
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
	verdict := resultWith("model-right-sizing", analyze.ConfidenceHigh, nil, nil)
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
