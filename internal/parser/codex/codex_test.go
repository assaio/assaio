package codex

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "update golden files")

func TestParseGolden(t *testing.T) {
	f, err := os.Open("testdata/rollout.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	recs, _, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.MarshalIndent(recs, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')

	const golden = "testdata/rollout.golden"
	if *update {
		if err := os.WriteFile(golden, got, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("golden mismatch:\n got=%s\nwant=%s", got, want)
	}
}

func TestDeltaAndModel(t *testing.T) {
	f, err := os.Open("testdata/rollout.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	recs, _, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records want 2", len(recs))
	}
	if recs[1].InputTokens != 300 || recs[1].CacheReadTokens != 200 ||
		recs[1].OutputTokens != 200 || recs[1].ReasoningTokens != 30 {
		t.Fatalf("delta wrong: %+v", recs[1])
	}
	if recs[0].Model != "gpt-5.1" || recs[1].Model != "gpt-5.1" {
		t.Fatalf("model not carried forward: %+v", recs)
	}
	for _, r := range recs {
		if r.Project != "app" || r.GitBranch != "" || r.Entrypoint != "" || r.Granularity != "turn" {
			t.Fatalf("dimensions = %+v, want Project=app GitBranch=\"\" Entrypoint=\"\" Granularity=turn", r)
		}
		// Cwd is excluded from the golden file (json:"-", never persisted); assert it here instead.
		if r.Cwd != "/home/dev/app" {
			t.Fatalf("Cwd = %q, want /home/dev/app", r.Cwd)
		}
	}
}

func TestNegativeDeltaFieldsAreDroppedNotPropagated(t *testing.T) {
	const rollout = `{"type":"session_meta","payload":{"id":"c1","cwd":"/home/dev/app","timestamp":"2026-07-01T09:00:00Z"}}
{"type":"turn_context","payload":{"model":"gpt-5.1"}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"cached_input_tokens":200,"output_tokens":300,"reasoning_output_tokens":50,"total_tokens":1600}}}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1500,"cached_input_tokens":50,"output_tokens":500,"reasoning_output_tokens":80,"total_tokens":2400}}}}
`
	recs, _, err := Parse(strings.NewReader(rollout))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records want 2", len(recs))
	}
	rec := recs[1]
	if rec.InputTokens < 0 || rec.CacheReadTokens < 0 || rec.OutputTokens < 0 || rec.ReasoningTokens < 0 {
		t.Fatalf("negative token field survived clamp: %+v", rec)
	}
	if rec.CacheReadTokens != 0 {
		t.Fatalf("CacheReadTokens = %d, want 0 (cached dropped 200 -> 50)", rec.CacheReadTokens)
	}
}

// TestCacheDominantTurnUnderCountsInputByDesign documents the honest-clamp tradeoff: when
// a turn's cached delta exceeds its input delta (true input is nonzero but the model's
// cumulative accounting makes the cached share look larger), InputTokens clamps to 0
// rather than going negative, undercounting non-cached input on that turn.
func TestCacheDominantTurnUnderCountsInputByDesign(t *testing.T) {
	const rollout = `{"type":"session_meta","payload":{"id":"c1","cwd":"/home/dev/app","timestamp":"2026-07-01T09:00:00Z"}}
{"type":"turn_context","payload":{"model":"gpt-5.1"}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"cached_input_tokens":100,"output_tokens":300,"reasoning_output_tokens":50,"total_tokens":1600}}}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1050,"cached_input_tokens":180,"output_tokens":400,"reasoning_output_tokens":60,"total_tokens":1800}}}}
`
	recs, _, err := Parse(strings.NewReader(rollout))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records want 2", len(recs))
	}
	rec := recs[1]
	if rec.InputTokens != 0 {
		t.Fatalf("InputTokens = %d, want 0 (documented under-count: delta.input=50 < delta.cached=80, clamped)", rec.InputTokens)
	}
	if rec.CacheReadTokens != 80 {
		t.Fatalf("CacheReadTokens = %d, want 80 (cache delta preserved)", rec.CacheReadTokens)
	}
}

func TestModelFromSessionMeta(t *testing.T) {
	const rollout = `{"type":"session_meta","payload":{"id":"c1","cwd":"/home/dev/app","timestamp":"2026-07-01T09:00:00Z","model":"gpt-5.1"}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"cached_input_tokens":200,"output_tokens":300,"reasoning_output_tokens":50,"total_tokens":1600}}}}
`
	recs, _, err := Parse(strings.NewReader(rollout))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records want 1", len(recs))
	}
	if recs[0].Model != "gpt-5.1" {
		t.Fatalf("Model = %q, want %q", recs[0].Model, "gpt-5.1")
	}
}

func TestParseSkipsMalformedLineAndCountsIt(t *testing.T) {
	const rollout = `{"type":"session_meta","payload":{"id":"c1","cwd":"/home/dev/app","timestamp":"2026-07-01T09:00:00Z","model":"gpt-5.1"}}
{not valid json
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"cached_input_tokens":200,"output_tokens":300,"reasoning_output_tokens":50,"total_tokens":1600}}}}
`
	recs, skipped, err := Parse(strings.NewReader(rollout))
	if err != nil {
		t.Fatalf("a malformed line must not abort the parse: %v", err)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d want 1", skipped)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records want 1 (record around the malformed line still parsed)", len(recs))
	}
}

