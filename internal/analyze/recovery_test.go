package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// The correction this detector exists to make. Per *step*, the aftermath of a failure looks
// expensive purely because the steps following one are mostly assistant turns while the window's
// average step is half tool calls carrying no tokens. Per *turn*, the same fixture is flat.
//
// The fixture is built so the two answers cannot coincide: every turn costs exactly 1,000 tokens,
// so the honest ratio is 1.00 and any step-based reading must come out above it.
func TestRecoveryComparesTurnsToTurnsRatherThanStepsToSteps(t *testing.T) {
	const spec = "a r1 r1 c!err a a c r1 r1 c r1 r1 c r1 r1 a r1 r1 c r1 r1 c r1 r1 a r1 r1 c r1 r1 c" +
		" a r1 r1 c r1 r1 c a r1 r1 c r1 r1 c a r1 r1 c r1 r1 c a r1 r1 c!err a a c r1 r1 c r1 r1 c" +
		" a r1 r1 c r1 r1 c a r1 r1 c r1 r1 c a r1 r1 c r1 r1 c a r1 r1 c r1 r1 c a r1 r1 c r1 r1 c" +
		" a r1 r1 c!err a a c r1 r1 c r1 r1 c a r1 r1 c r1 r1 c a r1 r1 c r1 r1 c a r1 r1 c r1 r1 c"
	in := Input{Trace: traceOf(sequence(t, "s", "assaio", spec))}

	a := buildAftermath(interactiveView(&in.Trace), in.Trace.Newest)
	ratio, ok := a.CostRatio()
	if !ok {
		t.Fatalf("no ratio from %d turns after %d failures", a.AfterTurns, a.Failures)
	}
	if ratio != 1 {
		t.Errorf("cost ratio = %v, want exactly 1: every turn in this fixture costs the same, so any "+
			"other answer is the sample's composition rather than a cost", ratio)
	}
}

// The aftermath is a bounded window, not "the rest of the session": a turn far past the failure is
// ordinary work, and counting it would dilute every ratio toward 1 by construction.
func TestRecoveryCountsOnlyTheTurnsInsideItsWindow(t *testing.T) {
	near := "c!err a:5000" + strings.Repeat(" o", recoveryWindow+4) + " a:1000"
	in := Input{Trace: traceOf(sequence(t, "s", "assaio", near))}

	a := buildAftermath(interactiveView(&in.Trace), in.Trace.Newest)

	if a.AfterTurns != 1 || a.AfterTokens != 5000 {
		t.Errorf("aftermath = %d turns / %d tokens, want the one turn inside the window",
			a.AfterTurns, a.AfterTokens)
	}
	if a.Turns != 2 || a.Tokens != 6000 {
		t.Errorf("baseline = %d turns / %d tokens, want both turns", a.Turns, a.Tokens)
	}
}

// A session's ending is only readable once it has stopped. A live session's last step is whatever
// it is doing right now, and on the audited store the sessions inside that tail outnumber the ones
// that really ended on a failure five to one.
func TestRecoveryExcludesSessionsStillRunningFromTheEndingFigure(t *testing.T) {
	settled := sequenceEndedAgo(t, "settled", "assaio", "a c!err", 6*time.Hour)
	live := sequence(t, "live", "assaio", "a c!err")
	finished := sequenceEndedAgo(t, "finished", "assaio", "a c", 6*time.Hour)
	in := Input{Trace: traceOf(settled, live, finished)}

	a := buildAftermath(interactiveView(&in.Trace), in.Trace.Newest)

	if a.Open != 1 {
		t.Errorf("open sessions = %d, want the one whose last step is at the newest", a.Open)
	}
	if len(a.Abandoned) != 1 || a.Abandoned[0].SessionID != "settled" {
		t.Errorf("abandoned = %+v, want only the settled one that ended on a failure", a.Abandoned)
	}
}

// A truncated response produced work and hit a ceiling; an unstated outcome is the source saying
// nothing. Reading either as a failure would invent sessions that gave up.
func TestRecoveryTreatsOnlyErrorsAndDenialsAsFailures(t *testing.T) {
	tests := []struct {
		name      string
		spec      string
		failures  int
		abandoned int
	}{
		{name: "an error", spec: "a c!err", failures: 1, abandoned: 1},
		{name: "a denial", spec: "a c!den", failures: 1, abandoned: 1},
		{name: "a truncated turn", spec: "a a!trunc", failures: 0, abandoned: 0},
		{name: "an unstated outcome", spec: "a o", failures: 0, abandoned: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seq := sequenceEndedAgo(t, "s", "assaio", tt.spec, 6*time.Hour)
			// A second, later sequence moves the newest step away from this one's tail.
			set := traceOf(seq, sequence(t, "later", "assaio", "a"))
			a := buildAftermath(interactiveView(&set), set.Newest)
			if a.Failures != tt.failures || len(a.Abandoned) != tt.abandoned {
				t.Errorf("failures/abandoned = %d/%d, want %d/%d",
					a.Failures, len(a.Abandoned), tt.failures, tt.abandoned)
			}
		})
	}
}

