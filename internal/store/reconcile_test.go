package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// oldSchemaDDL stands in for the very first shipped usage_record shape: missing subpath,
// every activity counter, compactions, and rework_lines.
const oldSchemaDDL = `CREATE TABLE usage_record (
    id                 INTEGER PRIMARY KEY,
    tool               TEXT    NOT NULL,
    session_id         TEXT    NOT NULL,
    ts                 TEXT    NOT NULL,
    model              TEXT    NOT NULL,
    input_tokens       INTEGER NOT NULL,
    output_tokens      INTEGER NOT NULL,
    cache_read_tokens  INTEGER NOT NULL,
    cache_write_tokens INTEGER NOT NULL,
    reasoning_tokens   INTEGER NOT NULL,
    dedupe_key         TEXT    NOT NULL,
    project            TEXT    NOT NULL DEFAULT '',
    git_branch         TEXT    NOT NULL DEFAULT '',
    entrypoint         TEXT    NOT NULL DEFAULT '',
    granularity        TEXT    NOT NULL DEFAULT 'turn',
    UNIQUE(tool, dedupe_key)
)`

// seedOldSchemaDB creates, at path, a database whose usage_record predates every
// post-original column and whose schema_migration already records 0001_init.sql as
// applied. That is exactly the state migrate() leaves behind for a database created by
// an older build once 0001_init.sql has since been edited in place: migrate() sees
// "0001_init.sql" already applied and skips it, so the edited CREATE TABLE never runs.
func seedOldSchemaDB(t *testing.T, path string) {
	t.Helper()
	ctx := context.Background()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	for _, stmt := range []string{
		`CREATE TABLE schema_migration (name TEXT PRIMARY KEY)`,
		`INSERT INTO schema_migration(name) VALUES ('0001_init.sql')`,
		oldSchemaDDL,
	} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("seed old schema, statement %q: %v", stmt, err)
		}
	}
}

// TestReconcileColumnsHealsPreReleaseSchemaDrift reproduces the maintainer's bug end to
// end: an upgraded database still on the pre-activity-columns schema must come up
// working, not crash with "no such column".
func TestReconcileColumnsHealsPreReleaseSchemaDrift(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	seedOldSchemaDB(t, path)

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open() on partial-schema db: %v", err)
	}
	defer func() { _ = s.Close() }()
	ctx := context.Background()

	rec := usage.Record{
		Tool: "claude-code", SessionID: "s1", Timestamp: time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC),
		Model: "m", InputTokens: 1, DedupeKey: "heal-check",
		Subpath: "apps/api", LinesAdded: 4, LinesRemoved: 1, Edits: 1, ToolCalls: 2, Rejected: 1,
		Compactions: 1, ReworkLines: 3, Member: "alice",
	}
	if _, err := s.Insert(ctx, []usage.Record{rec}); err != nil {
		t.Fatalf("Insert after reconcile: %v", err)
	}

	rows, err := s.Usage(ctx, rec.Timestamp.Add(-time.Hour))
	if err != nil {
		t.Fatalf("Usage query using healed columns: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("Usage = %+v, want 1 row", rows)
	}
	row := rows[0]
	if row.Compactions != 1 || row.ReworkLines != 3 || row.Edits != 1 || row.ToolCalls != 2 || row.Rejected != 1 {
		t.Fatalf("healed activity sums = %+v, want Compactions=1 ReworkLines=3 Edits=1 ToolCalls=2 Rejected=1", row)
	}

	var subpath, member string
	err = s.db.QueryRowContext(ctx, `SELECT subpath, member FROM usage_record WHERE dedupe_key = ?`, "heal-check").
		Scan(&subpath, &member)
	if err != nil {
		t.Fatalf("SELECT subpath, member after reconcile: %v", err)
	}
	if subpath != "apps/api" {
		t.Fatalf("subpath = %q, want apps/api", subpath)
	}
	if member != "alice" {
		t.Fatalf("member = %q, want alice", member)
	}
}

