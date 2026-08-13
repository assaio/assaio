package claude

import (
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/assaio/assaio/internal/usage"
)

// FuzzParseSteps holds the step reading to the same invariants the store's boundary enforces,
// so a malformed transcript can produce a shorter sequence but never an uninterpretable one.
func FuzzParseSteps(f *testing.F) {
	if seed, err := os.ReadFile("testdata/session.jsonl"); err == nil {
		f.Add(seed)
	}
	f.Add([]byte(""))
	f.Add([]byte("{}"))
	f.Add([]byte(`{"type":"assistant","uuid":"a1","sessionId":"s1","message":{"id":"m1","model":"m","usage":{"input_tokens":9223372036854775807,"output_tokens":9223372036854775807,"cache_read_input_tokens":9223372036854775807,"cache_creation_input_tokens":9223372036854775807},"stop_reason":"max_tokens"}}`))
	f.Add([]byte(`{"type":"assistant","uuid":"a2","sessionId":"s1","message":{"id":"m2","model":"m","usage":{"input_tokens":-9223372036854775808,"output_tokens":-1}}}`))
	f.Add([]byte(`{"type":"assistant","uuid":"a3","sessionId":"s1","message":{"id":"m3","model":"m","content":[{"type":"tool_use","id":"t1","name":"Read"},{"type":"tool_use","id":"t1","name":"Bash"},{"type":"tool_use","name":9}],"usage":{}}}
{"type":"user","uuid":"u1","message":{"content":[{"type":"tool_result","tool_use_id":"t1","is_error":true},{"type":"tool_result","tool_use_id":"nope"}]}}`))
	f.Add([]byte(`{"type":"user","uuid":"e1","toolUseResult":{"filePath":"�","structuredPatch":[]},"message":{"content":[{"type":"tool_result","tool_use_id":"t1"}]}}`))
	f.Add([]byte(`{"uuid":"c1","sessionId":"s1","subtype":"compact_boundary"}
{"uuid":"c2","sessionId":"s1","isCompactSummary":true}`))
	f.Add([]byte(`{"uuid":"d1","sessionId":"s1","toolDenialKind":"user-rejected"}`))
	// The shapes a call's own input arrives in. A non-object input must cost only its own target,
	// never the content array it sits in: typed as a struct it would fail that whole unmarshal and
	// drop every block on the line, which is how 477 sub-agent records were once lost.
	f.Add([]byte(`{"type":"assistant","uuid":"i1","sessionId":"s1","cwd":"/repo","message":{"id":"m4","model":"m","content":[{"type":"tool_use","id":"t1","name":"Read","input":"not-an-object"},{"type":"tool_use","id":"t2","name":"Edit","input":{"file_path":9}},{"type":"tool_use","id":"t3","name":"Edit","input":{"file_path":"../../etc/hosts"}},{"type":"tool_use","id":"t4","name":"NotebookEdit","input":{"notebook_path":"/a/b.ipynb"}},{"type":"tool_use","id":"t5","name":"Read","input":{"file_path":""}}],"usage":{}}}`))
	f.Add([]byte(`{"type":"assistant","uuid":"i2","sessionId":"s1","cwd":"�","message":{"id":"m5","model":"m","content":[{"type":"tool_use","id":"t6","name":"Read","input":{"file_path":"�"}}],"usage":{}}}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		steps, skipped, err := ParseSteps(strings.NewReader(string(data)))
		if err != nil {
			return
		}
		if skipped < 0 {
			t.Fatalf("skipped = %d, want >= 0", skipped)
		}
		for i := range steps {
			s := &steps[i]
			if !usage.ValidStepKind(s.Kind) {
				t.Fatalf("kind %q is outside the vocabulary: %+v", s.Kind, s)
			}
			if !usage.ValidStepOutcome(s.Outcome) {
				t.Fatalf("outcome %q is outside the vocabulary: %+v", s.Outcome, s)
			}
			if s.Ordinal != int64(i)+1 {
				t.Fatalf("ordinal %d at index %d breaks the sequence", s.Ordinal, i)
			}
			if s.Tokens < 0 {
				t.Fatalf("negative tokens: %+v", s)
			}
			if s.TargetRef < 0 {
				t.Fatalf("negative target ref: %+v", s)
			}
			// Refs are drawn from first-seen order over distinct paths, so one can never outrun
			// the steps that named them. A ref that does means the counter moved per call
			// instead of per file, which reads as "nine different files" where one was revisited.
			if s.TargetRef > int64(len(steps)) {
				t.Fatalf("target ref %d exceeds the %d steps that could have named a file: %+v",
					s.TargetRef, len(steps), s)
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
	})
}
