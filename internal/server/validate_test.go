package server

import (
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
