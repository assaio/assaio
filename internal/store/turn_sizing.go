package store

import (
	"context"
	"time"
)

// ModelTurns is per-model turn counts for the model-right-sizing metric: turns that
// produced output, and how many of those produced fewer than smallMax output tokens.
type ModelTurns struct {
	Model      string
	Turns      int64
	SmallTurns int64
}

// TurnSizing counts, per model, output-producing turns and how many produced fewer than
// smallMax output tokens. It reads the raw per-record grain that Usage's daily GROUP BY
// hides, so the model-right-sizing metric measures actual turns rather than day-aggregates.
func (s *Store) TurnSizing(ctx context.Context, since time.Time, smallMax int64) ([]ModelTurns, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT model,
            COALESCE(SUM(CASE WHEN output_tokens > 0 THEN 1 ELSE 0 END), 0) AS turns,
            COALESCE(SUM(CASE WHEN output_tokens > 0 AND output_tokens < ? THEN 1 ELSE 0 END), 0) AS small
        FROM usage_record
        WHERE ts >= ? AND granularity = 'turn'
        GROUP BY model`, smallMax, since.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []ModelTurns
	for rows.Next() {
		var m ModelTurns
		if err := rows.Scan(&m.Model, &m.Turns, &m.SmallTurns); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
