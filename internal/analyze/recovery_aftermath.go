package analyze

import (
	"time"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/trace"
	"github.com/assaio/assaio/internal/usage"
)

const (
	// recoveryWindow is how many steps after a failure count as its aftermath. Positional, not
	// by ordinal: the horizon can cut a sequence's opening away, and "the next few steps we can
	// see" is the only distance that survives that.
	recoveryWindow = 10
	// recoveryOpenTail is how close to the newest step a sequence's last step may be before it is
	// treated as still running. A live session's last step is whatever it is doing right now, and
	// a failure it resolves a minute later would otherwise be published as an abandoned session:
	// 19 of 290 sequences on the audited store sit inside this tail against 4 that end on a
	// failure, so the exclusion is larger than the finding it protects.
	recoveryOpenTail = time.Hour
)

// aftermath is what one scope's sequences did after something went wrong.
//
// Turns and Tokens count assistant steps only, and so do AfterTurns and AfterTokens. That is the
// whole correctness of this metric: measured over *all* steps, the aftermath of a failure reads
// 1.06x the window's average step over this window and 1.35x over a three-step one, but only
// because the steps following a failure are more heavily assistant turns than the window is
// (49.2% and 62.2% against 47.7%) while a tool call carries no tokens at all. That ratio reports
// the composition of the sample. Turn against turn, the same corpus reads 1.02x.
type aftermath struct {
	Sequences int
	// Open is how many sequences were left out of Abandoned for still running.
	Open int
	// Failures counts steps that failed or were declined; Compactions counts the context losses.
	// Kept apart because they are different events: one is work that did not land, the other is
	// the agent losing what it knew.
	Failures, Compactions int
	Steps                 int
	StepsAfterCompaction  int
	Turns, Tokens         int64
	AfterTurns            int64
	AfterTokens           int64
	// Abandoned are the sequences whose last visible step failed or was declined, having excluded
	// the ones still running.
	Abandoned []store.Timeline
}

// buildAftermath reads the scope's sequences. newest is the set's own latest step rather than the
// wall clock, so a store ingested last week does not read every sequence in it as live.
func buildAftermath(v *trace.View, newest time.Time) aftermath {
	var a aftermath
	a.Sequences = len(v.Sequences)
	for i := range v.Sequences {
		a.foldSequence(&v.Sequences[i], newest)
	}
	return a
}

func (a *aftermath) foldSequence(t *store.Timeline, newest time.Time) {
	sinceFailure := recoveryWindow + 1
	compacted := false
	for i := range t.Steps {
		s := &t.Steps[i]
		a.Steps++
		if compacted {
			a.StepsAfterCompaction++
		}
		if s.Kind == usage.StepAssistant {
			a.Turns++
			a.Tokens += s.Tokens
			if sinceFailure <= recoveryWindow {
				a.AfterTurns++
				a.AfterTokens += s.Tokens
			}
		}
		sinceFailure++
		switch {
		case s.Kind == usage.StepCompaction:
			a.Compactions++
			compacted = true
			sinceFailure = 1
		case failedOutcome(s.Outcome):
			a.Failures++
			sinceFailure = 1
		}
	}
	a.countEnding(t, newest)
}

// countEnding classifies how the sequence stopped. A sequence with no steps cannot have ended in
// anything, and one whose last step is inside the open tail has not ended at all.
func (a *aftermath) countEnding(t *store.Timeline, newest time.Time) {
	if len(t.Steps) == 0 {
		return
	}
	last := &t.Steps[len(t.Steps)-1]
	if !last.Timestamp.Before(newest.Add(-recoveryOpenTail)) {
		a.Open++
		return
	}
	if failedOutcome(last.Outcome) {
		a.Abandoned = append(a.Abandoned, *t)
	}
}

// failedOutcome reports whether a step did not do what it was asked. A truncated response is not
// one: it produced work and hit a length ceiling. "" is the source saying nothing, which is never
// read as a failure.
func failedOutcome(outcome string) bool {
	return outcome == usage.OutcomeError || outcome == usage.OutcomeDenied
}

// TokensPerTurn is the window's own baseline: what an assistant turn costs anywhere in it.
func (a *aftermath) TokensPerTurn() float64 { return perUnit(a.Tokens, a.Turns) }

// TokensPerTurnAfter is what an assistant turn costs inside the aftermath of a failure.
func (a *aftermath) TokensPerTurnAfter() float64 { return perUnit(a.AfterTokens, a.AfterTurns) }

// CostRatio is the aftermath's cost against the window's own, and whether both sides had turns to
// divide by. 1.0 means a failure changed nothing about what the next turns cost.
func (a *aftermath) CostRatio() (ratio float64, ok bool) {
	base, after := a.TokensPerTurn(), a.TokensPerTurnAfter()
	if base <= 0 || a.AfterTurns == 0 {
		return 0, false
	}
	return after / base, true
}

// CompactedShare is the share of the scope's steps that ran after their sequence lost its context.
func (a *aftermath) CompactedShare() float64 {
	return shareOf(int64(a.StepsAfterCompaction), int64(a.Steps))
}

// AbandonedShare is the share of the sequences that could be judged which ended on a failure.
func (a *aftermath) AbandonedShare() float64 {
	return shareOf(int64(len(a.Abandoned)), int64(a.Sequences-a.Open))
}

func perUnit(total, n int64) float64 {
	if n <= 0 {
		return 0
	}
	return float64(total) / float64(n)
}
