package store

import "context"

// Size is the database file's page accounting: what it occupies, and how much of that is
// space SQLite freed but is holding on to. Deleting rows only ever moves bytes from the
// first number to the second -- nothing returns them to the filesystem but Vacuum.
type Size struct {
	Bytes, Reclaimable int64
}

// Size reports the store's on-disk footprint and its reclaimable share, so growth is
// legible before it becomes a surprise.
func (s *Store) Size(ctx context.Context) (Size, error) {
	var pageCount, pageSize, freelist int64
	for _, q := range []struct {
		sql string
		dst *int64
	}{
		{`PRAGMA page_count`, &pageCount},
		{`PRAGMA page_size`, &pageSize},
		{`PRAGMA freelist_count`, &freelist},
	} {
		if err := s.db.QueryRowContext(ctx, q.sql).Scan(q.dst); err != nil {
			return Size{}, err
		}
	}
	return Size{Bytes: pageCount * pageSize, Reclaimable: freelist * pageSize}, nil
}

// Vacuum rewrites the database without its free pages and truncates the write-ahead log,
// which is the only thing that actually shrinks the file. It is deliberately never
// implicit: rewriting needs roughly twice the store's size in temporary disk space, a cost
// no ordinary command should impose without being asked.
func (s *Store) Vacuum(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `VACUUM`); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
	return err
}

// PruneIngestState drops the tool's per-input rows whose path is not in keep, returning
// how many were removed. Vendors rotate their own logs, so without this the table would
// accumulate a row for every transcript ever seen and grow with install age rather than
// with what is actually on disk.
func (s *Store) PruneIngestState(ctx context.Context, tool string, keep map[string]bool) (int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT path FROM ingest_file WHERE tool = ?`, tool)
	if err != nil {
		return 0, err
	}
	var gone []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			_ = rows.Close()
			return 0, err
		}
		if !keep[path] {
			gone = append(gone, path)
		}
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return s.deletePaths(ctx, gone)
}

// deletePaths removes one batch of ingest_file rows by path in a single transaction.
func (s *Store) deletePaths(ctx context.Context, paths []string) (int, error) {
	if len(paths) == 0 {
		return 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.PrepareContext(ctx, `DELETE FROM ingest_file WHERE path = ?`)
	if err != nil {
		return 0, err
	}
	defer func() { _ = stmt.Close() }()
	for _, p := range paths {
		if _, err := stmt.ExecContext(ctx, p); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(paths), nil
}
