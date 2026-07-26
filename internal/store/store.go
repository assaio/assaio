// Package store provides embedded SQLite persistence for usage records.
package store

import (
	"context"
	"database/sql"
	"time"

	_ "modernc.org/sqlite"

	"github.com/assaio/assaio/internal/usage"
)

// Store persists usage.Record values in an embedded SQLite database.
type Store struct{ db *sql.DB }

// UsageRow is one day/tool/model/project/entrypoint/member group with summed token and
// activity counts. Member is "" for purely local usage; it is only ever non-empty on a
// central store a team has synced into (see internal/server).
type UsageRow struct {
	Day, Tool, Model, Project, Entrypoint, Member                     string
	In, Out, CacheRead, CacheWrite, Reasoning                         int64
	LinesAdded, LinesRemoved, Edits, ToolCalls, Rejected, Compactions int64
	ReworkLines                                                       int64
	// ToolReads..ToolOther split ToolCalls by purpose; ToolErrors counts failed calls. All
	// zero for tools whose logs don't name their tool calls -- see usage.Record.
	ToolReads, ToolSearches, ToolCommands, ToolWrites, ToolOther int64
	ToolErrors                                                   int64
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

// restateSignalsSQL fills the activity columns migration 0002 added onto a row that carries
// none of them. The WHERE clause is what makes this a repair rather than an overwrite: a
// row whose signals were already captured is never rewritten, so a re-parse that extracts
// less can not degrade the store. Writing zeros over zeros is a harmless no-op.
const restateSignalsSQL = `
        UPDATE usage_record SET
            tool_reads = ?, tool_searches = ?, tool_commands = ?, tool_writes = ?,
            tool_other = ?, tool_errors = ?, sidechain = ?, skill = ?, agent = ?
        WHERE tool = ? AND dedupe_key = ?
          AND tool_reads = 0 AND tool_searches = 0 AND tool_commands = 0
          AND tool_writes = 0 AND tool_other = 0 AND tool_errors = 0
          AND sidechain = 0 AND skill = '' AND agent = ''`

// Insert writes recs, skipping any that duplicate an existing (tool, dedupe_key) pair, and
// returns the number of rows actually inserted -- new rows, never a restatement, so the
// figure backfill and sync print keeps meaning "records that did not exist before".
//
// A skipped duplicate is still offered to restateSignals, which fills in the activity
// columns added in 0002 when the stored row has none of them. That is how history parsed by
// an older build gains the new signals on the next backfill instead of keeping zeros
// forever; it is a repair of existing rows, so it is deliberately not counted here.
func (s *Store) Insert(ctx context.Context, recs []usage.Record) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	restate, err := tx.PrepareContext(ctx, restateSignalsSQL)
	if err != nil {
		return 0, err
	}
	defer func() { _ = restate.Close() }()
	stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO usage_record
          (tool, session_id, ts, model, input_tokens, output_tokens,
           cache_read_tokens, cache_write_tokens, reasoning_tokens, dedupe_key,
           project, subpath, git_branch, entrypoint, granularity,
           lines_added, lines_removed, edits, tool_calls, rejected, compactions, rework_lines,
           member,
           tool_reads, tool_searches, tool_commands, tool_writes, tool_other,
           tool_errors, sidechain, skill, agent)
        VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
        ON CONFLICT(tool, dedupe_key) DO NOTHING`)
	if err != nil {
		return 0, err
	}
	defer func() { _ = stmt.Close() }()
	inserted := 0
	for i := range recs {
		r := &recs[i]
		res, err := stmt.ExecContext(ctx, r.Tool, r.SessionID, r.Timestamp.UTC().Format(time.RFC3339),
			r.Model, r.InputTokens, r.OutputTokens, r.CacheReadTokens, r.CacheWriteTokens,
			r.ReasoningTokens, r.DedupeKey, r.Project, r.Subpath, r.GitBranch, r.Entrypoint, r.Granularity,
			r.LinesAdded, r.LinesRemoved, r.Edits, r.ToolCalls, r.Rejected, r.Compactions, r.ReworkLines,
			r.Member,
			r.ToolReads, r.ToolSearches, r.ToolCommands, r.ToolWrites, r.ToolOther,
			r.ToolErrors, r.Sidechain, r.Skill, r.Agent)
		if err != nil {
			return inserted, err
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			inserted++
			continue
		}
		if _, err := restate.ExecContext(ctx,
			r.ToolReads, r.ToolSearches, r.ToolCommands, r.ToolWrites, r.ToolOther,
			r.ToolErrors, r.Sidechain, r.Skill, r.Agent, r.Tool, r.DedupeKey); err != nil {
			return inserted, err
		}
	}
	if err := tx.Commit(); err != nil {
		return inserted, err
	}
	return inserted, nil
}

// Usage returns per-day/tool/model/project/entrypoint/member token totals for records
// with timestamp >= since. Member is "" on every row of a purely local store, so
// grouping by it is a no-op there; it only fans a group out further on a central store
// synced from more than one team member.
func (s *Store) Usage(ctx context.Context, since time.Time) ([]UsageRow, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT substr(ts,1,10) AS day, tool, model, project, entrypoint, member,
               SUM(input_tokens), SUM(output_tokens),
               SUM(cache_read_tokens), SUM(cache_write_tokens), SUM(reasoning_tokens),
               SUM(lines_added), SUM(lines_removed), SUM(edits), SUM(tool_calls), SUM(rejected), SUM(compactions),
               SUM(rework_lines),
               SUM(tool_reads), SUM(tool_searches), SUM(tool_commands), SUM(tool_writes), SUM(tool_other),
               SUM(tool_errors)
        FROM usage_record
        WHERE ts >= ?
        GROUP BY day, tool, model, project, entrypoint, member
        ORDER BY day, tool, model, project, entrypoint, member`, since.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []UsageRow
	for rows.Next() {
		var u UsageRow
		if err := rows.Scan(&u.Day, &u.Tool, &u.Model, &u.Project, &u.Entrypoint, &u.Member,
			&u.In, &u.Out, &u.CacheRead, &u.CacheWrite, &u.Reasoning,
			&u.LinesAdded, &u.LinesRemoved, &u.Edits, &u.ToolCalls, &u.Rejected, &u.Compactions,
			&u.ReworkLines,
			&u.ToolReads, &u.ToolSearches, &u.ToolCommands, &u.ToolWrites, &u.ToolOther,
			&u.ToolErrors); err != nil {
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
