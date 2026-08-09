package calibration_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/calibration"
	"github.com/assaio/assaio/internal/parser"
)

// TestEverySourceIsCalibrated holds the clause v1.0 rests on: a source either calibrates
// every signal its depth row claims, or the row is a promise nothing checks. A capability
// added to the matrix without a trace that pins it fails here, in the same change that
// added it.
func TestEverySourceIsCalibrated(t *testing.T) {
	answers, err := calibration.LoadAdjudicated("testdata")
	if err != nil {
		t.Fatal(err)
	}
	calibrated := make(map[string]map[string]bool)
	for _, a := range answers {
		if calibrated[a.Source] == nil {
			calibrated[a.Source] = make(map[string]bool)
		}
		for _, id := range a.Calibrates {
			calibrated[a.Source][id] = true
		}
	}
	for _, tool := range parser.Tools() {
		t.Run(tool, func(t *testing.T) {
			var missing []string
			for _, id := range parser.SignalsAnsweredBy(tool) {
				if !calibrated[tool][id] {
					missing = append(missing, id)
				}
			}
			sort.Strings(missing)
			if len(missing) > 0 {
				t.Errorf("depth row claims %d signal(s) no adjudicated trace pins: %s",
					len(missing), strings.Join(missing, ", "))
			}
		})
	}
}

// TestCalibratesOnlyDeclaredSignals: a trace claiming to pin a signal its source does not
// produce would report coverage assaio does not have, which is the failure mode the depth
// matrix exists to prevent.
func TestCalibratesOnlyDeclaredSignals(t *testing.T) {
	answers, err := calibration.LoadAdjudicated("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range answers {
		for _, id := range a.Calibrates {
			if !parser.Answers(a.Source, id) {
				t.Errorf("%s: trace claims to calibrate %s, which the depth row does not answer",
					a.Source, id)
			}
		}
	}
}

// TestCaptureIsDeclared: a constructed sample proves the reading, not the shape -- only a
// real capture can show that the vendor still writes what the parser expects. The two are
// different guarantees, so a trace has to say which one it offers.
func TestCaptureIsDeclared(t *testing.T) {
	answers, err := calibration.LoadAdjudicated("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range answers {
		if a.Capture != "real" && a.Capture != "constructed" {
			t.Errorf("%s: capture is %q, want real or constructed", a.Source, a.Capture)
		}
	}
}
