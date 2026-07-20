package codex

import (
	"strings"
	"testing"
)

// TestTokenCountInfoNullDoesNotResetBaseline covers the codex-rs rate-limit update, which
// arrives as {"type":"token_count","info":null}. It must be ignored outright: if it reset
// the cumulative baseline, the next real token_count would emit a delta equal to the whole
// session so far, double-counting every earlier turn.
func TestTokenCountInfoNullDoesNotResetBaseline(t *testing.T) {
	rollout := strings.Join([]string{
		`{"type":"session_meta","payload":{"id":"s1","model":"gpt","cwd":"/p"}}`,
		`{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"cached_input_tokens":200,"output_tokens":300,"reasoning_output_tokens":50}}}}`,
		`{"type":"event_msg","payload":{"type":"token_count","info":null}}`,
		`{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1500,"cached_input_tokens":400,"output_tokens":500,"reasoning_output_tokens":80}}}}`,
	}, "\n")

	recs, _, err := Parse(strings.NewReader(rollout))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2 (the info:null update emits nothing)", len(recs))
	}
	// Second record is the delta from the first's totals, not from a reset-to-zero baseline.
	got := recs[1]
	if got.InputTokens != 300 || got.CacheReadTokens != 200 || got.OutputTokens != 200 || got.ReasoningTokens != 30 {
		t.Fatalf("second record double-counts the session: got in=%d cacheRead=%d out=%d reasoning=%d, want 300/200/200/30",
			got.InputTokens, got.CacheReadTokens, got.OutputTokens, got.ReasoningTokens)
	}
}
