package analyze

import (
	"testing"
	"time"

	"github.com/assaio/assaio/internal/layer"
	"github.com/assaio/assaio/internal/store"
)

// withheldWindows are the two shapes that make a validator withhold: a window with nothing
// in it, and one whose rows come from a source that answers none of the signals a metric
// needs. The second reaches the branches the first short-circuits past.
func withheldWindows(now time.Time) map[string]Input {
	return map[string]Input{
		"empty window": {Now: now},
		"a source that answers nothing": {
			Now:      now,
			Usage:    []store.UsageRow{{Day: now.Format("2006-01-02"), Tool: "gemini-cli", Out: 10}},
			Sessions: []store.SessionRow{{SessionID: "s1", Tool: "gemini-cli", FirstTs: now, LastTs: now, Turns: 1}},
		},
	}
}

// TestWithheldVerdictDrawsTheNeutralGauge holds the class B136 closed in v0.14 and B3
// reopened at v0.21: a validator that withholds its verdict must not also draw a gauge. An
// empty bar reads as a bad result and a full one as good, and a withheld verdict is neither.
// Asserted over the whole registry rather than per validator, because both times the defect
// arrived through a new call site rather than a changed one.
func TestWithheldVerdictDrawsTheNeutralGauge(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	withheld := 0
	for name, in := range withheldWindows(now) {
		for _, v := range Validators() {
			r := v.Analyze(in)
			if r.Read != noDataRead {
				continue
			}
			withheld++
			if r.Purity != neutralPurity {
				t.Errorf("%s on %s: withheld the verdict but drew a %.2f gauge, want %.2f",
					v.Name(), name, r.Purity, neutralPurity)
			}
		}
	}
	if withheld == 0 {
		t.Fatal("no validator withheld a verdict on either window, so the assertion above never ran")
	}
}

// TestEveryValidatorStatesItsLayer is ROADMAP.md's "a metric states which one it is" made
// mechanical on the validator side. Register already panics on an invalid declaration, so this
// asserts the other half: that Evaluate stamps the declaration onto the Result a reader sees,
// rather than leaving it empty for the renderers to guess at.
func TestEveryValidatorStatesItsLayer(t *testing.T) {
	in := Input{Now: time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)}
	for _, v := range Validators() {
		got := Evaluate(v, &in)
		if !layer.Valid(got.Layer) {
			t.Errorf("%s: Result.Layer = %q, not one of activity|output|outcome|impact", v.Name(), got.Layer)
		}
		if got.Layer != v.Layer() {
			t.Errorf("%s: Result says %q while the validator declares %q", v.Name(), got.Layer, v.Layer())
		}
	}
}

// TestNoValidatorClaimsAnOutcome records what the layers make visible rather than asserting a
// preference: assaio's built-in reads are activity and output, and not one of them is evidence
// that code held. Promoting an output metric to an outcome claim is the failure ROADMAP.md
// names, so a validator arriving on the outcome layer has to be a deliberate decision with the
// evidence behind it, not a line changed in passing.
func TestNoValidatorClaimsAnOutcome(t *testing.T) {
	for _, v := range Validators() {
		if l := v.Layer(); l == layer.Outcome || l == layer.Impact {
			t.Errorf("%s claims layer %q; nothing local proves an outcome, and this needs the server stage plus an ADR", v.Name(), l)
		}
	}
}
