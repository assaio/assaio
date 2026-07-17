package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// reconcileColumns adds any usage_record column that 0001_init.sql defines but an
// already-migrated database is still missing. Pre-1.0, schema changes are made by
// editing 0001_init.sql in place (see RELEASING.md), so a database that recorded
// "0001_init.sql" as applied under an older build never gets the columns added since;
// this self-heals it. Post-1.0, schema changes ship as new immutable migration files
// and this function becomes a no-op for every column.
func reconcileColumns(ctx context.Context, db *sql.DB) error {
	want, err := schemaColumns()
	if err != nil {
		return err
	}
	have, err := existingColumns(ctx, db)
	if err != nil {
		return err
	}
	for _, col := range want {
		if have[col.name] {
			continue
		}
		// col.name/col.def come from our own embedded migrations/0001_init.sql, never
		// from external input.
		stmt := fmt.Sprintf(`ALTER TABLE usage_record ADD COLUMN %s %s`, col.name, col.def)
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("add column %s: %w", col.name, err)
		}
	}
	return nil
}

// existingColumns returns the set of column names usage_record actually has.
func existingColumns(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(usage_record)`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	have := make(map[string]bool)
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		have[name] = true
	}
	return have, rows.Err()
}

// column is one usage_record column as declared in 0001_init.sql: its name and the
// rest of its definition (type plus constraints), verbatim for use after ADD COLUMN.
type column struct{ name, def string }

// schemaColumns parses the usage_record column list out of the embedded 0001_init.sql,
// so "expected schema" has one source of truth instead of a second hand-maintained list
// that could drift from it.
func schemaColumns() ([]column, error) {
	raw, err := migrations.ReadFile("migrations/0001_init.sql")
	if err != nil {
		return nil, err
	}
	body := string(raw)
	const open = "usage_record ("
	start := strings.Index(body, open)
	if start < 0 {
		return nil, fmt.Errorf("schemaColumns: %q not found in 0001_init.sql", open)
	}
	start += len(open)
	end := strings.Index(body[start:], "\n);")
	if end < 0 {
		return nil, errors.New(`schemaColumns: closing ")" not found in 0001_init.sql`)
	}

	var cols []column
	for _, line := range strings.Split(body[start:start+end], "\n") {
		if col, ok := parseColumnLine(line); ok {
			cols = append(cols, col)
		}
	}
	return cols, nil
}

// parseColumnLine parses one usage_record column-definition line from 0001_init.sql into
// a column, reporting false for a blank line or a UNIQUE(...) table constraint. def keeps
// everything after the column name verbatim, save for trimming the whitespace that
// separates them -- splitting on strings.Fields and rejoining, as this used to, would
// collapse any internal whitespace inside a quoted DEFAULT instead of preserving it.
func parseColumnLine(line string) (column, bool) {
	line = strings.TrimSuffix(strings.TrimSpace(line), ",")
	if line == "" || strings.HasPrefix(strings.ToUpper(line), "UNIQUE") {
		return column{}, false
	}
	i := strings.IndexFunc(line, unicode.IsSpace)
	if i < 0 {
		return column{name: line}, true
	}
	return column{name: line[:i], def: strings.TrimSpace(line[i:])}, true
}
