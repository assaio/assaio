package store

import (
	"context"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

func TestSubpathsGroupsByProjectAndSubpath(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	ts := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "m", DedupeKey: "a1", Granularity: "turn",
			Project: "web", Subpath: "apps/mobile", LinesAdded: 100,
		},
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: ts.Add(time.Minute), Model: "m", DedupeKey: "a2", Granularity: "turn",
			Project: "web", Subpath: "apps/mobile", LinesAdded: 50,
		},
		{
			Tool: "claude-code", SessionID: "s2", Timestamp: ts, Model: "m", DedupeKey: "a3", Granularity: "turn",
			Project: "web", Subpath: "", LinesAdded: 10,
		},
		{
			Tool: "claude-code", SessionID: "s3", Timestamp: ts, Model: "m", DedupeKey: "a4", Granularity: "turn",
			Project: "other", Subpath: "ignored", LinesAdded: 999,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	rows, err := s.Subpaths(ctx, "web", ts.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("Subpaths = %+v, want 2 rows (apps/mobile and root)", rows)
	}
	// Ranked by lines descending: apps/mobile (150) before root (10).
	if rows[0].Subpath != "apps/mobile" || rows[0].Lines != 150 || rows[0].Sessions != 1 {
		t.Fatalf("rows[0] = %+v, want {apps/mobile 150 1}", rows[0])
	}
	if rows[1].Subpath != "" || rows[1].Lines != 10 || rows[1].Sessions != 1 {
		t.Fatalf("rows[1] = %+v, want {\"\" 10 1}", rows[1])
	}
}

func TestSubpathsExcludesOtherProjectsAndOldRows(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	ts := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "m", DedupeKey: "a1", Granularity: "turn", Project: "web", Subpath: "x", LinesAdded: 5},
		{Tool: "claude-code", SessionID: "s2", Timestamp: ts, Model: "m", DedupeKey: "a2", Granularity: "turn", Project: "other", Subpath: "x", LinesAdded: 5},
		{
			Tool: "claude-code", SessionID: "s3", Timestamp: ts.Add(-24 * time.Hour), Model: "m", DedupeKey: "a3", Granularity: "turn",
			Project: "web", Subpath: "x", LinesAdded: 5,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	rows, err := s.Subpaths(ctx, "web", ts.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Lines != 5 {
		t.Fatalf("Subpaths = %+v, want a single row with 5 lines (other project and old row excluded)", rows)
	}
}

func TestSubpathsDistinctSessionCountNotRowCount(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	ts := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "m", DedupeKey: "a1", Granularity: "turn", Project: "web", Subpath: "x", LinesAdded: 1},
		{Tool: "claude-code", SessionID: "s1", Timestamp: ts.Add(time.Minute), Model: "m", DedupeKey: "a2", Granularity: "turn", Project: "web", Subpath: "x", LinesAdded: 1},
		{Tool: "claude-code", SessionID: "s1", Timestamp: ts.Add(2 * time.Minute), Model: "m", DedupeKey: "a3", Granularity: "turn", Project: "web", Subpath: "x", LinesAdded: 1},
	})
	if err != nil {
		t.Fatal(err)
	}

	rows, err := s.Subpaths(ctx, "web", ts.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Sessions != 1 {
		t.Fatalf("Subpaths = %+v, want Sessions=1 (three rows, one session)", rows)
	}
}

// TestSubpathsSumAcrossMembers is the panel's own shape: it has a subpath column and no member
// column, so one subpath is one row. Grouping in member printed the same subpath once per person,
// each row holding one person's lines, and a reader took the top one for the subpath's total.
func TestSubpathsSumAcrossMembers(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	ts := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{
			Tool: "claude-code", SessionID: "alice-1", Timestamp: ts, Model: "m", DedupeKey: "alice:1", Granularity: "turn",
			Project: "web", Subpath: "apps/api", LinesAdded: 100, Member: "alice",
		},
		{
			Tool: "claude-code", SessionID: "bob-1", Timestamp: ts, Model: "m", DedupeKey: "bob:1", Granularity: "turn",
			Project: "web", Subpath: "apps/api", LinesAdded: 50, Member: "bob",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	rows, err := s.Subpaths(ctx, "web", ts.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("Subpaths = %+v, want one row for one subpath", rows)
	}
	if rows[0].Lines != 150 || rows[0].Sessions != 2 {
		t.Fatalf("row = %+v, want Lines=150 Sessions=2 -- the subpath's own totals, not one member's", rows[0])
	}
}

func TestSubpathsEmptyForUnknownProject(t *testing.T) {
	s := newStore(t)
	rows, err := s.Subpaths(context.Background(), "does-not-exist", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("Subpaths = %+v, want empty for a project with no rows", rows)
	}
}
