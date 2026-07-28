package store

import (
	"context"
	"testing"
	"time"
)

func TestIngestStateEmptyStore(t *testing.T) {
	st := openTempStore(t)
	got, err := st.IngestState(context.Background(), "v1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("want empty state, got %d entries", len(got))
	}
}

func TestIngestStateRoundTrip(t *testing.T) {
	st := openTempStore(t)
	ctx := context.Background()
	at := time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC)
	entries := []IngestEntry{
		{Path: "/logs/a.jsonl", Tool: "claude-code", Size: 12, MtimeNS: 345},
		{Path: "/logs/b.jsonl", Tool: "codex", Size: 67, MtimeNS: 890},
	}
	if err := st.RecordIngest(ctx, "v1", at, entries); err != nil {
		t.Fatal(err)
	}
	got, err := st.IngestState(ctx, "v1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 entries, got %d", len(got))
	}
	a := got["/logs/a.jsonl"]
	if a.Tool != "claude-code" || a.Size != 12 || a.MtimeNS != 345 {
		t.Errorf("round-trip mismatch: %+v", a)
	}
}

// TestIngestStateIsScopedToVersion is the guard that keeps Insert's restateSignals repair
// reachable: a build that did not write a row must not be told the input is up to date.
func TestIngestStateIsScopedToVersion(t *testing.T) {
	st := openTempStore(t)
	ctx := context.Background()
	at := time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC)
	if err := st.RecordIngest(ctx, "v1", at, []IngestEntry{
		{Path: "/logs/a.jsonl", Tool: "claude-code", Size: 12, MtimeNS: 345},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := st.IngestState(ctx, "v2")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("v2 must not see v1's rows, got %d", len(got))
	}
}

func TestRecordIngestOverwritesSamePath(t *testing.T) {
	st := openTempStore(t)
	ctx := context.Background()
	at := time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC)
	path := "/logs/a.jsonl"
	if err := st.RecordIngest(ctx, "v1", at, []IngestEntry{
		{Path: path, Tool: "claude-code", Size: 12, MtimeNS: 345},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.RecordIngest(ctx, "v1", at.Add(time.Hour), []IngestEntry{
		{Path: path, Tool: "claude-code", Size: 99, MtimeNS: 999},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := st.IngestState(ctx, "v1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 entry, got %d", len(got))
	}
	if e := got[path]; e.Size != 99 || e.MtimeNS != 999 {
		t.Errorf("want last write to win, got %+v", e)
	}
}

func TestRecordIngestEmptyIsNoOp(t *testing.T) {
	st := openTempStore(t)
	if err := st.RecordIngest(context.Background(), "v1", time.Now(), nil); err != nil {
		t.Fatal(err)
	}
}

func TestIngestFreshnessPerTool(t *testing.T) {
	st := openTempStore(t)
	ctx := context.Background()
	early := time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC)
	late := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	if err := st.RecordIngest(ctx, "v1", early, []IngestEntry{
		{Path: "/logs/a.jsonl", Tool: "claude-code"},
		{Path: "/logs/c.jsonl", Tool: "codex"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.RecordIngest(ctx, "v1", late, []IngestEntry{
		{Path: "/logs/b.jsonl", Tool: "claude-code"},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := st.IngestFreshness(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !got["claude-code"].Equal(late) {
		t.Errorf("claude-code freshness = %v, want %v", got["claude-code"], late)
	}
	if !got["codex"].Equal(early) {
		t.Errorf("codex freshness = %v, want %v", got["codex"], early)
	}
	if _, ok := got["gemini-cli"]; ok {
		t.Error("a never-ingested tool must be absent, not zero")
	}
}
