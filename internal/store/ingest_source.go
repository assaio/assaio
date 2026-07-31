package store

import (
	"context"
	"time"
)

// maxSourceRuns is how many ingest runs this table retains per tool. The drift canaries
// need a baseline, not an archive, so anything older is pruned on write -- which is what
// keeps the table's size independent of how long assaio has been installed.
const maxSourceRuns = 10

// SourceRun is one source's ingest pass: what was on disk (Discovered), what this run
// actually read (Parsed), and what came out of it. Ratios derived from these are the
// baseline a later run is compared against, so a run that parsed almost nothing stays
// comparable to one that parsed everything.
type SourceRun struct {
	Tool                                            string
	RanAt                                           time.Time
	Discovered, Parsed, Records, Skipped, ZeroToken int
}

// RecordSourceRun appends one row per source and prunes every touched tool back to
// maxSourceRuns, both in one transaction: the bound is enforced at write time rather than
// left to a cleanup that never runs.
func (s *Store) RecordSourceRun(ctx context.Context, runs []SourceRun) error {
	if len(runs) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	insert, err := tx.PrepareContext(ctx, `
        INSERT INTO ingest_source (tool, ran_at, discovered, parsed, records, skipped, zero_token)
        VALUES (?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer func() { _ = insert.Close() }()
	prune, err := tx.PrepareContext(ctx, `
        DELETE FROM ingest_source WHERE tool = ? AND id NOT IN (
            SELECT id FROM ingest_source WHERE tool = ? ORDER BY id DESC LIMIT ?)`)
	if err != nil {
		return err
	}
	defer func() { _ = prune.Close() }()
	for i := range runs {
		r := &runs[i]
		stamp := r.RanAt.UTC().Format(time.RFC3339)
		if _, err := insert.ExecContext(ctx, r.Tool, stamp,
			r.Discovered, r.Parsed, r.Records, r.Skipped, r.ZeroToken); err != nil {
			return err
		}
		if _, err := prune.ExecContext(ctx, r.Tool, r.Tool, maxSourceRuns); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SourceHistory returns every tool's retained ingest runs, oldest first, so the last
// element of a tool's slice is its most recent run -- the ordering the drift canaries
// read as "current" against everything before it.
func (s *Store) SourceHistory(ctx context.Context) (map[string][]SourceRun, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT tool, ran_at, discovered, parsed, records, skipped, zero_token
        FROM ingest_source ORDER BY tool, id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make(map[string][]SourceRun)
	for rows.Next() {
		var r SourceRun
		var stamp string
		if err := rows.Scan(&r.Tool, &stamp, &r.Discovered, &r.Parsed,
			&r.Records, &r.Skipped, &r.ZeroToken); err != nil {
			return nil, err
		}
		r.RanAt, _ = time.Parse(time.RFC3339, stamp)
		out[r.Tool] = append(out[r.Tool], r)
	}
	return out, rows.Err()
}
