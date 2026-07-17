package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestOpenErrorsWhenParentDirMissing(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "nonexistent-subdir", "t.db")
	s, err := Open(bad)
	if err == nil {
		_ = s.Close()
		t.Fatal("Open() err = nil, want an error when the parent directory does not exist")
	}
}

func TestInsertIsIdempotent(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	rec := usage.Record{
		Tool: "claude-code", SessionID: "s1",
		Timestamp: time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC),
		Model:     "claude-opus-4-5", InputTokens: 100, OutputTokens: 200, DedupeKey: "a1",
	}

	n1, err := s.Insert(ctx, []usage.Record{rec})
	if err != nil || n1 != 1 {
		t.Fatalf("first insert n=%d err=%v", n1, err)
	}
	n2, err := s.Insert(ctx, []usage.Record{rec})
	if err != nil || n2 != 0 {
		t.Fatalf("duplicate insert n=%d err=%v (want 0)", n2, err)
	}
	total, _ := s.Count(ctx)
	if total != 1 {
		t.Fatalf("Count = %d want 1", total)
	}
}

func TestUsageGroups(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	ts := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "m", InputTokens: 10, OutputTokens: 5, DedupeKey: "1"},
		{Tool: "claude-code", SessionID: "s1", Timestamp: ts.Add(time.Hour), Model: "m", InputTokens: 20, OutputTokens: 7, DedupeKey: "2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := s.Usage(ctx, ts.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].In != 30 || rows[0].Out != 12 || rows[0].Day != "2026-07-01" {
		t.Fatalf("Usage = %+v", rows)
	}
}

func TestInsertRoundTripsEmptyStringDims(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	ts := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	rec := usage.Record{
		Tool: "gemini-cli", SessionID: "s1", Timestamp: ts, Model: "m",
		InputTokens: 1, OutputTokens: 1, DedupeKey: "empty-dims",
		Project: "", GitBranch: "", Entrypoint: "", Granularity: "",
	}
	n, err := s.Insert(ctx, []usage.Record{rec})
	if err != nil || n != 1 {
		t.Fatalf("Insert n=%d err=%v", n, err)
	}
	rows, err := s.Usage(ctx, ts.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("Usage = %+v, want 1 row", rows)
	}
	row := rows[0]
	if row.Project != "" || row.Entrypoint != "" {
		t.Fatalf("row = %+v, want empty-string Project and Entrypoint preserved", row)
	}
	if row.In != 1 || row.Out != 1 {
		t.Fatalf("row totals = %+v, want In=1 Out=1", row)
	}
}

func TestUsageSinceBoundaryIsInclusive(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	since := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{Tool: "claude-code", SessionID: "s1", Timestamp: since, Model: "m", InputTokens: 1, DedupeKey: "at-since"},
		{Tool: "claude-code", SessionID: "s1", Timestamp: since.Add(-time.Second), Model: "m", InputTokens: 100, DedupeKey: "before-since"},
	})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := s.Usage(ctx, since)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("Usage = %+v, want 1 row (record exactly at since is included, since is >=)", rows)
	}
	if rows[0].In != 1 {
		t.Fatalf("Usage[0].In = %d, want 1 (the before-since record must be excluded)", rows[0].In)
	}
}

