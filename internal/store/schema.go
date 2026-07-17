package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
)

//go:embed migrations/*.sql
var migrations embed.FS

// migrate applies any embedded migration files not yet recorded in schema_migration, in
// filename order.
func migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migration (name TEXT PRIMARY KEY)`); err != nil {
		return err
	}
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		var seen string
		err := db.QueryRowContext(ctx, `SELECT name FROM schema_migration WHERE name = ?`, name).Scan(&seen)
		if err == nil {
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		body, err := migrations.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, string(body)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO schema_migration(name) VALUES (?)`, name); err != nil {
			return err
		}
	}
	return nil
}
