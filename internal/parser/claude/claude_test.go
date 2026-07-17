package claude

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "update golden files")

func TestParseGolden(t *testing.T) {
	f, err := os.Open("testdata/session.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	recs, _, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.MarshalIndent(recs, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')

	const golden = "testdata/session.golden"
	if *update {
		if err := os.WriteFile(golden, got, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("golden mismatch:\n got=%s\nwant=%s", got, want)
	}
}

func TestParseDedupesAndFilters(t *testing.T) {
	f, err := os.Open("testdata/session.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	recs, _, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 4 {
		t.Fatalf("got %d records, want 4 (3 assistant turns deduped + 1 sub-agent record, non-assistant lines otherwise filtered)", len(recs))
	}
	for _, r := range recs {
		if r.Project != "app" || r.GitBranch != "main" || r.Entrypoint != "cli" || r.Granularity != "turn" {
			t.Fatalf("dimensions = %+v, want Project=app GitBranch=main Entrypoint=cli Granularity=turn", r)
		}
		// Cwd is excluded from the golden file (json:"-", never persisted); assert it here instead.
		if r.Cwd != "/home/dev/app" {
			t.Fatalf("Cwd = %q, want /home/dev/app", r.Cwd)
		}
	}
}

func TestParseSkipsAssistantLineMissingUUID(t *testing.T) {
	const log = `{"type":"assistant","message":{"model":"claude-opus-4-5","usage":{"input_tokens":1,"output_tokens":1}}}
`
	recs, skipped, err := Parse(strings.NewReader(log))
	if err != nil {
		t.Fatalf("a missing uuid must not abort the parse: %v", err)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d want 1", skipped)
	}
	if len(recs) != 0 {
		t.Fatalf("got %d records want 0 (a record with an empty DedupeKey must never be emitted): %+v", len(recs), recs)
	}
}

func TestParseUserLineWithStringContentIsNotSkipped(t *testing.T) {
	// Real transcripts give ordinary user messages a plain-string message.content
	// (only assistant turns use the block-array form); that must not fail the line's
	// unmarshal and drop its other fields (here, cwd for carry-forward).
	const log = `{"type":"user","uuid":"u1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","cwd":"/home/dev/app","message":{"role":"user","content":"plain text message"}}
{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:05Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":10,"output_tokens":5}}}
`
	recs, skipped, err := Parse(strings.NewReader(log))
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0 (a user line with plain-string content is valid, not corrupt)", skipped)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1: %+v", len(recs), recs)
	}
	if recs[0].Project != "app" {
		t.Fatalf("Project = %q, want %q (cwd on the string-content user line must still carry forward)", recs[0].Project, "app")
	}
}

