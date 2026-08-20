package store

import (
	"context"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

func aggregateRow(key string) usage.Record {
	return usage.Record{
		Tool: "claude-code", SessionID: "s1", Timestamp: time.Now().UTC(), Model: "m",
		DedupeKey: key, Granularity: "session", Sidechain: 1,
		InputTokens: 100, OutputTokens: 50,
	}
}

// TestSupersededAggregatesFindsOnlyCoveredOnes: the parent transcript summarizes a completed
// sub-agent as one aggregate row, and the sub-agent's own transcript holds the same work per
// turn. Suppressing the aggregate at parse time only keeps a *new* one out -- a row written
// before that file existed stayed beside the detailed turns and counted the work twice. An
// aggregate whose transcript is still absent is the only record of that run and must stay.
func TestSupersededAggregatesFindsOnlyCoveredOnes(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)

	recs := []usage.Record{
		aggregateRow("agent:has-a-file"),
		aggregateRow("agent:no-file-yet"),
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: time.Now().UTC(), Model: "m",
			DedupeKey: "msg_1", Granularity: "turn", InputTokens: 1, OutputTokens: 1,
		},
	}
	if _, _, err := st.InsertLocal(ctx, recs); err != nil {
		t.Fatal(err)
	}

	covered := map[string]struct{}{"has-a-file": {}}
	got, err := st.SupersededAggregates(ctx, "claude-code", covered)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "agent:has-a-file" {
		t.Fatalf("superseded = %v, want only agent:has-a-file", got)
	}

	deleted, err := st.DeleteDedupeKeys(ctx, "claude-code", got)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	if n := countRows(t, st); n != 2 {
		t.Fatalf("rows left = %d, want 2 -- the uncovered aggregate and the ordinary turn", n)
	}

	// Idempotent: a second backfill finds nothing left to drop.
	again, err := st.SupersededAggregates(ctx, "claude-code", covered)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 0 {
		t.Fatalf("second pass = %v, want nothing", again)
	}
}

func TestSupersededAggregatesWithNothingCovered(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	if _, _, err := st.InsertLocal(ctx, []usage.Record{aggregateRow("agent:x")}); err != nil {
		t.Fatal(err)
	}
	got, err := st.SupersededAggregates(ctx, "claude-code", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("superseded = %v, want nothing when no transcript is on disk", got)
	}
	if n, err := st.DeleteDedupeKeys(ctx, "claude-code", nil); err != nil || n != 0 {
		t.Fatalf("DeleteDedupeKeys(nil) = (%d, %v), want (0, nil)", n, err)
	}
}

func countRows(t *testing.T, st *Store) int64 {
	t.Helper()
	n, err := st.Count(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return n
}
