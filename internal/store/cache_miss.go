package store

import (
	"context"
	"time"
)

// CacheMissRow is one tool's count of turns the vendor could not serve from its prompt
// cache, grouped by the reason the tool itself gave. Reason is that tool's own closed
// vocabulary, copied through rather than interpreted.
type CacheMissRow struct {
	Tool, Reason string
	Turns        int64
}

// The two queries are spelled out in full rather than assembled at run time, for the reason
// attribution.go states: SQL in this package is never built by concatenating anything but
// compile-time constants. A turn that hit cache states no reason and is absent here, as is
// every turn from a source that reports none -- the depth matrix, not an empty string, is
// where "this source cannot answer" lives.
const (
	cacheMissSelect = `
        SELECT tool, cache_miss_reason, COUNT(*)
        FROM usage_record
        WHERE ts >= ? AND cache_miss_reason <> ''`

	cacheMissGroup = `
        GROUP BY tool, cache_miss_reason
        ORDER BY COUNT(*) DESC, tool, cache_miss_reason`

	cacheMissQuery = cacheMissSelect + cacheMissGroup

	cacheMissFilteredQuery = cacheMissSelect + labelSubquery + cacheMissGroup
)

// CacheMisses returns the stated cache-miss reasons for records with timestamp >= since.
func (s *Store) CacheMisses(ctx context.Context, since time.Time) ([]CacheMissRow, error) {
	return s.cacheMisses(ctx, since, LabelFilter{})
}

// CacheMissesFiltered is CacheMisses restricted to sessions carrying filter's annotations,
// so a per-kind-of-work analysis does not read one subset's cache behaviour beside the whole
// window's causes.
func (s *Store) CacheMissesFiltered(ctx context.Context, since time.Time, filter LabelFilter) ([]CacheMissRow, error) {
	return s.cacheMisses(ctx, since, filter)
}

func (s *Store) cacheMisses(ctx context.Context, since time.Time, filter LabelFilter) ([]CacheMissRow, error) {
	query, args := cacheMissQuery, []any{since.UTC().Format(time.RFC3339)}
	if !filter.Empty() {
		query = cacheMissFilteredQuery
		args = append(args, filter.args()...)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []CacheMissRow
	for rows.Next() {
		var r CacheMissRow
		if err := rows.Scan(&r.Tool, &r.Reason, &r.Turns); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
