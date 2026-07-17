package store

import (
	"context"
	"time"
)

// SubpathRow is one repository subpath's summed AI lines and distinct session count
// within a single project, for the dashboard's per-project drill-down. Member is "" for
// purely local usage; non-empty only on a central store synced from a team member (see
// internal/server). Part of the GROUP BY key alongside subpath (see Subpaths), the same
// member-awareness Usage and Sessions apply, so two members' rows for the same subpath
// never blend and a coincidentally shared session_id is never undercounted.
type SubpathRow struct {
	Subpath  string
	Member   string
	Lines    int64
	Sessions int64
}

// Subpaths returns per-subpath, per-member AI-line and session totals for project's
// usage_record rows with timestamp >= since, ranked by lines descending. Sessions counts
// distinct session_id values within the (subpath, member) group, not usage_record rows;
// grouping in member keeps two different members' rows apart even in the never-expected
// case that their locally generated session_ids collide, instead of silently blending
// their activity (or undercounting COUNT(DISTINCT session_id)) into one row under
// whichever member sorts last -- the same reasoning as Store.Sessions.
func (s *Store) Subpaths(ctx context.Context, project string, since time.Time) ([]SubpathRow, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT subpath, member, SUM(lines_added), COUNT(DISTINCT session_id)
        FROM usage_record
        WHERE project = ? AND ts >= ?
        GROUP BY subpath, member
        ORDER BY SUM(lines_added) DESC, subpath, member`, project, since.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []SubpathRow
	for rows.Next() {
		var r SubpathRow
		if err := rows.Scan(&r.Subpath, &r.Member, &r.Lines, &r.Sessions); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
