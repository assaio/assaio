package store

import (
	"context"
	"time"
)

// SubpathRow is one repository subpath's summed AI lines and distinct session count within a
// single project, for the dashboard's per-project drill-down. One row per subpath across every
// member: the panel it feeds has no member column, so grouping in member printed the same
// subpath once per person, each row holding one person's lines, with the top one reading as the
// subpath's total -- and an unlabelled per-member output ranking under a heading that says
// repository.
type SubpathRow struct {
	Subpath  string
	Lines    int64
	Sessions int64
}

// Subpaths returns per-subpath AI-line and session totals for project's usage_record rows with
// timestamp >= since, ranked by lines descending. Sessions counts distinct session_id values,
// not rows, so a session touching a subpath across many turns counts once.
func (s *Store) Subpaths(ctx context.Context, project string, since time.Time) ([]SubpathRow, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT subpath, SUM(lines_added), COUNT(DISTINCT session_id)
        FROM usage_record
        WHERE project = ? AND ts >= ?
        GROUP BY subpath
        ORDER BY SUM(lines_added) DESC, subpath`, project, since.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []SubpathRow
	for rows.Next() {
		var r SubpathRow
		if err := rows.Scan(&r.Subpath, &r.Lines, &r.Sessions); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
