package codex

import (
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/usage"
)

// step is the part of a usage.Step a case asserts on, so a table row reads as the sequence it
// describes rather than as a struct literal.
type step struct {
	kind, outcome string
	target        int64
	tokens        int64
}

func got(t *testing.T, lines string) []step {
	t.Helper()
	_, steps, _, err := ParseAll(strings.NewReader(lines))
	if err != nil {
		t.Fatalf("ParseAll() err = %v", err)
	}
	out := make([]step, 0, len(steps))
	for i := range steps {
		if steps[i].Ordinal != int64(i)+1 {
			t.Fatalf("ordinal %d at index %d breaks the sequence", steps[i].Ordinal, i)
		}
		out = append(out, step{steps[i].Kind, steps[i].Outcome, steps[i].TargetRef, steps[i].Tokens})
	}
	return out
}

const (
	meta      = `{"type":"session_meta","timestamp":"2026-03-02T09:00:00Z","payload":{"id":"c1","cwd":"/w/app","source":"cli","timestamp":"2026-03-02T09:00:00Z"}}`
	reasoning = `{"type":"response_item","timestamp":"2026-03-02T09:00:01Z","payload":{"type":"reasoning"}}`
	assistant = `{"type":"response_item","timestamp":"2026-03-02T09:00:02Z","payload":{"type":"message","role":"assistant"}}`
	developer = `{"type":"response_item","timestamp":"2026-03-02T09:00:01Z","payload":{"type":"message","role":"developer"}}`
	user      = `{"type":"response_item","timestamp":"2026-03-02T09:00:01Z","payload":{"type":"message","role":"user"}}`
	call      = `{"type":"response_item","timestamp":"2026-03-02T09:00:03Z","payload":{"type":"custom_tool_call","name":"exec","status":"completed","call_id":"k1"}}`
	failed    = `{"type":"response_item","timestamp":"2026-03-02T09:00:03Z","payload":{"type":"custom_tool_call","name":"exec","status":"failed","call_id":"k2"}}`
	turn1     = `{"type":"event_msg","timestamp":"2026-03-02T09:00:04Z","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":40,"output_tokens":10,"reasoning_output_tokens":2}}}}`
	turn2     = `{"type":"event_msg","timestamp":"2026-03-02T09:00:08Z","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":300,"cached_input_tokens":40,"output_tokens":30,"reasoning_output_tokens":4}}}}`
	rateLimit = `{"type":"event_msg","timestamp":"2026-03-02T09:00:05Z","payload":{"type":"token_count","info":null}}`
	compacted = `{"type":"compacted","timestamp":"2026-03-02T09:00:06Z","payload":{}}`
	twoFiles  = `{"type":"event_msg","timestamp":"2026-03-02T09:00:07Z","payload":{"type":"patch_apply_end","success":true,"changes":{"/w/app/z.go":{"type":"update","unified_diff":"@@ -1,1 +1,1 @@\n+x"},"/w/app/a.go":{"type":"update","unified_diff":"@@ -1,1 +1,1 @@\n+y"}}}}`
	badPatch  = `{"type":"event_msg","timestamp":"2026-03-02T09:00:07Z","payload":{"type":"patch_apply_end","success":false,"changes":{"/w/app/a.go":{"type":"update","unified_diff":"@@ -1,1 +1,1 @@\n+y"}}}}`
	// A build that names its patch call. Unobserved on the audited corpus -- every write there
	// arrives as patch_apply_end -- but the record path carries a guard against exactly this
	// shape, and the golden rollout fixture holds it.
	namedPatch = `{"type":"response_item","timestamp":"2026-03-02T09:00:03Z","payload":{"type":"function_call","name":"apply_patch","status":"completed","call_id":"k3"}}`
)

