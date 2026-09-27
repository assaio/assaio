package plugin

import (
	"strings"
	"testing"
)

func TestParseRecordLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantErr bool
	}{
		{"valid turn", `{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","output_tokens":5,"dedupe_key":"s1:0","granularity":"turn"}`, false},
		{"valid session", `{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","input_tokens":5,"dedupe_key":"s1:0","granularity":"session"}`, false},
		{"a stated zero counter", `{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","input_tokens":0,"dedupe_key":"s1:0","granularity":"turn"}`, false},
		{"trailing whitespace", `{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","output_tokens":5,"dedupe_key":"s1:0","granularity":"turn"}  `, false},
		{"no token counter", `{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","dedupe_key":"s1:0","granularity":"turn"}`, true},
		{"reasoning_tokens alone", `{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","reasoning_tokens":0,"dedupe_key":"s1:0","granularity":"turn"}`, true},
		{"null counters only", `{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","input_tokens":null,"output_tokens":null,"dedupe_key":"s1:0","granularity":"turn"}`, true},
		{"a second object on the line", `{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","output_tokens":5,"dedupe_key":"s1:0","granularity":"turn"}{"session_id":"s1","timestamp":"2026-07-01T10:01:00Z","model":"m","output_tokens":5,"dedupe_key":"s1:1","granularity":"turn"}`, true},
		{"a duplicate key's null wins", `{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","output_tokens":5,"output_tokens":null,"dedupe_key":"s1:0","granularity":"turn"}`, true},
		{"keys match case-insensitively", `{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","OUTPUT_TOKENS":5,"dedupe_key":"s1:0","granularity":"turn"}`, false},
		{"trailing garbage", `{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","output_tokens":5,"dedupe_key":"s1:0","granularity":"turn"} trailing`, true},
		{"invalid JSON", `not json`, true},
		{"empty session_id", `{"session_id":"","timestamp":"2026-07-01T10:00:00Z","model":"m","dedupe_key":"s1:0","granularity":"turn"}`, true},
		{"empty dedupe_key", `{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","dedupe_key":"","granularity":"turn"}`, true},
		{"bad timestamp", `{"session_id":"s1","timestamp":"not-a-time","model":"m","dedupe_key":"s1:0","granularity":"turn"}`, true},
		{"invalid granularity", `{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","dedupe_key":"s1:0","granularity":"weekly"}`, true},
		{"negative input_tokens", `{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","input_tokens":-1,"dedupe_key":"s1:0","granularity":"turn"}`, true},
		{"negative output_tokens", `{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","output_tokens":-1,"dedupe_key":"s1:0","granularity":"turn"}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseRecordLine([]byte(tt.line), "demo")
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseRecordLine() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestParseRecordLineRejectsOversizedStringField guards the boundary cap: a plugin cannot
// smuggle a multi-kilobyte string field (here Model) past the record boundary into the
// store, matching the metric-result boundary's own length caps.
func TestParseRecordLineRejectsOversizedStringField(t *testing.T) {
	big := strings.Repeat("x", maxWireStringLen+1)
	line := `{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"` + big + `","output_tokens":5,"dedupe_key":"s1:0","granularity":"turn"}`
	if _, err := parseRecordLine([]byte(line), "demo"); err == nil {
		t.Fatal("want error for an oversized string field, got nil")
	}
}

func TestParseRecordLineNamespacesTool(t *testing.T) {
	rec, err := parseRecordLine([]byte(`{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","output_tokens":5,"dedupe_key":"s1:0","granularity":"turn"}`), "mytool")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Tool != "plugin:mytool" {
		t.Fatalf("Tool = %q, want plugin:mytool", rec.Tool)
	}
}
