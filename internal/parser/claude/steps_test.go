package claude

import (
	"os"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/usage"
)

// The two passes over a transcript must agree about what is in it. This is the guard on the
// claim that reading records and reading steps stays one reading of the format: if a tool name
// is classified in one pass and not the other, or a response folds in one and splits in the
// other, the totals part company here. Verified on the whole real corpus before being written
// down: 5,707 transcripts, 212,319 tool calls on each side, no file disagreeing.
func TestStepsAndRecordsAgreeOnOneTranscript(t *testing.T) {
	body, err := os.ReadFile("testdata/session.jsonl")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	recs, _, err := Parse(strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	steps, _, err := ParseSteps(strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("ParseSteps: %v", err)
	}

	var fromRecords int64
	for i := range recs {
		fromRecords += recs[i].ToolCalls
	}
	var fromSteps int64
	for i := range steps {
		switch steps[i].Kind {
		case usage.StepAssistant, usage.StepCompaction:
		default:
			fromSteps++
		}
	}
	if fromRecords != fromSteps {
		t.Errorf("tool calls disagree: records say %d, steps say %d", fromRecords, fromSteps)
	}

	// Tokens too, not only calls. The records fold a multi-line response by taking each field's
	// maximum; a step that summed per line and took the maximum of those would agree only while
	// every field stays monotone across the response. Comparing calls alone would never notice.
	recTokens := map[string]int64{}
	for i := range recs {
		r := &recs[i]
		recTokens[r.DedupeKey] = r.InputTokens + r.OutputTokens + r.CacheReadTokens + r.CacheWriteTokens
	}
	for i := range steps {
		if steps[i].Kind != usage.StepAssistant {
			continue
		}
		want, ok := recTokens[steps[i].DedupeKey]
		if !ok {
			continue
		}
		if steps[i].Tokens != want {
			t.Errorf("response %s: step says %d tokens, record says %d",
				steps[i].DedupeKey, steps[i].Tokens, want)
		}
	}
	if len(steps) == 0 {
		t.Fatal("no steps parsed: the fixture would prove nothing")
	}
}

// Claude writes one API response as several lines, each carrying one content block and each
// repeating the response's usage. The records fold those into one row; the steps must still
// see every call. Keying a call by its position within the *line* made three real calls share
// one key and lost 44,111 of them on the maintainer's corpus, and the single-transcript
// agreement test above was too small to notice.
func TestOneResponseSpreadOverLinesKeepsEveryCall(t *testing.T) {
	const transcript = `{"uuid":"l1","sessionId":"s","type":"assistant","message":{"id":"m1","model":"claude-opus-5","usage":{"input_tokens":10,"output_tokens":1},"content":[{"type":"tool_use","id":"t1","name":"Read"}]}}
{"uuid":"l2","sessionId":"s","type":"assistant","message":{"id":"m1","model":"claude-opus-5","usage":{"input_tokens":10,"output_tokens":2},"content":[{"type":"tool_use","id":"t2","name":"Bash"}]}}
{"uuid":"l3","sessionId":"s","type":"assistant","message":{"id":"m1","model":"claude-opus-5","usage":{"input_tokens":10,"output_tokens":3},"content":[{"type":"tool_use","id":"t3","name":"Edit"}]}}`
	recs, _, err := Parse(strings.NewReader(transcript))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	steps, _, err := ParseSteps(strings.NewReader(transcript))
	if err != nil {
		t.Fatalf("ParseSteps: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("records = %d, want 1: the three lines are one response", len(recs))
	}
	keys := map[string]bool{}
	var calls int64
	var assistants int
	for _, s := range steps {
		if keys[s.DedupeKey] {
			t.Errorf("dedupe key %q is used twice: two steps would collapse into one row", s.DedupeKey)
		}
		keys[s.DedupeKey] = true
		switch s.Kind {
		case usage.StepAssistant:
			assistants++
		default:
			calls++
		}
	}
	if assistants != 1 {
		t.Errorf("assistant steps = %d, want 1: one response is one step", assistants)
	}
	if calls != recs[0].ToolCalls {
		t.Errorf("tool steps = %d, but the record counts %d calls", calls, recs[0].ToolCalls)
	}
}

func TestEveryStepIsWellFormed(t *testing.T) {
	body, err := os.ReadFile("testdata/session.jsonl")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	steps, _, err := ParseSteps(strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("ParseSteps: %v", err)
	}
	for i, s := range steps {
		switch {
		case !usage.ValidStepKind(s.Kind):
			t.Errorf("step %d: kind %q is outside the vocabulary", i, s.Kind)
		case !usage.ValidStepOutcome(s.Outcome):
			t.Errorf("step %d: outcome %q is outside the vocabulary", i, s.Outcome)
		case s.Ordinal != int64(i)+1:
			t.Errorf("step %d: ordinal %d breaks the sequence", i, s.Ordinal)
		case s.DedupeKey == "":
			t.Errorf("step %d: no dedupe key, so re-reading would append a second copy", i)
		case s.SessionID == "":
			t.Errorf("step %d: no session, so it belongs to no timeline", i)
		}
	}
}

// A compaction is written as two adjacent markers and is one event. The records collapse the
// pair; if the steps ever stop doing the same, a friction signal doubles.
func TestAdjacentCompactionMarkersAreOneStep(t *testing.T) {
	const transcript = `{"uuid":"a","sessionId":"s","type":"assistant","message":{"id":"m1","model":"claude-opus-5","usage":{"input_tokens":1,"output_tokens":1}}}
{"uuid":"b","sessionId":"s","subtype":"compact_boundary"}
{"uuid":"c","sessionId":"s","isCompactSummary":true}
{"uuid":"d","sessionId":"s","type":"assistant","message":{"id":"m2","model":"claude-opus-5","usage":{"input_tokens":1,"output_tokens":1}}}
{"uuid":"e","sessionId":"s","subtype":"compact_boundary"}`
	steps, _, err := ParseSteps(strings.NewReader(transcript))
	if err != nil {
		t.Fatalf("ParseSteps: %v", err)
	}
	var got int
	for _, s := range steps {
		if s.Kind == usage.StepCompaction {
			got++
		}
	}
	if want := 2; got != want {
		t.Errorf("compaction steps = %d, want %d (one per overflow, not one per marker)", got, want)
	}
}

// A tool result names the call it answers. Attributing an error to the nearest step instead
// would put friction on whichever call happened to be last, which is a wrong number rather
// than a missing one.
//
// The failing call is deliberately NOT the last one: with the error on the final step,
// "attribute to the named call" and "attribute to the most recent call" give the same answer
// and the test passes either way. It did, until a negative control replaced the lookup with
// len(steps)-1 and nothing went red.
func TestOutcomeLandsOnTheCallItAnswers(t *testing.T) {
	const transcript = `{"uuid":"a","sessionId":"s","type":"assistant","message":{"id":"m1","model":"claude-opus-5","usage":{"input_tokens":1,"output_tokens":1},"content":[{"type":"tool_use","id":"t1","name":"Bash"},{"type":"tool_use","id":"t2","name":"Read"}]}}
{"uuid":"b","sessionId":"s","type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t1","is_error":true}]}}`
	steps, _, err := ParseSteps(strings.NewReader(transcript))
	if err != nil {
		t.Fatalf("ParseSteps: %v", err)
	}
	var read, command usage.Step
	for _, s := range steps {
		switch s.Kind {
		case usage.StepRead:
			read = s
		case usage.StepCommand:
			command = s
		}
	}
	if command.Outcome != usage.OutcomeError {
		t.Errorf("the failing call's outcome = %q, want %q", command.Outcome, usage.OutcomeError)
	}
	if read.Outcome != "" {
		t.Errorf("the later, unanswered call's outcome = %q, want it left unstated", read.Outcome)
	}
}

// The store must never receive a path. A target is an integer assigned in first-seen order,
// so repetition stays visible and identity stays unrecoverable.
func TestTargetsAreIntegersAssignedInFirstSeenOrder(t *testing.T) {
	const transcript = `{"uuid":"a","sessionId":"s","type":"assistant","message":{"id":"m1","model":"claude-opus-5","usage":{"input_tokens":1,"output_tokens":1},"content":[{"type":"tool_use","id":"t1","name":"Edit"},{"type":"tool_use","id":"t2","name":"Edit"},{"type":"tool_use","id":"t3","name":"Edit"}]}}
{"uuid":"b","sessionId":"s","type":"user","toolUseResult":{"filePath":"/repo/one.go","structuredPatch":[]},"message":{"content":[{"type":"tool_result","tool_use_id":"t1"}]}}
{"uuid":"c","sessionId":"s","type":"user","toolUseResult":{"filePath":"/repo/two.go","structuredPatch":[]},"message":{"content":[{"type":"tool_result","tool_use_id":"t2"}]}}
{"uuid":"d","sessionId":"s","type":"user","toolUseResult":{"filePath":"/repo/one.go","structuredPatch":[]},"message":{"content":[{"type":"tool_result","tool_use_id":"t3"}]}}`
	steps, _, err := ParseSteps(strings.NewReader(transcript))
	if err != nil {
		t.Fatalf("ParseSteps: %v", err)
	}
	var refs []int64
	for _, s := range steps {
		if s.Kind == usage.StepEdit {
			refs = append(refs, s.TargetRef)
		}
	}
	want := []int64{1, 2, 1}
	if len(refs) != len(want) {
		t.Fatalf("edit steps = %d, want %d", len(refs), len(want))
	}
	for i := range want {
		if refs[i] != want[i] {
			t.Fatalf("target refs = %v, want %v (the third edit revisits the first file)", refs, want)
		}
	}
}

// A sub-agent transcript records its PARENT's sessionId, so the session alone does not identify
// a sequence. Without the timeline separating them, the main loop and every sub-agent it
// launched all numbered from 1 into one key: 127,139 of 334,288 rows in the maintainer's store
// collided, and the worst session had 222 steps at position 1.
func TestASubAgentGetsItsOwnTimelineUnderTheParentSession(t *testing.T) {
	const parent = `{"uuid":"p1","sessionId":"S","type":"assistant","message":{"id":"m1","model":"claude-opus-5","usage":{"input_tokens":1,"output_tokens":1},"content":[{"type":"tool_use","id":"t1","name":"Read"}]}}`
	const sub = `{"uuid":"s1","sessionId":"S","agentId":"ag7","isSidechain":true,"type":"assistant","message":{"id":"m2","model":"claude-opus-5","usage":{"input_tokens":1,"output_tokens":1},"content":[{"type":"tool_use","id":"t2","name":"Bash"}]}}`

	_, parentSteps, _, err := ParseAll(strings.NewReader(parent))
	if err != nil {
		t.Fatalf("ParseAll(parent): %v", err)
	}
	_, subSteps, _, err := ParseAll(strings.NewReader(sub))
	if err != nil {
		t.Fatalf("ParseAll(sub): %v", err)
	}
	if len(parentSteps) == 0 || len(subSteps) == 0 {
		t.Fatal("one of the transcripts produced no steps: the test would prove nothing")
	}
	for _, s := range parentSteps {
		if s.SessionID != "S" || s.Timeline != "" {
			t.Errorf("parent step = (session %q, timeline %q), want (S, \"\")", s.SessionID, s.Timeline)
		}
	}
	for _, s := range subSteps {
		if s.SessionID != "S" {
			t.Errorf("sub-agent step session = %q, want S: the link to the parent is the session", s.SessionID)
		}
		if s.Timeline != "ag7" {
			t.Errorf("sub-agent step timeline = %q, want ag7", s.Timeline)
		}
	}
	// Both sequences legitimately start at ordinal 1; only the timeline keeps them apart.
	seen := map[[2]any]bool{}
	for _, s := range append(append([]usage.Step{}, parentSteps...), subSteps...) {
		k := [2]any{s.Timeline, s.Ordinal}
		if seen[k] {
			t.Errorf("(timeline %q, ordinal %d) occurs twice: the sequences are not separated", s.Timeline, s.Ordinal)
		}
		seen[k] = true
	}
}

// The denial line names the call it declined. Attributing it to the most recent unsettled step
// put 42 of 497 real denials on a different call; 30 of those were then overwritten by that
// call's own result and left the timeline entirely.
func TestDenialLandsOnTheCallTheLineNames(t *testing.T) {
	const transcript = `{"uuid":"a","sessionId":"s","type":"assistant","message":{"id":"m1","model":"claude-opus-5","usage":{"input_tokens":1,"output_tokens":1},"content":[{"type":"tool_use","id":"t1","name":"Bash"},{"type":"tool_use","id":"t2","name":"Read"}]}}
{"uuid":"b","sessionId":"s","toolDenialKind":"user-rejected","message":{"content":[{"type":"tool_result","tool_use_id":"t1"}]}}
{"uuid":"c","sessionId":"s","type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t2"}]}}`
	_, steps, _, err := ParseAll(strings.NewReader(transcript))
	if err != nil {
		t.Fatalf("ParseAll: %v", err)
	}
	var denied, ok string
	for _, s := range steps {
		switch s.Kind {
		case usage.StepCommand:
			denied = s.Outcome
		case usage.StepRead:
			ok = s.Outcome
		}
	}
	if denied != usage.OutcomeDenied {
		t.Errorf("the declined call's outcome = %q, want %q", denied, usage.OutcomeDenied)
	}
	if ok != usage.OutcomeOK {
		t.Errorf("the answered call's outcome = %q, want %q", ok, usage.OutcomeOK)
	}
}
