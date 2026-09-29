package store

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"sort"
	"testing"
)

// migrateThrough applies the embedded migrations up to and including last to db, the way a
// store written by the release that shipped last looks on disk.
func migrateThrough(t *testing.T, db *sql.DB, last string) {
	t.Helper()
	ctx := context.Background()
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	if _, err := db.ExecContext(ctx, `CREATE TABLE schema_migration (name TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if name > last {
			break
		}
		body, err := migrations.ReadFile("migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, string(body)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO schema_migration(name) VALUES (?)`, name); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUpgradeFromV0340LeavesRowsUnresolvedAndRereadsSourcesWithADirectory(t *testing.T) {
	for _, dropped := range []bool{false, true} {
		name := "with the legacy archive"
		if dropped {
			name = "after the legacy archive was dropped"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			path := filepath.Join(t.TempDir(), "v0.34.0.db")
			db, err := sql.Open("sqlite", storeDSN(path))
			if err != nil {
				t.Fatal(err)
			}
			migrateThrough(t, db, projectConflictMigration)
			if dropped {
				if _, err := db.ExecContext(ctx, `DROP TABLE usage_record_pre_response_grain`); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := db.ExecContext(ctx, `
                WITH RECURSIVE rows(n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM rows WHERE n < 20000)
                INSERT INTO usage_record
                    (tool, session_id, ts, model, input_tokens, output_tokens, cache_read_tokens,
                     cache_write_tokens, reasoning_tokens, dedupe_key, project, granularity)
                SELECT 'claude-code', 's1', '2026-09-01T09:00:00Z', 'm', 10, 5, 0, 0, 0,
                       printf('k%d', n), 'api', 'turn'
                FROM rows`); err != nil {
				t.Fatal(err)
			}
			for _, tool := range []string{"claude-code", "codex", "copilot-cli", "gemini-cli", "cline"} {
				if _, err := db.ExecContext(ctx, `
                    INSERT INTO ingest_file (path, tool, size, mtime_ns, version, ingested_at)
                    VALUES (?, ?, 1, 1, 'dev', '2026-09-01T09:00:00Z')`, "/logs/"+tool, tool); err != nil {
					t.Fatal(err)
				}
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			before := fileSize(t, path)

			st, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}

			var resolved, archived int
			if err := st.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM usage_record WHERE repo_id <> 0`).Scan(&resolved); err != nil || resolved != 0 {
				t.Fatalf("rows with a repository after upgrade = %d, %v; want 0", resolved, err)
			}
			if err := st.db.QueryRowContext(ctx, `SELECT COUNT(repo_id) FROM usage_record_pre_response_grain`).Scan(&archived); err != nil {
				t.Fatalf("archive lacks repo_id: %v", err)
			}
			if salt, err := st.RepositorySalt(ctx); err != nil || len(salt) != 32 {
				t.Fatalf("salt = %d bytes, %v; want 32", len(salt), err)
			}
			if got := shownTokens(t, st); len(got) != 1 || got["api"] != 200000 {
				t.Fatalf("shown = %v, want every row still under api", got)
			}
			rows, err := st.db.QueryContext(ctx, `SELECT tool FROM ingest_file ORDER BY tool`)
			if err != nil {
				t.Fatal(err)
			}
			var kept []string
			for rows.Next() {
				var tool string
				if err := rows.Scan(&tool); err != nil {
					t.Fatal(err)
				}
				kept = append(kept, tool)
			}
			_ = rows.Close()
			if len(kept) != 2 || kept[0] != "cline" || kept[1] != "gemini-cli" {
				t.Fatalf("watermarks kept = %v, want only the sources that record no directory", kept)
			}
			if err := st.Close(); err != nil {
				t.Fatal(err)
			}
			if growth := fileSize(t, path) - before; growth > 64*1024 {
				t.Fatalf("migration grew a 20,000-row store by %d bytes; want at most 64 KiB", growth)
			}
		})
	}
}
