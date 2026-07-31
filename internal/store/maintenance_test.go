package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

func TestPruneIngestStateDropsVanishedPaths(t *testing.T) {
	st := openTempStore(t)
	ctx := context.Background()
	at := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	if err := st.RecordIngest(ctx, "v1", at, []IngestEntry{
		{Path: "/logs/keep.jsonl", Tool: "claude-code"},
		{Path: "/logs/gone.jsonl", Tool: "claude-code"},
	}); err != nil {
		t.Fatal(err)
	}
	n, err := st.PruneIngestState(ctx, "claude-code", map[string]bool{"/logs/keep.jsonl": true})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("pruned %d row(s), want 1", n)
	}
	state, err := st.IngestState(ctx, "v1")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := state["/logs/gone.jsonl"]; ok {
		t.Error("a vanished path must not keep its row")
	}
	if _, ok := state["/logs/keep.jsonl"]; !ok {
		t.Error("a path still on disk must keep its row")
	}
}

// TestPruneIngestStateIsScopedToOneTool keeps one source's prune from touching another's
// state -- they are discovered independently and can drift independently.
func TestPruneIngestStateIsScopedToOneTool(t *testing.T) {
	st := openTempStore(t)
	ctx := context.Background()
	at := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	if err := st.RecordIngest(ctx, "v1", at, []IngestEntry{
		{Path: "/logs/a.jsonl", Tool: "claude-code"},
		{Path: "/logs/b.jsonl", Tool: "codex"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.PruneIngestState(ctx, "claude-code", nil); err != nil {
		t.Fatal(err)
	}
	state, err := st.IngestState(ctx, "v1")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := state["/logs/b.jsonl"]; !ok {
		t.Error("pruning claude-code must not drop codex state")
	}
}

func TestSizeReportsBytesAndReclaimableSpace(t *testing.T) {
	st := openTempStore(t)
	ctx := context.Background()
	if _, err := st.Insert(ctx, bulkRecords(2000)); err != nil {
		t.Fatal(err)
	}
	before, err := st.Size(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if before.Bytes <= 0 {
		t.Fatalf("Size = %+v, want a positive byte count", before)
	}
	if _, err := st.Clear(ctx, time.Time{}, ""); err != nil {
		t.Fatal(err)
	}
	after, err := st.Size(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if after.Reclaimable <= 0 {
		t.Fatalf("Size = %+v, want deleted pages reported as reclaimable", after)
	}
}

// TestVacuumReturnsFreedPagesToTheFilesystem is the point of the whole exercise: SQLite
// holds deleted pages on a freelist, so "deleted 120000 records" leaves the file exactly
// as large as it was until something explicitly reclaims them.
func TestVacuumReturnsFreedPagesToTheFilesystem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.db")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	if _, err := st.Insert(ctx, bulkRecords(2000)); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Clear(ctx, time.Time{}, ""); err != nil {
		t.Fatal(err)
	}
	before, err := st.Size(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Vacuum(ctx); err != nil {
		t.Fatal(err)
	}
	after, err := st.Size(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if after.Bytes >= before.Bytes {
		t.Fatalf("vacuum did not shrink the store: %d -> %d bytes", before.Bytes, after.Bytes)
	}
	if after.Reclaimable != 0 {
		t.Errorf("after vacuum Reclaimable = %d, want 0", after.Reclaimable)
	}
}

// bulkRecords builds n distinct usage records, enough to grow the file past its initial
// page allocation so a delete leaves something measurable to reclaim.
func bulkRecords(n int) []usage.Record {
	recs := make([]usage.Record, 0, n)
	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	for i := range n {
		recs = append(recs, usage.Record{
			Tool: "claude-code", SessionID: "s1", Timestamp: base.Add(time.Duration(i) * time.Second),
			Model: "claude-opus-4-5", InputTokens: 10, OutputTokens: 20,
			DedupeKey: "k" + time.Duration(i).String(), Granularity: "turn",
			Project: "app", Entrypoint: "cli",
		})
	}
	return recs
}