func TestClearScopes(t *testing.T) {
	ctx := context.Background()
	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	seed := func(t *testing.T) *Store {
		t.Helper()
		s := newStore(t)
		_, err := s.Insert(ctx, []usage.Record{
			{Tool: "claude-code", SessionID: "s1", Timestamp: base, Model: "m", InputTokens: 1, DedupeKey: "old-claude"},
			{Tool: "codex", SessionID: "s2", Timestamp: base, Model: "m", InputTokens: 1, DedupeKey: "old-codex"},
			{Tool: "claude-code", SessionID: "s1", Timestamp: base.Add(48 * time.Hour), Model: "m", InputTokens: 1, DedupeKey: "new-claude"},
			{Tool: "codex", SessionID: "s2", Timestamp: base.Add(48 * time.Hour), Model: "m", InputTokens: 1, DedupeKey: "new-codex"},
		})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	cutoff := base.Add(24 * time.Hour)

	t.Run("all", func(t *testing.T) {
		s := seed(t)
		n, err := s.Clear(ctx, time.Time{}, "")
		if err != nil {
			t.Fatal(err)
		}
		if n != 4 {
			t.Fatalf("Clear rows = %d, want 4", n)
		}
		total, _ := s.Count(ctx)
		if total != 0 {
			t.Fatalf("Count after Clear = %d, want 0", total)
		}
	})

	t.Run("before-only", func(t *testing.T) {
		s := seed(t)
		n, err := s.Clear(ctx, cutoff, "")
		if err != nil {
			t.Fatal(err)
		}
		if n != 2 {
			t.Fatalf("Clear rows = %d, want 2 (both old-* records)", n)
		}
		total, _ := s.Count(ctx)
		if total != 2 {
			t.Fatalf("Count after Clear = %d, want 2 (the new-* records remain)", total)
		}
	})

	t.Run("tool-only", func(t *testing.T) {
		s := seed(t)
		n, err := s.Clear(ctx, time.Time{}, "claude-code")
		if err != nil {
			t.Fatal(err)
		}
		if n != 2 {
			t.Fatalf("Clear rows = %d, want 2 (both claude-code records)", n)
		}
		total, _ := s.Count(ctx)
		if total != 2 {
			t.Fatalf("Count after Clear = %d, want 2 (the codex records remain)", total)
		}
	})

	t.Run("before-and-tool", func(t *testing.T) {
		s := seed(t)
		n, err := s.Clear(ctx, cutoff, "claude-code")
		if err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatalf("Clear rows = %d, want 1 (only old-claude)", n)
		}
		total, _ := s.Count(ctx)
		if total != 3 {
			t.Fatalf("Count after Clear = %d, want 3", total)
		}
	})
}

func TestMigrateIsIdempotentOnReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "t.db")

	s1, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s1.Insert(ctx, []usage.Record{
		{Tool: "claude-code", SessionID: "s1", Timestamp: time.Now(), Model: "m", InputTokens: 1, DedupeKey: "a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s1.Close(); err != nil {
		t.Fatal(err)
	}

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen must not fail on already-migrated schema: %v", err)
	}
	defer func() { _ = s2.Close() }()

	total, err := s2.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("Count after reopen = %d, want 1 (data preserved, no duplicate migration)", total)
	}
}

func TestUsageRoundTripsDimensions(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	ts := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "m", InputTokens: 10, OutputTokens: 5,
			DedupeKey: "1", Project: "app", GitBranch: "main", Entrypoint: "cli", Granularity: "turn",
		},
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "m", InputTokens: 3, OutputTokens: 1,
			DedupeKey: "2", Project: "other", GitBranch: "dev", Entrypoint: "sdk-py", Granularity: "turn",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := s.Usage(ctx, ts.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("Usage = %+v, want 2 rows (grouped by project/entrypoint)", rows)
	}
	byProject := map[string]UsageRow{}
	for _, r := range rows {
		byProject[r.Project] = r
	}
	app, ok := byProject["app"]
	if !ok || app.Entrypoint != "cli" || app.In != 10 {
		t.Fatalf("app row = %+v, ok=%v", app, ok)
	}
	other, ok := byProject["other"]
	if !ok || other.Entrypoint != "sdk-py" || other.In != 3 {
		t.Fatalf("other row = %+v, ok=%v", other, ok)
	}
}

// TestInsertPersistsSubpathNotCwd guards the privacy boundary: Subpath (relative, safe)
// is a real column and round-trips; Cwd (the full path) is never written anywhere, even
// when a record sets it.
func TestInsertPersistsSubpathNotCwd(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	ts := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	rec := usage.Record{
		Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "m", InputTokens: 1,
		DedupeKey: "cwd-check", Cwd: "/Users/alice/work/monorepo/apps/mobile",
		Project: "monorepo", Subpath: "apps/mobile",
	}
	n, err := s.Insert(ctx, []usage.Record{rec})
	if err != nil || n != 1 {
		t.Fatalf("Insert n=%d err=%v", n, err)
	}

	var project, subpath string
	err = s.db.QueryRowContext(ctx, `SELECT project, subpath FROM usage_record WHERE dedupe_key = ?`, "cwd-check").
		Scan(&project, &subpath)
	if err != nil {
		t.Fatal(err)
	}
	if project != "monorepo" || subpath != "apps/mobile" {
		t.Fatalf("project=%q subpath=%q, want monorepo/apps/mobile", project, subpath)
	}

	if _, err := s.db.ExecContext(ctx, `SELECT cwd FROM usage_record`); err == nil {
		t.Fatal("SELECT cwd succeeded, want an error: cwd must not be a column")
	}
}

