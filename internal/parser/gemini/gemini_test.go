package gemini

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/pricing"
)

var update = flag.Bool("update", false, "update golden files")

func TestParseGolden(t *testing.T) {
	f, err := os.Open("testdata/session.jsonl")
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

	const golden = "testdata/session.golden"
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

func TestTokenMappingAndPerMessageModel(t *testing.T) {
	f, err := os.Open("testdata/session.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	recs, _, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 3 {
		t.Fatalf("got %d records want 3", len(recs))
	}

	first := recs[0]
	if first.Model != "gemini-2.5-pro" {
		t.Fatalf("Model = %q, want gemini-2.5-pro", first.Model)
	}
	if first.InputTokens != 800 || first.CacheReadTokens != 200 ||
		first.OutputTokens != 370 || first.ReasoningTokens != 50 {
		t.Fatalf("token mapping wrong (OutputTokens must include the 50 thinking tokens): %+v", first)
	}
	if first.CacheWriteTokens != 0 || first.Granularity != "turn" || first.Tool != tool {
		t.Fatalf("dimensions wrong: %+v", first)
	}

	if recs[1].Model != "gemini-2.5-flash" {
		t.Fatalf("Model = %q, want gemini-2.5-flash (per-message switch)", recs[1].Model)
	}
	if recs[2].Model != "gemini-2.5-pro" {
		t.Fatalf("Model = %q, want gemini-2.5-pro (switched back)", recs[2].Model)
	}

	third := recs[2]
	if third.InputTokens != 400 || third.CacheReadTokens != 400 ||
		third.OutputTokens != 190 || third.ReasoningTokens != 30 {
		t.Fatalf("token mapping wrong (OutputTokens must include the 30 thinking tokens): %+v", third)
	}
}

func TestDedupeKeyDeterministic(t *testing.T) {
	f, err := os.Open("testdata/session.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	first, _, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}

	f2, err := os.Open("testdata/session.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f2.Close() }()

	second, _, err := Parse(f2)
	if err != nil {
		t.Fatal(err)
	}

	if len(first) != len(second) {
		t.Fatalf("record count differs across re-parse: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].DedupeKey != second[i].DedupeKey {
			t.Fatalf("DedupeKey not deterministic at %d: %q vs %q", i, first[i].DedupeKey, second[i].DedupeKey)
		}
	}
	wantSuffix := []string{":g1:0", ":g1:1", ":g1:2"}
	for i, suffix := range wantSuffix {
		if !strings.HasSuffix(first[i].DedupeKey, suffix) {
			t.Fatalf("DedupeKey[%d] = %q, want suffix %q", i, first[i].DedupeKey, suffix)
		}
	}
	// All records from the same file must share the same fingerprint prefix.
	fp := strings.TrimSuffix(first[0].DedupeKey, wantSuffix[0])
	for i, suffix := range wantSuffix {
		if strings.TrimSuffix(first[i].DedupeKey, suffix) != fp {
			t.Fatalf("DedupeKey[%d] = %q, fingerprint prefix differs within one file", i, first[i].DedupeKey)
		}
	}
}

// TestDedupeKeyScopedToFileAcrossResumedSession covers the collision this parser used to
// have: two different files (e.g. an original and a resumed-session log) that reuse the
// same session id must not produce the same DedupeKey, or the second file's records would
// silently fail to insert (ON CONFLICT(tool, dedupe_key) DO NOTHING).
func TestDedupeKeyScopedToFileAcrossResumedSession(t *testing.T) {
	const log1 = `{"sessionId":"resumed","timestamp":"2026-07-01T10:00:00Z","model":"gemini-2.5-pro","tokens":{"input":100,"output":50,"cached":0,"thoughts":0,"tool":0,"total":150}}
`
	const log2 = `{"sessionId":"resumed","timestamp":"2026-07-02T10:00:00Z","model":"gemini-2.5-pro","tokens":{"input":100,"output":50,"cached":0,"thoughts":0,"tool":0,"total":150}}
`
	recs1, _, err := Parse(strings.NewReader(log1))
	if err != nil {
		t.Fatal(err)
	}
	recs2, _, err := Parse(strings.NewReader(log2))
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

func TestSkipsMessagesWithoutTokens(t *testing.T) {
	const log = `{"sessionId":"g1","timestamp":"2026-07-01T10:00:00Z","model":"gemini-2.5-pro","tokens":{"input":0,"output":0,"cached":0,"thoughts":0,"tool":0,"total":0}}
{"sessionId":"g1","timestamp":"2026-07-01T10:00:05Z","model":"gemini-2.5-pro","tokens":{"input":100,"output":50,"cached":0,"thoughts":0,"tool":0,"total":150}}
`
	recs, _, err := Parse(strings.NewReader(log))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records want 1", len(recs))
	}
	if !strings.HasSuffix(recs[0].DedupeKey, ":g1:0") {
		t.Fatalf("DedupeKey = %q, want suffix :g1:0 (index counts emitted records only)", recs[0].DedupeKey)
	}
}

func TestParseSkipsMalformedLineAndCountsIt(t *testing.T) {
	const log = `{"sessionId":"g1","timestamp":"2026-07-01T10:00:00Z","model":"gemini-2.5-pro","tokens":{"input":100,"output":50,"cached":0,"thoughts":0,"tool":0,"total":150}}
{not valid json
{"sessionId":"g1","timestamp":"2026-07-01T10:00:05Z","model":"gemini-2.5-pro","tokens":{"input":200,"output":80,"cached":0,"thoughts":0,"tool":0,"total":280}}
`
	recs, skipped, err := Parse(strings.NewReader(log))
	if err != nil {
		t.Fatalf("a malformed line must not abort the parse: %v", err)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d want 1", skipped)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records want 2 (records around the malformed line still parsed)", len(recs))
	}
}

// TestThinkingTokensContributeToCost proves the fix end to end: before it, a Gemini
// record's thinking (thoughts) tokens sat only in ReasoningTokens, which pricing.Cost
// never reads, so they were silently unpriced. They must now be folded into OutputTokens
// and billed at the output rate, like Codex's reasoning tokens already are.
func TestThinkingTokensContributeToCost(t *testing.T) {
	f, err := os.Open("testdata/session.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	recs, _, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	// recs[0]: gemini-2.5-pro, output=300 tool=20 thoughts=50 -> OutputTokens=370.
	rec := recs[0]
	if rec.OutputTokens != 370 {
		t.Fatalf("OutputTokens = %d, want 370 (300 output + 20 tool + 50 thoughts)", rec.OutputTokens)
	}
	if rec.ReasoningTokens != 50 {
		t.Fatalf("ReasoningTokens = %d, want 50 (kept as an informational subset)", rec.ReasoningTokens)
	}

	const prices = `{"gemini-2.5-pro":{"input_cost_per_token":1.25e-06,"output_cost_per_token":1e-05,"cache_read_input_token_cost":1.25e-07}}`
	tbl, err := pricing.LoadReader(strings.NewReader(prices))
	if err != nil {
		t.Fatal(err)
	}

	got, ok := tbl.Cost(&rec)
	if !ok {
		t.Fatal("expected priced")
	}
	// The old (buggy) cost excluded the 50 thinking tokens from OutputTokens (320 vs 370).
	oldCost := float64(rec.InputTokens)*1.25e-6 + 320*1e-5 + float64(rec.CacheReadTokens)*1.25e-7
	want := oldCost + 50*1e-5
	if diff := got - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("Cost = %.10f want %.10f", got, want)
	}
	if got <= oldCost {
		t.Fatalf("Cost = %.10f, want > %.10f (old formula undercounted by ignoring thinking tokens)", got, oldCost)
	}
}

// TestHeaderSessionIDCarriesForwardToMessages covers the real Gemini recording shape: the
// session id rides the file's header line only, message lines omit it, and $set/$rewindTo
// control records carry no tokens. Every emitted record must inherit the header's session
// id, and the control records must be skipped without being counted as parse failures.
func TestHeaderSessionIDCarriesForwardToMessages(t *testing.T) {
	const log = `{"sessionId":"hdr-1","projectHash":"abc","startTime":"2026-07-01T10:00:00Z","kind":"main"}
{"$set":{"summary":"a summary update"}}
{"id":"m1","type":"gemini","timestamp":"2026-07-01T10:00:05Z","model":"gemini-2.5-pro","tokens":{"input":100,"output":50,"cached":0,"thoughts":0,"tool":0,"total":150}}
{"$rewindTo":"m0"}
{"id":"m2","type":"gemini","timestamp":"2026-07-01T10:00:10Z","model":"gemini-2.5-flash","tokens":{"input":200,"output":80,"cached":0,"thoughts":0,"tool":0,"total":280}}
`
	recs, skipped, err := Parse(strings.NewReader(log))
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0 (header and control records are valid JSON, just token-less)", skipped)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2 (only the two token-bearing message lines)", len(recs))
	}
	for i, rec := range recs {
		if rec.SessionID != "hdr-1" {
			t.Fatalf("recs[%d].SessionID = %q, want the header session id 'hdr-1'", i, rec.SessionID)
		}
		if !strings.Contains(rec.DedupeKey, ":hdr-1:") {
			t.Fatalf("recs[%d].DedupeKey = %q, want the header session id in the key", i, rec.DedupeKey)
		}
	}
}
