// Package trace turns the stored step sequences into the view a detector reads: the sequences of
// one declared scope, and the share of the window that scope leaves out. It exists so exactly one
// place computes that denominator. A detector filtering the set itself would report a rate over a
// population it never named -- which on the audited store means calling 2,279 one-shot SDK
// invocations sessions, and publishing "89% of sessions end without an edit" as a finding about
// how people work.
package trace

import (
	"time"

	"github.com/assaio/assaio/internal/store"
)

// Set is every sequence a window holds, with the horizon behind them. A detector never filters it
// itself: it asks for the scope it declares and gets a View back carrying what asking left out.
type Set struct {
	sequences []store.Timeline
	steps     int
	// Oldest and Newest are the horizon of the sequences held: how far back the history any
	// figure read off them actually goes, which a trend has to publish rather than imply (B156).
	Oldest, Newest time.Time
}

// New builds the set from what the store returned. Empty is the normal state on a store whose
// steps all predate the window, and on every source that has no step reading at all.
func New(sequences []store.Timeline) Set {
	s := Set{sequences: sequences}
	for i := range sequences {
		steps := sequences[i].Steps
		s.steps += len(steps)
		for j := range steps {
			at := steps[j].Timestamp
			if at.IsZero() {
				continue
			}
			if s.Oldest.IsZero() || at.Before(s.Oldest) {
				s.Oldest = at
			}
			if at.After(s.Newest) {
				s.Newest = at
			}
		}
	}
	return s
}

// Sequences is how many sequences the set holds, across every scope.
func (s *Set) Sequences() int { return len(s.sequences) }

// ForSessions narrows the set to the sequences of the sessions given, keeping the one invariant
// every surface depends on: the sequences describe exactly the sessions the figures beside them
// do. Without it a label-filtered run and a per-project drill would each print a detector's
// window-wide finding inside a panel headed by a subset -- two numbers on one page, describing
// two different populations, with nothing saying so.
//
// A caller holding every session gets the same set back, which is why this is safe to apply
// unconditionally rather than only on the filtered paths.
func (s *Set) ForSessions(sessions []store.SessionRow) Set {
	keep := make(map[sessionKey]struct{}, len(sessions))
	for i := range sessions {
		keep[sessionKey{sessions[i].SessionID, sessions[i].Member}] = struct{}{}
	}
	kept := make([]store.Timeline, 0, len(s.sequences))
	for i := range s.sequences {
		t := &s.sequences[i]
		if _, ok := keep[sessionKey{t.SessionID, t.Member}]; ok {
			kept = append(kept, *t)
		}
	}
	if len(kept) == len(s.sequences) {
		return *s
	}
	// The horizon travels with the set rather than being recomputed from the subset: it says how
	// far back the *store's* window reaches, and how recently anything at all happened. Recomputing
	// it made the newest step of whichever project a drill selected into "now", so the same session
	// counted as still running on one panel and as finished on another.
	narrowed := New(kept)
	narrowed.Oldest, narrowed.Newest = s.Oldest, s.Newest
	return narrowed
}

// sessionKey is how every read correlates a session: the id is unique per member, never alone.
type sessionKey struct{ session, member string }

// All is every sequence, for the one kind of caller that must carry the whole set rather than
// read a scope of it: a boundary that hands the sequences to something outside this process, and
// therefore cannot know which scope the far side will declare. A metric reading them here asks
// for its scope instead.
func (s *Set) All() []store.Timeline { return s.sequences }

// Steps is how many steps the set holds, across every scope.
func (s *Set) Steps() int { return s.steps }

// Empty reports whether the set holds nothing to read -- a store with no step history, or a
// window before it starts.
func (s *Set) Empty() bool { return s.steps == 0 }

// View is the sequences of one scope, and what asking for that scope left behind.
type View struct {
	Scope     string
	Sequences []store.Timeline
	Steps     int
	// Oldest is the earliest step *this scope* holds, which is not the set's: a figure read over
	// one scope of one project reaches back exactly as far as that scope's own sequences do, and
	// borrowing the set's made a narrowed panel claim coverage its sequences never had.
	Oldest time.Time
	// ExcludedSequences and ExcludedSteps are what the set holds outside this scope. Both are
	// kept, not one: they answer by an order of magnitude differently -- 89% of sequences against
	// 5.7% of steps on the audited store -- and a figure quoting whichever flatters it is the
	// trap this type exists to close.
	ExcludedSequences, ExcludedSteps int
}

// Scope returns the view of one scope. An unknown scope name yields an empty view rather than
// every sequence: a typo must lose the data loudly, not silently widen the population.
func (s *Set) Scope(scope string) View {
	v := View{Scope: scope}
	for i := range s.sequences {
		t := &s.sequences[i]
		if Scope(t) == scope {
			v.Sequences = append(v.Sequences, *t)
			v.Steps += len(t.Steps)
			for j := range t.Steps {
				at := t.Steps[j].Timestamp
				if !at.IsZero() && (v.Oldest.IsZero() || at.Before(v.Oldest)) {
					v.Oldest = at
				}
			}
			continue
		}
		v.ExcludedSequences++
		v.ExcludedSteps += len(t.Steps)
	}
	return v
}

// Empty reports whether this scope holds no steps in the window.
func (v *View) Empty() bool { return v.Steps == 0 }

// ExcludedStepShare is the share of the set's steps this scope does not cover, 0..1.
func (v *View) ExcludedStepShare() float64 {
	return share(v.ExcludedSteps, v.ExcludedSteps+v.Steps)
}

// ExcludedSequenceShare is the share of the set's sequences this scope does not cover, 0..1.
func (v *View) ExcludedSequenceShare() float64 {
	return share(v.ExcludedSequences, v.ExcludedSequences+len(v.Sequences))
}

func share(part, whole int) float64 {
	if whole <= 0 {
		return 0
	}
	return float64(part) / float64(whole)
}
