package claude

import (
	"os"
	"strings"
	"testing"
	"unicode/utf8"
)

func FuzzParse(f *testing.F) {
	if seed, err := os.ReadFile("testdata/session.jsonl"); err == nil {
		f.Add(seed)
	}
	f.Add([]byte(""))
	f.Add([]byte("\n"))
	f.Add([]byte("{}"))
	f.Add([]byte(`{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:05Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":9223372036854775807,"output_tokens":9223372036854775807,"cache_creation_input_tokens":9223372036854775807,"cache_read_input_tokens":9223372036854775807}}}`))
	f.Add([]byte(`{"type":"assistant","uuid":"a2","timestamp":"2026-07-01T10:00:05Z","sessionId"`))
	f.Add([]byte("{\"type\":\"assistant\",\"uuid\":\"a3\",\"message\":{\"model\":\xff\xfe}}"))
	f.Add([]byte(`{"type":"user","uuid":"sa1","toolUseResult":{"agentId":"ag1","resolvedModel":"claude-haiku-4-5","usage":{"input_tokens":9223372036854775807,"output_tokens":9223372036854775807},"toolStats":{"linesAdded":9223372036854775807,"linesRemoved":9223372036854775807}}}`))
	f.Add([]byte(`{"type":"user","uuid":"sa-stub","toolUseResult":{"agentId":"ag1"}}`))
	f.Add([]byte(`{"type":"user","uuid":"tr1","toolUseResult":{"structuredPatch":[{"lines":["+a","-b"," c"]}]}}`))
	f.Add([]byte(`{"type":"user","uuid":"tr2","toolUseResult":{"filePath":"/x","structuredPatch":[{"lines":["+a","-b"," c"]}]}}
{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:05Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":1,"output_tokens":1}}}
{"type":"user","uuid":"tr3","toolUseResult":{"filePath":"/x","structuredPatch":[{"lines":["+a","+b"]}]}}
{"type":"user","uuid":"tr4","toolUseResult":{"filePath":"/x","structuredPatch":[{"lines":["-a","-b","-c"]}]}}`))
	f.Add([]byte(`{"type":"user","uuid":"d1","toolDenialKind":"user-rejected"}`))
	f.Add([]byte(`{"type":"user","uuid":"u1","cwd":"/x","message":{"role":"user","content":"plain text"}}`))
	f.Add([]byte(`{"type":"system","uuid":"cb1","subtype":"compact_boundary"}`))
	f.Add([]byte(`{"type":"user","uuid":"cs1","isCompactSummary":true}`))

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
			if r.LinesAdded < 0 || r.LinesRemoved < 0 || r.Edits < 0 || r.ToolCalls < 0 || r.Rejected < 0 || r.Compactions < 0 {
				t.Fatalf("negative activity field: %+v", r)
			}
			if r.ReworkLines < 0 {
				t.Fatalf("ReworkLines = %d, want >= 0: %+v", r.ReworkLines, r)
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
			if !utf8.ValidString(r.DedupeKey) {
				t.Fatalf("DedupeKey not valid UTF-8: %+v", r)
			}
		}
	})
}
