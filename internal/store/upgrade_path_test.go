package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// The upgrade a real user performs: a store written by the per-line parser, then opened by
// this build. Open must clear both the inflated rows and the watermarks that would otherwise
// let the next plain backfill skip every transcript and never rebuild them. Exercised through
// Open rather than by running the migration body by hand, because the runner is the part that
// has to work.
func TestOpeningAPreResponseGrainStoreClearsItForRebuild(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.db")
	ctx := context.Background()

	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Insert(ctx, []usage.Record{
		{Tool: "claude-code", SessionID: "s", Timestamp: time.Now().UTC(), Model: "m", DedupeKey: "line-uuid", Granularity: "turn", OutputTokens: 9},
		{Tool: "codex", SessionID: "s", Timestamp: time.Now().UTC(), Model: "m", DedupeKey: "codex-1", Granularity: "turn", OutputTokens: 9},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.RecordIngest(ctx, "dev", time.Now().UTC(), []IngestEntry{
		{Path: "/t/a.jsonl", Tool: "claude-code", Size: 1, MtimeNS: 1},
		{Path: "/t/b.jsonl", Tool: "codex", Size: 1, MtimeNS: 1},
	}); err != nil {
		t.Fatal(err)
	}
	// Rewind to "0008 has not run here yet" -- the state every pre-0.12 store is in.
	if _, err := st.db.ExecContext(ctx,
		`DELETE FROM schema_migration WHERE name = '0008_response_grain_claude.sql'`); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	up, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = up.Close() }()

	rows, err := up.Usage(ctx, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if rows[i].Tool == "claude-code" {
			t.Errorf("claude-code row survived the upgrade: %+v", rows[i])
		}
	}
	if len(rows) != 1 || rows[0].Tool != "codex" {
		t.Errorf("rows = %+v, want only the codex row", rows)
	}
	state, err := up.IngestState(ctx, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := state["/t/a.jsonl"]; ok {
		t.Error("claude-code watermark survived; the next plain backfill would skip the transcript and leave the history empty")
	}
	if _, ok := state["/t/b.jsonl"]; !ok {
		t.Error("codex watermark was cleared; only claude-code needs re-reading")
	}
}
