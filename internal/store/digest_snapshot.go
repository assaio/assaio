package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// DigestSnapshotsKept bounds the table: a snapshot is small, but nothing that grows once a
// week may grow without a ceiling. Twelve is a quarter of weekly runs, enough to answer
// "since last time" after a few skipped weeks and no more.
const DigestSnapshotsKept = 12

// DigestSnapshot is one digest's record of what it reported, so the next run can say what
// moved. ParsedBy is the build that read the data behind it -- the field that lets a digest
// tell a real change from a parser correction reaching history.
type DigestSnapshot struct {
	TakenAt  time.Time
	ParsedBy string
	Window   string
	Payload  []byte
}

const (
	insertDigestSnapshot = `
        INSERT OR REPLACE INTO digest_snapshot(taken_at, parsed_by, window, payload)
        VALUES (?, ?, ?, ?)`

	// Trimmed per window, because the read is per window: a global newest-N would let twelve
	// daily runs evict the monthly basis, and that digest would silently become a first run.
	trimDigestSnapshots = `
        DELETE FROM digest_snapshot
        WHERE window = ? AND taken_at NOT IN (
            SELECT taken_at FROM digest_snapshot WHERE window = ? ORDER BY taken_at DESC LIMIT ?
        )`

	// Matched on window as well as time: "what moved" only means something against a basis
	// covering the same span, and a 7d run comparing itself to last month's 30d run would be
	// arithmetic on two different questions.
	selectLatestDigestSnapshot = `
        SELECT taken_at, parsed_by, window, payload
        FROM digest_snapshot
        WHERE taken_at < ? AND window = ?
        ORDER BY taken_at DESC
        LIMIT 1`
)

// SaveDigestSnapshot records this run and trims the table to DigestSnapshotsKept in the same
// transaction, so the bound holds even if the process dies immediately after.
func (s *Store) SaveDigestSnapshot(ctx context.Context, snap *DigestSnapshot) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	at := snap.TakenAt.UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx, insertDigestSnapshot, at, snap.ParsedBy, snap.Window, string(snap.Payload)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, trimDigestSnapshots, snap.Window, snap.Window, DigestSnapshotsKept); err != nil {
		return err
	}
	return tx.Commit()
}

// PreviousDigestSnapshot returns the newest snapshot of the same window taken strictly
// before at, or ok=false when this is the first digest of that window. Excluding the current
// run's own snapshot is what lets the caller save first and compare after without reading
// itself.
func (s *Store) PreviousDigestSnapshot(ctx context.Context, at time.Time, window string) (snap DigestSnapshot, ok bool, err error) {
	var taken, payload string
	row := s.db.QueryRowContext(ctx, selectLatestDigestSnapshot, at.UTC().Format(time.RFC3339), window)
	if err := row.Scan(&taken, &snap.ParsedBy, &snap.Window, &payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DigestSnapshot{}, false, nil
		}
		return DigestSnapshot{}, false, err
	}
	// A timestamp this build cannot read is treated as no basis rather than an error: the
	// caller then reports a first run, which is the honest reading of "there is nothing here
	// I can compare against" and never fails a cron job over one unreadable row.
	snap.TakenAt, err = time.Parse(time.RFC3339, taken)
	if err != nil {
		return DigestSnapshot{}, false, nil
	}
	snap.Payload = []byte(payload)
	return snap, true, nil
}
