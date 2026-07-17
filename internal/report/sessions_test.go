package report

import (
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

var sessionsNow = time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)

func almostEqual(a, b float64) bool {
	diff := a - b
	return diff < 1e-9 && diff > -1e-9
}

// sessionsFixtureRows are 5 sessions with known turns (1..5), output tokens (100..500),
// peak context (1000..5000), and active minutes (0,5,12,30,60). Edits are set on 3 of 5
// (code sessions), compactions on 2 of 5, and the first two share a calendar day so there
// are 4 distinct active days.
func sessionsFixtureRows() []store.SessionRow {
	base := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	return []store.SessionRow{
		{FirstTs: base, Turns: 1, OutputTokens: 100, PeakContextTokens: 1000, ActiveMinutes: 0, Edits: 0, Compactions: 0},
		{FirstTs: base.Add(time.Hour), Turns: 2, OutputTokens: 200, PeakContextTokens: 2000, ActiveMinutes: 5, Edits: 1, Compactions: 1},
		{FirstTs: base.AddDate(0, 0, 1), Turns: 3, OutputTokens: 300, PeakContextTokens: 3000, ActiveMinutes: 12, Edits: 2, Compactions: 0},
		{FirstTs: base.AddDate(0, 0, 2), Turns: 4, OutputTokens: 400, PeakContextTokens: 4000, ActiveMinutes: 30, Edits: 0, Compactions: 1},
		{FirstTs: base.AddDate(0, 0, 3), Turns: 5, OutputTokens: 500, PeakContextTokens: 5000, ActiveMinutes: 60, Edits: 3, Compactions: 0},
	}
}

func TestBuildSessionStatsHonestMetrics(t *testing.T) {
	stats := BuildSessionStats(sessionsFixtureRows(), sessionsNow)

	if stats.Count != 5 {
		t.Fatalf("Count = %d, want 5", stats.Count)
	}
	if stats.MedianTurns != 3 || stats.P90Turns != 5 {
		t.Fatalf("turns = median %d / p90 %d, want 3 / 5", stats.MedianTurns, stats.P90Turns)
	}
	if stats.MedianOutputTokens != 300 {
		t.Fatalf("MedianOutputTokens = %d, want 300 (work produced, not cumulative cache)", stats.MedianOutputTokens)
	}
	if stats.MedianPeakContextTokens != 3000 {
		t.Fatalf("MedianPeakContextTokens = %d, want 3000 (median of per-session peaks)", stats.MedianPeakContextTokens)
	}
	if !almostEqual(stats.MedianActiveMinutes, 12) {
		t.Fatalf("MedianActiveMinutes = %v, want 12", stats.MedianActiveMinutes)
	}
	if !almostEqual(stats.CodeSessionShare, 0.6) {
		t.Fatalf("CodeSessionShare = %v, want 0.6 (3 of 5 sessions made an edit)", stats.CodeSessionShare)
	}
	if !almostEqual(stats.CompactionRate, 0.4) {
		t.Fatalf("CompactionRate = %v, want 0.4 (2 of 5 sessions compacted)", stats.CompactionRate)
	}
	if !almostEqual(stats.SessionsPerActiveDay, 1.25) {
		t.Fatalf("SessionsPerActiveDay = %v, want 1.25 (5 sessions over 4 distinct active days)", stats.SessionsPerActiveDay)
	}
}

func TestBuildSessionStatsEmptyInputIsZeroValue(t *testing.T) {
	for _, rows := range [][]store.SessionRow{nil, {}} {
		stats := BuildSessionStats(rows, sessionsNow)
		if stats != (SessionStats{}) {
			t.Fatalf("empty input must yield a zero-value SessionStats, got %+v", stats)
		}
	}
}

func TestBuildSessionStatsSingleSessionEdge(t *testing.T) {
	base := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	rows := []store.SessionRow{
		{FirstTs: base, Turns: 8, OutputTokens: 1200, PeakContextTokens: 85000, ActiveMinutes: 15, Edits: 4, Compactions: 1},
	}

	stats := BuildSessionStats(rows, sessionsNow)
	if stats.Count != 1 {
		t.Fatalf("Count = %d, want 1", stats.Count)
	}
	if stats.MedianTurns != 8 || stats.P90Turns != 8 {
		t.Fatalf("single-session turns = %d/%d, want 8/8 (no interpolation with n=1)", stats.MedianTurns, stats.P90Turns)
	}
	if stats.MedianOutputTokens != 1200 || stats.MedianPeakContextTokens != 85000 {
		t.Fatalf("single-session output/peak = %d/%d, want 1200/85000", stats.MedianOutputTokens, stats.MedianPeakContextTokens)
	}
	if !almostEqual(stats.MedianActiveMinutes, 15) {
		t.Fatalf("MedianActiveMinutes = %v, want 15", stats.MedianActiveMinutes)
	}
	if !almostEqual(stats.CodeSessionShare, 1) || !almostEqual(stats.CompactionRate, 1) {
		t.Fatalf("single edited+compacted session = code %v / compaction %v, want 1 / 1", stats.CodeSessionShare, stats.CompactionRate)
	}
	if !almostEqual(stats.SessionsPerActiveDay, 1) {
		t.Fatalf("SessionsPerActiveDay = %v, want 1", stats.SessionsPerActiveDay)
	}
}

func TestBuildSessionStatsAllConversationalZeroCodeShare(t *testing.T) {
	base := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	rows := []store.SessionRow{
		{FirstTs: base, Turns: 2, OutputTokens: 100, PeakContextTokens: 500, ActiveMinutes: 3, Edits: 0, Compactions: 0},
		{FirstTs: base.Add(time.Hour), Turns: 3, OutputTokens: 150, PeakContextTokens: 700, ActiveMinutes: 4, Edits: 0, Compactions: 0},
	}
	stats := BuildSessionStats(rows, sessionsNow)
	if stats.CodeSessionShare != 0 {
		t.Fatalf("CodeSessionShare = %v, want 0 (no session made an edit)", stats.CodeSessionShare)
	}
	if stats.CompactionRate != 0 {
		t.Fatalf("CompactionRate = %v, want 0", stats.CompactionRate)
	}
}
