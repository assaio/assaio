package calibration_test

import (
	"testing"

	"github.com/assaio/assaio/internal/calibration"
	"github.com/assaio/assaio/internal/usage"
)

// parseTrace runs the source's own parser over a trace file. This is the only place in the
// package where internal/parser is called: everything it is compared against was counted
// somewhere else.
func parseTrace(t *testing.T, source, path string) ([]usage.Record, int) {
	t.Helper()
	parse, ok := parsers[source]
	if !ok {
		t.Fatalf("no parser registered for source %q", source)
	}
	recs, skipped, err := parse(path)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return recs, skipped
}

// TestAdjudicatedTotals is the check a golden file cannot be: the expected answer was
// counted from the trace's own bytes, so a parser that reads the log at the wrong semantic
// grain fails here instead of writing its mistake into a golden and preserving it.
func TestAdjudicatedTotals(t *testing.T) {
	answers, err := calibration.LoadAdjudicated("testdata")
	if err != nil {
		t.Fatal(err)
	}
	if len(answers) == 0 {
		t.Fatal("no adjudicated traces found")
	}
	for _, a := range answers {
		t.Run(a.Source+"/"+a.Trace, func(t *testing.T) {
			recs, skipped := parseTrace(t, a.Source, a.Trace)
			got := calibration.Sum(recs, skipped)
			for _, m := range calibration.Diff(&got, &a.Totals) {
				t.Errorf("%s: parser %d, adjudicated %d\n    how it was adjudicated: %s",
					m.Figure, m.Got, m.Want, a.Derivations[m.Figure])
			}
		})
	}
}

// TestEveryAdjudicatedFigureIsDerived: a total nobody can explain is a golden file with
// extra steps. Every figure has to carry the reasoning that produced it.
func TestEveryAdjudicatedFigureIsDerived(t *testing.T) {
	answers, err := calibration.LoadAdjudicated("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range answers {
		for _, name := range calibration.FigureNames() {
			if a.Derivations[name] == "" {
				t.Errorf("%s: figure %q has no derivation", a.Source, name)
			}
		}
		if a.Origin == "" || a.Method == "" {
			t.Errorf("%s: a trace must say where it came from and how it was counted", a.Source)
		}
	}
}
