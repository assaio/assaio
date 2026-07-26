package claude

import (
	"os"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/parser"
)

func toolUseTurn(blocks string) string {
	return `{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","message":{"model":"claude-opus-4-5","content":[` +
		blocks + `],"usage":{"input_tokens":1,"output_tokens":1}}}`
}

func TestParseSplitsToolCallsByPurpose(t *testing.T) {
	cases := []struct {
		name      string
		blocks    string
		want      parser.ToolCounts
		wantEdits int64
	}{
		{"reads", `{"type":"tool_use","name":"Read"},{"type":"tool_use","name":"WebFetch"}`, parser.ToolCounts{Reads: 2}, 0},
		{"searches", `{"type":"tool_use","name":"Grep"},{"type":"tool_use","name":"WebSearch"}`, parser.ToolCounts{Searches: 2}, 0},
		{"commands", `{"type":"tool_use","name":"Bash"}`, parser.ToolCounts{Commands: 1}, 0},
		{"writes", `{"type":"tool_use","name":"Edit"},{"type":"tool_use","name":"Write"}`, parser.ToolCounts{Writes: 2}, 2},
		// StructuredOutput returns a payload to the caller and writes no file; counting it
		// as production inflated the produce share of any turn that answered structurally.
		{"structured output is not a write", `{"type":"tool_use","name":"StructuredOutput"}`, parser.ToolCounts{Other: 1}, 0},
		{"unknown and mcp tools are other", `{"type":"tool_use","name":"mcp__srv__tool"},{"type":"tool_use","name":"NotATool"},{"type":"tool_use","name":"TodoWrite"}`, parser.ToolCounts{Other: 3}, 0},
		{"mixed", `{"type":"tool_use","name":"Read"},{"type":"tool_use","name":"Glob"},{"type":"tool_use","name":"Bash"},{"type":"tool_use","name":"MultiEdit"},{"type":"tool_use","name":"Agent"}`, parser.ToolCounts{Reads: 1, Searches: 1, Commands: 1, Writes: 1, Other: 1}, 1},
		{"non-tool blocks count for nothing", `{"type":"text","text":"hi"},{"type":"thinking","thinking":"hm"}`, parser.ToolCounts{}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			recs, _, err := Parse(strings.NewReader(toolUseTurn(c.blocks)))
			if err != nil {
				t.Fatal(err)
			}
			if len(recs) != 1 {
				t.Fatalf("got %d records, want 1: %+v", len(recs), recs)
			}
			r := recs[0]
			got := parser.ToolCounts{Reads: r.ToolReads, Searches: r.ToolSearches, Commands: r.ToolCommands, Writes: r.ToolWrites, Other: r.ToolOther}
			if got != c.want {
				t.Fatalf("split = %+v, want %+v", got, c.want)
			}
			if got.Total() != r.ToolCalls {
				t.Fatalf("split sums to %d but ToolCalls = %d", got.Total(), r.ToolCalls)
			}
			if r.Edits != c.wantEdits {
				t.Fatalf("Edits = %d, want %d", r.Edits, c.wantEdits)
			}
		})
	}
}

func TestParseGoldenSplitSumsToToolCalls(t *testing.T) {
	f, err := os.Open("testdata/session.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	recs, _, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range recs {
		sum := r.ToolReads + r.ToolSearches + r.ToolCommands + r.ToolWrites + r.ToolOther
		if sum != r.ToolCalls {
			t.Fatalf("%s: purpose split sums to %d, want ToolCalls=%d: %+v", r.DedupeKey, sum, r.ToolCalls, r)
		}
	}
}

func TestParseCountsFailedToolResultsOnTheCallingTurn(t *testing.T) {
	const log = `{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","message":{"model":"claude-opus-4-5","content":[{"type":"tool_use","name":"Bash"},{"type":"tool_use","name":"Read"}],"usage":{"input_tokens":1,"output_tokens":1}}}
{"type":"user","uuid":"tr1","timestamp":"2026-07-01T10:00:01Z","sessionId":"s1","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","is_error":true},{"type":"tool_result","tool_use_id":"t2","is_error":false}]}}
{"type":"user","uuid":"d1","timestamp":"2026-07-01T10:00:02Z","sessionId":"s1","toolDenialKind":"user-rejected","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t3","is_error":true}]}}
`
	recs, skipped, err := Parse(strings.NewReader(log))
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1: %+v", len(recs), recs)
	}
	if recs[0].ToolErrors != 1 {
		t.Fatalf("ToolErrors = %d, want 1 (only the is_error result counts, and never the denied call)", recs[0].ToolErrors)
	}
	if recs[0].Rejected != 1 {
		t.Fatalf("Rejected = %d, want 1", recs[0].Rejected)
	}
}

func TestParseDropsFailedToolResultBeforeAnyTurn(t *testing.T) {
	const log = `{"type":"user","uuid":"tr1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","is_error":true}]}}
{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:01Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":1,"output_tokens":1}}}
`
	recs, _, err := Parse(strings.NewReader(log))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1: %+v", len(recs), recs)
	}
	if recs[0].ToolErrors != 0 {
		t.Fatalf("ToolErrors = %d, want 0 (the failed result preceded any assistant record)", recs[0].ToolErrors)
	}
}
