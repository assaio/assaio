package codex

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestParseActivityFieldsPerTurn pins the exact per-turn activity numbers the shared
// golden fixture (testdata/rollout.jsonl) produces, so a golden diff alone can't hide a
// wrong-but-stable regression in the activity fields.
func TestParseActivityFieldsPerTurn(t *testing.T) {
	f, err := os.Open("testdata/rollout.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	recs, skipped, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2: %+v", len(recs), recs)
	}

	turn1, turn2 := recs[0], recs[1]
	if turn1.LinesAdded != 5 || turn1.LinesRemoved != 0 || turn1.Edits != 1 || turn1.ToolCalls != 2 || turn1.Compactions != 0 || turn1.ReworkLines != 0 {
		t.Fatalf("turn1 = %+v, want LinesAdded=5 LinesRemoved=0 Edits=1 ToolCalls=2 Compactions=0 ReworkLines=0", turn1)
	}
	// turn2's patch_apply_end re-edits a.ts (5 added in turn1): +3/-2 there plus +5/-1 on
	// the new b.ts = +8/-3 total; 2 of a.ts's 2 removed lines undo turn1's own additions.
	if turn2.LinesAdded != 8 || turn2.LinesRemoved != 3 || turn2.Edits != 1 || turn2.ToolCalls != 1 || turn2.Compactions != 1 || turn2.ReworkLines != 2 {
		t.Fatalf("turn2 = %+v, want LinesAdded=8 LinesRemoved=3 Edits=1 ToolCalls=1 Compactions=1 ReworkLines=2", turn2)
	}
}

func TestParsePatchApplyEndFailureIgnored(t *testing.T) {
	const rollout = `{"type":"session_meta","payload":{"id":"c2","cwd":"/home/dev/app2","timestamp":"2026-07-01T09:00:00Z"}}
{"type":"turn_context","payload":{"model":"gpt-5.1"}}
{"type":"event_msg","payload":{"type":"patch_apply_end","success":false,"changes":{"/repo/x.ts":{"type":"update","unified_diff":"@@ -1,3 +1,4 @@\n-a\n-b\n-c\n+d\n+e\n+f\n+g"}}}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":0,"output_tokens":50,"reasoning_output_tokens":0,"total_tokens":150}}}}
`
	recs, _, err := Parse(strings.NewReader(rollout))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1: %+v", len(recs), recs)
	}
	if r := recs[0]; r.LinesAdded != 0 || r.LinesRemoved != 0 || r.Edits != 0 {
		t.Fatalf("recs[0] = %+v, want LinesAdded=0 LinesRemoved=0 Edits=0 (success:false must contribute nothing)", r)
	}
}

func TestParseToolCallOutputsNotDoubleCounted(t *testing.T) {
	const rollout = `{"type":"session_meta","payload":{"id":"c3","cwd":"/home/dev/app3","timestamp":"2026-07-01T09:00:00Z"}}
{"type":"turn_context","payload":{"model":"gpt-5.1"}}
{"type":"response_item","payload":{"type":"function_call","id":"fc1","name":"wait","call_id":"call_1"}}
{"type":"response_item","payload":{"type":"function_call_output","call_id":"call_1","output":"done"}}
{"type":"response_item","payload":{"type":"custom_tool_call","id":"ctc1","status":"completed","call_id":"call_2","name":"exec"}}
{"type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"call_2","output":[]}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":0,"output_tokens":50,"reasoning_output_tokens":0,"total_tokens":150}}}}
`
	recs, _, err := Parse(strings.NewReader(rollout))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1: %+v", len(recs), recs)
	}
	if recs[0].ToolCalls != 2 {
		t.Fatalf("ToolCalls = %d, want 2 (one function_call + one custom_tool_call; their _output lines must not add to this)", recs[0].ToolCalls)
	}
}

