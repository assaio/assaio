package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// restateRecord is one turn identified by dedupe key "k1", carrying whatever activity
// signals the caller wants to write for it.
func restateRecord(reads, writes int64, skill string) usage.Record {
	return usage.Record{
		Tool: "claude-code", SessionID: "s1", Timestamp: time.Now().UTC(), Model: "m",
		DedupeKey: "k1", Granularity: "turn", InputTokens: 10, OutputTokens: 5,
		ToolCalls: reads + writes, ToolReads: reads, ToolWrites: writes, Skill: skill,
	}
}

func openTempStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// firstToolReads returns the stored row's captured signals, for asserting what a re-insert
// did or did not change.
func firstToolReads(t *testing.T, st *Store) (reads, writes int64, skill string) {
	t.Helper()
	row := st.db.QueryRowContext(context.Background(),
		`SELECT tool_reads, tool_writes, skill FROM usage_record WHERE dedupe_key = 'k1'`)
	if err := row.Scan(&reads, &writes, &skill); err != nil {
		t.Fatal(err)
	}
	return reads, writes, skill
}

// TestInsertRestatesRowsMissingTheNewSignals covers the upgrade path: history ingested by a
// build that could not extract the tool-call taxonomy must gain it on the next backfill,
// rather than staying zero forever behind a plain ON CONFLICT DO NOTHING.
func TestInsertRestatesRowsMissingTheNewSignals(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)

	if _, err := st.Insert(ctx, []usage.Record{restateRecord(0, 0, "")}); err != nil {
		t.Fatal(err)
	}
	n, err := st.Insert(ctx, []usage.Record{restateRecord(7, 3, "code-review")})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("inserted = %d, want 0: a restatement repairs an existing row, it is not a new record", n)
	}
	reads, writes, skill := firstToolReads(t, st)
	if reads != 7 || writes != 3 || skill != "code-review" {
		t.Fatalf("stored = (reads %d, writes %d, skill %q), want the newly captured signals", reads, writes, skill)
	}
}

// TestInsertNeverOverwritesCapturedSignals asserts a row that already carries signals is
// left alone, so a re-parse that extracts less can never silently degrade stored data.
func TestInsertNeverOverwritesCapturedSignals(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)

	if _, err := st.Insert(ctx, []usage.Record{restateRecord(7, 3, "code-review")}); err != nil {
		t.Fatal(err)
	}
	n, err := st.Insert(ctx, []usage.Record{restateRecord(1, 0, "")})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("inserted = %d, want 0: an already-captured row must not be rewritten", n)
	}
	reads, writes, skill := firstToolReads(t, st)
	if reads != 7 || writes != 3 || skill != "code-review" {
		t.Fatalf("stored = (reads %d, writes %d, skill %q), want the original signals intact", reads, writes, skill)
	}
}

// TestInsertSteadyStateRerunWritesNothing asserts an unchanged re-run still reports zero, so
// the count keeps meaning "new rows" once every row carries its signals.
func TestInsertSteadyStateRerunWritesNothing(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	rec := restateRecord(7, 3, "code-review")

	if _, err := st.Insert(ctx, []usage.Record{rec}); err != nil {
		t.Fatal(err)
	}
	n, err := st.Insert(ctx, []usage.Record{rec})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("inserted on an unchanged re-run = %d, want 0", n)
	}
}
