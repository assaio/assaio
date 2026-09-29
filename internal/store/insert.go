package store

import (
	"context"
	"database/sql"
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
	inserted, _, err := s.insertWith(ctx, recs, restateSignalsSQL, signalsRestateArgs, false)
	return inserted, err
}

// signalsRestateArgs binds r to restateSignalsSQL's placeholders; the repository is not one of
// the columns it fills.
func signalsRestateArgs(r *usage.Record, _ int64) []any {
	return []any{
		r.ToolReads, r.ToolSearches, r.ToolCommands, r.ToolWrites, r.ToolOther,
		r.ToolErrors, r.Sidechain, r.Skill, r.Agent, r.Tool, r.DedupeKey,
	}
}

// insertWith inserts recs idempotently and hands every skipped duplicate to restateSQL,
// which decides what a re-read is allowed to correct on a row that already exists. watched
// reads each duplicate before its restate (watchSQL): assigned columns let a re-read move a
// figure down or replace a name, which is the point -- a corrected rule has to reach history --
// and also the one way a parser regression erases evidence with nothing to show for it. Off on
// the paths whose restate cannot do either, or whose operator cannot act on it.
func (s *Store) insertWith(ctx context.Context, recs []usage.Record, restateSQL string,
	restateArgs func(*usage.Record, int64) []any, watched bool,
) (inserted int, w Restated, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, w, err
	}
	defer func() { _ = tx.Rollback() }()
	ids, err := registerRepositories(ctx, tx, recs)
	if err != nil {
		return 0, w, err
	}
	restate, err := tx.PrepareContext(ctx, restateSQL)
	if err != nil {
		return 0, w, err
	}
	var look *sql.Stmt
	if watched {
		look, err = tx.PrepareContext(ctx, watchSQL)
		if err != nil {
			return 0, w, err
		}
		defer func() { _ = look.Close() }()
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
           cache_write_1h, cache_miss_reason, repo_id)
        VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
        ON CONFLICT(tool, dedupe_key) DO NOTHING`)
	if err != nil {
		return 0, w, err
	}
	defer func() { _ = stmt.Close() }()
	for i := range recs {
		r := &recs[i]
		id := repoID(ids, r)
		res, err := stmt.ExecContext(ctx, r.Tool, r.SessionID, r.Timestamp.UTC().Format(time.RFC3339),
			r.Model, r.InputTokens, r.OutputTokens, r.CacheReadTokens, r.CacheWriteTokens,
			r.ReasoningTokens, r.DedupeKey, r.Project, r.Subpath, r.GitBranch, r.Entrypoint, r.Granularity,
			r.LinesAdded, r.LinesRemoved, r.Edits, r.ToolCalls, r.Rejected, r.Compactions, r.ReworkLines,
			r.Member,
			r.ToolReads, r.ToolSearches, r.ToolCommands, r.ToolWrites, r.ToolOther,
			r.ToolErrors, r.Sidechain, r.Skill, r.Agent,
			r.CacheWrite1hTokens, r.CacheMissReason, id)
		if err != nil {
			return inserted, w, err
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			inserted++
			continue
		}
		if look != nil {
			down, c, err := watch(ctx, look, r, id)
			if err != nil {
				return inserted, w, err
			}
			if down {
				w.Lowered++
			}
			w.Identity.Add(c)
		}
		if _, err := restate.ExecContext(ctx, restateArgs(r, id)...); err != nil {
			return inserted, w, err
		}
	}
	if err := tx.Commit(); err != nil {
		return inserted, w, err
	}
	return inserted, w, nil
}
