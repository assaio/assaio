package calibration_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/assaio/assaio/internal/calibration"
	"github.com/assaio/assaio/internal/parser/claude"
	"github.com/assaio/assaio/internal/parser/codex"
	"github.com/assaio/assaio/internal/usage"
)

// property is one metamorphic law: rewriting a trace one way must leave the named figures
// where they were.
type property struct {
	name      string
	trace     string
	rewrite   func([]byte) []byte
	parse     func(io.Reader) ([]usage.Record, int, error)
	unchanged func(a, b *calibration.Totals) []string
}

func tokensUnchanged(a, b *calibration.Totals) []string {
	var out []string
	if a.Input != b.Input || a.Output != b.Output || a.CacheRead != b.CacheRead || a.CacheWrite != b.CacheWrite {
		out = append(out, "token totals moved")
	}
	if a.Records != b.Records {
		out = append(out, "record count moved")
	}
	return out
}

func linesUnchanged(a, b *calibration.Totals) []string {
	if a.LinesAdded != b.LinesAdded || a.LinesRemoved != b.LinesRemoved {
		return []string{"line counts moved"}
	}
	return nil
}

// TestMetamorphicProperties: each law below would have caught a defect that shipped. They
// are the checks that need no expected answer at all -- only that two ways of writing the
// same fact are read the same way.
func TestMetamorphicProperties(t *testing.T) {
	props := []property{
		{
			// v0.12: one response written as several content-block lines, each repeating the
			// response's usage, was billed once per line.
			name:  "splitting a response across blocks does not change its tokens",
			trace: "testdata/claude-code/session.jsonl", rewrite: calibration.SplitResponseBlocks,
			parse: claude.Parse, unchanged: tokensUnchanged,
		},
		{
			// v0.14: a creation arrives as the file's body beside an empty patch, and only
			// the patch was read.
			name:  "a Claude creation counts the same written as a patch",
			trace: "testdata/claude-code/session.jsonl", rewrite: calibration.CreationAsPatch,
			parse: claude.Parse, unchanged: linesUnchanged,
		},
		{
			// v0.13 (B119): the same defect in Codex, one release earlier.
			name:  "a Codex creation counts the same written as a diff",
			trace: "testdata/codex/compaction.jsonl", rewrite: calibration.CreationAsDiff,
			parse: codex.Parse, unchanged: linesUnchanged,
		},
	}
	for _, p := range props {
		t.Run(p.name, func(t *testing.T) {
			raw, err := os.ReadFile(p.trace)
			if err != nil {
				t.Fatal(err)
			}
			before := totalsOf(t, p.parse, raw)
			after := totalsOf(t, p.parse, p.rewrite(raw))
			for _, d := range p.unchanged(&before, &after) {
				t.Errorf("%s\n    before: %+v\n    after:  %+v", d, before, after)
			}
		})
	}
}

// TestRewritesActuallyRewrite guards the property tests themselves: a transform that matched
// nothing would compare a trace against itself and pass forever.
func TestRewritesActuallyRewrite(t *testing.T) {
	cases := []struct {
		name    string
		trace   string
		rewrite func([]byte) []byte
	}{
		{"split blocks", "testdata/claude-code/session.jsonl", calibration.SplitResponseBlocks},
		{"creation as patch", "testdata/claude-code/session.jsonl", calibration.CreationAsPatch},
		{"creation as diff", "testdata/codex/compaction.jsonl", calibration.CreationAsDiff},
	}
	for _, c := range cases {
		raw, err := os.ReadFile(c.trace)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(raw, c.rewrite(raw)) {
			t.Errorf("%s: the rewrite changed nothing, so the property it guards is untested", c.name)
		}
	}
}

func totalsOf(t *testing.T, parse func(io.Reader) ([]usage.Record, int, error), raw []byte) calibration.Totals {
	t.Helper()
	recs, skipped, err := parse(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	return calibration.Sum(recs, skipped)
}
