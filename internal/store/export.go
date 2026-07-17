package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// Export returns every stored usage_record row with timestamp >= since as raw,
// individual usage.Record values -- not aggregated, unlike Usage. This is the source
// data `assaio-agent sync` reads and pushes to a team server.
func (s *Store) Export(ctx context.Context, since time.Time) ([]usage.Record, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT tool, session_id, ts, model, input_tokens, output_tokens,
               cache_read_tokens, cache_write_tokens, reasoning_tokens, dedupe_key,
               project, subpath, git_branch, entrypoint, granularity,
               lines_added, lines_removed, edits, tool_calls, rejected, compactions,
               rework_lines, member
        FROM usage_record
        WHERE ts >= ?
        ORDER BY ts`, since.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []usage.Record
	for rows.Next() {
		r, err := scanExportRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// scanExportRow scans one Export row, parsing its RFC3339 timestamp. Cwd is never
// scanned: it is not a stored column (see usage.Record.Cwd's privacy note).
func scanExportRow(rows *sql.Rows) (usage.Record, error) {
	var r usage.Record
	var ts string
	if err := rows.Scan(&r.Tool, &r.SessionID, &ts, &r.Model, &r.InputTokens, &r.OutputTokens,
		&r.CacheReadTokens, &r.CacheWriteTokens, &r.ReasoningTokens, &r.DedupeKey,
		&r.Project, &r.Subpath, &r.GitBranch, &r.Entrypoint, &r.Granularity,
		&r.LinesAdded, &r.LinesRemoved, &r.Edits, &r.ToolCalls, &r.Rejected, &r.Compactions,
		&r.ReworkLines, &r.Member); err != nil {
		return usage.Record{}, err
	}
	parsed, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return usage.Record{}, err
	}
	r.Timestamp = parsed
	return r, nil
}
