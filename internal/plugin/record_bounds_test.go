package plugin

import (
	"encoding/json"
	"testing"
	"time"
)

// TestRecordTimestampIsBounded closes the gap between the two boundaries a record can cross
// into assaio: the sync endpoint bounded the timestamp and the plugin protocol did not, so the
// same year-9999 record was rejected over HTTP and accepted from a subprocess -- where it then
// sat inside every --since window forever, since every query is `ts >= ?` with no ceiling.
func TestRecordTimestampIsBounded(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		timestamp string
		wantErr   bool
	}{
		{"within the window", "2026-08-09T09:00:00Z", false},
		{"a day old", "2026-08-08T09:00:00Z", false},
		{"inside the clock-skew allowance", "2026-08-10T09:00:00Z", false},
		{"far future", "9999-01-01T00:00:00Z", true},
		{"before any AI coding tool existed", "1999-01-01T00:00:00Z", true},
		{"past the clock-skew allowance", "2026-08-12T09:00:00Z", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := wireRecord{
				SessionID: "s1", DedupeKey: "s1:0", Model: "m",
				Granularity: "turn", Timestamp: tt.timestamp,
			}
			_, err := w.toRecordAt("demo", now)
			if (err != nil) != tt.wantErr {
				t.Fatalf("toRecordAt(%s) err = %v, wantErr %v", tt.timestamp, err, tt.wantErr)
			}
		})
	}
}

// TestRecordRejectsReasoningAboveOutput holds the subset invariant at the plugin boundary:
// reasoning tokens are a part of output, and a record claiming more of them renders as a
// reasoning share above 100%.
func TestRecordRejectsReasoningAboveOutput(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	w := wireRecord{
		SessionID: "s1", DedupeKey: "s1:0", Model: "m", Granularity: "turn",
		Timestamp: "2026-08-09T09:00:00Z", OutputTokens: 100, ReasoningTokens: 200,
	}
	if _, err := w.toRecordAt("demo", now); err == nil {
		t.Fatal("want error for reasoning_tokens above output_tokens, got nil")
	}
}

// TestMetricWireCarriesTheCacheTier: a metric plugin re-pricing the rows it is handed has to
// see both the 1-hour portion and its rate, or it necessarily bills every cache write at the
// cheaper 5-minute rate and reports a cost the core does not agree with.
func TestMetricWireCarriesTheCacheTier(t *testing.T) {
	const field = "cacheWrite1h"
	if got := wireField(t, metricUsageRow{CacheWrite: 100, CacheWrite1h: 40}, field); got != float64(40) {
		t.Errorf("usage row %s = %v, want 40", field, got)
	}
	if got := wireField(t, metricPrice{CacheWrite: 1, CacheWrite1h: 2}, field); got != float64(2) {
		t.Errorf("price %s = %v, want 2", field, got)
	}
}

func wireField(t *testing.T, v any, field string) any {
	t.Helper()
	encoded, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded[field]
}