func TestParseSubAgentAndActivity(t *testing.T) {
	data, err := os.ReadFile("testdata/session.jsonl")
	if err != nil {
		t.Fatal(err)
	}

	recs, _, err := Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	assistantIdx, subAgentIdx, agCount := -1, -1, 0
	for i := range recs {
		switch recs[i].DedupeKey {
		case "a3":
			assistantIdx = i
		case "agent:ag1":
			subAgentIdx, agCount = i, agCount+1
		}
	}

	if assistantIdx == -1 {
		t.Fatalf("no record for assistant turn a3: %+v", recs)
	}
	assistant := recs[assistantIdx]
	if assistant.ToolCalls != 2 || assistant.Edits != 1 {
		t.Fatalf("a3 ToolCalls/Edits = %d/%d, want 2/1", assistant.ToolCalls, assistant.Edits)
	}
	if assistant.LinesAdded != 5 || assistant.LinesRemoved != 2 {
		t.Fatalf("a3 LinesAdded/LinesRemoved = %d/%d, want 5/2 (from the following edit-result line)", assistant.LinesAdded, assistant.LinesRemoved)
	}
	if assistant.Rejected != 1 {
		t.Fatalf("a3 Rejected = %d, want 1", assistant.Rejected)
	}
	if assistant.Compactions != 1 {
		t.Fatalf("a3 Compactions = %d, want 1 (from the following compact_boundary line)", assistant.Compactions)
	}

	if agCount != 1 {
		t.Fatalf("records with DedupeKey=agent:ag1 = %d, want 1 (the async-launch stub must not itself emit a record)", agCount)
	}
	subAgent := recs[subAgentIdx]
	if subAgent.Tool != tool {
		t.Fatalf("sub-agent Tool = %q, want %q", subAgent.Tool, tool)
	}
	if subAgent.Model != "claude-haiku-4-5-20251001" {
		t.Fatalf("sub-agent Model = %q, want claude-haiku-4-5-20251001", subAgent.Model)
	}
	if subAgent.InputTokens != 500 || subAgent.OutputTokens != 30 {
		t.Fatalf("sub-agent tokens = in=%d out=%d, want in=500 out=30", subAgent.InputTokens, subAgent.OutputTokens)
	}
	if subAgent.LinesAdded != 12 || subAgent.LinesRemoved != 3 {
		t.Fatalf("sub-agent LinesAdded/LinesRemoved = %d/%d, want 12/3 (from its own toolStats)", subAgent.LinesAdded, subAgent.LinesRemoved)
	}
	if subAgent.Edits != 0 || subAgent.ToolCalls != 0 {
		t.Fatalf("sub-agent Edits/ToolCalls = %d/%d, want 0/0 (its own tool calls are not double-counted)", subAgent.Edits, subAgent.ToolCalls)
	}
	if subAgent.Granularity != "turn" {
		t.Fatalf("sub-agent Granularity = %q, want turn", subAgent.Granularity)
	}

	recs2, _, err := Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	subAgent2Idx := -1
	for i := range recs2 {
		if recs2[i].DedupeKey == "agent:ag1" {
			subAgent2Idx = i
		}
	}
	if subAgent2Idx == -1 {
		t.Fatal("second Parse call produced no agent:ag1 record")
	}
	subAgent2 := recs2[subAgent2Idx]
	if subAgent2.DedupeKey != subAgent.DedupeKey || subAgent2.Model != subAgent.Model ||
		subAgent2.InputTokens != subAgent.InputTokens || subAgent2.OutputTokens != subAgent.OutputTokens ||
		subAgent2.LinesAdded != subAgent.LinesAdded || subAgent2.LinesRemoved != subAgent.LinesRemoved {
		t.Fatalf("sub-agent record not stable across two Parse calls:\n first=%+v\nsecond=%+v", subAgent, subAgent2)
	}
}

func TestParseSkipsMalformedLineAndCountsIt(t *testing.T) {
	const log = `{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:05Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":100,"output_tokens":200}}}
{not valid json
{"type":"assistant","uuid":"a2","timestamp":"2026-07-01T10:01:00Z","sessionId":"s1","message":{"model":"claude-haiku-4-5","usage":{"input_tokens":10,"output_tokens":20}}}
`
	recs, skipped, err := Parse(strings.NewReader(log))
	if err != nil {
		t.Fatalf("a malformed line must not abort the parse: %v", err)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d want 1", skipped)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records want 2 (records around the malformed line still parsed)", len(recs))
	}
}

func TestParseCompactionAttribution(t *testing.T) {
	// Two compaction lines using each of the two ground-truth discriminators
	// (isCompactSummary and subtype=="compact_boundary") both attribute to the same
	// most-recently-emitted assistant record, incrementing rather than overwriting.
	const log = `{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":1,"output_tokens":1}}}
{"type":"user","uuid":"cs1","timestamp":"2026-07-01T10:00:01Z","sessionId":"s1","isCompactSummary":true}
{"type":"system","uuid":"cb1","timestamp":"2026-07-01T10:00:02Z","sessionId":"s1","subtype":"compact_boundary"}
{"type":"assistant","uuid":"a2","timestamp":"2026-07-01T10:00:03Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":1,"output_tokens":1}}}
`
	recs, skipped, err := Parse(strings.NewReader(log))
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2 (compaction lines never emit their own record): %+v", len(recs), recs)
	}
	if recs[0].Compactions != 2 {
		t.Fatalf("a1 Compactions = %d, want 2 (both compaction markers attributed to it)", recs[0].Compactions)
	}
	if recs[1].Compactions != 0 {
		t.Fatalf("a2 Compactions = %d, want 0 (compactions occurred before it was emitted)", recs[1].Compactions)
	}
}

func TestParseCompactionDroppedWithoutAssistant(t *testing.T) {
	// A compaction marker before any assistant turn has been emitted has nothing to
	// attribute to and must be dropped, like an edit or denial in the same position.
	const log = `{"type":"system","uuid":"cb1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","subtype":"compact_boundary"}
{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:01Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":1,"output_tokens":1}}}
`
	recs, _, err := Parse(strings.NewReader(log))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1: %+v", len(recs), recs)
	}
	if recs[0].Compactions != 0 {
		t.Fatalf("a1 Compactions = %d, want 0 (the compaction marker preceded any assistant record)", recs[0].Compactions)
	}
}
