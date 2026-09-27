package store

import (
	"context"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

const insertStepSQL = `
        INSERT OR IGNORE INTO session_step
            (tool, session_id, timeline, dedupe_key, ts, ordinal, kind, outcome, model,
             tokens, target_ref)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

// StepWrite is what one InsertSteps call did: rows new, refused at the vocabulary boundary, and
// stored steps whose time, kind, position, target or model a re-read changed.
type StepWrite struct {
	Inserted, Rejected, Changed int
}

// InsertSteps writes a session's step sequence, restating a step already stored, and reports
// new, refused and changed steps so neither growth, loss nor a rewrite goes unreported -- the
// skip-and-count policy the parsers follow.
//
// Steps are written only for files this store read itself, exactly like InsertLocal: the
// caller owns the input, so a re-read is the store's own better knowledge of the same step.
func (s *Store) InsertSteps(ctx context.Context, steps []usage.Step) (StepWrite, error) {
	var w StepWrite
	if len(steps) == 0 {
		return w, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return w, err
	}
	defer func() { _ = tx.Rollback() }()

	ins, err := tx.PrepareContext(ctx, insertStepSQL)
	if err != nil {
		return w, err
	}
	defer func() { _ = ins.Close() }()
	watched, err := tx.PrepareContext(ctx, restateStepWatchedSQL)
	if err != nil {
		return w, err
	}
	defer func() { _ = watched.Close() }()
	rest, err := tx.PrepareContext(ctx, restateStepRestSQL)
	if err != nil {
		return w, err
	}
	defer func() { _ = rest.Close() }()

	for i := range steps {
		st := &steps[i]
		if !usage.ValidStepKind(st.Kind) || !usage.ValidStepOutcome(st.Outcome) {
			// A vocabulary the readers cannot interpret is rejected at the boundary rather
			// than stored and rendered as a category nobody defined.
			w.Rejected++
			continue
		}
		res, err := ins.ExecContext(ctx, st.Tool, st.SessionID, st.Timeline, st.DedupeKey,
			st.Timestamp.UTC().Format(time.RFC3339), st.Ordinal, st.Kind, st.Outcome,
			st.Model, st.Tokens, st.TargetRef)
		if err != nil {
			return w, err
		}
		if n, err := res.RowsAffected(); err != nil {
			return w, err
		} else if n > 0 {
			w.Inserted++
			continue
		}
		changed, err := restateStep(ctx, watched, rest, st)
		if err != nil {
			return w, err
		}
		if changed {
			w.Changed++
		}
	}
	return w, tx.Commit()
}

// PruneSteps drops steps older than before and reports how many went. This is the bound the
// table ships with, re-measured on the maintainer's store for v0.26: 136.3 MB of table and
// indexes against usage_record's 69.7 MB, and 2.14 steps per record over the 30 days both cover.
// The ratio must be age-matched -- session_step is pruned and usage_record is not, so their raw
// totals compare a bounded table against an unbounded one.
//
// Deleting does not shrink the file -- only Vacuum does. That is why this returns a count the
// caller can report rather than pretending the space came back.
func (s *Store) PruneSteps(ctx context.Context, before time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM session_step WHERE ts < ?`, before.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// StepHorizon is the oldest and newest step the store holds, and how many there are. Zero
// times mean the table is empty, which is a different fact from a session with no steps.
type StepHorizon struct {
	Oldest, Newest time.Time
	Steps          int64
}

// Steps reports the horizon of the stored sequence, so every figure read off it can say how
// far back the history behind it actually goes.
func (s *Store) Steps(ctx context.Context) (StepHorizon, error) {
	var h StepHorizon
	var oldest, newest *string
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*), MIN(ts), MAX(ts) FROM session_step`).Scan(&h.Steps, &oldest, &newest)
	if err != nil {
		return StepHorizon{}, err
	}
	h.Oldest = parseStoredTime(oldest)
	h.Newest = parseStoredTime(newest)
	return h, nil
}

func parseStoredTime(v *string) time.Time {
	if v == nil {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, *v)
	if err != nil {
		return time.Time{}
	}
	return t
}
