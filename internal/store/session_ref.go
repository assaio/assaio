package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Naming one stored session, which is a different job from reading its annotations: these
// lookups run against usage_record and know nothing about labels, and `mark` is only their
// first caller.

// SessionRef identifies one session the way store.Sessions groups them, with just enough
// context for a command to confirm which session it acted on. Member is "" for purely local
// usage. Only SessionID and Member identify the row; Project and LastTs are descriptive.
type SessionRef struct {
	SessionID string
	Member    string
	Project   string
	LastTs    time.Time
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

// scanSessionRef scans the identity-plus-context shape both lookups select.
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
