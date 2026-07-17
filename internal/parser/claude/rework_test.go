package claude

import (
	"strings"
	"testing"
)

func TestParseReworkLaterEditToSameFileCounted(t *testing.T) {
	// a1 adds 10 lines to a.ts; a2, later, removes 4 of them. The thrash is attributed
	// to a2 -- the turn that did the reworking removal -- not to a1, which only added.
	const log = `{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":1,"output_tokens":1}}}
{"type":"user","uuid":"e1","timestamp":"2026-07-01T10:00:01Z","sessionId":"s1","toolUseResult":{"filePath":"/repo/a.ts","structuredPatch":[{"lines":["+l1","+l2","+l3","+l4","+l5","+l6","+l7","+l8","+l9","+l10"]}]}}
{"type":"assistant","uuid":"a2","timestamp":"2026-07-01T10:01:00Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":1,"output_tokens":1}}}
{"type":"user","uuid":"e2","timestamp":"2026-07-01T10:01:01Z","sessionId":"s1","toolUseResult":{"filePath":"/repo/a.ts","structuredPatch":[{"lines":["-l1","-l2","-l3","-l4"]}]}}
`
	recs, skipped, err := Parse(strings.NewReader(log))
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2: %+v", len(recs), recs)
	}

	a1, a2 := recs[0], recs[1]
	if a1.LinesAdded != 10 || a1.LinesRemoved != 0 || a1.ReworkLines != 0 {
		t.Fatalf("a1 = %+v, want LinesAdded=10 LinesRemoved=0 ReworkLines=0 (the addition itself is not rework)", a1)
	}
	if a2.LinesAdded != 0 || a2.LinesRemoved != 4 || a2.ReworkLines != 4 {
		t.Fatalf("a2 = %+v, want LinesAdded=0 LinesRemoved=4 ReworkLines=4 (all 4 removed lines undo a1's own additions)", a2)
	}
}

func TestParseReworkRemovalInUntouchedFileIsNotCounted(t *testing.T) {
	// Removing lines from a file AI never added to in this transcript is not rework --
	// it's deleting pre-existing (human or prior-session) code. The cap at addedSoFar
	// must hold even when addedSoFar is zero.
	const log = `{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":1,"output_tokens":1}}}
{"type":"user","uuid":"e1","timestamp":"2026-07-01T10:00:01Z","sessionId":"s1","toolUseResult":{"filePath":"/repo/b.ts","structuredPatch":[{"lines":["-x1","-x2","-x3"]}]}}
`
	recs, _, err := Parse(strings.NewReader(log))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1: %+v", len(recs), recs)
	}
	if recs[0].LinesRemoved != 3 {
		t.Fatalf("LinesRemoved = %d, want 3 (the removal itself is still counted)", recs[0].LinesRemoved)
	}
	if recs[0].ReworkLines != 0 {
		t.Fatalf("ReworkLines = %d, want 0 (nothing was previously AI-added to b.ts in this transcript)", recs[0].ReworkLines)
	}
}

func TestParseReworkInterleavedFilesTrackedIndependently(t *testing.T) {
	// a1 adds 6 lines to a.ts and 3 to b.ts. a2 then removes 2 from a.ts (all rework,
	// under the a.ts cap of 6) and 5 from b.ts (only 3 count as rework, capped at what
	// was actually added to b.ts -- the other 2 removed b.ts lines are not AI's own).
	// Each file's running total must not leak into the other's cap.
	const log = `{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":1,"output_tokens":1}}}
{"type":"user","uuid":"e1","timestamp":"2026-07-01T10:00:01Z","sessionId":"s1","toolUseResult":{"filePath":"/repo/a.ts","structuredPatch":[{"lines":["+a1","+a2","+a3","+a4","+a5","+a6"]}]}}
{"type":"user","uuid":"e2","timestamp":"2026-07-01T10:00:02Z","sessionId":"s1","toolUseResult":{"filePath":"/repo/b.ts","structuredPatch":[{"lines":["+b1","+b2","+b3"]}]}}
{"type":"assistant","uuid":"a2","timestamp":"2026-07-01T10:01:00Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":1,"output_tokens":1}}}
{"type":"user","uuid":"e3","timestamp":"2026-07-01T10:01:01Z","sessionId":"s1","toolUseResult":{"filePath":"/repo/a.ts","structuredPatch":[{"lines":["-a1","-a2"]}]}}
{"type":"user","uuid":"e4","timestamp":"2026-07-01T10:01:02Z","sessionId":"s1","toolUseResult":{"filePath":"/repo/b.ts","structuredPatch":[{"lines":["-b1","-b2","-b3","-b4","-b5"]}]}}
`
	recs, _, err := Parse(strings.NewReader(log))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2: %+v", len(recs), recs)
	}

	a1, a2 := recs[0], recs[1]
	if a1.LinesAdded != 9 || a1.ReworkLines != 0 {
		t.Fatalf("a1 = %+v, want LinesAdded=9 (6+3 across both files) ReworkLines=0", a1)
	}
	if a2.LinesRemoved != 7 {
		t.Fatalf("a2.LinesRemoved = %d, want 7 (2 from a.ts + 5 from b.ts)", a2.LinesRemoved)
	}
	if a2.ReworkLines != 5 {
		t.Fatalf("a2.ReworkLines = %d, want 5 (2 capped at a.ts's 6 + 3 capped at b.ts's 3, independently)", a2.ReworkLines)
	}
}