// TestInsertRoundTripsMember guards the team-server dimension: a record pushed under a
// given member must persist that member exactly, distinct from the "" purely-local
// default.
func TestInsertRoundTripsMember(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	rec := usage.Record{
		Tool: "claude-code", SessionID: "s1", Timestamp: time.Now(), Model: "m",
		InputTokens: 1, DedupeKey: "member-check", Member: "alice",
	}
	n, err := s.Insert(ctx, []usage.Record{rec})
	if err != nil || n != 1 {
		t.Fatalf("Insert n=%d err=%v", n, err)
	}
	var member string
	err = s.db.QueryRowContext(ctx, `SELECT member FROM usage_record WHERE dedupe_key = ?`, "member-check").Scan(&member)
	if err != nil {
		t.Fatal(err)
	}
	if member != "alice" {
		t.Fatalf("member = %q, want alice", member)
	}
}

// TestInsertDefaultsMemberToEmptyString guards local (non-synced) behavior: a record
// inserted the normal way (backfill, no server involved) must keep Member == "".
func TestInsertDefaultsMemberToEmptyString(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	rec := usage.Record{
		Tool: "claude-code", SessionID: "s1", Timestamp: time.Now(), Model: "m",
		InputTokens: 1, DedupeKey: "no-member",
	}
	if _, err := s.Insert(ctx, []usage.Record{rec}); err != nil {
		t.Fatal(err)
	}
	var member string
	err := s.db.QueryRowContext(ctx, `SELECT member FROM usage_record WHERE dedupe_key = ?`, "no-member").Scan(&member)
	if err != nil {
		t.Fatal(err)
	}
	if member != "" {
		t.Fatalf("member = %q, want empty (local usage default)", member)
	}
}

// TestUsageGroupsByMember guards the team-view dimension end to end: Usage must group
// separately per member, and purely local rows (member "") must remain their own group,
// distinct from any synced member's rows -- never silently merged together.
func TestUsageGroupsByMember(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	ts := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "m",
			InputTokens: 10, DedupeKey: "alice:1", Member: "alice", LinesAdded: 5,
		},
		{
			Tool: "claude-code", SessionID: "s2", Timestamp: ts, Model: "m",
			InputTokens: 20, DedupeKey: "bob:1", Member: "bob", LinesAdded: 7,
		},
		{
			Tool: "claude-code", SessionID: "s3", Timestamp: ts, Model: "m",
			InputTokens: 30, DedupeKey: "local:1", LinesAdded: 9,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := s.Usage(ctx, ts.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("Usage = %+v, want 3 rows: one per synced member plus the local \"\" group", rows)
	}
	byMember := map[string]UsageRow{}
	for _, r := range rows {
		byMember[r.Member] = r
	}
	if alice, ok := byMember["alice"]; !ok || alice.In != 10 || alice.LinesAdded != 5 {
		t.Fatalf("alice row = %+v, ok=%v, want In=10 LinesAdded=5", alice, ok)
	}
	if bob, ok := byMember["bob"]; !ok || bob.In != 20 || bob.LinesAdded != 7 {
		t.Fatalf("bob row = %+v, ok=%v, want In=20 LinesAdded=7", bob, ok)
	}
	if local, ok := byMember[""]; !ok || local.In != 30 || local.LinesAdded != 9 {
		t.Fatalf("local row = %+v, ok=%v, want In=30 LinesAdded=9", local, ok)
	}
}

func TestUsageSumsActivityFields(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	ts := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "m", InputTokens: 10,
			DedupeKey: "1", LinesAdded: 5, LinesRemoved: 2, Edits: 1, ToolCalls: 2, Rejected: 1, Compactions: 1,
			ReworkLines: 2,
		},
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: ts.Add(time.Hour), Model: "m", InputTokens: 20,
			DedupeKey: "2", LinesAdded: 3, LinesRemoved: 1, Edits: 0, ToolCalls: 0, Rejected: 0, Compactions: 0,
			ReworkLines: 1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := s.Usage(ctx, ts.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("Usage = %+v, want 1 row", rows)
	}
	row := rows[0]
	if row.LinesAdded != 8 || row.LinesRemoved != 3 || row.Edits != 1 || row.ToolCalls != 2 || row.Rejected != 1 || row.Compactions != 1 {
		t.Fatalf("activity sums = %+v, want LinesAdded=8 LinesRemoved=3 Edits=1 ToolCalls=2 Rejected=1 Compactions=1", row)
	}
	if row.ReworkLines != 3 {
		t.Fatalf("ReworkLines sum = %d, want 3", row.ReworkLines)
	}
}
