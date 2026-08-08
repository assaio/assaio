package store

import (
	"context"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// seedClearable builds a store holding two tools' records on two days, one ingest
// watermark per tool, and one label per session -- everything the scope flags can select.
func seedClearable(t *testing.T, base time.Time) *Store {
	t.Helper()
	ctx := context.Background()
	s := newStore(t)
	if _, err := s.Insert(ctx, []usage.Record{
		{Tool: "claude-code", SessionID: "s1", Timestamp: base, Model: "m", InputTokens: 1, DedupeKey: "old-claude"},
		{Tool: "claude-code", SessionID: "s1", Timestamp: base.Add(48 * time.Hour), Model: "m", InputTokens: 1, DedupeKey: "new-claude"},
		{Tool: "codex", SessionID: "s2", Timestamp: base, Model: "m", InputTokens: 1, DedupeKey: "old-codex"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordIngest(ctx, "v1", base, []IngestEntry{
		{Path: "/logs/a.jsonl", Tool: "claude-code", Size: 1, MtimeNS: 1},
		{Path: "/logs/b.jsonl", Tool: "codex", Size: 1, MtimeNS: 1},
	}); err != nil {
		t.Fatal(err)
	}
	for _, ref := range []SessionRef{{SessionID: "s1"}, {SessionID: "s2"}} {
		if err := s.Mark(ctx, ref, SessionLabel{Task: "feature", MarkedAt: base}); err != nil {
			t.Fatal(err)
		}
	}
	return s
}

func ingestPaths(t *testing.T, s *Store) []string {
	t.Helper()
	state, err := s.IngestState(context.Background(), "v1")
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(state))
	for p := range state {
		out = append(out, p)
	}
	return out
}

// TestClearAllForgetsWhatItRead is B121: emptying usage_record without clearing the
// per-input watermarks leaves every input matching on size/mtime/version, so the next
// backfill reports "unchanged" and imports nothing.
func TestClearAllForgetsWhatItRead(t *testing.T) {
	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	s := seedClearable(t, base)
	if _, err := s.Clear(context.Background(), time.Time{}, ""); err != nil {
		t.Fatal(err)
	}
	if got := ingestPaths(t, s); len(got) != 0 {
		t.Fatalf("ingest watermarks after clear --all = %v, want none (a plain backfill must re-read)", got)
	}
}

func TestClearToolForgetsOnlyThatToolsInputs(t *testing.T) {
	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	s := seedClearable(t, base)
	if _, err := s.Clear(context.Background(), time.Time{}, "codex"); err != nil {
		t.Fatal(err)
	}
	got := ingestPaths(t, s)
	if len(got) != 1 || got[0] != "/logs/a.jsonl" {
		t.Fatalf("ingest watermarks after clear --tool codex = %v, want only claude-code's", got)
	}
}

// TestClearOlderThanKeepsWatermarks pins the deliberate other half: pruning history is a
// request to forget records, not to re-import them on the next run.
func TestClearOlderThanKeepsWatermarks(t *testing.T) {
	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	s := seedClearable(t, base)
	if _, err := s.Clear(context.Background(), base.Add(24*time.Hour), ""); err != nil {
		t.Fatal(err)
	}
	if got := ingestPaths(t, s); len(got) != 2 {
		t.Fatalf("ingest watermarks after a time-scoped clear = %v, want both kept", got)
	}
}

// TestDeleteLabelsHonoursToolScope is B122: --labels beside --tool must not destroy the
// annotations of every other tool, which no re-import can rebuild.
func TestDeleteLabelsHonoursToolScope(t *testing.T) {
	ctx := context.Background()
	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	s := seedClearable(t, base)
	n, err := s.DeleteLabels(ctx, time.Time{}, "codex")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("DeleteLabels(tool=codex) = %d, want 1", n)
	}
	if _, ok, err := s.Label(ctx, SessionRef{SessionID: "s1"}); err != nil || !ok {
		t.Fatalf("claude-code's label survived = %v (err %v), want it kept", ok, err)
	}
}

// TestDeleteLabelsKeepsASessionTheClearOnlyHalfRemoves covers the conservative rule for a
// time scope: a session with records on both sides of the cutoff still exists afterwards,
// so its annotation is not the clear's to delete.
func TestDeleteLabelsKeepsASessionTheClearOnlyHalfRemoves(t *testing.T) {
	ctx := context.Background()
	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	s := seedClearable(t, base)
	n, err := s.DeleteLabels(ctx, base.Add(24*time.Hour), "")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("DeleteLabels(before=+24h) = %d, want 1 (only the fully-pruned session s2)", n)
	}
	if _, ok, _ := s.Label(ctx, SessionRef{SessionID: "s1"}); !ok {
		t.Fatal("s1 spans the cutoff and survives the clear, so its label must survive too")
	}
}

func TestDeleteLabelsUnscopedDeletesEverything(t *testing.T) {
	ctx := context.Background()
	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	s := seedClearable(t, base)
	// A label whose session has no usage rows at all: only the unscoped delete may take it,
	// because no scope can prove it belongs to the tool or window being cleared.
	if err := s.Mark(ctx, SessionRef{SessionID: "orphan"}, SessionLabel{Task: "chore", MarkedAt: base}); err != nil {
		t.Fatal(err)
	}
	n, err := s.DeleteLabels(ctx, time.Time{}, "")
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("DeleteLabels(unscoped) = %d, want 3", n)
	}
}

func TestDeleteLabelsScopedLeavesAnOrphanAlone(t *testing.T) {
	ctx := context.Background()
	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	s := seedClearable(t, base)
	if err := s.Mark(ctx, SessionRef{SessionID: "orphan"}, SessionLabel{Task: "chore", MarkedAt: base}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DeleteLabels(ctx, time.Time{}, "codex"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Label(ctx, SessionRef{SessionID: "orphan"}); !ok {
		t.Fatal("a label with no usage rows cannot be attributed to --tool codex, so it must survive")
	}
}
