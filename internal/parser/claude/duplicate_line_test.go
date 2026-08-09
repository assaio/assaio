package claude

import (
	"strings"
	"testing"
)

// TestRepeatedLineIsFoldedInOnce covers a transcript that writes the same line twice -- a
// streamed retry. The guard used to sit inside appendAssistant, so only assistant lines were
// protected by it, while every count that actually lives on a *user* line went in twice: an
// edit result's added, removed and rework lines, a tool denial, a failed tool result, a
// compaction boundary. Measured across 5,575 real transcripts: 329 repeated edit results
// carrying 460 added and 656 removed lines, and 5 repeated denials.
func TestRepeatedLineIsFoldedInOnce(t *testing.T) {
	const assistantLine = `{"type":"assistant","uuid":"a1","sessionId":"s1","timestamp":"2026-08-01T10:00:00Z","cwd":"/w/app","message":{"id":"m1","model":"claude-opus-5","usage":{"input_tokens":10,"output_tokens":5},"content":[{"type":"tool_use","name":"Edit"}]}}`
	const editResult = `{"type":"user","uuid":"u1","sessionId":"s1","timestamp":"2026-08-01T10:00:01Z","toolUseResult":{"filePath":"/w/app/x.go","structuredPatch":[{"lines":["+one","+two","-gone"]}]}}`
	const denial = `{"type":"user","uuid":"u2","sessionId":"s1","timestamp":"2026-08-01T10:00:02Z","toolDenialKind":"tool_use_rejected"}`

	transcript := strings.Join([]string{assistantLine, editResult, editResult, denial, denial}, "\n") + "\n"
	recs, skipped, err := Parse(strings.NewReader(transcript))
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1", len(recs))
	}
	r := recs[0]
	if r.LinesAdded != 2 {
		t.Errorf("LinesAdded = %d, want 2 -- the repeated edit result must not count again", r.LinesAdded)
	}
	if r.LinesRemoved != 1 {
		t.Errorf("LinesRemoved = %d, want 1", r.LinesRemoved)
	}
	if r.ReworkLines != 0 {
		t.Errorf("ReworkLines = %d, want 0 -- a repeated removal must not respend the budget", r.ReworkLines)
	}
	if r.Rejected != 1 {
		t.Errorf("Rejected = %d, want 1", r.Rejected)
	}
}

// TestLinesWithoutUUIDStillCount: there is nothing to recognise such a line by, so the dedupe
// guard must let every one of them through rather than collapsing them onto each other.
func TestLinesWithoutUUIDStillCount(t *testing.T) {
	const assistantLine = `{"type":"assistant","uuid":"a1","sessionId":"s1","timestamp":"2026-08-01T10:00:00Z","message":{"id":"m1","model":"claude-opus-5","usage":{"input_tokens":10,"output_tokens":5},"content":[]}}`
	const editNoUUID = `{"type":"user","sessionId":"s1","timestamp":"2026-08-01T10:00:01Z","toolUseResult":{"filePath":"/w/app/x.go","structuredPatch":[{"lines":["+one"]}]}}`

	transcript := strings.Join([]string{assistantLine, editNoUUID, editNoUUID}, "\n") + "\n"
	recs, _, err := Parse(strings.NewReader(transcript))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1", len(recs))
	}
	if recs[0].LinesAdded != 2 {
		t.Errorf("LinesAdded = %d, want 2 -- two distinct edits that happen to carry no uuid", recs[0].LinesAdded)
	}
}
