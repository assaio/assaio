package trace

import (
	"strconv"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

var origin = time.Date(2026, 5, 4, 9, 0, 0, 0, time.UTC)

// seq builds one sequence of n steps a minute apart from start. timeline is the sub-agent id
// ("" for a session's main loop) and entrypoint is what the source recorded; together they are
// everything Scope reads.
func seq(session, timeline, entrypoint string, start time.Time, n int) store.Timeline {
	t := store.Timeline{
		Tool: "claude-code", SessionID: session, Timeline: timeline, Entrypoint: entrypoint,
	}
	for i := range n {
		t.Steps = append(t.Steps, store.TimelineStep{
			Ordinal:   int64(i + 1),
			Timestamp: start.Add(time.Duration(i) * time.Minute),
		})
	}
	return t
}

// The invariant every detector's denominator rests on: asking for a scope partitions the set.
// Each sequence is either in the view or counted as excluded -- never both, never neither -- and
// the four scopes together cover the set exactly once. Without it a rate published over one
// declared population would be measured over another.
func TestScopePartitionsTheSet(t *testing.T) {
	set := New([]store.Timeline{
		seq("s1", "", entrypointCLI, origin, 4),
		seq("s2", "", entrypointSDKPy, origin, 100),
		seq("s3", "", entrypointSDKCLIRs, origin, 7),
		seq("s4", "agent-1", entrypointCLI, origin, 3),
		seq("s5", "", "", origin, 2),
		seq("s6", "", "wasm-repl", origin, 1),
	})

	held, heldSteps := 0, 0
	for _, scope := range []string{Interactive, SubAgent, Programmatic, Unstated} {
		v := set.Scope(scope)
		if got := len(v.Sequences) + v.ExcludedSequences; got != set.Sequences() {
			t.Errorf("%s: %d held + %d excluded = %d sequences, want the set's %d",
				scope, len(v.Sequences), v.ExcludedSequences, got, set.Sequences())
		}
		if got := v.Steps + v.ExcludedSteps; got != set.Steps() {
			t.Errorf("%s: %d held + %d excluded = %d steps, want the set's %d",
				scope, v.Steps, v.ExcludedSteps, got, set.Steps())
		}
		for i := range v.Sequences {
			if got := Scope(&v.Sequences[i]); got != scope {
				t.Errorf("the %s view holds session %s, which is %s",
					scope, v.Sequences[i].SessionID, got)
			}
		}
		held += len(v.Sequences)
		heldSteps += v.Steps
	}
	if held != set.Sequences() || heldSteps != set.Steps() {
		t.Errorf("the four scopes cover %d/%d sequences and %d/%d steps, want every one exactly once",
			held, set.Sequences(), heldSteps, set.Steps())
	}
}

// A scope name no constant defines must lose the data, not silently widen the population to
// every sequence in the window.
func TestAnUnknownScopeHoldsNothing(t *testing.T) {
	set := New([]store.Timeline{seq("s1", "", entrypointCLI, origin, 4)})

	v := set.Scope("intractive")
	if !v.Empty() || len(v.Sequences) != 0 {
		t.Fatalf("a misspelled scope held %d sequence(s) / %d step(s), want none",
			len(v.Sequences), v.Steps)
	}
	if v.ExcludedSequences != 1 || v.ExcludedSteps != 4 {
		t.Errorf("excluded %d sequence(s) / %d step(s), want the whole set", v.ExcludedSequences, v.ExcludedSteps)
	}
	if got := v.ExcludedStepShare(); got != 1 {
		t.Errorf("ExcludedStepShare() = %v, want 1", got)
	}
}

// Empty is a statement about steps, not about sequences: a window can hold sequences whose steps
// all fall outside it, and there is nothing there for a detector to read.
func TestEmptySetAndSteplessSequences(t *testing.T) {
	empty := New(nil)
	if !empty.Empty() || empty.Sequences() != 0 || empty.Steps() != 0 {
		t.Fatalf("New(nil): empty=%v sequences=%d steps=%d", empty.Empty(), empty.Sequences(), empty.Steps())
	}
	if !empty.Oldest.IsZero() || !empty.Newest.IsZero() {
		t.Errorf("New(nil) horizon = %v..%v, want zero", empty.Oldest, empty.Newest)
	}
	v := empty.Scope(Interactive)
	if !v.Empty() {
		t.Errorf("a scope of the empty set holds %d step(s)", v.Steps)
	}
	// Both shares divide by a whole that is zero here; the guard must answer 0 rather than NaN,
	// which would print as a percentage and compare false against every threshold.
	if got := v.ExcludedStepShare(); got != 0 {
		t.Errorf("ExcludedStepShare() = %v on an empty set, want 0", got)
	}
	if got := v.ExcludedSequenceShare(); got != 0 {
		t.Errorf("ExcludedSequenceShare() = %v on an empty set, want 0", got)
	}

	stepless := New([]store.Timeline{
		seq("s1", "", entrypointCLI, origin, 0),
		seq("s2", "", entrypointSDKPy, origin, 0),
	})
	if !stepless.Empty() {
		t.Errorf("a set of step-free sequences reports %d step(s), want none", stepless.Steps())
	}
	if stepless.Sequences() != 2 {
		t.Errorf("Sequences() = %d, want the 2 sequences the set holds", stepless.Sequences())
	}
}

// The horizon is read off the steps' own stamps, so a step the source left unstamped must not
// drag the window back to year 1.
func TestHorizonIgnoresUnstampedSteps(t *testing.T) {
	stamped := seq("s1", "", entrypointCLI, origin, 3)
	stamped.Steps = append(stamped.Steps, store.TimelineStep{Ordinal: 4})

	set := New([]store.Timeline{stamped})
	if !set.Oldest.Equal(origin) {
		t.Errorf("Oldest = %v, want %v", set.Oldest, origin)
	}
	if want := origin.Add(2 * time.Minute); !set.Newest.Equal(want) {
		t.Errorf("Newest = %v, want %v", set.Newest, want)
	}
	if set.Steps() != 4 {
		t.Errorf("Steps() = %d, want 4 -- an unstamped step is still a step", set.Steps())
	}
}

// View.Oldest answers for the scope, not for the set. A panel narrowed to one scope reaches back
// exactly as far as that scope's own steps; borrowing the set's would claim coverage its
// sequences never had.
func TestViewOldestIsTheScopesOwn(t *testing.T) {
	old := origin.AddDate(0, 0, -30)
	set := New([]store.Timeline{
		seq("script", "", entrypointSDKPy, old, 5),
		seq("person", "", entrypointCLI, origin, 5),
	})

	v := set.Scope(Interactive)
	if !v.Oldest.Equal(origin) {
		t.Errorf("the interactive view reaches back to %v, want %v -- the SDK run is not its history",
			v.Oldest, origin)
	}
	if !set.Oldest.Equal(old) {
		t.Errorf("the set's own horizon = %v, want %v", set.Oldest, old)
	}
}

// The two exclusion shares answer by an order of magnitude differently -- on the audited store,
// 89% of sequences against 5.7% of steps -- which is why the View keeps both and no caveat may
// substitute one for the other.
func TestTheTwoExcludedSharesAreNotInterchangeable(t *testing.T) {
	sequences := []store.Timeline{seq("person", "", entrypointCLI, origin, 200)}
	for i := range 20 {
		sequences = append(sequences, seq("script"+strconv.Itoa(i), "", entrypointSDKPy, origin, 1))
	}

	set := New(sequences)
	v := set.Scope(Interactive)
	if got, want := v.ExcludedSequenceShare(), 20.0/21.0; got != want {
		t.Errorf("ExcludedSequenceShare() = %v, want %v", got, want)
	}
	if got, want := v.ExcludedStepShare(), 20.0/220.0; got != want {
		t.Errorf("ExcludedStepShare() = %v, want %v", got, want)
	}
}
