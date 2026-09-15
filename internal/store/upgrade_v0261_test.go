package store

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

const projectConflictMigration = "0013_project_conflict.sql"

func TestUpgradeFromV0261AddsProjectConflictWithoutChangingRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v0.26.1.db")
	db, err := sql.Open("sqlite", storeDSN(path))
	if err != nil {
		t.Fatal(err)
	}
	createV0261Schema(t, db)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `
        WITH RECURSIVE rows(n) AS (
            SELECT 1
            UNION ALL
            SELECT n + 1 FROM rows WHERE n < 20000
        )
        INSERT INTO usage_record
            (tool, session_id, ts, model, input_tokens, output_tokens,
             cache_read_tokens, cache_write_tokens, reasoning_tokens,
             dedupe_key, project, granularity)
		SELECT 'claude-code', 's1', '2026-09-01T09:00:00Z', 'm', 10, 5,
		       0, 0, 0, printf('agent:%d', n), 'alpha', 'session'
		FROM rows`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	before := fileSize(t, path)

	upgraded, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if count, err := upgraded.Count(context.Background()); err != nil || count != 20000 {
		t.Fatalf("Count() = %d, %v; want 20000, nil", count, err)
	}
	var project string
	var conflict int
	err = upgraded.db.QueryRowContext(context.Background(),
		`SELECT project, project_conflict FROM usage_record WHERE dedupe_key = 'agent:1'`).
		Scan(&project, &conflict)
	if err != nil {
		t.Fatal(err)
	}
	if project != "alpha" || conflict != 0 {
		t.Fatalf("upgraded row = (%q, %d), want (alpha, 0)", project, conflict)
	}
	if err := upgraded.Close(); err != nil {
		t.Fatal(err)
	}
	after := fileSize(t, path)
	growth := after - before
	t.Logf("v0.26.1-shaped store growth: %d bytes", growth)
	if growth > 64*1024 {
		t.Fatalf("migration grew a 20,000-row store by %d bytes; want at most 64 KiB", growth)
	}
}

func fileSize(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Size()
}

func createV0261Schema(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE schema_migration (name TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == projectConflictMigration {
			break
		}
		body, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, string(body)); err != nil {
			t.Fatalf("apply %s: %v", entry.Name(), err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO schema_migration(name) VALUES (?)`, entry.Name()); err != nil {
			t.Fatal(err)
		}
	}
}