func TestSequence(t *testing.T) {
	cases := []struct {
		name  string
		lines []string
		want  []step
	}{
		{
			name:  "a turn opens on the model's first output and is priced by its token_count",
			lines: []string{meta, reasoning, call, turn1},
			want: []step{
				{usage.StepAssistant, "", 0, 110},
				{usage.StepCommand, "", 0, 0},
			},
		},
		{
			name:  "the turn's input opens nothing",
			lines: []string{meta, developer, user, reasoning, turn1},
			want:  []step{{usage.StepAssistant, "", 0, 110}},
		},
		{
			name:  "an assistant message inside an open turn is the same response",
			lines: []string{meta, reasoning, assistant, turn1},
			want:  []step{{usage.StepAssistant, "", 0, 110}},
		},
		{
			name:  "a turn that never reports tokens is not a turn the usage table has",
			lines: []string{meta, reasoning, call},
			want:  []step{{usage.StepCommand, "", 0, 0}},
		},
		{
			name:  "a rate-limit update reports no totals and ends no turn",
			lines: []string{meta, reasoning, rateLimit, turn1},
			want:  []step{{usage.StepAssistant, "", 0, 110}},
		},
		{
			name:  "a completed call is not a call that succeeded",
			lines: []string{meta, call, failed, turn1},
			want: []step{
				{usage.StepAssistant, "", 0, 110},
				{usage.StepCommand, "", 0, 0},
				{usage.StepCommand, usage.OutcomeError, 0, 0},
			},
		},
		{
			name:  "one patch, one step per file, in path order",
			lines: []string{meta, twoFiles},
			want: []step{
				{usage.StepEdit, usage.OutcomeOK, 1, 0},
				{usage.StepEdit, usage.OutcomeOK, 2, 0},
			},
		},
		{
			name:  "a patch that did not apply says so",
			lines: []string{meta, badPatch},
			want:  []step{{usage.StepEdit, usage.OutcomeError, 1, 0}},
		},
		{
			name:  "a file patched twice keeps its target across the sequence",
			lines: []string{meta, twoFiles, badPatch},
			want: []step{
				{usage.StepEdit, usage.OutcomeOK, 1, 0},
				{usage.StepEdit, usage.OutcomeOK, 2, 0},
				{usage.StepEdit, usage.OutcomeError, 1, 0},
			},
		},
		{
			name:  "a named patch call and the event reporting it are one application",
			lines: []string{meta, namedPatch, twoFiles, turn1},
			want: []step{
				{usage.StepAssistant, "", 0, 110},
				{usage.StepEdit, "", 0, 0},
			},
		},
		{
			name:  "the suppression ends with the turn that set it",
			lines: []string{meta, namedPatch, turn1, twoFiles},
			want: []step{
				{usage.StepAssistant, "", 0, 110},
				{usage.StepEdit, "", 0, 0},
				{usage.StepEdit, usage.OutcomeOK, 1, 0},
				{usage.StepEdit, usage.OutcomeOK, 2, 0},
			},
		},
		{
			name:  "a compaction is where the context was lost",
			lines: []string{meta, reasoning, turn1, compacted, reasoning, turn2},
			want: []step{
				{usage.StepAssistant, "", 0, 110},
				{usage.StepCompaction, "", 0, 0},
				{usage.StepAssistant, "", 0, 220},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			steps := got(t, strings.Join(tc.lines, "\n")+"\n")
			if len(steps) != len(tc.want) {
				t.Fatalf("got %d step(s) %+v, want %d %+v", len(steps), steps, len(tc.want), tc.want)
			}
			for i := range steps {
				if steps[i] != tc.want[i] {
					t.Errorf("step %d = %+v, want %+v", i+1, steps[i], tc.want[i])
				}
			}
		})
	}
}

