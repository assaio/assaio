package plugin

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// FuzzParseRecord asserts parseRecordLineAt's invariants over arbitrary lines: it never panics,
// and every accepted record is namespaced, identified, in range and internally consistent.
func FuzzParseRecord(f *testing.F) {
	for _, doc := range catalogueSeeds(f, "parser-record") {
		f.Add(doc)
	}
	f.Add(`{"session_id":"s1","timestamp":"2026-08-12T09:00:00Z","model":"m","output_tokens":1,"dedupe_key":"k","granularity":"turn"} {}`)
	f.Add(`{"session_id":"s1","timestamp":"2026-08-12T09:00:00Z","output_tokens":null,"dedupe_key":"k","granularity":"turn"}`)
	f.Add("")

	now := time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC)
	f.Fuzz(func(t *testing.T, line string) {
		rec, err := parseRecordLineAt([]byte(line), "demo", now)
		if err != nil {
			return
		}
		if !json.Valid(bytes.TrimSpace([]byte(line))) {
			t.Fatal("accepted a line that is not exactly one JSON value")
		}
		if err := usage.CheckTimestamp(rec.Timestamp, now); err != nil {
			t.Fatalf("accepted an out-of-range timestamp: %v", err)
		}
		for _, v := range []string{rec.SessionID, rec.Model, rec.DedupeKey, rec.Project, rec.GitBranch, rec.Entrypoint} {
			if len(v) > maxWireStringLen {
				t.Fatalf("accepted a %d-byte string field", len(v))
			}
		}
		if rec.Tool != "plugin:demo" {
			t.Fatalf("accepted record has Tool %q, want the stamped namespace", rec.Tool)
		}
		if rec.SessionID == "" || rec.DedupeKey == "" {
			t.Fatal("accepted a record without its identity")
		}
		if rec.Granularity != "turn" && rec.Granularity != "session" {
			t.Fatalf("accepted granularity %q", rec.Granularity)
		}
		for _, n := range []int64{rec.InputTokens, rec.OutputTokens, rec.CacheReadTokens, rec.CacheWriteTokens, rec.ReasoningTokens} {
			if n < 0 {
				t.Fatalf("accepted a negative count: %+v", rec)
			}
		}
		if rec.ReasoningTokens > rec.OutputTokens {
			t.Fatalf("accepted reasoning above output: %+v", rec)
		}
		if !statesACounter(line) {
			t.Fatal("accepted a record that states no token counter")
		}
	})
}

// statesACounter re-reads the line the way encoding/json matches keys -- unescaped and
// case-insensitive -- and reports whether any of the four counters carries a non-null value.
func statesACounter(line string) bool {
	var m map[string]json.RawMessage
	if json.Unmarshal([]byte(line), &m) != nil {
		return false
	}
	for k, v := range m {
		for _, want := range []string{"input_tokens", "output_tokens", "cache_read_tokens", "cache_write_tokens"} {
			if strings.EqualFold(k, want) && string(v) != "null" {
				return true
			}
		}
	}
	return false
}
