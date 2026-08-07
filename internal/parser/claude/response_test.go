package claude

import (
	"strings"
	"testing"
)

// Claude writes one JSONL line per content block of a single API response. Every line
// repeats that response's usage verbatim, and the output count grows to its true total only
// on the last one -- so a record per line billed one request several times. Measured on
// 5,724 real transcripts before the fix: 354,904 assistant lines were 159,175 responses,
// inflating output tokens 1.97x and cache-write tokens 2.81x.
func TestOneResponseIsCountedOnce(t *testing.T) {
	const transcript = `{"type":"assistant","uuid":"u1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","message":{"id":"msg_1","model":"claude-opus-4-5","content":[{"type":"text"}],"usage":{"input_tokens":8,"output_tokens":2,"cache_creation_input_tokens":2400,"cache_read_input_tokens":100}}}
{"type":"assistant","uuid":"u2","timestamp":"2026-07-01T10:00:01Z","sessionId":"s1","message":{"id":"msg_1","model":"claude-opus-4-5","content":[{"type":"tool_use","name":"Read"}],"usage":{"input_tokens":8,"output_tokens":2,"cache_creation_input_tokens":2400,"cache_read_input_tokens":100}}}
{"type":"assistant","uuid":"u3","timestamp":"2026-07-01T10:00:02Z","sessionId":"s1","message":{"id":"msg_1","model":"claude-opus-4-5","content":[{"type":"tool_use","name":"Edit"}],"usage":{"input_tokens":8,"output_tokens":158,"cache_creation_input_tokens":2400,"cache_read_input_tokens":100}}}`

	recs, skipped, err := Parse(strings.NewReader(transcript))
	if err != nil || skipped != 0 {
		t.Fatalf("Parse = %d skipped, %v", skipped, err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1 -- three lines are one API response", len(recs))
	}
	r := &recs[0]
	// Tokens are the response's, taken once. Output is the largest, not the sum: the early
	// lines carry a partial count.
	if r.InputTokens != 8 {
		t.Errorf("InputTokens = %d, want 8 (counted once, not 24)", r.InputTokens)
	}
	if r.CacheWriteTokens != 2400 {
		t.Errorf("CacheWriteTokens = %d, want 2400 (counted once, not 7200)", r.CacheWriteTokens)
	}
	if r.CacheReadTokens != 100 {
		t.Errorf("CacheReadTokens = %d, want 100 (counted once, not 300)", r.CacheReadTokens)
	}
	if r.OutputTokens != 158 {
		t.Errorf("OutputTokens = %d, want 158 (the final total, not 2 and not 162)", r.OutputTokens)
	}
	// Activity is each line's own and is summed: two blocks made two calls.
	if r.ToolCalls != 2 || r.ToolReads != 1 || r.ToolWrites != 1 {
		t.Errorf("tool activity = %d calls (%d read, %d write), want 2 (1 read, 1 write)",
			r.ToolCalls, r.ToolReads, r.ToolWrites)
	}
	if r.Edits != 1 {
		t.Errorf("Edits = %d, want 1", r.Edits)
	}
	if r.DedupeKey != "msg_1" {
		t.Errorf("DedupeKey = %q, want the response id", r.DedupeKey)
	}
	if r.Timestamp.Format("15:04:05") != "10:00:00" {
		t.Errorf("Timestamp = %s, want the response's first line", r.Timestamp)
	}
}

// Two responses stay two records, and a later tool result attributes to the response that
// made the call rather than to its first block.
func TestSeparateResponsesStaySeparate(t *testing.T) {
	const transcript = `{"type":"assistant","uuid":"u1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","message":{"id":"msg_1","model":"claude-opus-4-5","content":[{"type":"text"}],"usage":{"output_tokens":10,"cache_creation_input_tokens":100}}}
{"type":"assistant","uuid":"u2","timestamp":"2026-07-01T10:00:01Z","sessionId":"s1","message":{"id":"msg_2","model":"claude-opus-4-5","content":[{"type":"tool_use","name":"Edit"}],"usage":{"output_tokens":20,"cache_creation_input_tokens":200}}}
{"type":"user","uuid":"u3","toolUseResult":{"filePath":"/x/a.go","structuredPatch":[{"lines":["+a","+b","-c"]}]}}`

	recs, _, err := Parse(strings.NewReader(transcript))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}
	if recs[0].CacheWriteTokens != 100 || recs[1].CacheWriteTokens != 200 {
		t.Errorf("cache writes = %d, %d; want 100, 200", recs[0].CacheWriteTokens, recs[1].CacheWriteTokens)
	}
	if recs[1].LinesAdded != 2 || recs[1].LinesRemoved != 1 {
		t.Errorf("edit attributed to +%d/-%d on the second response, want +2/-1",
			recs[1].LinesAdded, recs[1].LinesRemoved)
	}
	if recs[0].LinesAdded != 0 {
		t.Errorf("first response gained %d lines it did not write", recs[0].LinesAdded)
	}
}