// Compaction is counted apart from failure: losing the context is a different event from a call
// that did not land, and the share of work done afterwards is the figure the published Copilot
// trace study makes interesting.
func TestRecoveryCountsTheStepsRunAfterAContextLoss(t *testing.T) {
	in := Input{Trace: traceOf(sequence(t, "s", "assaio", "a e1 k a e1 c"))}

	a := buildAftermath(interactiveView(&in.Trace), in.Trace.Newest)

	if a.Compactions != 1 {
		t.Fatalf("compactions = %d, want 1", a.Compactions)
	}
	if a.StepsAfterCompaction != 3 {
		t.Errorf("steps after the compaction = %d, want the 3 that follow it", a.StepsAfterCompaction)
	}
	if a.CompactedShare() != 0.5 {
		t.Errorf("compacted share = %v, want half of the six steps", a.CompactedShare())
	}
}

// A window where nothing failed must say so, not report a ratio of zero or an all-clear it never
// tested.
func TestRecoveryOnAWindowWhereNothingFailed(t *testing.T) {
	in := Input{Trace: traceOf(sequence(t, "s", "assaio", "a e1 c a e2"))}

	got := Evaluate(recoveryValidator{}, &in)

	if figureValue(t, &got, "cost of the turns after a failure") != "—" {
		t.Errorf("figures = %+v, want a dash rather than a ratio against nothing", got.Figures)
	}
	if got.Read != noDataRead {
		t.Errorf("Read = %+v, want no verdict", got.Read)
	}
	if !strings.Contains(got.Takeaway, "no recovery to measure") {
		t.Errorf("Takeaway = %q, want it to say nothing failed", got.Takeaway)
	}
}

// The refusal is part of the contract, and so is the composition warning: the number this detector
// does not publish is the one a reader is most likely to recompute wrongly.
func TestRecoveryDeclaresWhatItCannotDistinguish(t *testing.T) {
	in := Input{Trace: favorableTrace(t)}
	got := Evaluate(recoveryValidator{}, &in)
	for _, want := range []string{"Cannot distinguish", "only the last one *stored*", "composition"} {
		if !hasCaveatContaining(got.Caveats, want) {
			t.Errorf("caveats do not mention %q: %v", want, got.Caveats)
		}
	}
}

// sequenceEndedAgo builds a sequence whose steps all sit d earlier, so a test can place one
// sequence's ending well before the newest step in the set -- which is what separates a session
// that stopped from one still running.
func sequenceEndedAgo(t *testing.T, id, project, spec string, d time.Duration) store.Timeline {
	t.Helper()
	seq := sequence(t, id, project, spec)
	for i := range seq.Steps {
		seq.Steps[i].Timestamp = seq.Steps[i].Timestamp.Add(-d)
	}
	return seq
}

// TestRecoveryWhenEveryTurnFollowsAFailure is the case the baseline fix made reachable: with
// the aftermath out of the denominator, a scope whose every assistant turn follows a failure
// has no baseline left. Reporting that as "nothing follows a failure" says the opposite of what
// happened.
func TestRecoveryWhenEveryTurnFollowsAFailure(t *testing.T) {
	var a aftermath
	a.Turns, a.AfterTurns = 4, 4
	a.Tokens, a.AfterTokens = 400, 400
	a.Failures = 1

	if _, ok := a.CostRatio(); ok {
		t.Fatal("a ratio was computed against an empty baseline")
	}
	if note := recoveryUnratedNote(&a); !strings.Contains(note, "no baseline left") {
		t.Fatalf("note = %q, want it to name the empty baseline", note)
	}
	if got := recoveryTakeaway(false, 0, &a); !strings.Contains(got, "nothing here to compare") {
		t.Fatalf("takeaway = %q, want it to say what is missing", got)
	}
}

// TestRecoveryFigureNamesTheDenominatorItActuallyUses guards the conflation the baseline fix
// exists to end.
func TestRecoveryFigureNamesTheDenominatorItActuallyUses(t *testing.T) {
	var a aftermath
	a.Turns, a.AfterTurns = 100, 20
	a.Tokens, a.AfterTokens = 10000, 2400
	ratio, ok := a.CostRatio()
	if !ok {
		t.Fatal("expected a ratio")
	}
	f := recoveryCostFigure(&a, ratio, true)
	if strings.Contains(f.Value, "the window's own turn") {
		t.Fatalf("figure = %q, still names the denominator the fix removed", f.Value)
	}
	if !strings.Contains(f.Value, "follows none") {
		t.Fatalf("figure = %q, want the baseline named as turns following no failure", f.Value)
	}
}
