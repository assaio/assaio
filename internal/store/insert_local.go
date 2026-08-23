package store

import (
	"context"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// restateActivitySQL corrects a stored turn from a re-read of the file it came from. It
// reaches every column a later parser can extract better -- the token counts, the activity,
// the cache tier and miss reason 0007 added, and since v0.24 the identity columns too. Most
// take
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
// model is the one identity column here, and only as a repair: a source can legitimately
// emit a record before it knows the model (Cline reads it from a task_metadata.json sidecar
// that may not exist yet), and a first read that saw none must not pin those tokens as
// unpriceable forever. A stored name is never overwritten -- the CASE only fills a blank --
// so the "first read is the authority" rule still holds wherever there was an answer.
//
// The line drawn between the two halves below is where the number comes from. A token count
// is the vendor's own figure, read off the log: a later build reads the same number, so MAX
// costs nothing and still guards a truncated or rotated file. Every activity count is
// assaio's, derived by an attribution rule -- which lines are edits, which turn a result
// belongs to, which removal undoes an earlier addition -- and a rule assaio wrote is a rule
// assaio can get wrong. v0.13 learned this on rework_lines and moved that one column; v0.14
// found the same shape one layer up, in which *lines* the rules are applied to, and corrected
// four more counts downward. MAX would have pinned every stored row at the inflated figure
// forever, which is the failure ROADMAP.md's "a correction reaches history" names as a v1.0
// condition.
//
// Assignment is safe for a session still being written for the same reason it was for rework:
// each rule is monotone in the prefix read. A longer transcript never yields fewer edits,
// fewer tool calls or less rework for an already-emitted turn -- later lines attach to later
// turns -- so a re-read restates upward exactly as MAX did. Assigning the purpose split
// together also keeps it summing to tool_calls, which taking each column's own maximum could
// not promise across a build that reclassifies a tool.
//
// granularity is assigned for its own reason: it is a claim about what a record *is*, and the
// current parse is the authority on that. A build that learns a record summarizes a whole run
// rather than one turn has to be able to say so, and MAX would keep 'turn' forever because it
// sorts above 'session'. Re-labelling coarser is the documented direction
// (docs/format-resilience.md); it is never a way to claim finer detail, because only a re-read
// of the same file can set it.
//
// ts, project, entrypoint and git_branch were the hole B116 named: stamped once and correctable
// by nothing, so a parser fix to any of them could not reach a single stored row and no canary
// looked at them. They are corrected now, on the rule that decided granularity -- a re-read of
// the same file is the authority on what that file says. The three text columns overwrite only
// when the re-read has an answer: a sidecar that has not been written yet, or a repository moved
// out from under a path, would otherwise clear a name that was correctly captured, and clearing
// is not correcting. ts carries no such empty case -- a record without a timestamp never reaches
// the store -- so it is assigned outright, which is what lets a fix to timestamp extraction move
// rows back into the days they belong to.
//
// subpath travels with project because one call to projectid.Resolve sets both, and its guard
// is the *project's* emptiness rather than its own: "" is the correct subpath for a session at
// a repository root, so guarding subpath on itself discards exactly the correction that matters
// -- a re-read resolving a deeper root (a `git init` in a subdirectory, a submodule) offers
// (deeper-project, ""), the project moves and the path stays relative to a root it no longer
// belongs to.
const restateActivitySQL = `
        UPDATE usage_record SET
            granularity = ?,
            ts = ?,
            model = CASE WHEN model = '' THEN ? ELSE model END,
            project = CASE WHEN ? <> '' THEN ? ELSE project END,
            subpath = CASE WHEN ? <> '' THEN ? ELSE subpath END, -- guarded on project, see below
            entrypoint = CASE WHEN ? <> '' THEN ? ELSE entrypoint END,
            git_branch = CASE WHEN ? <> '' THEN ? ELSE git_branch END,
            input_tokens = MAX(input_tokens, ?), output_tokens = MAX(output_tokens, ?),
            cache_read_tokens = MAX(cache_read_tokens, ?),
            cache_write_tokens = MAX(cache_write_tokens, ?),
            reasoning_tokens = MAX(reasoning_tokens, ?),
            cache_write_1h = MAX(cache_write_1h, ?),
            lines_added = ?, lines_removed = ?, rework_lines = ?,
            edits = ?, tool_calls = ?, rejected = ?, compactions = ?,
            tool_reads = ?, tool_searches = ?, tool_commands = ?, tool_writes = ?,
            tool_other = ?, tool_errors = ?,
            sidechain = MAX(sidechain, ?), skill = MAX(skill, ?), agent = MAX(agent, ?),
            cache_miss_reason = MAX(cache_miss_reason, ?)
        WHERE tool = ? AND dedupe_key = ?`

// InsertLocal writes records parsed from a local session file, restating the activity of a
// turn that is already stored. Use it only for files this store reads itself: the caller
// owns the input, so re-reading it is the store's own better knowledge of the same turn.
// Records arriving from elsewhere go through Insert, which stays first-write-wins.
// It returns how many rows it inserted and how many stored rows the re-read moved a figure
// *down* on -- see lowerRestateSQL for why that number is worth having.
func (s *Store) InsertLocal(ctx context.Context, recs []usage.Record) (inserted, lowered int, err error) {
	return s.insertWith(ctx, recs, restateActivitySQL, activityRestateArgs, lowerRestateSQL)
}

// InsertSynced writes records a team member pushed from their own local store, restating a
// turn that is already stored exactly as InsertLocal does. The rows are the pusher's own
// re-read of their own append-only logs, and no other member can reach them: the sync
// endpoint prefixes every dedupe_key with "<member>:" and the member charset excludes the
// colon, so each row has exactly one possible writer (internal/server/handlers.go). Routing
// this through first-write-wins Insert instead made the team dashboard the one surface where
// a partial figure could never be corrected -- a Claude response synced mid-stream, whose
// output count only reaches its true total on the response's last line, stayed at the
// partial number for good.
func (s *Store) InsertSynced(ctx context.Context, recs []usage.Record) (int, error) {
	// The lowering watch is the local ingest's: a server operator cannot act on a member's
	// parser regression, and the member's own backfill already reports it to whoever can.
	inserted, _, err := s.insertWith(ctx, recs, restateActivitySQL, activityRestateArgs, "")
	return inserted, err
}

// activityRestateArgs binds r to restateActivitySQL's placeholders.
func activityRestateArgs(r *usage.Record) []any {
	return []any{
		r.Granularity,
		r.Timestamp.UTC().Format(time.RFC3339),
		r.Model,
		r.Project, r.Project,
		r.Project, r.Subpath,
		r.Entrypoint, r.Entrypoint,
		r.GitBranch, r.GitBranch,
		r.InputTokens, r.OutputTokens, r.CacheReadTokens, r.CacheWriteTokens, r.ReasoningTokens,
		r.CacheWrite1hTokens,
		r.LinesAdded, r.LinesRemoved, r.ReworkLines,
		r.Edits, r.ToolCalls, r.Rejected, r.Compactions,
		r.ToolReads, r.ToolSearches, r.ToolCommands, r.ToolWrites, r.ToolOther, r.ToolErrors,
		r.Sidechain, r.Skill, r.Agent,
		r.CacheMissReason,
		r.Tool, r.DedupeKey,
	}
}

// lowerRestateSQL matches a stored row that the offered record would restate *downward* on any
// column the restate assigns. Those columns are assaio's own attribution rules, and a corrected
// rule reaching history is exactly why they are assigned rather than MAX'd (B116) -- which
// leaves no difference, from the store's side, between a fix landing and a parser regression
// quietly erasing evidence. Counting the moves is what makes the second one visible.
// LIMIT 1 because the question is whether this row moved down, not by how much on how many
// columns.
const lowerRestateSQL = `
        SELECT 1 FROM usage_record
        WHERE tool = ? AND dedupe_key = ?
          AND (lines_added > ? OR lines_removed > ? OR rework_lines > ?
               OR edits > ? OR tool_calls > ? OR rejected > ? OR compactions > ?
               OR tool_reads > ? OR tool_searches > ? OR tool_commands > ?
               OR tool_writes > ? OR tool_other > ? OR tool_errors > ?)
        LIMIT 1`

// lowerArgs binds r to lowerRestateSQL's placeholders.
func lowerArgs(r *usage.Record) []any {
	return []any{
		r.Tool, r.DedupeKey,
		r.LinesAdded, r.LinesRemoved, r.ReworkLines,
		r.Edits, r.ToolCalls, r.Rejected, r.Compactions,
		r.ToolReads, r.ToolSearches, r.ToolCommands, r.ToolWrites, r.ToolOther, r.ToolErrors,
	}
}
