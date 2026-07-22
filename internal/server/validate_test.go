package server

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// newValidRecord returns a record that passes validateRecord, for tests to mutate one
// field of.
func newValidRecord() usage.Record {
	return usage.Record{
		Tool: "claude-code", SessionID: "s1", Timestamp: time.Now(), Model: "m",
		DedupeKey: "a1", Granularity: "turn", InputTokens: 1,
	}
}

func TestValidateRecordAcceptsKnownTools(t *testing.T) {
	for _, tool := range []string{"claude-code", "codex", "cline", "gemini-cli", "plugin:my-plugin"} {
		r := newValidRecord()
		r.Tool = tool
		if err := validateRecord(&r); err != nil {
			t.Errorf("tool %q: unexpected error %v", tool, err)
		}
	}
}

func TestValidateRecordRejectsUnknownTool(t *testing.T) {
	tests := []string{"", "gpt-5-cli", "plugin:", "plugin:Has Spaces", "plugin:trailing/slash", "plugin:UPPER"}
	for _, tool := range tests {
		r := newValidRecord()
		r.Tool = tool
		if err := validateRecord(&r); err == nil {
			t.Errorf("tool %q: want error, got nil", tool)
		}
	}
}

func TestValidateRecordRejectsOutOfRangeTimestamps(t *testing.T) {
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	cases := map[string]time.Time{
		"zero value":   {},
		"before floor": time.Date(2019, 12, 31, 0, 0, 0, 0, time.UTC),
		"far future":   time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC),
		"beyond skew":  now.Add(72 * time.Hour),
	}
	for name, ts := range cases {
		r := newValidRecord()
		r.Timestamp = ts
		if err := validateRecordAt(&r, now); err == nil {
			t.Errorf("%s: want error, got nil", name)
		}
	}
}

func TestValidateRecordAcceptsInSkewTimestamp(t *testing.T) {
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	r := newValidRecord()
	r.Timestamp = now.Add(1 * time.Hour) // mild clock skew, within tolerance
	if err := validateRecordAt(&r, now); err != nil {
		t.Errorf("in-skew timestamp: unexpected error %v", err)
	}
}

func TestValidateRecordAcceptsKnownGranularities(t *testing.T) {
	for _, g := range []string{"turn", "session"} {
		r := newValidRecord()
		r.Granularity = g
		if err := validateRecord(&r); err != nil {
			t.Errorf("granularity %q: unexpected error %v", g, err)
		}
	}
}

func TestValidateRecordRejectsUnknownGranularity(t *testing.T) {
	for _, g := range []string{"", "weekly", "TURN", "Session"} {
		r := newValidRecord()
		r.Granularity = g
		if err := validateRecord(&r); err == nil {
			t.Errorf("granularity %q: want error, got nil", g)
		}
	}
}

func TestValidateRecordRejectsNegativeFields(t *testing.T) {
	tests := []struct {
		name string
		mut  func(*usage.Record)
	}{
		{"negative input tokens", func(r *usage.Record) { r.InputTokens = -1 }},
		{"negative output tokens", func(r *usage.Record) { r.OutputTokens = -1 }},
		{"negative cache read tokens", func(r *usage.Record) { r.CacheReadTokens = -1 }},
		{"negative lines added", func(r *usage.Record) { r.LinesAdded = -1 }},
		{"negative rework lines", func(r *usage.Record) { r.ReworkLines = -1 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newValidRecord()
			tt.mut(&r)
			if err := validateRecord(&r); err == nil {
				t.Fatal("want error, got nil")
			}
		})
	}
}

func TestValidateRecordRejectsOverflowMagnitude(t *testing.T) {
	r := newValidRecord()
	r.InputTokens = maxFieldValue + 1
	if err := validateRecord(&r); err == nil {
		t.Fatal("want error for overflow-magnitude token count, got nil")
	}
}

func TestValidateRecordAcceptsBoundaryMagnitude(t *testing.T) {
	r := newValidRecord()
	r.InputTokens = maxFieldValue
	if err := validateRecord(&r); err != nil {
		t.Fatalf("boundary value maxFieldValue should be accepted: %v", err)
	}
}

// TestValidateRecordRejectsEmptyDedupeKey guards the honesty rule: an empty dedupe_key
// collapses every such row into one under ON CONFLICT(tool, dedupe_key), silently
// undercounting a member. An empty session_id is tolerated -- the Claude parser doesn't
// guarantee it, and rejecting the whole push over one stale row would break a client's sync.
func TestValidateRecordRejectsEmptyDedupeKey(t *testing.T) {
	r := newValidRecord()
	r.DedupeKey = ""
	if err := validateRecord(&r); err == nil {
		t.Fatal("empty dedupe_key: want error, got nil")
	}

	r = newValidRecord()
	r.SessionID = ""
	if err := validateRecord(&r); err != nil {
		t.Fatalf("empty session_id should be tolerated, got %v", err)
	}
}

// TestValidateRecordRejectsOversizedStringField guards the boundary cap: a token-holding
// client cannot push a multi-megabyte string field into the shared store and the
// unauthenticated dashboard it feeds -- including Tool, whose "plugin:<name>" regex is
// otherwise length-unbounded.
func TestValidateRecordRejectsOversizedStringField(t *testing.T) {
	r := newValidRecord()
	r.Model = strings.Repeat("x", maxStringField+1)
	if err := validateRecord(&r); err == nil {
		t.Fatal("want error for an oversized Model field, got nil")
	}

	r = newValidRecord()
	r.Tool = "plugin:" + strings.Repeat("a", maxStringField)
	if err := validateRecord(&r); err == nil {
		t.Fatal("want error for an oversized Tool field, got nil")
	}
}