// TestPerLineTimestampStampsEachRecord covers the fix for records inheriting the
// session-start timestamp: every rollout line carries its own top-level timestamp, and
// each emitted record must be stamped with the timestamp of the line that closed it, not
// session_meta's. The two token_count lines here cross a day boundary, which is exactly
// the case that used to bucket both turns under the same --by day.
func TestPerLineTimestampStampsEachRecord(t *testing.T) {
	const rollout = `{"type":"session_meta","timestamp":"2026-07-01T23:00:00Z","payload":{"id":"c10","cwd":"/home/dev/app10","timestamp":"2026-07-01T23:00:00Z"}}
{"type":"turn_context","timestamp":"2026-07-01T23:00:01Z","payload":{"model":"gpt-5.1"}}
{"type":"event_msg","timestamp":"2026-07-01T23:00:02Z","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":0,"output_tokens":50,"reasoning_output_tokens":0,"total_tokens":150}}}}
{"type":"event_msg","timestamp":"2026-07-02T00:00:05Z","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":200,"cached_input_tokens":0,"output_tokens":90,"reasoning_output_tokens":0,"total_tokens":290}}}}
`
	recs, _, err := Parse(strings.NewReader(rollout))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2: %+v", len(recs), recs)
	}
	want0 := time.Date(2026, 7, 1, 23, 0, 2, 0, time.UTC)
	want1 := time.Date(2026, 7, 2, 0, 0, 5, 0, time.UTC)
	if !recs[0].Timestamp.Equal(want0) {
		t.Fatalf("recs[0].Timestamp = %v, want %v (its own line's timestamp, not session_meta's)", recs[0].Timestamp, want0)
	}
	if !recs[1].Timestamp.Equal(want1) {
		t.Fatalf("recs[1].Timestamp = %v, want %v (crosses midnight from recs[0])", recs[1].Timestamp, want1)
	}
	if recs[0].Timestamp.Equal(recs[1].Timestamp) {
		t.Fatal("both records share one timestamp: the session-start-inherited bug is back")
	}
}

// TestPerLineTimestampFallsBackToLastKnownWhenLineLacksOne covers a line that omits its
// own top-level timestamp (permitted by the fix's spec): the record it closes must fall
// back to the last known timestamp rather than a zero time.
func TestPerLineTimestampFallsBackToLastKnownWhenLineLacksOne(t *testing.T) {
	const rollout = `{"type":"session_meta","timestamp":"2026-07-01T09:00:00Z","payload":{"id":"c11","cwd":"/home/dev/app11","timestamp":"2026-07-01T09:00:00Z"}}
{"type":"event_msg","timestamp":"2026-07-01T10:30:00Z","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":0,"output_tokens":50,"reasoning_output_tokens":0,"total_tokens":150}}}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":200,"cached_input_tokens":0,"output_tokens":90,"reasoning_output_tokens":0,"total_tokens":290}}}}
`
	recs, _, err := Parse(strings.NewReader(rollout))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2: %+v", len(recs), recs)
	}
	want := time.Date(2026, 7, 1, 10, 30, 0, 0, time.UTC)
	if !recs[1].Timestamp.Equal(want) {
		t.Fatalf("recs[1].Timestamp = %v, want %v (line lacks its own timestamp; must fall back to the last known one, not zero)", recs[1].Timestamp, want)
	}
}

// TestSessionMetaEnvelopeTimestampSurvivesEmptyPayloadTimestamp guards the applySessionMeta
// fix: when session_meta carries an envelope timestamp but its payload omits one, a later
// line that also lacks its own timestamp must inherit the envelope's timestamp, not be
// stamped with the zero time (the old code overwrote st.ts with the empty payload value).
func TestSessionMetaEnvelopeTimestampSurvivesEmptyPayloadTimestamp(t *testing.T) {
	const rollout = `{"type":"session_meta","timestamp":"2026-07-01T09:00:00Z","payload":{"id":"c12","cwd":"/home/dev/app12"}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":0,"output_tokens":50,"reasoning_output_tokens":0,"total_tokens":150}}}}
`
	recs, _, err := Parse(strings.NewReader(rollout))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1: %+v", len(recs), recs)
	}
	want := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	if !recs[0].Timestamp.Equal(want) {
		t.Fatalf("recs[0].Timestamp = %v, want %v (envelope ts must survive an empty payload ts)", recs[0].Timestamp, want)
	}
}

// TestDedupeKeyScopedToFileAcrossResumedSession covers the collision this parser used to
// have: two different files (e.g. an original and a resumed-session rollout) that reuse
// the same session id must not produce the same DedupeKey, or the second file's records
// would silently fail to insert (ON CONFLICT(tool, dedupe_key) DO NOTHING).
func TestDedupeKeyScopedToFileAcrossResumedSession(t *testing.T) {
	const rollout1 = `{"type":"session_meta","payload":{"id":"resumed","cwd":"/home/dev/app","timestamp":"2026-07-01T09:00:00Z"}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":0,"output_tokens":50,"reasoning_output_tokens":0,"total_tokens":150}}}}
`
	const rollout2 = `{"type":"session_meta","payload":{"id":"resumed","cwd":"/home/dev/app","timestamp":"2026-07-02T09:00:00Z"}}
{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":0,"output_tokens":50,"reasoning_output_tokens":0,"total_tokens":150}}}}
`
	recs1, _, err := Parse(strings.NewReader(rollout1))
	if err != nil {
		t.Fatal(err)
	}
	recs2, _, err := Parse(strings.NewReader(rollout2))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs1) != 1 || len(recs2) != 1 {
		t.Fatalf("got %d/%d records, want 1/1", len(recs1), len(recs2))
	}
	if recs1[0].DedupeKey == recs2[0].DedupeKey {
		t.Fatalf("DedupeKey collided across two different files sharing session id %q: %q", "resumed", recs1[0].DedupeKey)
	}
}
