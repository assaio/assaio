package plugin

import "testing"

func TestParseRecordLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantErr bool
	}{
		{"valid turn", `{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","dedupe_key":"s1:0","granularity":"turn"}`, false},
		{"valid session", `{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","dedupe_key":"s1:0","granularity":"session"}`, false},
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

func TestParseRecordLineNamespacesTool(t *testing.T) {
	rec, err := parseRecordLine([]byte(`{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","dedupe_key":"s1:0","granularity":"turn"}`), "mytool")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Tool != "plugin:mytool" {
		t.Fatalf("Tool = %q, want plugin:mytool", rec.Tool)
	}
}
