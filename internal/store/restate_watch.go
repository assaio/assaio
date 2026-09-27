package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// Restated is what one InsertLocal call's re-reads changed on rows that already existed. From
// the store's side a corrected rule landing and a parser regression look identical, so both
// are counted rather than left to be inferred from a figure that quietly moved.
type Restated struct {
	// Lowered is rows on which an assigned activity figure went down.
	Lowered int
	// Identity is rows whose stored identity a re-read replaced or declined to report.
	Identity IdentityChanges
}

// IdentityChanges counts rows by the identity column a re-read changed. A row changing two
// counted columns counts once in Rows and once in each; Project includes a subpath-only change,
// and Label is skill, agent and cache-miss reason together. Filling a blank is not a change: a
// session read while it is still being written fills names later as a matter of course.
type IdentityChanges struct {
	Rows, Model, TS, Project, Entrypoint, Branch, Label int
	// Kept is rows on which a stored name was kept because the re-read offered none -- what a
	// parser that stopped reading a field looks like. Never applied, only counted.
	Kept int
}

// Add folds o into c.
func (c *IdentityChanges) Add(o IdentityChanges) {
	c.Rows += o.Rows
	c.Model += o.Model
	c.TS += o.TS
	c.Project += o.Project
	c.Entrypoint += o.Entrypoint
	c.Branch += o.Branch
	c.Label += o.Label
	c.Kept += o.Kept
}

// watchSQL reads, before the restate runs, whether it will lower an activity figure and the
// identity it will overwrite. One point lookup on the (tool, dedupe_key) unique index; the
// comparison happens in identityChanges, where each column's rule can be tested without a store.
const watchSQL = `
        SELECT (lines_added > ? OR lines_removed > ? OR rework_lines > ?
                OR edits > ? OR tool_calls > ? OR rejected > ? OR compactions > ?
                OR tool_reads > ? OR tool_searches > ? OR tool_commands > ?
                OR tool_writes > ? OR tool_other > ? OR tool_errors > ?),
               ts, model, project, subpath, entrypoint, git_branch,
               skill, agent, cache_miss_reason
        FROM usage_record WHERE tool = ? AND dedupe_key = ?`

func watchArgs(r *usage.Record) []any {
	return []any{
		r.LinesAdded, r.LinesRemoved, r.ReworkLines,
		r.Edits, r.ToolCalls, r.Rejected, r.Compactions,
		r.ToolReads, r.ToolSearches, r.ToolCommands, r.ToolWrites, r.ToolOther, r.ToolErrors,
		r.Tool, r.DedupeKey,
	}
}

// identityRow is the row's identity as it stands before the restate.
type identityRow struct {
	ts, model, project, subpath, entrypoint, branch string
	skill, agent, missReason                        string
}

// watch runs watchSQL for r. The row exists by construction -- the caller asks only after the
// insert reported a conflict inside the same transaction -- so a missing row is an error, not
// an agreement.
func watch(ctx context.Context, stmt *sql.Stmt, r *usage.Record) (lowered bool, c IdentityChanges, err error) {
	var s identityRow
	err = stmt.QueryRowContext(ctx, watchArgs(r)...).Scan(&lowered,
		&s.ts, &s.model, &s.project, &s.subpath, &s.entrypoint, &s.branch,
		&s.skill, &s.agent, &s.missReason)
	if err != nil {
		return false, IdentityChanges{}, fmt.Errorf("watching restate of %s %s: %w", r.Tool, r.DedupeKey, err)
	}
	return lowered, identityChanges(&s, r), nil
}

// identityChanges compares one stored row with the record about to restate it, applying the
// same rules restateActivitySQL applies. A sub-agent aggregate's context columns are not
// watched: every parent transcript that ran it carries it, so they follow whichever file was
// read last, and its project_conflict state is only ever set on those rows. Its model and time
// are watched: the model comes from the sub-agent itself, and the time settles on the earliest
// carrier.
func identityChanges(s *identityRow, r *usage.Record) IdentityChanges {
	var c IdentityChanges
	kept := false
	named := func(stored, offered string, n *int) {
		switch {
		case stored == "":
		case offered == "":
			kept = true
		case stored != offered:
			*n++
		}
	}
	named(s.model, r.Model, &c.Model)
	label := 0
	named(s.skill, r.Skill, &label)
	named(s.agent, r.Agent, &label)
	named(s.missReason, r.CacheMissReason, &label)
	c.Label = min(label, 1)
	if movesTime(s.ts, r) {
		c.TS = 1
	}
	if !isAgentAggregate(r.Tool, r.DedupeKey) {
		named(s.entrypoint, r.Entrypoint, &c.Entrypoint)
		named(s.branch, r.GitBranch, &c.Branch)
		if !r.ProjectGuessed {
			named(s.project, r.Project, &c.Project)
			if r.Project != "" && s.project == r.Project && s.subpath != r.Subpath {
				c.Project = 1
			}
		}
	}
	if c.Model+c.TS+c.Project+c.Entrypoint+c.Branch+c.Label > 0 {
		c.Rows = 1
	}
	if kept {
		c.Kept = 1
	}
	return c
}

// movesTime mirrors restateActivitySQL's ts rule: Claude Code keeps the earliest time offered,
// every other source takes the re-read's.
func movesTime(stored string, r *usage.Record) bool {
	offered := r.Timestamp.UTC().Format(time.RFC3339)
	if r.Tool == "claude-code" {
		return offered < stored
	}
	return offered != stored
}

// isAgentAggregate mirrors restateActivitySQL's `tool = 'claude-code' AND dedupe_key LIKE
// 'agent:%'`; the two must change together.
func isAgentAggregate(tool, dedupeKey string) bool {
	return tool == "claude-code" && strings.HasPrefix(dedupeKey, "agent:")
}
