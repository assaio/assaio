package usage

import (
	"testing"
	"time"
)

func TestCheckTimestamp(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		ts      time.Time
		wantErr bool
	}{
		{"now", now, false},
		{"the floor itself", TimestampFloor, false},
		{"a day before now", now.AddDate(0, 0, -1), false},
		{"within the skew", now.Add(FutureSkew - time.Hour), false},
		{"the zero time", time.Time{}, true},
		{"before the floor", TimestampFloor.Add(-time.Second), true},
		{"past the skew", now.Add(FutureSkew + time.Hour), true},
		{"year 9999", time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckTimestamp(tt.ts, now)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckTimestamp(%s) err = %v, wantErr %v", tt.ts, err, tt.wantErr)
			}
		})
	}
}

func TestCheckCounts(t *testing.T) {
	tests := []struct {
		name    string
		rec     Record
		wantErr bool
	}{
		{"empty record", Record{}, false},
		{"ordinary turn", Record{InputTokens: 100, OutputTokens: 50, ReasoningTokens: 20}, false},
		{"reasoning equal to output", Record{OutputTokens: 50, ReasoningTokens: 50}, false},
		{"cache tier equal to its whole", Record{CacheWriteTokens: 10, CacheWrite1hTokens: 10}, false},
		{"negative token", Record{InputTokens: -1}, true},
		{"overflow magnitude", Record{InputTokens: MaxCount + 1}, true},
		{"the cap itself", Record{InputTokens: MaxCount}, false},
		{"cache tier above its whole", Record{CacheWriteTokens: 10, CacheWrite1hTokens: 11}, true},
		{"reasoning above output", Record{OutputTokens: 50, ReasoningTokens: 51}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckCounts(&tt.rec)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckCounts err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
