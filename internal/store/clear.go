package store

import (
	"context"
	"database/sql"
	"time"
)

// Clear deletes usage records older than before (if non-zero) and/or matching tool (if
// non-empty), returning the number of rows removed.
//
// A clear that is not time-scoped also drops the ingest watermarks of the inputs it just
// unread, so a plain `backfill` rebuilds what was deleted. Without that, every input still
// matches on size/mtime/version and the re-import silently imports nothing. A time-scoped
// clear deliberately keeps them: pruning history is a request to forget records, not to
// re-read them on the next run.
func (s *Store) Clear(ctx context.Context, before time.Time, tool string) (int64, error) {
	hasBefore := !before.IsZero()
	hasTool := tool != ""

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	var res sql.Result
	switch {
	case hasBefore && hasTool:
		res, err = tx.ExecContext(ctx, `DELETE FROM usage_record WHERE ts < ? AND tool = ?`,
			before.UTC().Format(time.RFC3339), tool)
	case hasBefore:
		res, err = tx.ExecContext(ctx, `DELETE FROM usage_record WHERE ts < ?`,
			before.UTC().Format(time.RFC3339))
	case hasTool:
		res, err = tx.ExecContext(ctx, `DELETE FROM usage_record WHERE tool = ?`, tool)
	default:
		res, err = tx.ExecContext(ctx, `DELETE FROM usage_record`)
	}
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if !hasBefore {
		if err := forgetIngested(ctx, tx, tool); err != nil {
			return 0, err
		}
	}
	return n, tx.Commit()
}

// forgetIngested drops the per-input watermarks of tool (or of every tool when empty), so
// the next backfill re-reads those inputs instead of skipping them as unchanged.
func forgetIngested(ctx context.Context, tx *sql.Tx, tool string) error {
	var err error
	if tool == "" {
		_, err = tx.ExecContext(ctx, `DELETE FROM ingest_file`)
	} else {
		_, err = tx.ExecContext(ctx, `DELETE FROM ingest_file WHERE tool = ?`, tool)
	}
	return err
}

// rowsAffected folds the two-step Exec-then-count into one call for the delete paths whose
// return value is a count.
func rowsAffected(res sql.Result, err error) (int64, error) {
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
