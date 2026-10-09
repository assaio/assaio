package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

const migratedDigest = "member-v2-11111111111111111111111111111111"

func legacySyncStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "server.db")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.ExecContext(context.Background(), `DROP TABLE sync_protocol`); err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.ExecContext(context.Background(), `DELETE FROM schema_migration WHERE name = '0015_sync_protocol.sql'`); err != nil {
		t.Fatal(err)
	}
	record := usage.Record{
		Tool: "claude-code", SessionID: "s1", Timestamp: time.Now().UTC(), Model: "m",
		DedupeKey: "alice:k1", Member: "alice", GitBranch: "secret/branch",
		Granularity: "turn", InputTokens: 7,
	}
	if _, err := st.InsertSynced(context.Background(), []usage.Record{record}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.ExecContext(context.Background(), `INSERT INTO usage_record_pre_response_grain SELECT * FROM usage_record`); err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.ExecContext(context.Background(), `INSERT INTO session_label(session_id,member,task,outcome,difficulty,marked_at)
        VALUES ('s1','alice','feature','unknown','unknown',?)`, time.Now().UTC().Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.ExecContext(context.Background(), `INSERT INTO session_step(tool,session_id,timeline,dedupe_key,ts,ordinal,kind,member)
        VALUES ('claude-code','s1','','step-1',?,1,'assistant','alice')`, time.Now().UTC().Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	version, err := st.SyncProtocolVersion(context.Background())
	if err != nil || version != 1 {
		t.Fatalf("upgraded legacy store version=%d err=%v, want v1", version, err)
	}
	return st
}

func TestMigrateSyncIdentityPreservesRowsAndMakesRepushIdempotent(t *testing.T) {
	st := legacySyncStore(t)
	ctx := context.Background()
	change, err := st.MigrateSyncIdentity(ctx, map[string]string{"alice": migratedDigest})
	if err != nil {
		t.Fatal(err)
	}
	if change.Records != 1 || change.Archived != 1 || change.Labels != 1 || change.Steps != 1 {
		t.Fatalf("change = %+v", change)
	}
	version, err := st.SyncProtocolVersion(ctx)
	if err != nil || version != 2 {
		t.Fatalf("version=%d err=%v", version, err)
	}
	recs, err := st.Export(ctx, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].DedupeKey != migratedDigest+":k1" || recs[0].Member != migratedDigest || recs[0].GitBranch != "" || recs[0].InputTokens != 7 {
		t.Fatalf("migrated rows = %+v", recs)
	}
	for _, table := range []string{legacyArchiveTable, "session_label", "session_step"} {
		var member string
		if err := st.db.QueryRowContext(ctx, `SELECT member FROM `+table+` LIMIT 1`).Scan(&member); err != nil || member != migratedDigest {
			t.Fatalf("%s member=%q err=%v", table, member, err)
		}
	}
	var archivedBranch string
	if err := st.db.QueryRowContext(ctx, `SELECT git_branch FROM `+legacyArchiveTable+` LIMIT 1`).Scan(&archivedBranch); err != nil || archivedBranch != "" {
		t.Fatalf("archive branch=%q err=%v", archivedBranch, err)
	}
	repush := recs[0]
	repush.OutputTokens = 11
	if inserted, err := st.InsertSynced(ctx, []usage.Record{repush}); err != nil || inserted != 0 {
		t.Fatalf("re-push inserted=%d err=%v, want zero", inserted, err)
	}
	recs, err = st.Export(ctx, time.Time{})
	if err != nil || len(recs) != 1 || recs[0].OutputTokens != 11 {
		t.Fatalf("re-push rows=%+v err=%v", recs, err)
	}
}

func TestMigrateSyncIdentityRejectsIncompleteAndCollidingMapsWithoutWrites(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mapping map[string]string
		setup   func(*testing.T, *Store)
	}{
		{"missing", map[string]string{}, nil},
		{"invalid digest", map[string]string{"alice": "alice"}, nil},
		{"missing second member", map[string]string{"alice": migratedDigest}, func(t *testing.T, st *Store) {
			t.Helper()
			_, err := st.db.ExecContext(context.Background(), `INSERT INTO session_label(session_id,member,task,outcome,difficulty,marked_at)
                VALUES ('s2','bob','feature','unknown','unknown','2026-10-01T00:00:00Z')`)
			if err != nil {
				t.Fatal(err)
			}
		}},
		{"malformed prefix", map[string]string{"alice": migratedDigest}, func(t *testing.T, st *Store) {
			t.Helper()
			if _, err := st.db.ExecContext(context.Background(), `UPDATE usage_record SET dedupe_key='other:k1' WHERE member='alice'`); err != nil {
				t.Fatal(err)
			}
		}},
		{"unique collision", map[string]string{"alice": migratedDigest}, func(t *testing.T, st *Store) {
			t.Helper()
			if _, err := st.db.ExecContext(context.Background(), `INSERT INTO usage_record(tool,session_id,ts,model,input_tokens,output_tokens,
                cache_read_tokens,cache_write_tokens,reasoning_tokens,dedupe_key)
                VALUES ('claude-code','local','2026-10-01T00:00:00Z','m',1,0,0,0,0,?)`, migratedDigest+":k1"); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := legacySyncStore(t)
			if tc.setup != nil {
				tc.setup(t, st)
			}
			if _, err := st.MigrateSyncIdentity(context.Background(), tc.mapping); err == nil {
				t.Fatal("expected migration refusal")
			}
			version, err := st.SyncProtocolVersion(context.Background())
			if err != nil || version != 1 {
				t.Fatalf("version=%d err=%v", version, err)
			}
			var key, branch string
			if err := st.db.QueryRowContext(context.Background(), `SELECT dedupe_key,git_branch FROM usage_record WHERE member='alice'`).Scan(&key, &branch); err != nil || branch != "secret/branch" {
				t.Fatalf("legacy row key=%q branch=%q err=%v", key, branch, err)
			}
		})
	}
}
