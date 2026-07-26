package codex

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/usage"
)

// TestCodexToolNamesClassify pins Codex's own tool vocabulary against the shared
// allowlist: every name Codex emits must land in the purpose a report would expect, and
// anything unlisted must fall to other rather than be guessed at.
func TestCodexToolNamesClassify(t *testing.T) {
	tests := []struct {
		name  string
		names []string
		want  parser.ToolCounts
	}{
		{"writes", []string{"apply_patch", "str_replace_editor", "edit", "write"}, parser.ToolCounts{Writes: 4}},
		{"commands", []string{"shell", "exec", "bash", "run", "local_shell", "container.exec"}, parser.ToolCounts{Commands: 6}},
		{"searches", []string{"web_search", "grep", "search", "glob"}, parser.ToolCounts{Searches: 4}},
		{"reads", []string{"read", "read_file", "view", "open_page", "fetch"}, parser.ToolCounts{Reads: 5}},
		{"unlisted codex names are other", []string{"wait", "update_plan", "some_future_codex_tool", ""}, parser.ToolCounts{Other: 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got parser.ToolCounts
			for _, n := range tt.names {
				got.Add(n)
			}
			if got != tt.want {
				t.Fatalf("counts = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestFailedStatus(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{"failed", true},
		{"incomplete", true},
		{"completed", false},
		{"in_progress", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := failedStatus(tt.status); got != tt.want {
				t.Fatalf("failedStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

// TestParseToolBucketsSumToToolCalls pins the invariant every metric over the split
// depends on: a turn's five buckets account for exactly its tool calls, no more, no less.
func TestParseToolBucketsSumToToolCalls(t *testing.T) {
	rollouts := map[string]string{
		"golden fixture": readFixture(t),
		"unknown names only": header("s1") +
			toolCallLine("function_call", "definitely_not_a_known_tool", "") +
			toolCallLine("custom_tool_call", "", "completed") +
			tokenCountLine(100, 0, 50),
		"one of each bucket": header("s2") +
			toolCallLine("custom_tool_call", "read_file", "completed") +
			toolCallLine("function_call", "web_search", "") +
			toolCallLine("custom_tool_call", "exec", "completed") +
			toolCallLine("function_call", "apply_patch", "") +
			toolCallLine("function_call", "wait", "") +
			tokenCountLine(100, 0, 50),
		"outputs and non-call items ignored": header("s3") +
			toolCallLine("custom_tool_call", "exec", "completed") +
			`{"type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"x","output":[]}}` + "\n" +
			`{"type":"response_item","payload":{"type":"reasoning","id":"r1"}}` + "\n" +
			tokenCountLine(100, 0, 50),
	}
	for name, rollout := range rollouts {
		t.Run(name, func(t *testing.T) {
			recs, _, err := Parse(strings.NewReader(rollout))
			if err != nil {
				t.Fatal(err)
			}
			if len(recs) == 0 {
				t.Fatal("no records parsed")
			}
			for i := range recs {
				assertBucketsSum(t, &recs[i])
			}
		})
	}
}

func TestParseToolBucketsPerTurn(t *testing.T) {
	recs, _, err := Parse(strings.NewReader(readFixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}
	// turn1: exec + failed exec (commands), read_file, web_search, apply_patch, wait. It also
	// carries one patch_apply_end, which must NOT add a write: the turn already named its
	// patch call, so counting the event too would double-count the same edit.
	turn1 := recs[0]
	if turn1.ToolCommands != 2 || turn1.ToolReads != 1 || turn1.ToolSearches != 1 ||
		turn1.ToolWrites != 1 || turn1.ToolOther != 1 || turn1.ToolCalls != 6 {
		t.Fatalf("turn1 = %+v, want ToolCommands=2 ToolReads=1 ToolSearches=1 ToolWrites=1 ToolOther=1 ToolCalls=6", turn1)
	}
	if turn1.ToolErrors != 1 {
		t.Fatalf("turn1.ToolErrors = %d, want 1 (the status:failed exec)", turn1.ToolErrors)
	}
	// turn2: exec (command) + update_plan (unknown -> other), and two patch_apply_end events
	// (one applied, one failed) that no response_item names. Those are real writes and are
	// the only way a Codex window ever registers one, so they count as write calls -- and the
	// failed one now has a call for its error to be a share of.
	turn2 := recs[1]
	if turn2.ToolCommands != 1 || turn2.ToolOther != 1 || turn2.ToolWrites != 2 ||
		turn2.ToolCalls != 4 || turn2.ToolReads != 0 || turn2.ToolSearches != 0 {
		t.Fatalf("turn2 = %+v, want ToolCommands=1 ToolOther=1 ToolWrites=2 ToolCalls=4 and no reads/searches", turn2)
	}
	if turn2.ToolErrors != 1 {
		t.Fatalf("turn2.ToolErrors = %d, want 1 (the success:false patch_apply_end)", turn2.ToolErrors)
	}
}

func assertBucketsSum(t *testing.T, r *usage.Record) {
	t.Helper()
	sum := r.ToolReads + r.ToolSearches + r.ToolCommands + r.ToolWrites + r.ToolOther
	if sum != r.ToolCalls {
		t.Fatalf("buckets sum to %d, want ToolCalls=%d: %+v", sum, r.ToolCalls, r)
	}
}

func readFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/rollout.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func header(session string) string {
	return fmt.Sprintf(`{"type":"session_meta","payload":{"id":%q,"cwd":"/home/dev/app","timestamp":"2026-07-01T09:00:00Z"}}`, session) + "\n" +
		`{"type":"turn_context","payload":{"model":"gpt-5.1"}}` + "\n"
}

func toolCallLine(kind, name, status string) string {
	return fmt.Sprintf(`{"type":"response_item","payload":{"type":%q,"name":%q,"status":%q}}`, kind, name, status) + "\n"
}

func tokenCountLine(input, cached, output int64) string {
	return fmt.Sprintf(`{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":%d,"cached_input_tokens":%d,"output_tokens":%d,"reasoning_output_tokens":0}}}}`, input, cached, output) + "\n"
}
