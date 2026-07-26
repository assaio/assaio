package store

import (
	"context"
	"time"
)

// AttributionRow is one skill's or sub-agent's usage across the queried window. Name is the
// category label the tool itself assigned (e.g. "code-review", "general-purpose") -- never
// a prompt or any other content.
type AttributionRow struct {
	Name     string
	Tokens   int64
	Lines    int64
	Records  int64
	Sessions int64
}

// Attribution returns per-skill and per-sub-agent token totals for records with timestamp
// >= since, each sorted by Tokens descending. Rows carrying no label are excluded rather
// than bucketed under an empty name: a turn the tool didn't attribute is unknown, not a
// skill called "". Both slices are empty when no tool in the window reports attribution,
// which is why the validator over them states its own coverage.
func (s *Store) Attribution(ctx context.Context, since time.Time) (skills, agents []AttributionRow, err error) {
	skills, err = s.attributionRows(ctx, skillAttributionQuery, since)
	if err != nil {
		return nil, nil, err
	}
	agents, err = s.attributionRows(ctx, agentAttributionQuery, since)
	if err != nil {
		return nil, nil, err
	}
	return skills, agents, nil
}

// The two attribution queries are spelled out in full rather than sharing one built by
// interpolating a column name: an identifier cannot be a bind parameter, and assembling
// SQL by concatenation -- even from our own constants -- is the pattern to keep out of
// this package.
const (
	skillAttributionQuery = `
        SELECT skill AS name,
               SUM(input_tokens + output_tokens + cache_read_tokens + cache_write_tokens),
               SUM(lines_added), COUNT(*), COUNT(DISTINCT session_id)
        FROM usage_record
        WHERE ts >= ? AND skill <> ''
        GROUP BY name
        ORDER BY 2 DESC, name`

	agentAttributionQuery = `
        SELECT agent AS name,
               SUM(input_tokens + output_tokens + cache_read_tokens + cache_write_tokens),
               SUM(lines_added), COUNT(*), COUNT(DISTINCT session_id)
        FROM usage_record
        WHERE ts >= ? AND agent <> ''
        GROUP BY name
        ORDER BY 2 DESC, name`
)

// attributionRows runs one of the two attribution queries above and scans its rows.
func (s *Store) attributionRows(ctx context.Context, query string, since time.Time) ([]AttributionRow, error) {
	rows, err := s.db.QueryContext(ctx, query, since.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []AttributionRow
	for rows.Next() {
		var a AttributionRow
		if err := rows.Scan(&a.Name, &a.Tokens, &a.Lines, &a.Records, &a.Sessions); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