// TestParseTrailingActivityAfterLastTokenCountAttributedToLastRecord covers the flush
// this parser does at end-of-input: activity with no later token_count to close it must
// still land on the last emitted record, not be dropped.
func TestParseTrailingActivityAfterLastTokenCountAttributedToLastRecord(t *testing.T) {
	const rollout = `{"type":"session_meta","payload":{"id":"c4","cwd":"/home/dev/app4","timestamp":"2026-07-01T09:00:00Z"}}
{"type":"turn_context","payload":{"model":"gpt-5.1"}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":0,"output_tokens":50,"reasoning_output_tokens":0,"total_tokens":150}}}}
{"type":"event_msg","payload":{"type":"patch_apply_end","success":true,"changes":{"/repo/trailing.ts":{"type":"update","unified_diff":"@@ -1,1 +1,3 @@\n ctx\n+t1\n+t2"}}}}
`
	recs, _, err := Parse(strings.NewReader(rollout))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1: %+v", len(recs), recs)
	}
	if r := recs[0]; r.LinesAdded != 2 || r.Edits != 1 {
		t.Fatalf("recs[0] = %+v, want LinesAdded=2 Edits=1 (trailing patch after the only token_count must still attribute)", r)
	}
}

// TestParseReworkAccumulatesBeforeFirstRecord covers rework tracking across two
// patch_apply_end events that both occur before any token_count has fired, so no record
// exists yet to attribute to -- the rework math must still hold once one finally does.
func TestParseReworkAccumulatesBeforeFirstRecord(t *testing.T) {
	const rollout = `{"type":"session_meta","payload":{"id":"c5","cwd":"/home/dev/app5","timestamp":"2026-07-01T09:00:00Z"}}
{"type":"turn_context","payload":{"model":"gpt-5.1"}}
{"type":"event_msg","payload":{"type":"patch_apply_end","success":true,"changes":{"/repo/y.ts":{"type":"update","unified_diff":"@@ -1,1 +1,7 @@\n ctx\n+y1\n+y2\n+y3\n+y4\n+y5\n+y6"}}}}
{"type":"event_msg","payload":{"type":"patch_apply_end","success":true,"changes":{"/repo/y.ts":{"type":"update","unified_diff":"@@ -1,7 +1,5 @@\n ctx\n-y1\n-y2\n y3\n y4\n y5"}}}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":0,"output_tokens":50,"reasoning_output_tokens":0,"total_tokens":150}}}}
`
	recs, _, err := Parse(strings.NewReader(rollout))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1: %+v", len(recs), recs)
	}
	r := recs[0]
	if r.LinesAdded != 6 || r.LinesRemoved != 2 || r.Edits != 2 || r.ReworkLines != 2 {
		t.Fatalf("recs[0] = %+v, want LinesAdded=6 LinesRemoved=2 Edits=2 ReworkLines=2", r)
	}
}

// TestParseDiffLineCountsSkipsUnifiedDiffHeaders covers a unified diff that includes its
// "--- a/file" / "+++ b/file" file headers (not just the bare hunk Codex's other test
// fixtures happen to use): those headers start with the same "-"/"+" markers as removed/
// added body lines, but must never be counted as such.
func TestParseDiffLineCountsSkipsUnifiedDiffHeaders(t *testing.T) {
	const rollout = `{"type":"session_meta","payload":{"id":"c12","cwd":"/home/dev/app12","timestamp":"2026-07-01T09:00:00Z"}}
{"type":"turn_context","payload":{"model":"gpt-5.1"}}
{"type":"event_msg","payload":{"type":"patch_apply_end","success":true,"changes":{"/repo/z.ts":{"type":"update","unified_diff":"--- a/z.ts\n+++ b/z.ts\n@@ -1,2 +1,3 @@\n ctx\n-old\n+new\n+extra"}}}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":0,"output_tokens":50,"reasoning_output_tokens":0,"total_tokens":150}}}}
`
	recs, _, err := Parse(strings.NewReader(rollout))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1: %+v", len(recs), recs)
	}
	if r := recs[0]; r.LinesAdded != 2 || r.LinesRemoved != 1 {
		t.Fatalf("recs[0] = %+v, want LinesAdded=2 LinesRemoved=1 (the --- /+++ file headers must not count as body lines)", r)
	}
}

// TestParseNoFilePathLeaksIntoRecords is a belt-and-suspenders privacy check alongside
// the golden inspection: the changes map's file paths must never reach the marshaled
// output, even though they drive rework tracking internally.
func TestParseNoFilePathLeaksIntoRecords(t *testing.T) {
	f, err := os.Open("testdata/rollout.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	recs, _, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	blob, err := json.Marshal(recs)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/repo/a.ts", "/repo/b.ts", "/repo/c.ts", "/repo/"} {
		if strings.Contains(string(blob), path) {
			t.Fatalf("marshaled records leak file path %q: %s", path, blob)
		}
	}
}
