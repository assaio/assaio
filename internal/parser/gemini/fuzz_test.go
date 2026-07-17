package gemini

import (
	"os"
	"strings"
	"testing"
)

func FuzzParse(f *testing.F) {
	if seed, err := os.ReadFile("testdata/session.jsonl"); err == nil {
		f.Add(seed)
	}
	f.Add([]byte(""))
	f.Add([]byte("\n"))
	f.Add([]byte("{}"))
	f.Add([]byte(`{"sessionId":"g1","timestamp":"2026-07-01T10:00:00Z","model":"gemini-2.5-pro","tokens":{"input":9223372036854775807,"output":9223372036854775807,"cached":9223372036854775807,"thoughts":9223372036854775807,"tool":9223372036854775807,"total":9223372036854775807}}`))
	f.Add([]byte(`{"sessionId":"g2","timestamp":"2026-07-01T10:00:00Z","model":"gemini-2.5-flash","tokens":{"input":500,"output":100,"cached":0,"thoughts":75,"tool":0,"total":675}}`))
	f.Add([]byte(`{"sessionId":"g1","timestamp":"2026-07-01T10:00:00Z","model"`))
	f.Add([]byte("{\"sessionId\":\xff\xfe,\"tokens\":{}}"))

	f.Fuzz(func(t *testing.T, data []byte) {
		recs, skipped, err := Parse(strings.NewReader(string(data)))
		if err != nil {
			return
		}
		if skipped < 0 {
			t.Fatalf("skipped = %d, want >= 0", skipped)
		}
		for i := range recs {
			r := &recs[i]
			if r.InputTokens < 0 || r.OutputTokens < 0 || r.CacheReadTokens < 0 ||
				r.CacheWriteTokens < 0 || r.ReasoningTokens < 0 {
				t.Fatalf("negative token field: %+v", r)
			}
			if r.Tool != tool {
				t.Fatalf("Tool = %q, want %q", r.Tool, tool)
			}
			if r.DedupeKey == "" {
				t.Fatalf("DedupeKey empty: %+v", r)
			}
			if r.Granularity != "turn" {
				t.Fatalf("Granularity = %q, want turn", r.Granularity)
			}
		}
	})
}
