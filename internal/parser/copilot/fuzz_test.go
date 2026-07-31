package copilot

import (
	"os"
	"strings"
	"testing"
)

// FuzzParse holds the parser's invariants on arbitrary bytes: it never panics, never
// reports negative counts, and never emits a record without the identity a dedupe key
// needs.
func FuzzParse(f *testing.F) {
	if seed, err := os.ReadFile("testdata/session.jsonl"); err == nil {
		f.Add(string(seed))
	}
	f.Add(`{"type":"session.shutdown","data":{"modelMetrics":{"m":{"tokenDetails":{"input":{"tokenCount":-5}}}}}}`)
	f.Add(`{"type":"session.start","data":{"sessionId":"s"}}`)
	f.Add("")

	f.Fuzz(func(t *testing.T, in string) {
		recs, skipped, err := Parse(strings.NewReader(in))
		if err != nil {
			return
		}
		if skipped < 0 {
			t.Fatalf("skipped = %d", skipped)
		}
		for i := range recs {
			r := &recs[i]
			if r.InputTokens < 0 || r.OutputTokens < 0 || r.CacheReadTokens < 0 ||
				r.CacheWriteTokens < 0 || r.ReasoningTokens < 0 ||
				r.LinesAdded < 0 || r.LinesRemoved < 0 {
				t.Fatalf("negative count in %+v", r)
			}
			if r.Model == "" || r.DedupeKey == "" {
				t.Fatalf("record without identity: %+v", r)
			}
			if r.Granularity != "session" {
				t.Fatalf("granularity = %q", r.Granularity)
			}
		}
	})
}
