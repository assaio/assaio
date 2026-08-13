package store

import (
	"context"
	"time"
)

// HistoryStart is the earliest observation the store holds, ignoring any window. The zero time
// means the store is empty.
//
// It answers a question no windowed query can: whether the span a trend compares against existed
// to be compared. A window of 30 days over a store holding 12 reports growth from a base that was
// never there, and on a fresh install -- or after a source deletes its own transcripts, which
// Claude Code does at 30 days by default -- that is the ordinary case rather than the odd one
// (B156).
// tool narrows it to one source, "" spans every one. The narrowing is not a convenience: each
// source keeps its own logs for its own length of time, so comparing one tool's retention against
// every tool's span answers about a set that was never measured.
func (s *Store) HistoryStart(ctx context.Context, tool string) (time.Time, error) {
	query := `SELECT MIN(ts) FROM usage_record`
	var args []any
	if tool != "" {
		query += ` WHERE tool = ?`
		args = append(args, tool)
	}
	var oldest *string
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&oldest); err != nil {
		return time.Time{}, err
	}
	return parseStoredTime(oldest), nil
}
