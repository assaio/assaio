package claude

import (
	"strings"
	"testing"
)

// A cache write's 1-hour portion is a subset of the write it belongs to. The vendor
// disagreed with itself on 44 of 355,063 audited turns, so the parser clamps rather than
// trusting the pair -- an over-large portion would make the 5-minute remainder negative and
// price a turn above what it could have cost.
func TestCacheWrite1hIsClampedToItsWrite(t *testing.T) {
	tests := []struct {
		name  string
		usage string
		want  int64
	}{
		{"split reported", `"cache_creation_input_tokens":100,"cache_creation":{"ephemeral_5m_input_tokens":40,"ephemeral_1h_input_tokens":60}`, 60},
		{"no split object", `"cache_creation_input_tokens":100`, 0},
		{"all five-minute", `"cache_creation_input_tokens":100,"cache_creation":{"ephemeral_5m_input_tokens":100,"ephemeral_1h_input_tokens":0}`, 0},
		{"portion exceeds its write", `"cache_creation_input_tokens":10,"cache_creation":{"ephemeral_5m_input_tokens":0,"ephemeral_1h_input_tokens":99}`, 10},
		{"negative portion", `"cache_creation_input_tokens":100,"cache_creation":{"ephemeral_5m_input_tokens":0,"ephemeral_1h_input_tokens":-5}`, 0},
		{"portion of a negative write", `"cache_creation_input_tokens":-100,"cache_creation":{"ephemeral_5m_input_tokens":0,"ephemeral_1h_input_tokens":30}`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := `{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1",` +
				`"message":{"model":"claude-opus-4-5","usage":{` + tt.usage + `}}}`
			recs, skipped, err := Parse(strings.NewReader(line))
			if err != nil || skipped != 0 || len(recs) != 1 {
				t.Fatalf("Parse = %d recs, %d skipped, %v", len(recs), skipped, err)
			}
			if got := recs[0].CacheWrite1hTokens; got != tt.want {
				t.Errorf("CacheWrite1hTokens = %d, want %d", got, tt.want)
			}
			if recs[0].CacheWrite1hTokens > recs[0].CacheWriteTokens {
				t.Errorf("1h portion %d exceeds the write %d it is part of",
					recs[0].CacheWrite1hTokens, recs[0].CacheWriteTokens)
			}
		})
	}
}

// The miss reason is the vendor's own closed vocabulary, copied through as a label. An
// absent diagnostics block is "no reason stated", never a fabricated one.
func TestCacheMissReasonIsCopiedThroughVerbatim(t *testing.T) {
	tests := []struct {
		name, diagnostics, want string
	}{
		{"stated reason", `,"diagnostics":{"cache_miss_reason":{"type":"tools_changed"}}`, "tools_changed"},
		{"no diagnostics", ``, ""},
		{"empty diagnostics", `,"diagnostics":{}`, ""},
		{"reason without a type", `,"diagnostics":{"cache_miss_reason":{}}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := `{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1",` +
				`"message":{"model":"claude-opus-4-5","usage":{"input_tokens":1}` + tt.diagnostics + `}}`
			recs, _, err := Parse(strings.NewReader(line))
			if err != nil || len(recs) != 1 {
				t.Fatalf("Parse = %d recs, %v", len(recs), err)
			}
			if got := recs[0].CacheMissReason; got != tt.want {
				t.Errorf("CacheMissReason = %q, want %q", got, tt.want)
			}
		})
	}
}

// The miss reason is a vendor category label, rendered verbatim and grouped on. A log
// carrying anything but a vocabulary token in that field states no reason, so free text never
// reaches the store -- and a sync push can never be rejected for a value the parser wrote.
func TestCacheMissReasonIsBoundedAtTheParser(t *testing.T) {
	tests := []struct {
		name, raw, want string
	}{
		{"vendor token", "previous_message_not_found", "previous_message_not_found"},
		{"unseen vendor token", "some_future_reason9", "some_future_reason9"},
		{"prose", "the cache missed for reasons", ""},
		{"markup", "<script>x</script>", ""},
		{"uppercase", "Tools_Changed", ""},
		{"overlong", strings.Repeat("a", 65), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := `{"type":"assistant","uuid":"u1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1",` +
				`"message":{"id":"msg_1","model":"claude-opus-4-5","usage":{"input_tokens":1},` +
				`"diagnostics":{"cache_miss_reason":{"type":"` + tt.raw + `"}}}}`
			recs, _, err := Parse(strings.NewReader(line))
			if err != nil || len(recs) != 1 {
				t.Fatalf("Parse = %d recs, %v", len(recs), err)
			}
			if got := recs[0].CacheMissReason; got != tt.want {
				t.Errorf("CacheMissReason = %q, want %q", got, tt.want)
			}
		})
	}
}
