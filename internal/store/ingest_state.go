package store

import (
	"context"
	"time"
)

// IngestEntry identifies one parsed input: the path assaio read, the tool it belongs to,
// and the size and modification time it carried when parsed. A Cline entry's Path is the
// task directory, signed by the sum and maximum across the files inside it.
type IngestEntry struct {
	Path, Tool string
	Size       int64
	MtimeNS    int64
}

// IngestState returns every input already parsed by the build identified by version,
// keyed by path. Rows written by any other build are omitted rather than returned, so an
// upgraded binary re-parses everything once -- the pass that lets Insert's restateSignals
// fill signals an older parser could not extract.
func (s *Store) IngestState(ctx context.Context, version string) (map[string]IngestEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT path, tool, size, mtime_ns FROM ingest_file WHERE version = ?`, version)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make(map[string]IngestEntry)
	for rows.Next() {
		var e IngestEntry
		if err := rows.Scan(&e.Path, &e.Tool, &e.Size, &e.MtimeNS); err != nil {
			return nil, err
		}
		out[e.Path] = e
	}
	return out, rows.Err()
}

// RecordIngest upserts one row per input parsed in this run, stamping each with version
// and at. Unlike usage_record, this table is a cache of what was read rather than a
// record of usage, so last-write-wins is the correct conflict policy here.
func (s *Store) RecordIngest(ctx context.Context, version string, at time.Time, entries []IngestEntry) error {
	if len(entries) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO ingest_file (path, tool, size, mtime_ns, version, ingested_at)
        VALUES (?,?,?,?,?,?)
        ON CONFLICT(path) DO UPDATE SET
            tool = excluded.tool, size = excluded.size, mtime_ns = excluded.mtime_ns,
            version = excluded.version, ingested_at = excluded.ingested_at`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()
	stamp := at.UTC().Format(time.RFC3339)
	for i := range entries {
		e := &entries[i]
		if _, err := stmt.ExecContext(ctx, e.Path, e.Tool, e.Size, e.MtimeNS, version, stamp); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// IngestFreshness returns the most recent ingest time per tool, answering "how current is
// this number" for statusline and doctor. A tool absent from the map has never been
// ingested by any build.
func (s *Store) IngestFreshness(ctx context.Context) (map[string]time.Time, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT tool, MAX(ingested_at) FROM ingest_file GROUP BY tool`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make(map[string]time.Time)
	for rows.Next() {
		var tool, stamp string
		if err := rows.Scan(&tool, &stamp); err != nil {
			return nil, err
		}
		t, err := time.Parse(time.RFC3339, stamp)
		if err != nil {
			continue
		}
		out[tool] = t
	}
	return out, rows.Err()
}
