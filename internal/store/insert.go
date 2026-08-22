package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

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
// A skipped duplicate is offered to restateSignals, which fills in the activity columns
// added in 0002 when the stored row has none of them. That is how history parsed by an older
// build gains the new signals on the next backfill instead of keeping zeros forever; it is a
// repair of existing rows, so it is deliberately not counted here.
//
// This path stays first-write-wins because its callers do not own the rows they offer: exec
// parser plugins, `demo` and `share`. A file the store reads itself is a different contract --
// see InsertLocal, which the sync server writes through as InsertSynced.
func (s *Store) Insert(ctx context.Context, recs []usage.Record) (int, error) {
	// restateSignalsSQL only ever fills columns that are still zero, so this path cannot lower
	// a figure and has nothing to watch.
	inserted, _, err := s.insertWith(ctx, recs, restateSignalsSQL, signalsRestateArgs, "")
	return inserted, err
}

// signalsRestateArgs binds r to restateSignalsSQL's placeholders.
func signalsRestateArgs(r *usage.Record) []any {
	return []any{
		r.ToolReads, r.ToolSearches, r.ToolCommands, r.ToolWrites, r.ToolOther,
		r.ToolErrors, r.Sidechain, r.Skill, r.Agent, r.Tool, r.DedupeKey,
	}
}

// insertWith inserts recs idempotently and hands every skipped duplicate to restateSQL,
// which decides what a re-read is allowed to correct on a row that already exists. lowerSQL is
// the watch on that correction: assigned columns let a re-read move a figure *down*, which is
// the point -- a corrected rule has to reach history -- and also the one way a parser
// regression erases evidence with nothing to show for it. Empty on the paths that cannot lower
// anything.
func (s *Store) insertWith(ctx context.Context, recs []usage.Record, restateSQL string,
	restateArgs func(*usage.Record) []any, lowerSQL string,
) (inserted, lowered int, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback() }()
	restate, err := tx.PrepareContext(ctx, restateSQL)
	if err != nil {
		return 0, 0, err
	}
	var lower *sql.Stmt
	if lowerSQL != "" {
		lower, err = tx.PrepareContext(ctx, lowerSQL)
		if err != nil {
			return 0, 0, err
		}
		defer func() { _ = lower.Close() }()
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
           tool_errors, sidechain, skill, agent,
           cache_write_1h, cache_miss_reason)
        VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
        ON CONFLICT(tool, dedupe_key) DO NOTHING`)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = stmt.Close() }()
	for i := range recs {
		r := &recs[i]
		res, err := stmt.ExecContext(ctx, r.Tool, r.SessionID, r.Timestamp.UTC().Format(time.RFC3339),
			r.Model, r.InputTokens, r.OutputTokens, r.CacheReadTokens, r.CacheWriteTokens,
			r.ReasoningTokens, r.DedupeKey, r.Project, r.Subpath, r.GitBranch, r.Entrypoint, r.Granularity,
			r.LinesAdded, r.LinesRemoved, r.Edits, r.ToolCalls, r.Rejected, r.Compactions, r.ReworkLines,
			r.Member,
			r.ToolReads, r.ToolSearches, r.ToolCommands, r.ToolWrites, r.ToolOther,
			r.ToolErrors, r.Sidechain, r.Skill, r.Agent,
			r.CacheWrite1hTokens, r.CacheMissReason)
		if err != nil {
			return inserted, lowered, err
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			inserted++
			continue
		}
		if lower != nil {
			down, err := wouldLower(ctx, lower, r)
			if err != nil {
				return inserted, lowered, err
			}
			if down {
				lowered++
			}
		}
		if _, err := restate.ExecContext(ctx, restateArgs(r)...); err != nil {
			return inserted, lowered, err
		}
	}
	if err := tx.Commit(); err != nil {
		return inserted, lowered, err
	}
	return inserted, lowered, nil
}

// wouldLower reports whether restating r on its stored row moves any assigned activity figure
// down. Asked before the update, because afterwards the old figure is gone.
func wouldLower(ctx context.Context, stmt *sql.Stmt, r *usage.Record) (bool, error) {
	var one int
	err := stmt.QueryRowContext(ctx, lowerArgs(r)...).Scan(&one)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	case err != nil:
		return false, err
	}
	return true, nil
}
