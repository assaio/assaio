package calibration_test

import (
	"bufio"
	"os"
	"testing"

	"github.com/assaio/assaio/internal/calibration"
)

// TestInvariantsHoldOnEveryTrace runs the accounting rules over every calibrated trace.
// Unlike the adjudicated totals, these need no expected value, so the same code runs over a
// whole real corpus -- which is what TestInvariantsHoldOnTheRealCorpus below does when a
// machine has one.
func TestInvariantsHoldOnEveryTrace(t *testing.T) {
	answers, err := calibration.LoadAdjudicated("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range answers {
		t.Run(a.Source+"/"+a.Trace, func(t *testing.T) {
			recs, _, skipped := parseTrace(t, a.Source, a.Trace)
			for _, v := range calibration.Invariants(recs, skipped) {
				t.Error(v)
			}
		})
	}
}

// TestInvariantsHoldOnTheRealCorpus is the same rules over as many real session files as the
// caller points it at, which is the only way to exercise shapes no fixture contains. It is
// opt-in because the corpus is the developer's own machine: ASSAIO_CALIBRATION_CORPUS names a
// file listing one transcript path per line, and ASSAIO_CALIBRATION_SOURCE the parser to read
// them with.
func TestInvariantsHoldOnTheRealCorpus(t *testing.T) {
	manifest := os.Getenv("ASSAIO_CALIBRATION_CORPUS")
	if manifest == "" {
		t.Skip("set ASSAIO_CALIBRATION_CORPUS to a file listing session paths")
	}
	source := os.Getenv("ASSAIO_CALIBRATION_SOURCE")
	if _, ok := parsers[source]; !ok {
		t.Fatalf("ASSAIO_CALIBRATION_SOURCE=%q is not a calibrated source", source)
	}
	f, err := os.Open(manifest) //nolint:gosec // the operator names their own corpus manifest
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	var files, violations int
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		path := sc.Text()
		if path == "" {
			continue
		}
		recs, _, skipped, err := parsers[source](path)
		if err != nil {
			continue
		}
		files++
		for _, v := range calibration.Invariants(recs, skipped) {
			violations++
			if violations <= 20 {
				t.Errorf("%s: %s", path, v)
			}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	t.Logf("checked %d %s sessions, %d violations", files, source, violations)
}
