package agy

import (
	"os"
	"strings"
	"testing"
)

// FuzzParseTranscript holds the parser's invariants on arbitrary bytes. The conversation id is
// fuzzed alongside the transcript because it is the one input that does not come from the file:
// it is the directory name ParseDir reads, and it goes straight into every dedupe key.
func FuzzParseTranscript(f *testing.F) {
	for _, fixture := range []string{"testdata/session.jsonl", "testdata/interrupted.jsonl"} {
		if seed, err := os.ReadFile(fixture); err == nil { //nolint:gosec // a fixture path this file lists
			f.Add(string(seed), "conv-1")
		}
	}
	f.Add(`{"step_index":-9223372036854775808,"source":"MODEL","created_at":"2026-09-01T12:00:00Z"}`, "conv-1")
	f.Add(`{"step_index":1,"source":"MODEL","created_at":"2026-09-01T12:00:00Z","tool_calls":[{"name":"\xff"}]}`, "")
	f.Add(`{"step_index":1,"source":"MODEL"}`, "conv-1")
	f.Add("{}", "conv-1")
	f.Add("", "")

	f.Fuzz(func(t *testing.T, in, conversationID string) {
		recs, skipped, err := ParseTranscript(strings.NewReader(in), conversationID)
		if err != nil {
			return
		}
		if skipped < 0 {
			t.Fatalf("skipped = %d", skipped)
		}
		seen := make(map[string]bool, len(recs))
		for i := range recs {
			r := &recs[i]
			if r.InputTokens|r.OutputTokens|r.CacheReadTokens|r.CacheWriteTokens|
				r.CacheWrite1hTokens|r.ReasoningTokens != 0 {
				t.Fatalf("a source with no token counter produced one: %+v", r)
			}
			if r.ToolCalls < 0 || r.Edits < 0 || r.ToolReads < 0 || r.ToolSearches < 0 ||
				r.ToolCommands < 0 || r.ToolWrites < 0 || r.ToolOther < 0 {
				t.Fatalf("negative count in %+v", r)
			}
			// The class split is the same tool calls counted a second way, so a disagreement
			// means a name was counted twice or lost between the two.
			if got := r.ToolReads + r.ToolSearches + r.ToolCommands + r.ToolWrites + r.ToolOther; got != r.ToolCalls {
				t.Fatalf("class split %d does not sum to ToolCalls %d: %+v", got, r.ToolCalls, r)
			}
			if r.Edits > r.ToolCalls {
				t.Fatalf("edits %d exceed the tool calls they are part of: %+v", r.Edits, r)
			}
			if r.Tool != tool || r.SessionID == "" || r.DedupeKey == "" || r.Timestamp.IsZero() {
				t.Fatalf("record without identity: %+v", r)
			}
			if r.Granularity != "turn" {
				t.Fatalf("granularity = %q", r.Granularity)
			}
			if seen[r.DedupeKey] {
				t.Fatalf("duplicate dedupe key %q within one transcript", r.DedupeKey)
			}
			seen[r.DedupeKey] = true
		}
	})
}
