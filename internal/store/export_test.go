package store

import (
	"context"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

func TestExportRoundTripsRecords(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	ts := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	recs := []usage.Record{
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "m",
			InputTokens: 10, OutputTokens: 5, CacheReadTokens: 1, CacheWriteTokens: 2, ReasoningTokens: 3,
			DedupeKey: "e1", Project: "web", Subpath: "apps/api", GitBranch: "main", Entrypoint: "cli",
			Granularity: "turn", LinesAdded: 4, LinesRemoved: 1, Edits: 1, ToolCalls: 2, Rejected: 1,
			Compactions: 1, ReworkLines: 2, Member: "alice",
		},
		{
			Tool: "codex", SessionID: "s2", Timestamp: ts.Add(time.Hour), Model: "m2",
			InputTokens: 20, DedupeKey: "e2", Member: "bob",
		},
	}
	if _, err := s.Insert(ctx, recs); err != nil {
		t.Fatal(err)
	}

	got, err := s.Export(ctx, ts.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("Export = %+v, want 2 records", got)
	}
	byKey := map[string]usage.Record{}
	for _, r := range got {
		byKey[r.DedupeKey] = r
	}

	e1, ok := byKey["e1"]
	if !ok {
		t.Fatalf("Export missing e1: %+v", got)
	}
	if e1.Tool != "claude-code" || e1.Project != "web" || e1.Subpath != "apps/api" || e1.Member != "alice" ||
		e1.InputTokens != 10 || e1.LinesAdded != 4 || e1.ReworkLines != 2 || !e1.Timestamp.Equal(ts) {
		t.Fatalf("e1 = %+v, want round-tripped fields matching input", e1)
	}
	// Cwd is transient and never stored; a record read back through Export must not
	// resurrect it.
	if e1.Cwd != "" {
		t.Fatalf("e1.Cwd = %q, want empty (never persisted)", e1.Cwd)
	}

	e2, ok := byKey["e2"]
	if !ok || e2.Member != "bob" || e2.Tool != "codex" {
		t.Fatalf("e2 = %+v, ok=%v", e2, ok)
	}
}

func TestExportExcludesRowsBeforeSince(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	since := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{Tool: "claude-code", SessionID: "s1", Timestamp: since, Model: "m", DedupeKey: "at-since"},
		{Tool: "claude-code", SessionID: "s1", Timestamp: since.Add(-time.Second), Model: "m", DedupeKey: "before-since"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Export(ctx, since)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].DedupeKey != "at-since" {
		t.Fatalf("Export = %+v, want only the at-since record (inclusive boundary)", got)
	}
}

func TestExportEmptyStoreReturnsNoRecords(t *testing.T) {
	s := newStore(t)
	got, err := s.Export(context.Background(), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("Export = %+v, want empty", got)
	}
}