// A line repeated verbatim is a streamed retry: it must not add its block's activity a
// second time to the response it belongs to.
func TestAVerbatimRepeatOfALineIsDropped(t *testing.T) {
	const line = `{"type":"assistant","uuid":"u1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","message":{"id":"msg_1","model":"claude-opus-4-5","content":[{"type":"tool_use","name":"Read"}],"usage":{"output_tokens":10}}}`
	recs, _, err := Parse(strings.NewReader(line + "\n" + line))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1", len(recs))
	}
	if recs[0].ToolCalls != 1 {
		t.Errorf("ToolCalls = %d, want 1 -- the repeat is the same block, not a second one", recs[0].ToolCalls)
	}
}

// The 8-in-48,440 response whose blocks are not adjacent still belongs to one request, so
// it folds into the record already holding it rather than opening a second one that the
// store would drop on its duplicate key.
func TestANonAdjacentBlockRejoinsItsResponse(t *testing.T) {
	const transcript = `{"type":"assistant","uuid":"u1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","message":{"id":"msg_1","model":"claude-opus-4-5","content":[{"type":"tool_use","name":"Read"}],"usage":{"output_tokens":5,"cache_creation_input_tokens":50}}}
{"type":"assistant","uuid":"u2","timestamp":"2026-07-01T10:00:01Z","sessionId":"s1","message":{"id":"msg_2","model":"claude-opus-4-5","content":[{"type":"text"}],"usage":{"output_tokens":7,"cache_creation_input_tokens":70}}}
{"type":"assistant","uuid":"u3","timestamp":"2026-07-01T10:00:02Z","sessionId":"s1","message":{"id":"msg_1","model":"claude-opus-4-5","content":[{"type":"tool_use","name":"Bash"}],"usage":{"output_tokens":9,"cache_creation_input_tokens":50}}}`

	recs, _, err := Parse(strings.NewReader(transcript))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}
	if recs[0].ToolCalls != 2 || recs[0].ToolCommands != 1 {
		t.Errorf("first response = %d calls (%d command), want 2 (1 command)",
			recs[0].ToolCalls, recs[0].ToolCommands)
	}
	if recs[0].CacheWriteTokens != 50 {
		t.Errorf("CacheWriteTokens = %d, want 50 counted once", recs[0].CacheWriteTokens)
	}
	if recs[0].OutputTokens != 9 {
		t.Errorf("OutputTokens = %d, want 9", recs[0].OutputTokens)
	}
	keys := map[string]bool{}
	for i := range recs {
		if keys[recs[i].DedupeKey] {
			t.Fatalf("duplicate dedupe key %q -- the store would drop one of these",
				recs[i].DedupeKey)
		}
		keys[recs[i].DedupeKey] = true
	}
}

// A transcript predating the message.id field still parses: the line's uuid is the key and
// each line is its own record, which is exactly what it was before.
func TestALineWithoutAResponseIDFallsBackToItsUUID(t *testing.T) {
	const transcript = `{"type":"assistant","uuid":"u1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"output_tokens":10}}}
{"type":"assistant","uuid":"u2","timestamp":"2026-07-01T10:00:01Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"output_tokens":20}}}`
	recs, _, err := Parse(strings.NewReader(transcript))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}
	if recs[0].DedupeKey != "u1" || recs[1].DedupeKey != "u2" {
		t.Errorf("dedupe keys = %q, %q; want the line uuids", recs[0].DedupeKey, recs[1].DedupeKey)
	}
}
