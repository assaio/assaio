package store

import (
	"context"
	"time"
)

// RepositoryRow is one name the store shows sessions under, for `repos`: how many local
// sessions carry it, how many of those resolved to no single repository, and when the last of
// them was active. A session is counted under the name Sessions gives it, so every session is
// counted once.
type RepositoryRow struct {
	Name                 string
	Sessions, Unresolved int64
	LastTs               time.Time
}

// Repositories lists every name local sessions are shown under, sorted by name. The empty name
// is the sessions whose rows named two projects or none.
func (s *Store) Repositories(ctx context.Context) ([]RepositoryRow, error) {
	rows, err := s.db.QueryContext(ctx, `
        WITH`+shownNames+`,
        refs AS (`+sessionRefsSelect+`
            WHERE member = ''
            GROUP BY session_id, member
        ),
        named AS (`+sessionRefsNamed+`
        )
        SELECT named.name, COUNT(*), SUM(refs.repo_id = 0), MAX(named.last_ts)
        FROM named JOIN refs USING (session_id, member)
        GROUP BY named.name ORDER BY named.name`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []RepositoryRow
	for rows.Next() {
		var r RepositoryRow
		var last string
		if err := rows.Scan(&r.Name, &r.Sessions, &r.Unresolved, &last); err != nil {
			return nil, err
		}
		r.LastTs = parseStoredTime(&last)
		out = append(out, r)
	}
	return out, rows.Err()
}

// Coverage is how much of the local store names its repository, in sessions -- the unit `repos`
// lists -- and in usage rows, which is where the figures come from. The two differ widely when
// many short sessions ran outside any repository.
type Coverage struct {
	Sessions, SessionsResolved, Rows, RowsResolved int64
}

// IdentityCoverage counts the local sessions whose rows resolved to one repository and the local
// rows that did, each out of all of them.
func (s *Store) IdentityCoverage(ctx context.Context) (Coverage, error) {
	var c Coverage
	err := s.db.QueryRowContext(ctx, `
        WITH refs AS (`+sessionRefsSelect+`
            WHERE member = ''
            GROUP BY session_id, member
        )
        SELECT COALESCE(SUM(repo_id > 0), 0), COUNT(*),
               (SELECT COALESCE(SUM(repo_id > 0), 0) FROM usage_record WHERE member = ''),
               (SELECT COUNT(*) FROM usage_record WHERE member = '')
        FROM refs`).Scan(&c.SessionsResolved, &c.Sessions, &c.RowsResolved, &c.Rows)
	return c, err
}
