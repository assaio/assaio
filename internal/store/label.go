package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// SessionLabel is one session's annotations: what kind of work it was, how it ended, and
// how hard it was. Every field is a closed-vocabulary value from internal/label, and ""
// means that axis was never set -- which is what lets a session be marked with a task now
// and an outcome when the work actually ends.
type SessionLabel struct {
	Task       string
	Outcome    string
	Difficulty string
	MarkedAt   time.Time
}

// Set reports whether any axis carries a value; an all-empty label is not stored.
func (l SessionLabel) Set() bool { return l.Task != "" || l.Outcome != "" || l.Difficulty != "" }

// LabelFilter narrows a query to sessions carrying these annotations; an empty axis places
// no constraint on it. Its zero value matches everything, and every query below then runs
// the same SQL assaio has always run -- an unlabeled store is unaffected by this feature.
type LabelFilter struct {
	Task       string
	Outcome    string
	Difficulty string
}

// Empty reports whether the filter constrains nothing.
func (f LabelFilter) Empty() bool { return f.Task == "" && f.Outcome == "" && f.Difficulty == "" }

// labelSubquery restricts a usage_record query to annotated sessions. It correlates on
// (session_id, member) -- the label's own primary key -- because on a central store a bare
// session_id would let one member's annotation select another member's usage. It is a fixed
// fragment appended to a fixed base query, and every value it compares is bound, never
// interpolated -- no caller input reaches the SQL text. An empty LabelFilter selects the
// unfiltered base query instead of this one, so the subquery never silently narrows a
// window to "sessions that happen to be annotated".
const labelSubquery = `
          AND EXISTS (
              SELECT 1 FROM session_label
              WHERE session_label.session_id = usage_record.session_id
                AND session_label.member = usage_record.member
                AND (? = '' OR task = ?)
                AND (? = '' OR outcome = ?)
                AND (? = '' OR difficulty = ?)
          )`

// aliasedLabelSubquery is labelSubquery for the one query that aliases usage_record, where
// a bare session_id would be ambiguous across the joined tables. Spelled out rather than
// built from a shared fragment, for the reason attribution.go states.
const aliasedLabelSubquery = `
          AND EXISTS (
              SELECT 1 FROM session_label
              WHERE session_label.session_id = r.session_id
                AND session_label.member = r.member
                AND (? = '' OR task = ?)
                AND (? = '' OR outcome = ?)
                AND (? = '' OR difficulty = ?)
          )`

// args returns the six bind values labelSubquery consumes, each axis passed twice: once to
// test whether it constrains anything, once to compare.
func (f LabelFilter) args() []any {
	return []any{f.Task, f.Task, f.Outcome, f.Outcome, f.Difficulty, f.Difficulty}
}

// Mark upserts ref's annotations, replacing every axis with what set carries -- callers
// merge with the stored label first when they mean to change only one axis (see
// internal/cli.mark). Storing the whole row keeps this method free of per-axis update SQL.
func (s *Store) Mark(ctx context.Context, ref SessionRef, set SessionLabel) error {
	_, err := s.db.ExecContext(ctx, `
        INSERT INTO session_label(session_id, member, task, outcome, difficulty, marked_at)
        VALUES (?, ?, ?, ?, ?, ?)
        ON CONFLICT(session_id, member) DO UPDATE SET
            task = excluded.task, outcome = excluded.outcome,
            difficulty = excluded.difficulty, marked_at = excluded.marked_at`,
		ref.SessionID, ref.Member, set.Task, set.Outcome, set.Difficulty,
		set.MarkedAt.UTC().Format(time.RFC3339))
	return err
}

// Unmark deletes ref's annotations, reporting whether there were any.
func (s *Store) Unmark(ctx context.Context, ref SessionRef) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM session_label WHERE session_id = ? AND member = ?`, ref.SessionID, ref.Member)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// Label returns ref's stored annotations; ok is false when the session was never marked.
func (s *Store) Label(ctx context.Context, ref SessionRef) (l SessionLabel, ok bool, err error) {
	var markedAt string
	err = s.db.QueryRowContext(ctx,
		`SELECT task, outcome, difficulty, marked_at FROM session_label WHERE session_id = ? AND member = ?`,
		ref.SessionID, ref.Member).Scan(&l.Task, &l.Outcome, &l.Difficulty, &markedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return SessionLabel{}, false, nil
	}
	if err != nil {
		return SessionLabel{}, false, err
	}
	l.MarkedAt, _ = time.Parse(time.RFC3339, markedAt)
	return l, true, nil
}

// LabelCount is how many sessions carry annotations, for the commands that must say what
// they are about to keep or delete.
func (s *Store) LabelCount(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM session_label`).Scan(&n)
	return n, err
}

// DeleteLabels removes annotations within the same scope Clear applies to usage records,
// returning how many were deleted. Only ever called behind an explicit flag: nothing else
// in assaio deletes data a person typed.
//
// Unscoped (zero before, empty tool) it deletes every annotation, including one whose
// session has no usage rows left to identify it by. A scope narrows it to the sessions the
// matching Clear removes *entirely*: a session with records on both sides of a time cutoff
// still exists afterwards, so its annotation is not that clear's to take, and one that
// cannot be tied to the scope at all is never guessed at.
func (s *Store) DeleteLabels(ctx context.Context, before time.Time, tool string) (int64, error) {
	if before.IsZero() && tool == "" {
		return rowsAffected(s.db.ExecContext(ctx, `DELETE FROM session_label`))
	}
	cutoff := ""
	if !before.IsZero() {
		cutoff = before.UTC().Format(time.RFC3339)
	}
	// Bound twice per axis: once to ask whether it constrains anything, once to compare --
	// the same shape labelSubquery uses, and for the same reason.
	args := []any{cutoff, cutoff, tool, tool}
	return rowsAffected(s.db.ExecContext(ctx, `
        DELETE FROM session_label
        WHERE EXISTS (`+clearedRecords+`)
          AND NOT EXISTS (`+survivingRecords+`)`, append(args, args...)...))
}

// clearedRecords selects the session's usage rows a Clear with this scope removes;
// survivingRecords selects the ones it leaves behind. Spelled as two fragments over the
// same bound scope so the two can never describe different windows.
const (
	scopedRecord = `SELECT 1 FROM usage_record u
              WHERE u.session_id = session_label.session_id AND u.member = session_label.member`
	clearedRecords   = scopedRecord + ` AND (? = '' OR u.ts < ?) AND (? = '' OR u.tool = ?)`
	survivingRecords = scopedRecord + ` AND NOT ((? = '' OR u.ts < ?) AND (? = '' OR u.tool = ?))`
)
