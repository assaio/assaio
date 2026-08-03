package copilot

import (
	"os"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/usage"
)

func parseFixture(t *testing.T) []usage.Record {
	t.Helper()
	f, err := os.Open("testdata/session.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	recs, _, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	return recs
}

func byModel(recs []usage.Record) map[string]usage.Record {
	out := make(map[string]usage.Record, len(recs))
	for i := range recs {
		out[recs[i].Model] = recs[i]
	}
	return out
}

func TestParseEmitsOneRecordPerModel(t *testing.T) {
	recs := parseFixture(t)
	if len(recs) != 2 {
		t.Fatalf("want one record per model in the session, got %d", len(recs))
	}
	got := byModel(recs)
	if _, ok := got["claude-sonnet-5"]; !ok {
		t.Fatalf("models = %v", got)
	}
}

// TestInputTokensExcludeCacheWrites is the mapping that decides whether cost is right.
// Copilot's usage.inputTokens is the whole prompt including tokens written to cache
// (2520 = 100 + 2400 here), so storing it as InputTokens beside CacheWriteTokens would
// count those tokens twice. tokenDetails carries the split, and that is what we read.
func TestInputTokensExcludeCacheWrites(t *testing.T) {
	got := byModel(parseFixture(t))["claude-sonnet-5"]
	if got.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100 (the uncached share, not the 2520 total)", got.InputTokens)
	}
	if got.CacheWriteTokens != 2400 {
		t.Errorf("CacheWriteTokens = %d, want 2400", got.CacheWriteTokens)
	}
	if got.CacheReadTokens != 800 {
		t.Errorf("CacheReadTokens = %d, want 800", got.CacheReadTokens)
	}
	if got.OutputTokens != 40 {
		t.Errorf("OutputTokens = %d, want 40", got.OutputTokens)
	}
	if got.ReasoningTokens != 10 {
		t.Errorf("ReasoningTokens = %d, want 10", got.ReasoningTokens)
	}
}

// TestRecordsAreSessionGranularity keeps Copilot out of per-turn figures: its complete
// accounting only exists in the shutdown event, so one record covers a whole session.
func TestRecordsAreSessionGranularity(t *testing.T) {
	recs := parseFixture(t)
	for i := range recs {
		if r := &recs[i]; r.Granularity != "session" {
			t.Fatalf("Granularity = %q, want session", r.Granularity)
		}
	}
}

// TestLineChangesAreNotSplitAcrossModels guards against inventing a number: Copilot reports
// code changes once per session, with no per-model breakdown, so they are attributed whole
// to the model that made the most requests rather than divided by a ratio nobody measured.
func TestLineChangesAreNotSplitAcrossModels(t *testing.T) {
	got := byModel(parseFixture(t))
	if got["claude-sonnet-5"].LinesAdded != 25 || got["claude-sonnet-5"].LinesRemoved != 7 {
		t.Errorf("dominant model = +%d-%d, want +25-7",
			got["claude-sonnet-5"].LinesAdded, got["claude-sonnet-5"].LinesRemoved)
	}
	if got["gpt-5"].LinesAdded != 0 || got["gpt-5"].LinesRemoved != 0 {
		t.Errorf("secondary model = +%d-%d, want 0 rather than a share of the session total",
			got["gpt-5"].LinesAdded, got["gpt-5"].LinesRemoved)
	}
}

func TestRecordCarriesSessionIdentityAndCwd(t *testing.T) {
	got := byModel(parseFixture(t))["claude-sonnet-5"]
	if got.SessionID != "11111111-2222-3333-4444-555555555555" {
		t.Errorf("SessionID = %q", got.SessionID)
	}
	if got.Cwd != "/home/dev/app" {
		t.Errorf("Cwd = %q, want the session's working directory for project resolution", got.Cwd)
	}
	if got.Tool != "copilot-cli" {
		t.Errorf("Tool = %q, want copilot-cli", got.Tool)
	}
	if got.Timestamp.IsZero() {
		t.Error("Timestamp must anchor the session")
	}
}

// TestDedupeKeysAreStablePerModel is what makes a re-import idempotent.
func TestDedupeKeysAreStablePerModel(t *testing.T) {
	seen := map[string]bool{}
	recs := parseFixture(t)
	for i := range recs {
		r := &recs[i]
		if seen[r.DedupeKey] {
			t.Fatalf("duplicate dedupe key %q", r.DedupeKey)
		}
		if !strings.Contains(r.DedupeKey, r.Model) {
			t.Fatalf("dedupe key %q must distinguish the model", r.DedupeKey)
		}
		seen[r.DedupeKey] = true
	}
}

// TestSessionWithoutShutdownYieldsNothing covers a session still running or killed: the
// totals only exist at shutdown, so there is nothing to report yet -- and reporting zeros
// would be worse than reporting nothing.
func TestSessionWithoutShutdownYieldsNothing(t *testing.T) {
	recs, skipped, err := Parse(strings.NewReader(
		`{"type":"session.start","data":{"sessionId":"s1","context":{"cwd":"/x"}},"timestamp":"2026-07-31T11:00:00Z"}` + "\n",
	))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 0 || skipped != 0 {
		t.Fatalf("recs=%d skipped=%d, want a live session to yield nothing", len(recs), skipped)
	}
}

// The dedupe key is "<session>:<model>", so a session whose id never appears would key every
// such session identically and let ON CONFLICT DO NOTHING drop all but the first. Yielding
// nothing is the honest outcome: the usage happened, but nothing can identify it.
func TestSessionWithoutAnIdYieldsNothing(t *testing.T) {
	recs, skipped, err := Parse(strings.NewReader(
		`{"type":"session.shutdown","data":{"modelMetrics":{"m":{"requests":{"count":1},"tokenDetails":{"input":{"tokenCount":10}}}}},"timestamp":"2026-07-31T11:00:00Z"}` + "\n",
	))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 0 {
		t.Fatalf("recs = %+v, want nothing from a session with no id", recs)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d, want the unidentifiable session counted as skipped", skipped)
	}
}

// A record stamped with the zero time lands in every window that reaches back far enough and
// in none that does not; an unstamped session is dropped rather than dated to year one.
func TestSessionWithoutATimestampYieldsNothing(t *testing.T) {
	recs, skipped, err := Parse(strings.NewReader(
		`{"type":"session.shutdown","data":{"sessionId":"s1","modelMetrics":{"m":{"requests":{"count":1},"tokenDetails":{"input":{"tokenCount":10}}}}}}` + "\n",
	))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 0 {
		t.Fatalf("recs = %+v, want nothing from a session with no timestamp", recs)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d, want the undatable session counted as skipped", skipped)
	}
}

func TestCorruptLinesAreSkippedAndCounted(t *testing.T) {
	_, skipped, err := Parse(strings.NewReader("{not json\n" + `{"type":"session.start","data":{}}` + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d, want 1", skipped)
	}
}
