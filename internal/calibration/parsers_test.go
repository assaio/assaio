package calibration_test

import (
	"io"
	"os"

	"github.com/assaio/assaio/internal/parser/agy"
	"github.com/assaio/assaio/internal/parser/claude"
	"github.com/assaio/assaio/internal/parser/cline"
	"github.com/assaio/assaio/internal/parser/codex"
	"github.com/assaio/assaio/internal/parser/copilot"
	"github.com/assaio/assaio/internal/parser/gemini"
	"github.com/assaio/assaio/internal/usage"
)

// parsers maps a source to the entry point its trace is read through, taking a path rather
// than a reader because Cline stores a session as a directory. A source missing from this
// map has no calibrated trace at all, which TestEverySourceIsCalibrated reports.
//
// Every entry returns steps beside records, and a source with no step reading returns none: that
// absence is the claim its depth row makes, and adjudicating it as zero is how the claim is
// checked rather than assumed.
var parsers = map[string]func(path string) ([]usage.Record, []usage.Step, int, error){
	"claude-code": fromFile(claude.ParseAll),
	"codex":       recordsOnly(codex.Parse),
	"gemini-cli":  recordsOnly(gemini.Parse),
	"copilot-cli": recordsOnly(copilot.Parse),
	"cline":       dirRecordsOnly(cline.ParseDir),
	"agy":         dirRecordsOnly(agy.ParseDir),
}

func fromFile(parse func(io.Reader) ([]usage.Record, []usage.Step, int, error)) func(string) ([]usage.Record, []usage.Step, int, error) {
	return func(path string) ([]usage.Record, []usage.Step, int, error) {
		f, err := os.Open(path) //nolint:gosec // a fixed testdata path named by this package's own fixtures
		if err != nil {
			return nil, nil, 0, err
		}
		defer func() { _ = f.Close() }()
		return parse(f)
	}
}

// recordsOnly adapts a source that reads no sequence, so its trace adjudicates the step figures
// as absent rather than being exempt from them.
func recordsOnly(parse func(io.Reader) ([]usage.Record, int, error)) func(string) ([]usage.Record, []usage.Step, int, error) {
	return func(path string) ([]usage.Record, []usage.Step, int, error) {
		f, err := os.Open(path) //nolint:gosec // a fixed testdata path named by this package's own fixtures
		if err != nil {
			return nil, nil, 0, err
		}
		defer func() { _ = f.Close() }()
		recs, skipped, err := parse(f)
		return recs, nil, skipped, err
	}
}

func dirRecordsOnly(parse func(path string) ([]usage.Record, int, error)) func(string) ([]usage.Record, []usage.Step, int, error) {
	return func(path string) ([]usage.Record, []usage.Step, int, error) {
		recs, skipped, err := parse(path)
		return recs, nil, skipped, err
	}
}