// TestEntrypoint pins the field the scope vocabulary reads a sequence under, including the union
// that made a whole session_meta line unreadable when it was typed as a string (see sessionMeta).
func TestEntrypoint(t *testing.T) {
	cases := []struct {
		name, meta, want string
	}{
		{"the terminal UI", `{"type":"session_meta","payload":{"id":"c1","source":"cli"}}`, "cli"},
		{"a scripted run", `{"type":"session_meta","payload":{"id":"c1","source":"exec"}}`, "exec"},
		{"none written", `{"type":"session_meta","payload":{"id":"c1"}}`, ""},
		{"the object form", `{"type":"session_meta","payload":{"id":"c1","source":{"subagent":"review"}}}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recs, _, skipped, err := ParseAll(strings.NewReader(tc.meta + "\n" + turn1 + "\n"))
			if err != nil {
				t.Fatalf("ParseAll() err = %v", err)
			}
			if skipped != 0 {
				t.Fatalf("skipped = %d, want 0: the line is readable whichever form source takes", skipped)
			}
			if len(recs) != 1 {
				t.Fatalf("got %d record(s), want 1", len(recs))
			}
			if recs[0].Entrypoint != tc.want {
				t.Errorf("Entrypoint = %q, want %q", recs[0].Entrypoint, tc.want)
			}
			if recs[0].SessionID != "c1" {
				t.Errorf("SessionID = %q, want c1 -- a field read too confidently must not cost the session its identity", recs[0].SessionID)
			}
		})
	}
}

// TestDedupeKeysAreUniqueAndStable pins the contract two comments in this package state and
// nothing asserted: a key identifies one step at its source, so re-reading a transcript rewrites
// the same row instead of appending a second copy of the session. A repeated key does not error --
// the store's INSERT OR IGNORE drops the loser silently, leaving a hole in the ordinals.
func TestDedupeKeysAreUniqueAndStable(t *testing.T) {
	lines := []string{meta, reasoning, call, turn1, reasoning, failed, twoFiles, turn2, compacted, reasoning, call, turn2}
	full := strings.Join(lines, "\n") + "\n"

	_, steps, _, err := ParseAll(strings.NewReader(full))
	if err != nil {
		t.Fatalf("ParseAll() err = %v", err)
	}
	seen := map[string]bool{}
	for i := range steps {
		if seen[steps[i].DedupeKey] {
			t.Fatalf("dedupe key %q is used twice: two steps would collapse into one row", steps[i].DedupeKey)
		}
		seen[steps[i].DedupeKey] = true
	}

	// A transcript is read while it is still being written, so every prefix must key its steps
	// the way the finished file does. Only the ordinal may move -- a turn open at the cut is
	// withdrawn, and the store restates ordinals on the re-read.
	kind := map[string]string{}
	target := map[string]int64{}
	for i := range steps {
		kind[steps[i].DedupeKey] = steps[i].Kind
		target[steps[i].DedupeKey] = steps[i].TargetRef
	}
	for cut := 1; cut < len(lines); cut++ {
		prefix := strings.Join(lines[:cut], "\n") + "\n"
		_, early, _, err := ParseAll(strings.NewReader(prefix))
		if err != nil {
			t.Fatalf("prefix of %d line(s): err = %v", cut, err)
		}
		for i := range early {
			k := early[i].DedupeKey
			want, ok := kind[k]
			if !ok {
				t.Errorf("prefix of %d line(s): key %q is absent from the finished read", cut, k)
				continue
			}
			if early[i].Kind != want {
				t.Errorf("prefix of %d line(s): key %q is a %s here and a %s in the finished read",
					cut, k, early[i].Kind, want)
			}
			if early[i].TargetRef != target[k] {
				t.Errorf("prefix of %d line(s): key %q targets %d here and %d in the finished read",
					cut, k, early[i].TargetRef, target[k])
			}
		}
	}
}

// TestAssistantStepMatchesItsRecord holds the one invariant that keeps the two readings of a
// single scan from drifting apart by different arithmetic: a turn's step reports the tokens its
// record reports, and there is exactly one such step per priced turn.
func TestAssistantStepMatchesItsRecord(t *testing.T) {
	lines := strings.Join([]string{meta, reasoning, call, turn1, reasoning, turn2, reasoning}, "\n") + "\n"
	recs, steps, _, err := ParseAll(strings.NewReader(lines))
	if err != nil {
		t.Fatalf("ParseAll() err = %v", err)
	}
	var turns []usage.Step
	for i := range steps {
		if steps[i].Kind == usage.StepAssistant {
			turns = append(turns, steps[i])
		}
	}
	if len(turns) != len(recs) {
		t.Fatalf("got %d assistant step(s) for %d record(s): the sequence must hold one per priced turn",
			len(turns), len(recs))
	}
	for i := range recs {
		want := recs[i].InputTokens + recs[i].CacheReadTokens + recs[i].OutputTokens
		if turns[i].Tokens != want {
			t.Errorf("turn %d: step reports %d tokens, its record %d", i+1, turns[i].Tokens, want)
		}
	}
}
