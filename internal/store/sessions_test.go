package store

import (
	"context"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

func TestSessionsAggregatesPerSession(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)

	// Session s1: three turns. Gaps are 10min (A->B, counted) and 40min (B->C, a resume
	// gap that must be EXCLUDED). Peak context is the max single-turn cache_read+input.
	// Session s2: a single turn, so no inter-turn gap and zero active minutes.
	_, err := s.Insert(ctx, []usage.Record{
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: base, Model: "claude-opus-4-5",
			Project: "web", InputTokens: 100, OutputTokens: 50, CacheReadTokens: 1000,
			DedupeKey: "s1-a", Edits: 1, Compactions: 0,
		},
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: base.Add(10 * time.Minute), Model: "claude-opus-4-5",
			Project: "web", InputTokens: 200, OutputTokens: 80, CacheReadTokens: 5000,
			DedupeKey: "s1-b", Edits: 0, Compactions: 1,
		},
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: base.Add(50 * time.Minute), Model: "claude-opus-4-5",
			Project: "web", InputTokens: 50, OutputTokens: 30, CacheReadTokens: 8000,
			DedupeKey: "s1-c", Edits: 1, Compactions: 0,
		},
		{
			Tool: "claude-code", SessionID: "s2", Timestamp: base.Add(2 * time.Hour), Model: "claude-haiku-4-5",
			Project: "api", InputTokens: 30, OutputTokens: 10, CacheReadTokens: 0,
			DedupeKey: "s2-a", Edits: 0, Compactions: 0,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	rows, err := s.Sessions(ctx, base.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("Sessions = %+v, want 2 rows", rows)
	}
	bySession := map[string]SessionRow{}
	for _, r := range rows {
		bySession[r.SessionID] = r
	}

	s1, ok := bySession["s1"]
	if !ok {
		t.Fatalf("missing session s1: %+v", rows)
	}
	if s1.Turns != 3 || s1.OutputTokens != 160 {
		t.Fatalf("s1 = %+v, want Turns=3 OutputTokens=160", s1)
	}
	if s1.Edits != 2 || s1.Compactions != 1 {
		t.Fatalf("s1 = %+v, want Edits=2 Compactions=1", s1)
	}
	// Peak = max(100+1000, 200+5000, 50+8000) = 8050, not a running total.
	if s1.PeakContextTokens != 8050 {
		t.Fatalf("s1.PeakContextTokens = %d, want 8050 (max single-turn cache_read+input)", s1.PeakContextTokens)
	}
	// Active = 10min (A->B); the 40min B->C resume gap is excluded.
	if s1.ActiveMinutes < 9.999 || s1.ActiveMinutes > 10.001 {
		t.Fatalf("s1.ActiveMinutes = %v, want 10 (the 40min resume gap must be excluded)", s1.ActiveMinutes)
	}
	if !s1.FirstTs.Equal(base) || !s1.LastTs.Equal(base.Add(50*time.Minute)) {
		t.Fatalf("s1 FirstTs/LastTs = %v/%v, want %v/%v", s1.FirstTs, s1.LastTs, base, base.Add(50*time.Minute))
	}
	if s1.Project != "web" || s1.Tool != "claude-code" {
		t.Fatalf("s1 dims = %+v, want Project=web Tool=claude-code", s1)
	}

	s2, ok := bySession["s2"]
	if !ok {
		t.Fatalf("missing session s2: %+v", rows)
	}
	if s2.Turns != 1 || s2.OutputTokens != 10 || s2.PeakContextTokens != 30 {
		t.Fatalf("s2 = %+v, want Turns=1 OutputTokens=10 PeakContextTokens=30", s2)
	}
	if s2.ActiveMinutes != 0 {
		t.Fatalf("s2.ActiveMinutes = %v, want 0 (a single-turn session has no inter-turn gap)", s2.ActiveMinutes)
	}
}

func TestSessionsSinceBoundaryIsInclusive(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	since := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{Tool: "claude-code", SessionID: "s1", Timestamp: since, Model: "m", InputTokens: 1, DedupeKey: "at-since"},
		{Tool: "claude-code", SessionID: "s2", Timestamp: since.Add(-time.Second), Model: "m", InputTokens: 1, DedupeKey: "before-since"},
	})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := s.Sessions(ctx, since)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].SessionID != "s1" {
		t.Fatalf("Sessions = %+v, want only s1 (record exactly at since is included)", rows)
	}
}

// TestSessionsIncludesMember guards the team-view dimension: Sessions must carry each
// session's Member through, constant within the session like Project/Tool (unlike
// Model -- see SessionRow.Model).
func TestSessionsIncludesMember(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	ts := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "m", InputTokens: 1, DedupeKey: "1", Member: "alice"},
		{Tool: "claude-code", SessionID: "s2", Timestamp: ts, Model: "m", InputTokens: 1, DedupeKey: "2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := s.Sessions(ctx, ts.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	bySession := map[string]SessionRow{}
	for _, r := range rows {
		bySession[r.SessionID] = r
	}
	if bySession["s1"].Member != "alice" {
		t.Fatalf("s1.Member = %q, want alice", bySession["s1"].Member)
	}
	if bySession["s2"].Member != "" {
		t.Fatalf("s2.Member = %q, want empty (local, unsynced session)", bySession["s2"].Member)
	}
}

// TestSessionsSameSessionIDDifferentMembersStaySeparate guards against a real bug caught
// while building the team dashboard's per-member session counts: two members' pushed
// usage_record rows sharing the same raw session_id (never expected in practice --
// session_id is a locally generated UUID -- but reachable by, e.g., syncing the same
// local store twice under two --member names, which the demo/testing workflow does) must
// not collapse into one SessionRow attributed to whichever member sorts last. Each member
// must keep their own row and their own turn/activity counts.
func TestSessionsSameSessionIDDifferentMembersStaySeparate(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	ts := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{Tool: "claude-code", SessionID: "shared", Timestamp: ts, Model: "m", InputTokens: 1, DedupeKey: "alice:1", Member: "alice"},
		{Tool: "claude-code", SessionID: "shared", Timestamp: ts.Add(time.Minute), Model: "m", InputTokens: 1, DedupeKey: "alice:2", Member: "alice"},
		{Tool: "claude-code", SessionID: "shared", Timestamp: ts, Model: "m", InputTokens: 1, DedupeKey: "bob:1", Member: "bob"},
	})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := s.Sessions(ctx, ts.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("Sessions = %+v, want 2 rows: one per member, even though session_id collides", rows)
	}
	byMember := map[string]SessionRow{}
	for _, r := range rows {
		byMember[r.Member] = r
	}
	if alice, ok := byMember["alice"]; !ok || alice.Turns != 2 {
		t.Fatalf("alice row = %+v, ok=%v, want Turns=2 (bob's turn must not be counted in)", alice, ok)
	}
	if bob, ok := byMember["bob"]; !ok || bob.Turns != 1 {
		t.Fatalf("bob row = %+v, ok=%v, want Turns=1 (alice's turns must not be counted in)", bob, ok)
	}
}

func TestSessionsEmptyStore(t *testing.T) {
	s := newStore(t)
	rows, err := s.Sessions(context.Background(), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("Sessions on empty store = %+v, want empty", rows)
	}
}
