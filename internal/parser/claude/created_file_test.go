package claude

import (
	"strings"
	"testing"
)

const createAssistant = `{"type":"assistant","uuid":"a1","sessionId":"s1","timestamp":"2026-03-02T09:00:00Z","cwd":"/w/app","message":{"id":"m1","model":"claude-opus-5","usage":{"input_tokens":10,"output_tokens":5},"content":[{"type":"tool_use","name":"Write"}]}}`

// TestCreatedFileContributesItsLines: Claude Code describes a file it created as the whole
// body in `content` beside an empty structuredPatch, so a reader that only walks the patch
// counted every created file as zero added lines. Measured across 5,602 real transcripts:
// 5,599 creations carrying 625,012 lines the corpus never counted.
func TestCreatedFileContributesItsLines(t *testing.T) {
	const created = `{"type":"user","uuid":"u1","sessionId":"s1","timestamp":"2026-03-02T09:00:01Z","toolUseResult":{"type":"create","filePath":"/w/app/x.go","content":"package main\n\nfunc main() {}\n","structuredPatch":[]}}`

	recs, _, err := Parse(strings.NewReader(createAssistant + "\n" + created + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1", len(recs))
	}
	if recs[0].LinesAdded != 3 {
		t.Errorf("LinesAdded = %d, want 3 -- the created file's body is its added lines", recs[0].LinesAdded)
	}
	if recs[0].LinesRemoved != 0 {
		t.Errorf("LinesRemoved = %d, want 0", recs[0].LinesRemoved)
	}
}

// TestCreationShapeDoesNotChangeTheCount is the metamorphic property: the same creation
// written as a patch and as a body must produce the same line count. Either encoding read
// alone looks correct; only the pair shows a reader that counts one and not the other.
func TestCreationShapeDoesNotChangeTheCount(t *testing.T) {
	shapes := map[string]string{
		"as content": `{"type":"user","uuid":"u1","sessionId":"s1","timestamp":"2026-03-02T09:00:01Z","toolUseResult":{"type":"create","filePath":"/w/app/x.go","content":"one\ntwo\nthree\n","structuredPatch":[]}}`,
		"as patch":   `{"type":"user","uuid":"u1","sessionId":"s1","timestamp":"2026-03-02T09:00:01Z","toolUseResult":{"filePath":"/w/app/x.go","structuredPatch":[{"lines":["+one","+two","+three"]}]}}`,
	}
	counts := make(map[string]int64, len(shapes))
	for name, line := range shapes {
		recs, _, err := Parse(strings.NewReader(createAssistant + "\n" + line + "\n"))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(recs) != 1 {
			t.Fatalf("%s: got %d records, want 1", name, len(recs))
		}
		counts[name] = recs[0].LinesAdded
	}
	if counts["as content"] != counts["as patch"] {
		t.Errorf("as content = %d, as patch = %d -- the encoding must not change the count",
			counts["as content"], counts["as patch"])
	}
}

// TestCreatedFileFeedsTheReworkBudget: a file born with no counted additions enters rework
// tracking with nothing to undo, so a later removal in it is charged against a budget of
// zero and the churn rate reads as if the lines had never been written.
func TestCreatedFileFeedsTheReworkBudget(t *testing.T) {
	const created = `{"type":"user","uuid":"u1","sessionId":"s1","timestamp":"2026-03-02T09:00:01Z","toolUseResult":{"type":"create","filePath":"/w/app/x.go","content":"one\ntwo\nthree\n","structuredPatch":[]}}`
	const undone = `{"type":"user","uuid":"u2","sessionId":"s1","timestamp":"2026-03-02T09:00:02Z","toolUseResult":{"filePath":"/w/app/x.go","structuredPatch":[{"lines":["-one","-two"]}]}}`

	recs, _, err := Parse(strings.NewReader(strings.Join([]string{createAssistant, created, undone}, "\n") + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if recs[0].ReworkLines != 2 {
		t.Errorf("ReworkLines = %d, want 2 -- the creation's lines are what the removal undoes", recs[0].ReworkLines)
	}
}

// TestOneCompactionIsCountedOnce: Claude Code writes a single context overflow as two
// adjacent lines -- a system boundary and the user-side summary -- so a counter that fires
// per marker reports every compaction twice. Measured on the maintainer's corpus: 19 real
// events, 38 counted.
func TestOneCompactionIsCountedOnce(t *testing.T) {
	const boundary = `{"type":"system","uuid":"c1","sessionId":"s1","timestamp":"2026-03-02T09:00:02Z","subtype":"compact_boundary"}`
	const summary = `{"type":"user","uuid":"c2","sessionId":"s1","timestamp":"2026-03-02T09:00:03Z","isCompactSummary":true}`
	const later = `{"type":"assistant","uuid":"a2","sessionId":"s1","timestamp":"2026-03-02T09:00:04Z","message":{"id":"m2","model":"claude-opus-5","usage":{"input_tokens":1,"output_tokens":1},"content":[]}}`

	recs, _, err := Parse(strings.NewReader(strings.Join([]string{createAssistant, boundary, summary}, "\n") + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if recs[0].Compactions != 1 {
		t.Errorf("Compactions = %d, want 1 -- the boundary and its summary are one overflow", recs[0].Compactions)
	}

	second := strings.NewReplacer(`"c1"`, `"c3"`, `"c2"`, `"c4"`)
	twice := strings.Join([]string{
		createAssistant, boundary, summary, later,
		second.Replace(boundary), second.Replace(summary),
	}, "\n") + "\n"
	recs, _, err = Parse(strings.NewReader(twice))
	if err != nil {
		t.Fatal(err)
	}
	var total int64
	for i := range recs {
		total += recs[i].Compactions
	}
	if total != 2 {
		t.Errorf("Compactions = %d, want 2 -- two overflows separated by a turn are two events", total)
	}
}
