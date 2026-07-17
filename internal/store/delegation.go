package store

import (
	"context"
	"time"
)

// Delegation returns the sub-agent and total token counts for usage_record rows with
// timestamp >= since. subAgentTokens sums only claude-code rows whose dedupe_key marks a
// Task sub-agent turn: the "agent:" prefix internal/parser/claude/toolresult.go writes
// locally, or that same marker after a central store's member prefix -- handleUsage
// (internal/server/handlers.go) rewrites every synced dedupe_key to "<member>:" +
// original, turning "agent:x" into "<member>:agent:x". Matching is scoped to
// tool = 'claude-code' because another tool's own dedupe_key scheme (e.g. codex's
// "<session>:<turn>") could otherwise coincidentally start with "agent:" too, if a
// session or task happened to be literally named "agent"; every other tool's records are
// excluded regardless of shape. totalTokens sums every row, every tool, in the same
// window. This is the exact per-token delegation share, read directly from dedupe_key --
// not an approximation from model mix.
func (s *Store) Delegation(ctx context.Context, since time.Time) (subAgentTokens, totalTokens int64, err error) {
	err = s.db.QueryRowContext(ctx, `
        SELECT
            COALESCE(SUM(CASE WHEN tool = 'claude-code'
                AND (dedupe_key LIKE 'agent:%' OR dedupe_key LIKE '%:agent:%') THEN
                input_tokens + output_tokens + cache_read_tokens + cache_write_tokens + reasoning_tokens
                ELSE 0 END), 0),
            COALESCE(SUM(input_tokens + output_tokens + cache_read_tokens + cache_write_tokens + reasoning_tokens), 0)
        FROM usage_record
        WHERE ts >= ?`, since.UTC().Format(time.RFC3339)).Scan(&subAgentTokens, &totalTokens)
	return subAgentTokens, totalTokens, err
}
