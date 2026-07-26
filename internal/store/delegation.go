package store

import (
	"context"
	"time"
)

// Delegation returns the sub-agent and total token counts for usage_record rows with
// timestamp >= since. The window picks ONE definition of sub-agent work rather than
// accepting either: when any row in it carries sidechain = 1 the marker decides, and only
// otherwise does the older dedupe_key shape. Accepting both would double-count a store that
// holds a pre-upgrade `agent:` aggregate row *and* the per-turn sidechain rows later parsed
// from the same sub-agent transcript -- the aggregate is suppressed at parse time, but the
// row already written is never deleted. The fallback still lets a store that has not been
// re-backfilled report its real share instead of dropping to zero.
//
// The dedupe_key fallback matches the "agent:" prefix internal/parser/claude/toolresult.go writes
// locally, or that same marker after a central store's member prefix -- handleUsage
// (internal/server/handlers.go) rewrites every synced dedupe_key to "<member>:" +
// original, turning "agent:x" into "<member>:agent:x". Matching is scoped to
// tool = 'claude-code' because another tool's own dedupe_key scheme (e.g. codex's
// "<session>:<turn>") could otherwise coincidentally start with "agent:" too, if a
// session or task happened to be literally named "agent"; every other tool's records are
// excluded regardless of shape. totalTokens sums every row, every tool, in the same
// window. This is the exact per-token delegation share, read from the record itself --
// not an approximation from model mix.
func (s *Store) Delegation(ctx context.Context, since time.Time) (subAgentTokens, totalTokens int64, err error) {
	// reasoning_tokens is a subset of output_tokens (usage.Record) and is never re-added:
	// these totals must match rowTokens on every other surface.
	const q = `
        SELECT
            COALESCE(SUM(CASE WHEN
                CASE WHEN EXISTS (SELECT 1 FROM usage_record WHERE ts >= ? AND sidechain = 1)
                     THEN sidechain = 1
                     ELSE tool = 'claude-code'
                          AND (dedupe_key LIKE 'agent:%' OR dedupe_key LIKE '%:agent:%')
                END
                THEN input_tokens + output_tokens + cache_read_tokens + cache_write_tokens
                ELSE 0 END), 0),
            COALESCE(SUM(input_tokens + output_tokens + cache_read_tokens + cache_write_tokens), 0)
        FROM usage_record
        WHERE ts >= ?`
	ts := since.UTC().Format(time.RFC3339)
	err = s.db.QueryRowContext(ctx, q, ts, ts).Scan(&subAgentTokens, &totalTokens)
	return subAgentTokens, totalTokens, err
}
