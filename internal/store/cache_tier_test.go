package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

func cacheRec(key string, write, long int64, reason string) usage.Record {
	return usage.Record{
		Tool: "claude-code", SessionID: "s1", Timestamp: time.Now().UTC(),
		Model: "claude-opus-4-5", DedupeKey: key, Granularity: "turn",
		CacheWriteTokens: write, CacheWrite1hTokens: long, CacheMissReason: reason,
	}
}

// The tier and the miss reason survive a round trip and reach the aggregate a report reads.
func TestCacheTierAndMissReasonRoundTrip(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()
	if _, err := st.InsertLocal(ctx, []usage.Record{
		cacheRec("msg_1", 100, 60, "tools_changed"),
		cacheRec("msg_2", 40, 0, ""),
	}); err != nil {
		t.Fatal(err)
	}
	rows, err := st.Usage(ctx, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	var write, long int64
	for i := range rows {
		write += rows[i].CacheWrite
		long += rows[i].CacheWrite1h
	}
	if write != 140 || long != 60 {
		t.Errorf("cache write = %d (1h %d), want 140 (1h 60)", write, long)
	}

	misses, err := st.CacheMisses(ctx, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(misses) != 1 || misses[0].Reason != "tools_changed" || misses[0].Turns != 1 {
		t.Fatalf("CacheMisses = %+v, want one tools_changed turn", misses)
	}
}

// A turn that hit cache states no reason. Counting it under an empty reason would make
// "no reason given" look like a cause of its own.
func TestCacheMissesExcludesTurnsWithNoStatedReason(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()
	if _, err := st.InsertLocal(ctx, []usage.Record{cacheRec("msg_1", 10, 0, "")}); err != nil {
		t.Fatal(err)
	}
	misses, err := st.CacheMisses(ctx, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(misses) != 0 {
		t.Errorf("CacheMisses = %+v, want none", misses)
	}
}

// History parsed by a build that could not read the tier gains it on a re-read of the same
// file, which is how a local store heals rather than keeping zeros forever.
func TestARereadFillsInTheCacheTierOnStoredHistory(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()
	if _, err := st.InsertLocal(ctx, []usage.Record{cacheRec("msg_1", 100, 0, "")}); err != nil {
		t.Fatal(err)
	}
	// The same turn, re-read by a build that now extracts both fields.
	if _, err := st.InsertLocal(ctx, []usage.Record{cacheRec("msg_1", 100, 60, "model_changed")}); err != nil {
		t.Fatal(err)
	}
	rows, err := st.Usage(ctx, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].CacheWrite1h != 60 {
		t.Fatalf("rows = %+v, want one row whose 1h portion restated to 60", rows)
	}
	misses, err := st.CacheMisses(ctx, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(misses) != 1 || misses[0].Reason != "model_changed" {
		t.Fatalf("CacheMisses = %+v, want the restated reason", misses)
	}
}

// Migration 0008 clears every claude-code row the old per-line grain inflated, local and
// synced alike: nothing will ever collide with their uuid keys, so keeping a member's synced
// rows would leave a team server reporting the inflated total plus the rebuilt one. Another
// tool's rows are untouched. The watermarks go with the rows, because a source build's
// identity is constant and a plain backfill would otherwise find every input unchanged and
// never rebuild what this deleted.
func TestResponseGrainMigrationClearsClaudeRowsAndTheirWatermarks(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()
	rows := []usage.Record{
		{Tool: "claude-code", SessionID: "s", Timestamp: time.Now().UTC(), Model: "m", DedupeKey: "old-uuid", Granularity: "turn", OutputTokens: 9},
		{Tool: "claude-code", SessionID: "s", Timestamp: time.Now().UTC(), Model: "m", DedupeKey: "synced-uuid", Granularity: "turn", OutputTokens: 9, Member: "bob"},
		{Tool: "codex", SessionID: "s", Timestamp: time.Now().UTC(), Model: "m", DedupeKey: "codex-1", Granularity: "turn", OutputTokens: 9},
	}
	if _, err := st.Insert(ctx, rows); err != nil {
		t.Fatal(err)
	}
	if err := st.RecordIngest(ctx, "dev", time.Now().UTC(), []IngestEntry{
		{Path: "/t/a.jsonl", Tool: "claude-code", Size: 1, MtimeNS: 1},
		{Path: "/t/b.jsonl", Tool: "codex", Size: 1, MtimeNS: 1},
	}); err != nil {
		t.Fatal(err)
	}
	// The migration ran at Open, before these rows existed; re-running it is what the test is.
	body, err := migrations.ReadFile("migrations/0008_response_grain_claude.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.ExecContext(ctx, string(body)); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		key  string
		want bool
	}{
		{"old-uuid", false},
		{"synced-uuid", false},
		{"codex-1", true},
	} {
		var n int
		row := st.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM usage_record WHERE dedupe_key = ?`, tc.key)
		if err := row.Scan(&n); err != nil {
			t.Fatal(err)
		}
		if kept := n > 0; kept != tc.want {
			t.Errorf("row %q kept = %v, want %v", tc.key, kept, tc.want)
		}
	}
	// A watermark that survived would make the next plain backfill skip the transcript it
	// points at, leaving the rows above deleted and never rebuilt.
	state, err := st.IngestState(ctx, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := state["/t/a.jsonl"]; ok {
		t.Error("claude-code watermark survived; a plain backfill would skip the file and never rebuild it")
	}
	if _, ok := state["/t/b.jsonl"]; !ok {
		t.Error("another tool's watermark was cleared; only claude-code needs re-reading")
	}
}

// A response's blocks arrive across several lines and only the last carries its true output
// count, so a session read while it was still being written stores a partial figure under a
// response id that already exists. Without a token restate the completed read has nothing to
// insert and the undercount is permanent.
func TestARereadCorrectsAPartialOutputCount(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()
	partial := usage.Record{
		Tool: "claude-code", SessionID: "s", Timestamp: time.Now().UTC(), Model: "m",
		DedupeKey: "msg_1", Granularity: "turn", OutputTokens: 2,
		CacheWriteTokens: 2400, CacheWrite1hTokens: 2400,
	}
	if _, err := st.InsertLocal(ctx, []usage.Record{partial}); err != nil {
		t.Fatal(err)
	}
	complete := partial
	complete.OutputTokens = 158
	if _, err := st.InsertLocal(ctx, []usage.Record{complete}); err != nil {
		t.Fatal(err)
	}
	rows, err := st.Usage(ctx, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Out != 158 {
		t.Fatalf("rows = %+v, want one row whose output restated to 158", rows)
	}
	// The 1-hour portion is restated too, so it must never outgrow the write it is part of.
	if rows[0].CacheWrite1h > rows[0].CacheWrite {
		t.Errorf("1h portion %d exceeds the cache write %d it is part of",
			rows[0].CacheWrite1h, rows[0].CacheWrite)
	}
}

// The upgrade moves the pre-response-grain rows aside instead of destroying them: the rebuild
// only reaches as far back as the transcripts that still exist, and Claude Code rotates its
// own logs, so a silent DELETE would take history nothing could restore.
func TestUpgradeArchivesRatherThanDestroysClaudeHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.db")
	ctx := context.Background()

	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Insert(ctx, []usage.Record{
		{Tool: "claude-code", SessionID: "s", Timestamp: time.Now().UTC(), Model: "m", DedupeKey: "line-uuid", Granularity: "turn", OutputTokens: 9},
	}); err != nil {
		t.Fatal(err)
	}
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

	rows, err := up.LegacyArchiveRows(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if rows != 1 {
		t.Fatalf("archive holds %d rows, want the pre-0.12 row kept aside", rows)
	}
	if err := up.DropLegacyArchive(ctx); err != nil {
		t.Fatal(err)
	}
	if rows, err := up.LegacyArchiveRows(ctx); err != nil || rows != 0 {
		t.Errorf("archive holds %d rows after drop (err %v), want 0", rows, err)
	}
}

// A store created after 0.12 never has an archive, and asking about one is not an error.
func TestAFreshStoreHasNoLegacyArchive(t *testing.T) {
	st := newStore(t)
	rows, err := st.LegacyArchiveRows(context.Background())
	if err != nil || rows != 0 {
		t.Errorf("LegacyArchiveRows = (%d, %v), want (0, nil)", rows, err)
	}
}
