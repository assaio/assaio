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
	if err := clearSteps(ctx, tx, before, tool); err != nil {
		return 0, err
	}
	if !hasBefore {
		if err := forgetIngested(ctx, tx, tool); err != nil {
			return 0, err
		}
	}
	// A digest compares this window against what the last run reported. Rows deleted here are
	// not a change in how the tools were used, but nothing downstream can tell the difference
	// -- so the comparison basis goes with the records it described, and the next digest says
	// it has none rather than reporting the deletion as movement.
	if _, err := tx.ExecContext(ctx, `DELETE FROM digest_snapshot`); err != nil {
		return 0, err
	}
	return n, tx.Commit()
}

// clearSteps erases the step timeline under the same scope as the records. Without it a
// `clear` left every step behind -- session, timestamp, ordinal, kind, outcome, model and
// tokens for the whole history -- and the next backfill's INSERT OR IGNORE found them already
// present and printed steps=0, confirming to the person who asked to forget their history that
// there was nothing there.
func clearSteps(ctx context.Context, tx *sql.Tx, before time.Time, tool string) error {
	var err error
	switch {
	case !before.IsZero() && tool != "":
		_, err = tx.ExecContext(ctx, `DELETE FROM session_step WHERE ts < ? AND tool = ?`,
			before.UTC().Format(time.RFC3339), tool)
	case !before.IsZero():
		_, err = tx.ExecContext(ctx, `DELETE FROM session_step WHERE ts < ?`,
			before.UTC().Format(time.RFC3339))
	case tool != "":
		_, err = tx.ExecContext(ctx, `DELETE FROM session_step WHERE tool = ?`, tool)
	default:
		_, err = tx.ExecContext(ctx, `DELETE FROM session_step`)
	}
	return err
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