// TestReconcileColumnsIsIdempotent calls reconcileColumns repeatedly on an already-healed
// database (Open already ran it once) and requires every extra call to be a silent no-op,
// not a "duplicate column name" error.
func TestReconcileColumnsIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	seedOldSchemaDB(t, path)
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open() on partial-schema db: %v", err)
	}
	defer func() { _ = s.Close() }()
	ctx := context.Background()

	before, err := existingColumns(ctx, s.db)
	if err != nil {
		t.Fatal(err)
	}
	for i := range 2 {
		if err := reconcileColumns(ctx, s.db); err != nil {
			t.Fatalf("reconcileColumns() call #%d: %v, want nil (must be idempotent)", i+2, err)
		}
	}
	after, err := existingColumns(ctx, s.db)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != len(after) {
		t.Fatalf("column count changed across idempotent re-runs: before=%d after=%d", len(before), len(after))
	}
}

// TestReconcileColumnsNoopOnCurrentSchema requires a database already on the current
// schema to come out of reconcileColumns byte-for-byte unchanged.
func TestReconcileColumnsNoopOnCurrentSchema(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	before, err := existingColumns(ctx, s.db)
	if err != nil {
		t.Fatal(err)
	}
	if err := reconcileColumns(ctx, s.db); err != nil {
		t.Fatalf("reconcileColumns() on current schema: %v, want nil", err)
	}
	after, err := existingColumns(ctx, s.db)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != len(after) {
		t.Fatalf("column count changed on a current-schema db: before=%d after=%d", len(before), len(after))
	}
}

// TestSchemaColumnsMatchesInitSQL pins schemaColumns' parse of 0001_init.sql to the
// column list that file is currently known to declare, so a parsing regression fails
// loudly here instead of silently under-reconciling in production.
func TestSchemaColumnsMatchesInitSQL(t *testing.T) {
	cols, err := schemaColumns()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"id", "tool", "session_id", "ts", "model", "input_tokens", "output_tokens",
		"cache_read_tokens", "cache_write_tokens", "reasoning_tokens", "dedupe_key",
		"project", "subpath", "git_branch", "entrypoint", "granularity",
		"lines_added", "lines_removed", "edits", "tool_calls", "rejected", "compactions", "rework_lines",
		"member",
	}
	if len(cols) != len(want) {
		t.Fatalf("schemaColumns() = %d columns, want %d: %+v", len(cols), len(want), cols)
	}
	for i, c := range cols {
		if c.name != want[i] {
			t.Fatalf("schemaColumns()[%d].name = %q, want %q", i, c.name, want[i])
		}
	}
}

// TestParseColumnLine guards the ADD COLUMN def text against collapsing: unlike a
// strings.Fields-and-rejoin split, it must preserve whitespace inside a quoted DEFAULT
// verbatim, since that text is later spliced straight into an ALTER TABLE statement.
func TestParseColumnLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantCol column
		wantOK  bool
	}{
		{
			name:    "aligned column with trailing comma",
			line:    "    id                 INTEGER PRIMARY KEY,",
			wantCol: column{name: "id", def: "INTEGER PRIMARY KEY"},
			wantOK:  true,
		},
		{
			name:    "internal whitespace inside a quoted default is preserved, not collapsed",
			line:    "    note               TEXT    NOT NULL DEFAULT 'a  b',",
			wantCol: column{name: "note", def: "TEXT    NOT NULL DEFAULT 'a  b'"},
			wantOK:  true,
		},
		{
			name:   "blank line",
			line:   "   ",
			wantOK: false,
		},
		{
			name:   "UNIQUE table constraint is not a column",
			line:   "    UNIQUE(tool, dedupe_key)",
			wantOK: false,
		},
		{
			name:    "name with no def",
			line:    "bareword",
			wantCol: column{name: "bareword"},
			wantOK:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col, ok := parseColumnLine(tt.line)
			if ok != tt.wantOK {
				t.Fatalf("parseColumnLine(%q) ok = %v, want %v", tt.line, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if col != tt.wantCol {
				t.Fatalf("parseColumnLine(%q) = %+v, want %+v", tt.line, col, tt.wantCol)
			}
		})
	}
}
