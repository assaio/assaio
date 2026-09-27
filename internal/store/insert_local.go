package store

import (
	"context"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// restateActivitySQL corrects a stored turn from a re-read of the file it came from. It
// reaches every column a later parser can extract better -- the token counts, the activity,
// the cache tier and miss reason 0007 added, and the identity columns. The token counts take
// MAX(stored, offered), which is what makes a half-read turn repairable without ever degrading
// one: a log is append-only, so the later read is the more complete one.
//
// The token counts are here because a Claude response's blocks arrive across several lines
// and only the last carries its true output count: a session read while it was still being
// written stores the partial figure, and without a restate that undercount would be permanent
// -- the response id already exists, so the completed read has nothing new to insert. Keeping
// cache_write_tokens beside cache_write_1h also holds the subset invariant the 1-hour share
// divides by; restating one without the other could put the portion above its whole, and
// reasoning_tokens is here for the same reason against output_tokens.
//
// model, skill, agent and cache_miss_reason are names: a re-read that states one replaces the
// stored one, and a re-read that states none keeps it. A source can emit a record before it
// knows the name (Cline reads the model from a task_metadata.json sidecar that may not exist
// yet), so a blank is not an answer; a different name is, because a model decides the price and
// a label decides a grouping. MAX would only let a correction through when it sorts higher.
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
// ts, project, entrypoint and git_branch follow the rule that decides granularity: a re-read
// of the same file is the authority on what that file says, so a parser fix to any of them
// reaches stored rows. The three text columns overwrite only when the re-read has an answer: a
// sidecar that has not been written yet would otherwise clear a name that was correctly
// captured, and clearing is not correcting. A working directory that no longer exists is not an
// answer either -- what resolves for it now saw less than the read that stored the project
// (usage.Record.ProjectGuessed) -- so the args offer it as none. ts is assigned, except for
// Claude Code, where one response can be carried by several transcripts (a forked sub-agent
// replays its origin's prefix with the fork's later time): there the earliest line that carries
// it is when it happened, and assigning would leave the stored time to whichever file was read
// last. For Claude Code a fix that moves timestamps earlier reaches stored rows; one that moves
// them later does not. Every other source keys a row by position, so one key is one line.
//
// subpath travels with project because one call to projectid.Resolve sets both, and its guard
// is the *project's* emptiness rather than its own: "" is the correct subpath for a session at
// a repository root, so guarding subpath on itself discards exactly the correction that matters
// -- a re-read resolving a deeper root (a `git init` in a subdirectory, a submodule) offers
// (deeper-project, ""), the project moves and the path stays relative to a root it no longer
// belongs to. A completed Claude sub-agent aggregate has a narrower rule: `agent:<id>` is its
// source identity, so two non-empty projects on that identity are competing claims rather than
// a correction. project_conflict is sticky and keeps both project and subpath empty; applying
// that rule to ordinary turns would make the B116 correction path unreachable.
const restateActivitySQL = `
        UPDATE usage_record SET
            granularity = ?,
            ts = CASE WHEN tool = 'claude-code' THEN MIN(ts, ?) ELSE ? END,
            model = CASE WHEN ? <> '' THEN ? ELSE model END,
            project_conflict = CASE
                WHEN project_conflict = 1 THEN 1
                WHEN tool = 'claude-code' AND dedupe_key LIKE 'agent:%'
                     AND ? <> '' AND project <> '' AND project <> ? THEN 1
                ELSE 0
            END,
            project = CASE
                WHEN project_conflict = 1 THEN ''
                WHEN tool = 'claude-code' AND dedupe_key LIKE 'agent:%'
                     AND ? <> '' AND project <> '' AND project <> ? THEN ''
                WHEN ? <> '' THEN ?
                ELSE project
            END,
            subpath = CASE
                WHEN project_conflict = 1 THEN ''
                WHEN tool = 'claude-code' AND dedupe_key LIKE 'agent:%'
                     AND ? <> '' AND project <> '' AND project <> ? THEN ''
                WHEN ? <> '' THEN ?
                ELSE subpath
            END,
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
            sidechain = MAX(sidechain, ?),
            skill = CASE WHEN ? <> '' THEN ? ELSE skill END,
            agent = CASE WHEN ? <> '' THEN ? ELSE agent END,
            cache_miss_reason = CASE WHEN ? <> '' THEN ? ELSE cache_miss_reason END
        WHERE tool = ? AND dedupe_key = ?`

// InsertLocal writes records parsed from a local session file, restating the activity of a
// turn that is already stored. Use it only for files this store reads itself: the caller
// owns the input, so re-reading it is the store's own better knowledge of the same turn.
// Records arriving from elsewhere go through Insert, which stays first-write-wins.
// It returns how many rows it inserted and what the re-read changed on rows already stored
// (Restated).
func (s *Store) InsertLocal(ctx context.Context, recs []usage.Record) (int, Restated, error) {
	return s.insertWith(ctx, recs, restateActivitySQL, activityRestateArgs, true)
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
	// The restate watch is the local ingest's: a server operator cannot act on a member's
	// parser regression, and the member's own backfill already reports it to whoever can.
	inserted, _, err := s.insertWith(ctx, recs, restateActivitySQL, activityRestateArgs, false)
	return inserted, err
}

// activityRestateArgs binds r to restateActivitySQL's placeholders. A guessed project is
// offered as no answer, so the stored one is kept.
func activityRestateArgs(r *usage.Record) []any {
	project, subpath := r.Project, r.Subpath
	if r.ProjectGuessed {
		project, subpath = "", ""
	}
	return []any{
		r.Granularity,
		r.Timestamp.UTC().Format(time.RFC3339), r.Timestamp.UTC().Format(time.RFC3339),
		r.Model, r.Model,
		project, project,
		project, project, project, project,
		project, project, project, subpath,
		r.Entrypoint, r.Entrypoint,
		r.GitBranch, r.GitBranch,
		r.InputTokens, r.OutputTokens, r.CacheReadTokens, r.CacheWriteTokens, r.ReasoningTokens,
		r.CacheWrite1hTokens,
		r.LinesAdded, r.LinesRemoved, r.ReworkLines,
		r.Edits, r.ToolCalls, r.Rejected, r.Compactions,
		r.ToolReads, r.ToolSearches, r.ToolCommands, r.ToolWrites, r.ToolOther, r.ToolErrors,
		r.Sidechain,
		r.Skill, r.Skill, r.Agent, r.Agent,
		r.CacheMissReason, r.CacheMissReason,
		r.Tool, r.DedupeKey,
	}
}
