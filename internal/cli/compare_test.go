package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

// TestCompareWindowIsSymmetric guards that both compared windows span exactly N
// day-buckets. The earlier off-by-one gave the recent window N+1 buckets, biasing every
// mover's Δ upward.
func TestCompareWindowIsSymmetric(t *testing.T) {
	now := time.Date(2026, 7, 20, 15, 0, 0, 0, time.UTC)
	priorStart, cutoff := compareWindow(now, 7)

	if got := priorStart.Format("2006-01-02T15:04:05Z07:00"); got != "2026-07-07T00:00:00Z" {
		t.Fatalf("priorStart = %s, want 2026-07-07T00:00:00Z (midnight of prior's first bucket)", got)
	}
	if cutoff != "2026-07-14" {
		t.Fatalf("cutoff = %s, want 2026-07-14 (recent = today-6..today)", cutoff)
	}

	// One row per calendar day across both windows: recent and prior must get 7 each.
	var rows []store.UsageRow
	for d := range 14 {
		day := now.AddDate(0, 0, -d).Format("2006-01-02")
		rows = append(rows, store.UsageRow{Day: day, LinesAdded: 1})
	}
	recent, prior := splitByDay(rows, cutoff)
	if len(recent) != 7 || len(prior) != 7 {
		t.Fatalf("split = %d recent / %d prior, want 7 / 7 (symmetric windows)", len(recent), len(prior))
	}
}

func TestReportCompareShowsMovers(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	db := filepath.Join(t.TempDir(), "u.db")
	now := time.Now().UTC()
	seedStoreAt(t, db, []usage.Record{
		{Tool: "claude-code", SessionID: "r1", Timestamp: now.AddDate(0, 0, -2), Model: "claude-opus-4-5", InputTokens: 1000, OutputTokens: 2000, Project: "web", LinesAdded: 100, DedupeKey: "r1"},
		{Tool: "claude-code", SessionID: "p1", Timestamp: now.AddDate(0, 0, -10), Model: "claude-opus-4-5", InputTokens: 100, OutputTokens: 200, Project: "web", LinesAdded: 10, DedupeKey: "p1"},
	})
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"report", "--db", db, "--since", "7d", "--compare"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"top movers", "web", "Δ COST"} {
		if !strings.Contains(got, want) {
			t.Fatalf("report --compare output missing %q:\n%s", want, got)
		}
	}
}

func TestSplitByDayPartitionsAtCutoff(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "2026-07-16", Project: "recent"},
		{Day: "2026-07-10", Project: "recent"},
		{Day: "2026-07-09", Project: "prior"},
		{Day: "2026-07-01", Project: "prior"},
	}
	recent, prior := splitByDay(rows, "2026-07-10")
	if len(recent) != 2 || len(prior) != 2 {
		t.Fatalf("split at cutoff 2026-07-10 = %d recent, %d prior; want 2/2", len(recent), len(prior))
	}
	for _, r := range recent {
		if r.Project != "recent" {
			t.Fatalf("recent side has a prior row: %+v", r)
		}
	}
}
