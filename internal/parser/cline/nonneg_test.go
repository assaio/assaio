package cline

import (
	"strings"
	"testing"
)

// TestParseTaskClampsNegativeTokens covers the parser contract's NonNeg requirement: a
// well-formed api_req_started entry carrying a negative token count must never reach a
// record with that negative value, which would silently deflate SUM() totals downstream.
func TestParseTaskClampsNegativeTokens(t *testing.T) {
	ui := `[{"type":"say","say":"api_req_started","ts":1700000000000,` +
		`"text":"{\"tokensIn\":-5,\"tokensOut\":10,\"cacheReads\":-3,\"cacheWrites\":2}"}]`

	recs, _, err := ParseTask(strings.NewReader(ui), "task1", taskMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1", len(recs))
	}
	r := recs[0]
	if r.InputTokens != 0 || r.CacheReadTokens != 0 || r.OutputTokens != 10 || r.CacheWriteTokens != 2 {
		t.Fatalf("negative tokens not clamped: %+v", r)
	}
}
