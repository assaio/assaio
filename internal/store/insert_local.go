package store

import (
	"context"

	"github.com/assaio/assaio/internal/usage"
)

// restateActivitySQL corrects a stored turn from a re-read of the file it came from. Every
// column takes MAX(stored, offered), which is what makes a half-attributed turn repairable
// without ever degrading one: a session ingested while still being written yields a turn
// whose tool calls are known but whose errors, denials and edit counts a later line has not
// attributed yet, and a log is append-only, so the later read is the more complete one.
// MAX on the text columns keeps a captured skill or agent from being cleared by a re-read
// that saw none, since an empty string sorts below any name.
const restateActivitySQL = `
        UPDATE usage_record SET
            lines_added = MAX(lines_added, ?), lines_removed = MAX(lines_removed, ?),
            edits = MAX(edits, ?), tool_calls = MAX(tool_calls, ?),
            rejected = MAX(rejected, ?), compactions = MAX(compactions, ?),
            rework_lines = MAX(rework_lines, ?),
            tool_reads = MAX(tool_reads, ?), tool_searches = MAX(tool_searches, ?),
            tool_commands = MAX(tool_commands, ?), tool_writes = MAX(tool_writes, ?),
            tool_other = MAX(tool_other, ?), tool_errors = MAX(tool_errors, ?),
            sidechain = MAX(sidechain, ?), skill = MAX(skill, ?), agent = MAX(agent, ?)
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
		r.LinesAdded, r.LinesRemoved, r.Edits, r.ToolCalls, r.Rejected, r.Compactions,
		r.ReworkLines,
		r.ToolReads, r.ToolSearches, r.ToolCommands, r.ToolWrites, r.ToolOther, r.ToolErrors,
		r.Sidechain, r.Skill, r.Agent, r.Tool, r.DedupeKey,
	}
}
