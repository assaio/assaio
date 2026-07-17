package store

import (
	"context"
	"database/sql"
	"time"
)

// SessionRow is one session's aggregated activity across every usage_record row sharing
// its session_id: identity (SessionID/Project/Tool/Model), its first/last turn
// timestamps, turn/output/edit/compaction counts, the largest context window it reached
// (PeakContextTokens), and its resume-safe focused work time (ActiveMinutes). Model is
// the odd one out among the identity fields -- see its own doc comment below. The two
// derived signals below are computed in SQL, not from wall-clock duration, because a
// Claude session_id is long-lived and reused across --resume, so raw last-minus-first is
// the id's lifespan, not work time.
type SessionRow struct {
	SessionID string
	Project   string
	Tool      string
	// Model is representative, not constant: MAX(model) across the group, picked
	// arbitrarily by sort order. A Task sub-agent record shares its parent turn's
	// session_id but can carry a different model, so this is not a fact true of every
	// row in the group (see Sessions).
	Model string
	// Member is "" for purely local usage; non-empty only on a central store synced
	// from a team member (see internal/server). Part of the GROUP BY key alongside
	// session_id (see Sessions), so two different members' local installs coincidentally
	// reusing the same session_id -- never expected in practice, since each is a locally
	// generated UUID -- still get their own SessionRow rather than being blended into one.
	Member  string
	FirstTs time.Time
	LastTs  time.Time
	Turns   int64
	// OutputTokens is summed output tokens: work produced, not cumulative cache reads.
	OutputTokens int64
	// PeakContextTokens is the largest single-turn context window (cache_read + input)
	// the session reached: the honest answer to "how large did the context get".
	PeakContextTokens int64
	Edits             int64
	Compactions       int64
	// ActiveMinutes is the sum of inter-turn gaps that are <= 30 minutes: focused work
	// time that excludes the long idle spans a resumed session_id spans.
	ActiveMinutes float64
}

// activeGapCeilingMinutes bounds an inter-turn gap counted as focused work: a longer gap
// is treated as the session having been left and resumed, not continuous engagement.
const activeGapCeilingMinutes = 30

// Sessions groups usage_record rows with timestamp >= since by (session_id, member) and
// returns one SessionRow per group, ordered by session_id then member. Grouping on the
// pair rather than session_id alone matters only for a central store: it keeps two
// different members' usage_record rows apart even in the never-expected case that their
// locally generated session_ids collide, instead of silently blending their activity into
// one row under whichever member sorts last. Project/Tool/Model use MAX to keep the query
// strict-SQL and portable; Project and Tool are genuinely constant within a (session_id,
// member) group, but Model is not, since a Task sub-agent record shares its parent
// session_id under a different model -- see SessionRow.Model. ActiveMinutes sums, via a
// window function over each group's ordered turns, only the inter-turn gaps <=
// activeGapCeilingMinutes; ts is sliced to its second-precision prefix so julianday
// parses it without depending on the driver's timezone-suffix support.
func (s *Store) Sessions(ctx context.Context, since time.Time) ([]SessionRow, error) {
	sinceStr := since.UTC().Format(time.RFC3339)
	rows, err := s.db.QueryContext(ctx, `
        WITH gaps AS (
            SELECT session_id, member,
                   (julianday(substr(ts, 1, 19))
                    - LAG(julianday(substr(ts, 1, 19)))
                        OVER (PARTITION BY session_id, member ORDER BY ts)) * 1440.0 AS gap_min
            FROM usage_record
            WHERE ts >= ?
        ),
        active AS (
            SELECT session_id, member, SUM(gap_min) AS active_min
            FROM gaps
            WHERE gap_min IS NOT NULL AND gap_min <= ?
            GROUP BY session_id, member
        )
        SELECT r.session_id, MAX(r.project), MAX(r.tool), MAX(r.model), r.member,
               MIN(r.ts), MAX(r.ts), COUNT(*),
               SUM(r.output_tokens),
               MAX(r.cache_read_tokens + r.input_tokens),
               SUM(r.edits), SUM(r.compactions),
               COALESCE(MAX(a.active_min), 0.0)
        FROM usage_record r
        LEFT JOIN active a ON a.session_id = r.session_id AND a.member = r.member
        WHERE r.ts >= ?
        GROUP BY r.session_id, r.member
        ORDER BY r.session_id, r.member`, sinceStr, activeGapCeilingMinutes, sinceStr)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []SessionRow
	for rows.Next() {
		row, err := scanSessionRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// scanSessionRow scans one Sessions row, parsing its RFC3339 first/last timestamps.
func scanSessionRow(rows *sql.Rows) (SessionRow, error) {
	var r SessionRow
	var firstTs, lastTs string
	if err := rows.Scan(&r.SessionID, &r.Project, &r.Tool, &r.Model, &r.Member,
		&firstTs, &lastTs, &r.Turns, &r.OutputTokens, &r.PeakContextTokens,
		&r.Edits, &r.Compactions, &r.ActiveMinutes); err != nil {
		return SessionRow{}, err
	}
	first, err := time.Parse(time.RFC3339, firstTs)
	if err != nil {
		return SessionRow{}, err
	}
	last, err := time.Parse(time.RFC3339, lastTs)
	if err != nil {
		return SessionRow{}, err
	}
	r.FirstTs, r.LastTs = first, last
	return r, nil
}
