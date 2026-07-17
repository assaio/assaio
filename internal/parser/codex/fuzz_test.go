package codex

import (
	"os"
	"strings"
	"testing"
)

func FuzzParse(f *testing.F) {
	if seed, err := os.ReadFile("testdata/rollout.jsonl"); err == nil {
		f.Add(seed)
	}
	f.Add([]byte(""))
	f.Add([]byte("\n"))
	f.Add([]byte("{}"))
	f.Add([]byte(`{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":9223372036854775807,"cached_input_tokens":9223372036854775807,"output_tokens":9223372036854775807,"reasoning_output_tokens":9223372036854775807}}}}`))
	f.Add([]byte(`{"type":"session_meta","payload":{"id":"c1","cwd"`))
	f.Add([]byte("{\"type\":\"session_meta\",\"payload\":{\"id\":\xff\xfe}}"))
	f.Add([]byte(`{"type":"event_msg","payload":{"type":"patch_apply_end","success":true,"changes":{"/a":{"unified_diff":"+x\n-y"},"/b":9223372036854775807}}}
{"type":"response_item","payload":{"type":"function_call"}}
{"type":"response_item","payload":{"type":"custom_tool_call_output"}}
{"type":"compacted","payload":{}}`))
	f.Add([]byte(`{"type":"session_meta","timestamp":"2026-07-01T23:59:00Z","payload":{"id":"c9","cwd":"/home/dev/app9","timestamp":"2026-07-01T23:59:00Z"}}
{"type":"event_msg","timestamp":"2026-07-01T23:59:30Z","payload":{"type":"patch_apply_end","success":true,"changes":{"/x":{"unified_diff":"--- a/x\n+++ b/x\n@@ -1,1 +1,2 @@\n-old\n+new\n+extra"}}}}
{"type":"event_msg","timestamp":"2026-07-02T00:00:00Z","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":0,"output_tokens":50,"reasoning_output_tokens":0,"total_tokens":150}}}}`))
	f.Add([]byte(`{"type":"session_meta","timestamp":"not-a-timestamp","payload":{"id":"c13"}}`))

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
			if r.LinesAdded < 0 || r.LinesRemoved < 0 || r.Edits < 0 || r.ToolCalls < 0 ||
				r.Compactions < 0 || r.ReworkLines < 0 {
				t.Fatalf("negative activity field: %+v", r)
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
