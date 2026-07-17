package cline

import (
	"os"
	"strings"
	"testing"
)

func FuzzParseTask(f *testing.F) {
	if seed, err := os.ReadFile("testdata/ui_messages.json"); err == nil {
		f.Add(seed)
	}
	f.Add([]byte(""))
	f.Add([]byte("\n"))
	f.Add([]byte("{}"))
	f.Add([]byte(`[{"type":"say","say":"api_req_started","text":"{\"tokensIn\":9223372036854775807,\"tokensOut\":9223372036854775807,\"cacheWrites\":9223372036854775807,\"cacheReads\":9223372036854775807}","ts":1751360400000}]`))
	f.Add([]byte(`[{"type":"say","say":"api_req_started","text":"{\"tokensIn\":1`))
	f.Add([]byte("[{\"type\":\"say\",\"say\":\"api_req_started\",\"text\":\xff\xfe}]"))

	meta := taskMetadata{ModelUsage: []modelUsageEntry{{TS: 1751360400000, ModelID: "claude-sonnet-4-5"}}}

	f.Fuzz(func(t *testing.T, data []byte) {
		recs, skipped, err := ParseTask(strings.NewReader(string(data)), "fuzz-task", meta)
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
