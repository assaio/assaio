package store

import (
	"context"

	"github.com/assaio/assaio/internal/usage"
)

// restateActivitySQL corrects a stored turn from a re-read of the file it came from. It
// reaches the columns a later parser can extract better -- the token counts, activity, and
// the cache tier and miss reason 0007 added -- never the identity columns a first read is the
// authority on (ts, model, project, entrypoint, git_branch). Every column takes
// MAX(stored, offered), which is what makes a half-attributed turn repairable
// without ever degrading one: a session ingested while still being written yields a turn
// whose tool calls are known but whose errors, denials and edit counts a later line has not
// attributed yet, and a log is append-only, so the later read is the more complete one.
// MAX on the text columns keeps a captured skill or agent from being cleared by a re-read
// that saw none, since an empty string sorts below any name.
//
// The token counts are here because a Claude response's blocks arrive across several lines
// and only the last carries its true output count: a session read while it was still being
// written stores the partial figure, and without a restate that undercount would be permanent
// -- the response id already exists, so the completed read has nothing new to insert. Keeping
// cache_write_tokens beside cache_write_1h also holds the subset invariant the 1-hour share
// divides by; restating one without the other could put the portion above its whole, and
// reasoning_tokens is here for the same reason against output_tokens.
//
// granularity is the one column assigned rather than maximised. It is a claim about what a
// record *is*, and the current parse is the authority on that: a build that learns a record
// summarizes a whole run rather than one turn has to be able to say so, and MAX would keep
// 'turn' forever because it sorts above 'session'. Re-labelling coarser is the documented
// direction (docs/format-resilience.md); it is never a way to claim finer detail, because
// only a re-read of the same file can set it.
const restateActivitySQL = `
        UPDATE usage_record SET
            granularity = ?,
            input_tokens = MAX(input_tokens, ?), output_tokens = MAX(output_tokens, ?),
            cache_read_tokens = MAX(cache_read_tokens, ?),
            cache_write_tokens = MAX(cache_write_tokens, ?),
            reasoning_tokens = MAX(reasoning_tokens, ?),
            lines_added = MAX(lines_added, ?), lines_removed = MAX(lines_removed, ?),
            edits = MAX(edits, ?), tool_calls = MAX(tool_calls, ?),
            rejected = MAX(rejected, ?), compactions = MAX(compactions, ?),
            rework_lines = MAX(rework_lines, ?),
            tool_reads = MAX(tool_reads, ?), tool_searches = MAX(tool_searches, ?),
            tool_commands = MAX(tool_commands, ?), tool_writes = MAX(tool_writes, ?),
            tool_other = MAX(tool_other, ?), tool_errors = MAX(tool_errors, ?),
            sidechain = MAX(sidechain, ?), skill = MAX(skill, ?), agent = MAX(agent, ?),
            cache_write_1h = MAX(cache_write_1h, ?),
            cache_miss_reason = MAX(cache_miss_reason, ?)
        WHERE tool = ? AND dedupe_key = ?`

// InsertLocal writes records parsed from a local session file, restating the activity of a
// turn that is already stored. Use it only for files this store reads itself: the caller
// owns the input, so re-reading it is the store's own better knowledge of the same turn.
// Records arriving from elsewhere go through Insert, which stays first-write-wins.
func (s *Store) InsertLocal(ctx context.Context, recs []usage.Record) (int, error) {
	return s.insertWith(ctx, recs, restateActivitySQL, activityRestateArgs)
}

// activityRestateArgs binds r to restateActivitySQL's placeholders.
func activityRestateArgs(r *usage.Record) []any {
	return []any{
		r.Granularity,
		r.InputTokens, r.OutputTokens, r.CacheReadTokens, r.CacheWriteTokens, r.ReasoningTokens,
		r.LinesAdded, r.LinesRemoved, r.Edits, r.ToolCalls, r.Rejected, r.Compactions,
		r.ReworkLines,
		r.ToolReads, r.ToolSearches, r.ToolCommands, r.ToolWrites, r.ToolOther, r.ToolErrors,
		r.Sidechain, r.Skill, r.Agent,
		r.CacheWrite1hTokens, r.CacheMissReason,
		r.Tool, r.DedupeKey,
	}
}
