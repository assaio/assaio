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

// SessionRef identifies one session the way store.Sessions groups them, with just enough
// context for a command to confirm which session it acted on. Member is "" for purely local
// usage. Only SessionID and Member identify the row; Project and LastTs are descriptive.
type SessionRef struct {
	SessionID string
	Member    string
	Project   string
	LastTs    time.Time
}

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

// labelSubquery restricts a usage_record query to annotated sessions. It is a fixed
// fragment appended to a fixed base query, and every value it compares is bound, never
// interpolated -- no caller input reaches the SQL text. An empty LabelFilter selects the
// unfiltered base query instead of this one, so the subquery never silently narrows a
// window to "sessions that happen to be annotated".
const labelSubquery = `
          AND session_id IN (
              SELECT session_id FROM session_label
              WHERE (? = '' OR task = ?)
                AND (? = '' OR outcome = ?)
                AND (? = '' OR difficulty = ?)
          )`

// aliasedLabelSubquery is labelSubquery for the one query that aliases usage_record, where
// a bare session_id would be ambiguous across the joined tables. Spelled out rather than
// built from a shared fragment, for the reason attribution.go states.
const aliasedLabelSubquery = `
          AND r.session_id IN (
              SELECT session_id FROM session_label
              WHERE (? = '' OR task = ?)
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

// DeleteLabels removes every annotation, returning how many were deleted. Only ever called
// behind an explicit flag: nothing else in assaio deletes data a person typed.
func (s *Store) DeleteLabels(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM session_label`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// MatchSessions returns every stored session whose id starts with prefix, ordered by id.
// More than one match is an ambiguity the caller reports rather than resolving, the way git
// treats a short revision that names two objects.
func (s *Store) MatchSessions(ctx context.Context, prefix string) ([]SessionRef, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT session_id, member, MAX(project), MAX(ts) FROM usage_record
        WHERE session_id LIKE ? ESCAPE '\'
        GROUP BY session_id, member
        ORDER BY session_id, member`, escapeLike(prefix)+"%")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []SessionRef
	for rows.Next() {
		ref, err := scanSessionRef(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, rows.Err()
}

// scanSessionRef scans the identity-plus-context shape both lookups below select.
func scanSessionRef(row interface{ Scan(...any) error }) (SessionRef, error) {
	var ref SessionRef
	var lastTs string
	if err := row.Scan(&ref.SessionID, &ref.Member, &ref.Project, &lastTs); err != nil {
		return SessionRef{}, err
	}
	ref.LastTs, _ = time.Parse(time.RFC3339, lastTs)
	return ref, nil
}

// escapeLike neutralizes the wildcards a session id prefix must never be read as, so a
// prefix containing % or _ matches those characters literally instead of any character.
func escapeLike(s string) string {
	var b []byte
	for i := range len(s) {
		if c := s[i]; c == '%' || c == '_' || c == '\\' {
			b = append(b, '\\')
		}
		b = append(b, s[i])
	}
	return string(b)
}

// LatestSession returns the most recently active session, restricted to project when it is
// non-empty. This is what `mark` targets when the user names no session: the work they just
// finished in the repository they are standing in.
func (s *Store) LatestSession(ctx context.Context, project string) (ref SessionRef, ok bool, err error) {
	row := s.db.QueryRowContext(ctx, `
        SELECT session_id, member, MAX(project), MAX(ts) FROM usage_record
        WHERE (? = '' OR project = ?)
        GROUP BY session_id, member
        ORDER BY MAX(ts) DESC, session_id
        LIMIT 1`, project, project)
	ref, err = scanSessionRef(row)
	if errors.Is(err, sql.ErrNoRows) {
		return SessionRef{}, false, nil
	}
	if err != nil {
		return SessionRef{}, false, err
	}
	return ref, true, nil
}
