package codex

import (
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/assaio/assaio/internal/usage"
)

// FuzzParseSteps holds the sequence reading to the same invariants the store's boundary
// enforces, so a malformed rollout can produce a shorter sequence but never an
// uninterpretable one.
func FuzzParseSteps(f *testing.F) {
	if b, err := os.ReadFile("testdata/rollout.jsonl"); err == nil {
		f.Add(b)
	}
	// The calibration trace is the only checked-in Codex capture holding a turn, a named call and
	// a multi-file patch in one file, which is the shape every sequence invariant below turns on.
	if b, err := os.ReadFile("../../calibration/testdata/codex/sequence.jsonl"); err == nil {
		f.Add(b)
	}
	f.Add([]byte(""))
	f.Add([]byte("{}"))
	// The union that cost a rollout its session id when source was typed as a string.
	f.Add([]byte(`{"type":"session_meta","payload":{"id":"c1","source":{"subagent":"review"}}}
{"type":"session_meta","payload":{"id":"c1","source":9}}
{"type":"session_meta","payload":{"id":"c1","source":["cli"]}}`))
	// A turn opened and never closed, and one closed twice over.
	f.Add([]byte(`{"type":"response_item","payload":{"type":"reasoning"}}
{"type":"event_msg","payload":{"type":"token_count","info":null}}`))
	f.Add([]byte(`{"type":"response_item","payload":{"type":"message","role":"assistant"}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":9223372036854775807,"cached_input_tokens":9223372036854775807,"output_tokens":9223372036854775807}}}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":-9223372036854775808,"output_tokens":-1}}}}`))
	// Calls with no id, a repeated id, and a status outside the vocabulary.
	f.Add([]byte(`{"type":"response_item","payload":{"type":"custom_tool_call","name":"exec","call_id":"k"}}
{"type":"response_item","payload":{"type":"custom_tool_call","name":"exec","call_id":"k","status":"incomplete"}}
{"type":"response_item","payload":{"type":"function_call","name":"wait","status":"?"}}`))
	// A patch whose map is large enough for iteration order to matter, and one that failed.
	f.Add([]byte(`{"type":"event_msg","payload":{"type":"patch_apply_end","success":true,"changes":{"/d":{"type":"update","unified_diff":"@@\n+a"},"/c":{"type":"add","content":"x\ny"},"/b":{},"/a":9223372036854775807}}}
{"type":"event_msg","payload":{"type":"patch_apply_end","success":false,"changes":{"/d":{"type":"update","unified_diff":"@@\n-a"}}}}`))
	f.Add([]byte("{\"type\":\"event_msg\",\"payload\":{\"type\":\"patch_apply_end\",\"success\":true,\"changes\":{\"\xff\xfe\":{}}}}"))
	f.Add([]byte(`{"type":"compacted","payload":{}}
{"type":"compacted"}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, steps, skipped, err := ParseAll(strings.NewReader(string(data)))
		if err != nil {
			return
		}
		if skipped < 0 {
			t.Fatalf("skipped = %d, want >= 0", skipped)
		}
		targets := map[int64]bool{}
		for i := range steps {
			s := &steps[i]
			if !usage.ValidStepKind(s.Kind) {
				t.Fatalf("kind %q is outside the vocabulary: %+v", s.Kind, s)
			}
			if !usage.ValidStepOutcome(s.Outcome) {
				t.Fatalf("outcome %q is outside the vocabulary: %+v", s.Outcome, s)
			}
			// Ordinals are assigned after the withdrawn turns are dropped, so a sequence that
			// lost one must still number from 1 without a gap for a reader to explain.
			if s.Ordinal != int64(i)+1 {
				t.Fatalf("ordinal %d at index %d breaks the sequence", s.Ordinal, i)
			}
			if s.Tokens < 0 {
				t.Fatalf("negative tokens: %+v", s)
			}
			if s.TargetRef < 0 || s.TargetRef > int64(len(steps)) {
				t.Fatalf("target ref %d outside the %d steps that could have named a file: %+v",
					s.TargetRef, len(steps), s)
			}
			if s.TargetRef != 0 {
				targets[s.TargetRef] = true
			}
			// Only a patched file gives a step a target; a command's input is content this
			// reading never opens, so a target on one would mean a path was read after all.
			if s.TargetRef != 0 && s.Kind != usage.StepEdit {
				t.Fatalf("a %s step names a file: %+v", s.Kind, s)
			}
			if s.Tokens != 0 && s.Kind != usage.StepAssistant {
				t.Fatalf("a %s step carries tokens Codex prices per turn: %+v", s.Kind, s)
			}
			if s.Tool != tool {
				t.Fatalf("Tool = %q, want %q", s.Tool, tool)
			}
			if s.DedupeKey == "" {
				t.Fatalf("DedupeKey empty, so re-reading would append a second copy: %+v", s)
			}
			if !utf8.ValidString(s.DedupeKey) || !utf8.ValidString(s.Model) {
				t.Fatalf("DedupeKey/Model not valid UTF-8: %+v", s)
			}
		}
		// Refs are drawn from first-seen order over distinct paths, so the set they form is
		// 1..n with nothing missing. A gap means the counter moved per call instead of per
		// file, which reads as more distinct files than the sequence ever touched.
		for ref := int64(1); ref <= int64(len(targets)); ref++ {
			if !targets[ref] {
				t.Fatalf("target refs skip %d: %v", ref, targets)
			}
		}
	})
}
