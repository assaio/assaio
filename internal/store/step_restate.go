package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// restateStepSet corrects a step already stored from a re-read of the file it came from. Every
// column but model is assigned, because each is assaio's own claim rather than a figure read off
// the log: the position a step holds, the file identity behind it, the sum of four chosen token
// fields, the mapping of a stop reason or an attributed result to an outcome, and the
// classification of a tool call. A rule assaio wrote is a rule assaio can get wrong, and MAX or a
// fill-only CASE pins the old answer on every stored row forever. Ingest re-reads whole files,
// and each rule is monotone in the prefix read, so a half-written session still restates upward.
// ts follows usage_record.ts, which comes from the same line (for Claude Code the earliest
// offered wins), and PruneSteps reads it. Ingest drops a step whose offered time is past the
// horizon before it gets here, so a step whose earliest carrier is past it keeps a later carrier's
// time while its record keeps the earlier one. model follows usage_record's naming rule: a stated
// name replaces the stored one, a blank keeps it.
//
// It runs as two guarded statements. The first applies only when a watched column -- ts, kind,
// ordinal, target_ref or a stated model -- differs, and its row count is the change the caller
// reports; outcome and tokens move legitimately while a session is still being written, so they
// are applied by the second without being counted. A re-read that changes nothing writes nothing.
const restateStepSet = `
        UPDATE session_step SET
            ts = CASE WHEN tool = 'claude-code' THEN MIN(ts, ?) ELSE ? END,
            kind = ?, outcome = ?, target_ref = ?, tokens = ?, ordinal = ?,
            model = CASE WHEN ? <> '' THEN ? ELSE model END
        WHERE tool = ? AND timeline = ? AND dedupe_key = ?`

const (
	restateStepWatchedSQL = restateStepSet + `
          AND ((tool = 'claude-code' AND ts > ?) OR (tool <> 'claude-code' AND ts <> ?)
               OR kind <> ? OR ordinal <> ? OR target_ref <> ?
               OR (model <> '' AND ? <> '' AND model <> ?))`
	restateStepRestSQL = restateStepSet + `
          AND (outcome <> ? OR tokens <> ? OR (model = '' AND ? <> ''))`
)

func restateStepArgs(st *usage.Step, ts string) []any {
	return []any{
		ts, ts, st.Kind, st.Outcome, st.TargetRef, st.Tokens, st.Ordinal,
		st.Model, st.Model,
		st.Tool, st.Timeline, st.DedupeKey,
	}
}

// restateStep applies the two guarded statements to one stored step and reports whether a
// watched column changed.
func restateStep(ctx context.Context, watched, rest *sql.Stmt, st *usage.Step) (bool, error) {
	ts := st.Timestamp.UTC().Format(time.RFC3339)
	res, err := watched.ExecContext(ctx,
		append(restateStepArgs(st, ts), ts, ts, st.Kind, st.Ordinal, st.TargetRef, st.Model, st.Model)...)
	if err != nil {
		return false, err
	}
	if n, err := res.RowsAffected(); err != nil || n > 0 {
		return n > 0, err
	}
	_, err = rest.ExecContext(ctx, append(restateStepArgs(st, ts), st.Outcome, st.Tokens, st.Model)...)
	return false, err
}
