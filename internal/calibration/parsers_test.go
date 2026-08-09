package calibration_test

import (
	"io"
	"os"

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
var parsers = map[string]func(path string) ([]usage.Record, int, error){
	"claude-code": fromFile(claude.Parse),
	"codex":       fromFile(codex.Parse),
	"gemini-cli":  fromFile(gemini.Parse),
	"copilot-cli": fromFile(copilot.Parse),
	"cline":       cline.ParseDir,
}

func fromFile(parse func(io.Reader) ([]usage.Record, int, error)) func(string) ([]usage.Record, int, error) {
	return func(path string) ([]usage.Record, int, error) {
		f, err := os.Open(path) //nolint:gosec // a fixed testdata path named by this package's own fixtures
		if err != nil {
			return nil, 0, err
		}
		defer func() { _ = f.Close() }()
		return parse(f)
	}
}
