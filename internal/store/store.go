// Package store provides embedded SQLite persistence for usage records.
package store

import (
	"context"
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

// Store persists usage.Record values in an embedded SQLite database.
type Store struct{ db *sql.DB }

// UsageRow is one day/tool/model/project/entrypoint/member/granularity group with summed
// token and activity counts. Member is "" for purely local usage; it is only ever non-empty
// on a central store a team has synced into (see internal/server). Granularity is part of
// the group rather than a summary of it: a per-turn record and a session aggregate are
// different units, so summing them into one row would state a total nobody can interpret.
type UsageRow struct {
	Day, Tool, Model, Project, Entrypoint, Member, Granularity        string
	In, Out, CacheRead, CacheWrite, Reasoning                         int64
	LinesAdded, LinesRemoved, Edits, ToolCalls, Rejected, Compactions int64
	ReworkLines                                                       int64
	// ToolReads..ToolOther split ToolCalls by purpose; ToolErrors counts failed calls. All
	// zero for tools whose logs don't name their tool calls -- see usage.Record.
	ToolReads, ToolSearches, ToolCommands, ToolWrites, ToolOther int64
	ToolErrors                                                   int64
	// Task, Outcome and Difficulty are the session annotations this group belongs to, and
	// are populated only by UsageByLabel -- every other query leaves them "" because it did
	// not group by them. Empty from UsageByLabel means genuinely unannotated usage.
	Task, Outcome, Difficulty string
}

// Open opens (creating if needed) a WAL-mode SQLite database at path and applies any
// pending migrations.
func Open(path string) (*Store, error) {
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	// Migrations are local and fast; no caller cancellation semantics at open.
	if err := migrate(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, err
	}
	// Invariant: every column 0001_init.sql gains after its original set must carry a constant DEFAULT, or ADD COLUMN below fails on upgrade (post-1.0, real migrations replace this file entirely).
	if err := reconcileColumns(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error { return s.db.Close() }

// Usage returns per-day/tool/model/project/entrypoint/member/granularity token totals for
// records with timestamp >= since. Member is "" on every row of a purely local store, so
// grouping by it is a no-op there; it only fans a group out further on a central store
// synced from more than one team member. Granularity almost never fans a group out either,
// since a source emits one kind of record -- but when it does, the split is the point.
func (s *Store) Usage(ctx context.Context, since time.Time) ([]UsageRow, error) {
	return s.usage(ctx, since, LabelFilter{}, false)
}

// UsageFiltered is Usage restricted to sessions carrying filter's annotations. An empty
// filter returns exactly what Usage returns, so a caller need not special-case "no filter".
func (s *Store) UsageFiltered(ctx context.Context, since time.Time, filter LabelFilter) ([]UsageRow, error) {
	return s.usage(ctx, since, filter, false)
}

// UsageByLabel is Usage with the session annotations added to the grouping, so a report can
// be read per kind of work. Usage from a session nobody marked comes back with empty
// annotations -- it is grouped, never dropped, because a view that hid it would present a
// slice of the window as the whole of it.
func (s *Store) UsageByLabel(ctx context.Context, since time.Time) ([]UsageRow, error) {
	return s.usage(ctx, since, LabelFilter{}, true)
}

// usageAggregates are the sums every Usage variant reports, in one place so the plain and
// annotation-grouped queries cannot drift apart on what a total contains.
const usageAggregates = `
               SUM(input_tokens), SUM(output_tokens),
               SUM(cache_read_tokens), SUM(cache_write_tokens), SUM(reasoning_tokens),
               SUM(lines_added), SUM(lines_removed), SUM(edits), SUM(tool_calls), SUM(rejected), SUM(compactions),
               SUM(rework_lines),
               SUM(tool_reads), SUM(tool_searches), SUM(tool_commands), SUM(tool_writes), SUM(tool_other),
               SUM(tool_errors)`

// The three usage queries are spelled out in full rather than assembled at run time, for
// the reason attribution.go states: SQL in this package is never built by concatenating
// anything but compile-time constants. usageSelect and usageGroup are shared so the
// variants cannot drift apart on what a total contains or how it is keyed.
const (
	usageSelect = `
        SELECT substr(ts,1,10) AS day, tool, model, project, entrypoint, member, granularity,` +
		usageAggregates + `
        FROM usage_record
        WHERE ts >= ?`

	usageGroup = `
        GROUP BY day, tool, model, project, entrypoint, member, granularity
        ORDER BY day, tool, model, project, entrypoint, member, granularity`

	usageQuery = usageSelect + usageGroup

	usageFilteredQuery = usageSelect + labelSubquery + usageGroup

	// usageByLabelQuery adds the annotations to both the projection and the grouping. The
	// join is USING rather than ON so the two shared columns merge and every other column
	// stays unqualified. The annotation columns come last so one scan path serves them all.
	usageByLabelQuery = `
        SELECT substr(ts,1,10) AS day, tool, model, project, entrypoint, member, granularity,` +
		usageAggregates + `,
               COALESCE(task, ''), COALESCE(outcome, ''), COALESCE(difficulty, '')
        FROM usage_record LEFT JOIN session_label USING (session_id, member)
        WHERE ts >= ?
        GROUP BY day, tool, model, project, entrypoint, member, granularity, task, outcome, difficulty
        ORDER BY day, tool, model, project, entrypoint, member, granularity, task, outcome, difficulty`
)

func (s *Store) usage(ctx context.Context, since time.Time, filter LabelFilter, byLabel bool) ([]UsageRow, error) {
	query, args := usageQuery, []any{since.UTC().Format(time.RFC3339)}
	switch {
	case byLabel:
		query = usageByLabelQuery
	case !filter.Empty():
		query = usageFilteredQuery
		args = append(args, filter.args()...)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []UsageRow
	for rows.Next() {
		var u UsageRow
		dest := []any{
			&u.Day, &u.Tool, &u.Model, &u.Project, &u.Entrypoint, &u.Member, &u.Granularity,
			&u.In, &u.Out, &u.CacheRead, &u.CacheWrite, &u.Reasoning,
			&u.LinesAdded, &u.LinesRemoved, &u.Edits, &u.ToolCalls, &u.Rejected, &u.Compactions,
			&u.ReworkLines,
			&u.ToolReads, &u.ToolSearches, &u.ToolCommands, &u.ToolWrites, &u.ToolOther,
			&u.ToolErrors,
		}
		if byLabel {
			dest = append(dest, &u.Task, &u.Outcome, &u.Difficulty)
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// Count returns the total number of stored usage records.
func (s *Store) Count(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM usage_record`).Scan(&n)
	return n, err
}

// Clear deletes usage records older than before (if non-zero) and/or matching tool (if
// non-empty), returning the number of rows removed.
func (s *Store) Clear(ctx context.Context, before time.Time, tool string) (int64, error) {
	hasBefore := !before.IsZero()
	hasTool := tool != ""

	var res sql.Result
	var err error
	switch {
	case hasBefore && hasTool:
		res, err = s.db.ExecContext(ctx, `DELETE FROM usage_record WHERE ts < ? AND tool = ?`,
			before.UTC().Format(time.RFC3339), tool)
	case hasBefore:
		res, err = s.db.ExecContext(ctx, `DELETE FROM usage_record WHERE ts < ?`,
			before.UTC().Format(time.RFC3339))
	case hasTool:
		res, err = s.db.ExecContext(ctx, `DELETE FROM usage_record WHERE tool = ?`, tool)
	default:
		res, err = s.db.ExecContext(ctx, `DELETE FROM usage_record`)
	}
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
