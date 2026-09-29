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
        WITH`+shownNames+`,
        refs AS (`+sessionRefsSelect+`
            WHERE session_id LIKE ? ESCAPE '\'
            GROUP BY session_id, member
        )`+sessionRefsNamed+`
        ORDER BY refs.session_id, refs.member`, escapeLike(prefix)+"%")
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

// LatestSession returns the most recently active session anywhere in the store.
func (s *Store) LatestSession(ctx context.Context) (ref SessionRef, ok bool, err error) {
	return scanLatest(s.db.QueryRowContext(ctx, `
        WITH`+shownNames+`,
        refs AS (`+sessionRefsSelect+`
            GROUP BY session_id, member
        )`+sessionRefsNamed+`
        ORDER BY refs.last_ts DESC, refs.session_id
        LIMIT 1`))
}

// LatestSessionIn returns the most recently active session with a row resolved to repository
// id. This is what `mark` targets when the user names no session: the work they just finished
// in the repository they are standing in -- never a session that only shares its name.
func (s *Store) LatestSessionIn(ctx context.Context, id int64) (ref SessionRef, ok bool, err error) {
	return scanLatest(s.db.QueryRowContext(ctx, `
        WITH`+shownNames+`,
        mine AS (SELECT DISTINCT session_id, member FROM usage_record WHERE repo_id > 0 AND repo_id = ?),
        refs AS (`+sessionRefsSelect+`
            WHERE EXISTS (SELECT 1 FROM mine
                          WHERE mine.session_id = usage_record.session_id AND mine.member = usage_record.member)
            GROUP BY session_id, member
        )`+sessionRefsNamed+`
        ORDER BY refs.last_ts DESC, refs.session_id
        LIMIT 1`, id))
}

func scanLatest(row *sql.Row) (SessionRef, bool, error) {
	ref, err := scanSessionRef(row)
	if errors.Is(err, sql.ErrNoRows) {
		return SessionRef{}, false, nil
	}
	if err != nil {
		return SessionRef{}, false, err
	}
	return ref, true, nil
}

// sessionRefsSelect and sessionRefsNamed are the two halves of both lookups: one row per
// session with its label and repository, then the name a reader sees for them.
const (
	sessionRefsSelect = `
            SELECT session_id, member, ` + sessionLabelExpr + ` AS label,
                   ` + sessionRepoExpr + ` AS repo_id, MAX(ts) AS last_ts
            FROM usage_record`
	sessionRefsNamed = `
        SELECT refs.session_id, refs.member, COALESCE(shown_name.name, refs.label) AS name, refs.last_ts
        FROM refs
        LEFT JOIN shown_name ON shown_name.label = refs.label AND shown_name.repo_id = refs.repo_id`
)

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
