package report

import (
	"testing"
	"time"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// TestRecentWindowCoversExactlyItsDays holds the day-bucket count: "7 days" is today and the
// six before it, not today and the seven before it. Subtracting the whole duration and then
// truncating to a date made the boundary date recent too, so every Hot/GoingStale/DormantTools
// verdict compared an eight-day window against a six-day one and read the difference as a
// trend.
func TestRecentWindowCoversExactlyItsDays(t *testing.T) {
	now := time.Date(2026, 8, 9, 13, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		recent time.Duration
		want   string
	}{
		{"seven days ends six buckets back", 7 * 24 * time.Hour, "2026-08-03"},
		{"one day is today alone", 24 * time.Hour, "2026-08-09"},
		{"thirty days", 30 * 24 * time.Hour, "2026-07-11"},
		{"under a day still covers today", time.Hour, "2026-08-09"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := recentCutoff(now, tt.recent); got != tt.want {
				t.Errorf("recentCutoff = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSplitRecentPutsTheBoundaryDayOnTheRightSide(t *testing.T) {
	now := time.Date(2026, 8, 9, 13, 0, 0, 0, time.UTC)
	rows := []store.UsageRow{
		{Day: "2026-08-09", Tool: "claude-code"},
		{Day: "2026-08-03", Tool: "claude-code"},
		{Day: "2026-08-02", Tool: "claude-code"},
	}
	recent, prior := splitRecent(rows, now, 7*24*time.Hour)
	if len(recent) != 2 {
		t.Fatalf("recent = %d rows, want 2 (08-09 and 08-03)", len(recent))
	}
	if len(prior) != 1 || prior[0].Day != "2026-08-02" {
		t.Fatalf("prior = %+v, want only 2026-08-02", prior)
	}
}

// TestInventoryDoesNotCountUnattributedAsAProject covers the breadth signal adoption reads:
// a source that logs no working directory leaves every row's project empty, and counting that
// one nameless bucket made a single-repo user who also runs Gemini CLI look like a two-project
// one. The spend still counts; only the project name is unknown.
func TestInventoryDoesNotCountUnattributedAsAProject(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "2026-08-09", Tool: "claude-code", Project: "app", LinesAdded: 10},
		{Day: "2026-08-09", Tool: "gemini-cli", Project: "", LinesAdded: 0},
	}
	inv := BuildInventory(rows, pricing.Table{})
	if inv.Projects != 1 {
		t.Errorf("Projects = %d, want 1 -- the nameless bucket is not a project", inv.Projects)
	}
	if inv.Unattributed != 1 {
		t.Errorf("Unattributed = %d, want 1", inv.Unattributed)
	}
	if inv.TotalLinesAdded != 10 {
		t.Errorf("TotalLinesAdded = %d, want 10 -- unattributed usage still counts", inv.TotalLinesAdded)
	}
}

// TestProjectRankingsExcludeUnattributed keeps the nameless bucket out of the rankings too,
// where it would otherwise render as a blank-named project beside real ones.
func TestProjectRankingsExcludeUnattributed(t *testing.T) {
	now := time.Date(2026, 8, 9, 13, 0, 0, 0, time.UTC)
	rows := []store.UsageRow{
		{Day: "2026-08-09", Tool: "claude-code", Project: "app", In: 100},
		{Day: "2026-08-09", Tool: "gemini-cli", Project: "", In: 900},
	}
	got := BuildInsights(rows, pricing.Table{}, now, 7*24*time.Hour, 0)
	for _, g := range got.Hot {
		if g.Name == "" {
			t.Fatalf("Hot contains a nameless project: %+v", got.Hot)
		}
	}
	if len(got.Hot) != 1 || got.Hot[0].Name != "app" {
		t.Fatalf("Hot = %+v, want only app", got.Hot)
	}
}
