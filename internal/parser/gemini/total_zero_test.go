package gemini

import (
	"strings"
	"testing"
)

// TestMessageWithTokensButZeroTotalIsCounted covers a Gemini message that carries real
// input/output tokens but a missing or zero tokens.total: it must still produce a record,
// not vanish. Only a message with no usage at all (all counters zero) is skipped.
func TestMessageWithTokensButZeroTotalIsCounted(t *testing.T) {
	log := strings.Join([]string{
		`{"sessionId":"s1","model":"gemini-2.5-pro","tokens":{"input":120,"output":45,"total":0}}`,
		`{"sessionId":"s1","model":"gemini-2.5-pro","tokens":{"input":0,"output":0,"total":0}}`,
	}, "\n")

	recs, _, err := Parse(strings.NewReader(log))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1 (the tokened message kept, the empty one skipped)", len(recs))
	}
	if recs[0].InputTokens != 120 || recs[0].OutputTokens != 45 {
		t.Fatalf("unexpected tokens: %+v", recs[0])
	}
}
