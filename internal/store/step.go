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

// restateStepSQL corrects a step already stored from a re-read of the file it came from. The
// outcome and token total are the ones a longer prefix of the same transcript can complete, and
// each is monotone in the prefix read, so a re-read restates upward and never degrades a row --
// the same argument InsertLocal makes for the activity counts.
//
// ordinal and target_ref are assigned rather than kept, for the reason granularity is assigned
// in restateActivitySQL: each is a claim the current parse is the authority on, not a value a
// later parse can only improve. A parser change that reads one more step out of a transcript
// shifts every later position; widening which calls register a target renumbers first-seen
// order the same way. Keeping the stored value would leave one sequence holding a mix of two
// parsers' numbering -- a 7 beside a 3 that means the same file, which nothing downstream could
// detect as wrong.
const restateStepSQL = `
        UPDATE session_step SET
            outcome = CASE WHEN outcome = '' THEN ? ELSE outcome END,
            target_ref = ?,
            tokens = MAX(tokens, ?),
            ordinal = ?
        WHERE tool = ? AND timeline = ? AND dedupe_key = ?`

// InsertSteps writes a session's step sequence, restating a step already stored. It returns
// how many rows were new and how many were refused at the vocabulary boundary, so neither
// growth nor loss goes unreported -- the skip-and-count policy the parsers follow.
//
// Steps are written only for files this store read itself, exactly like InsertLocal: the
// caller owns the input, so a re-read is the store's own better knowledge of the same step.
func (s *Store) InsertSteps(ctx context.Context, steps []usage.Step) (inserted, rejected int, err error) {
	if len(steps) == 0 {
		return 0, 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback() }()

	ins, err := tx.PrepareContext(ctx, insertStepSQL)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = ins.Close() }()
	upd, err := tx.PrepareContext(ctx, restateStepSQL)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = upd.Close() }()

	for i := range steps {
		st := &steps[i]
		if !usage.ValidStepKind(st.Kind) || !usage.ValidStepOutcome(st.Outcome) {
			// A vocabulary the readers cannot interpret is rejected at the boundary rather
			// than stored and rendered as a category nobody defined.
			rejected++
			continue
		}
		res, err := ins.ExecContext(ctx, st.Tool, st.SessionID, st.Timeline, st.DedupeKey,
			st.Timestamp.UTC().Format(time.RFC3339), st.Ordinal, st.Kind, st.Outcome,
			st.Model, st.Tokens, st.TargetRef)
		if err != nil {
			return 0, 0, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return 0, 0, err
		}
		if n > 0 {
			inserted++
			continue
		}
		if _, err := upd.ExecContext(ctx, st.Outcome, st.TargetRef, st.Tokens, st.Ordinal,
			st.Tool, st.Timeline, st.DedupeKey); err != nil {
			return 0, 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return inserted, rejected, nil
}

// PruneSteps drops steps older than before and reports how many went. This is the bound the
// table ships with: a step row is 2.21 times as frequent as a usage record, and without a
// horizon the sequence would grow with install age while the reports over it only ever look
// at a recent window.
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
