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
			Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "m", DedupeKey: "a1",
			Project: "web", Subpath: "apps/mobile", LinesAdded: 100,
		},
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: ts.Add(time.Minute), Model: "m", DedupeKey: "a2",
			Project: "web", Subpath: "apps/mobile", LinesAdded: 50,
		},
		{
			Tool: "claude-code", SessionID: "s2", Timestamp: ts, Model: "m", DedupeKey: "a3",
			Project: "web", Subpath: "", LinesAdded: 10,
		},
		{
			Tool: "claude-code", SessionID: "s3", Timestamp: ts, Model: "m", DedupeKey: "a4",
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
		{Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "m", DedupeKey: "a1", Project: "web", Subpath: "x", LinesAdded: 5},
		{Tool: "claude-code", SessionID: "s2", Timestamp: ts, Model: "m", DedupeKey: "a2", Project: "other", Subpath: "x", LinesAdded: 5},
		{
			Tool: "claude-code", SessionID: "s3", Timestamp: ts.Add(-24 * time.Hour), Model: "m", DedupeKey: "a3",
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
		{Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "m", DedupeKey: "a1", Project: "web", Subpath: "x", LinesAdded: 1},
		{Tool: "claude-code", SessionID: "s1", Timestamp: ts.Add(time.Minute), Model: "m", DedupeKey: "a2", Project: "web", Subpath: "x", LinesAdded: 1},
		{Tool: "claude-code", SessionID: "s1", Timestamp: ts.Add(2 * time.Minute), Model: "m", DedupeKey: "a3", Project: "web", Subpath: "x", LinesAdded: 1},
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

// TestSubpathsKeepsMembersSeparate guards the central-store case: two members' rows for
// the same subpath must not blend into one, and a session_id collision across members
// (see TestSessionsSameSessionIDDifferentMembersStaySeparate) must not undercount
// COUNT(DISTINCT session_id).
func TestSubpathsKeepsMembersSeparate(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	ts := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{
			Tool: "claude-code", SessionID: "shared", Timestamp: ts, Model: "m", DedupeKey: "alice:1",
			Project: "web", Subpath: "apps/api", LinesAdded: 100, Member: "alice",
		},
		{
			Tool: "claude-code", SessionID: "shared", Timestamp: ts, Model: "m", DedupeKey: "bob:1",
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
	if len(rows) != 2 {
		t.Fatalf("Subpaths = %+v, want 2 rows: one per member, even though subpath and session_id collide", rows)
	}
	byMember := map[string]SubpathRow{}
	for _, r := range rows {
		byMember[r.Member] = r
	}
	if alice, ok := byMember["alice"]; !ok || alice.Lines != 100 || alice.Sessions != 1 {
		t.Fatalf("alice row = %+v, ok=%v, want Lines=100 Sessions=1 (bob's lines must not blend in)", alice, ok)
	}
	if bob, ok := byMember["bob"]; !ok || bob.Lines != 50 || bob.Sessions != 1 {
		t.Fatalf("bob row = %+v, ok=%v, want Lines=50 Sessions=1 (alice's lines must not blend in)", bob, ok)
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
