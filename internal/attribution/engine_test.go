package attribution

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/event"
)

// nearestCommit is the engine everyone writes first: for each session, link the commit
// closest in time after it ends. It is plausible, it is what a proximity heuristic alone
// produces, and the corpus exists to reject it -- so running it here is how the corpus
// proves it has teeth rather than merely having scenarios.
func nearestCommit(f *Fixture, sessions []sessionSpec) Links {
	out := Links{}
	for i := range sessions {
		end := epoch.Add(sessions[i].End)
		best, bestGap := "", time.Duration(0)
		for j := range f.Commits {
			at := f.Commits[j].OccurredAt
			if at.Before(end) {
				continue
			}
			if gap := at.Sub(end); best == "" || gap < bestGap {
				best, bestGap = f.Commits[j].ID, gap
			}
		}
		if best != "" {
			out[sessions[i].ID] = Link{Commits: []string{best}}
		}
	}
	return out
}

// A single confident answer is exactly what an ambiguous fixture must not accept. If this
// ever passes, the corpus has stopped defending the property it was written for.
func TestAForcingEngineFailsTheAmbiguousScenarios(t *testing.T) {
	for _, name := range []string{"genuinely-ambiguous", "overlapping-users"} {
		t.Run(name, func(t *testing.T) {
			s, ok := Get(name)
			if !ok {
				t.Fatalf("scenario %q is missing", name)
			}
			f := buildOrSkip(t, &s)

			violations := Check(&s, &f, nearestCommit(&f, s.Sessions))
			if len(violations) == 0 {
				t.Fatal("the corpus accepted an engine that picks one candidate and reports no ambiguity")
			}
			if !strings.Contains(Report(violations), "ambiguous") {
				t.Errorf("violations = %s, want them to name the ambiguity that was lost", Report(violations))
			}
		})
	}
}

// The other half of the same proof: an engine that keeps the alternatives and says so has to
// pass, or the corpus is unsatisfiable and tells an implementer nothing.
func TestAnEngineThatKeepsAmbiguityPassesEveryScenario(t *testing.T) {
	for _, s := range Corpus() {
		t.Run(s.Name, func(t *testing.T) {
			f := buildOrSkip(t, &s)

			if violations := Check(&s, &f, honestEngine(&s, &f)); len(violations) != 0 {
				t.Fatalf("an engine answering exactly what the scenario describes was rejected:\n%s",
					Report(violations))
			}
		})
	}
}

// honestEngine answers what each scenario states, confirmations first. It shows only that the
// expectations are consistent with the fixtures Check reads them against; that a real engine can
// satisfy them is TestMatchPassesEveryConformanceScenario's job, and the teeth are the engines
// below, which must fail.
func honestEngine(s *Scenario, f *Fixture) Links {
	out := Links{}
	for i := range s.Sessions {
		id := s.Sessions[i].ID
		if hash, corrected := f.Confirmed[id]; corrected {
			out[id] = Link{Commits: []string{hash}}
			continue
		}
		if want := s.Expect[id]; len(want.Candidates) > 0 {
			out[id] = Link{Commits: f.hashesFor(want.Candidates), Ambiguous: want.Ambiguous, Change: s.changeNumber(want.Change)}
		}
	}
	return out
}

// oneClockEngine is proximity matching that trusts a single time per commit: a commit whose
// clock falls inside a session links to it, and failing that, the commits in the following
// window do -- ambiguous when there is more than one. With the committer clock it judges a commit by
// when it reached its branch; with the author clock, by when it was first written.
func oneClockEngine(f *Fixture, sessions []sessionSpec, clock func(*event.Event) time.Time) Links {
	out := Links{}
	for i := range sessions {
		start, end := epoch.Add(sessions[i].Start), epoch.Add(sessions[i].End)
		var inside, following []string
		for j := range f.Commits {
			t := clock(&f.Commits[j])
			switch {
			case t.Before(start) || t.After(end.Add(DefaultMaxGap)):
			case !t.After(end):
				inside = append(inside, f.Commits[j].ID)
			default:
				following = append(following, f.Commits[j].ID)
			}
		}
		switch {
		case len(inside) > 0:
			out[sessions[i].ID] = Link{Commits: inside}
		case len(following) > 0:
			out[sessions[i].ID] = Link{Commits: following, Ambiguous: len(following) > 1}
		}
	}
	return out
}

func committerClock(e *event.Event) time.Time { return e.OccurredAt }

func authorClock(e *event.Event) time.Time {
	if c, ok := e.Payload.(event.Commit); ok && !c.AuthoredAt.IsZero() {
		return c.AuthoredAt
	}
	return e.OccurredAt
}

// Either clock alone is wrong somewhere: the committer time is the merge on a forge rebase and
// a forge merge, the author time forgets the session that amended a commit. If a single-clock
// engine ever passes these scenarios, they have stopped defending anything.
func TestASingleClockFailsTheLandingScenarios(t *testing.T) {
	for _, tt := range []struct {
		scenario string
		clock    func(*event.Event) time.Time
	}{
		{"rebase-merge", committerClock},
		{"forge-merge-commit", committerClock},
		{"amended-in-a-later-session", committerClock},
		{"amended-in-a-later-session", authorClock},
		{"forge-merge-commit", authorClock},
	} {
		t.Run(tt.scenario, func(t *testing.T) {
			s, ok := Get(tt.scenario)
			if !ok {
				t.Fatalf("scenario %q is missing", tt.scenario)
			}
			f := buildOrSkip(t, &s)
			if len(Check(&s, &f, oneClockEngine(&f, s.Sessions, tt.clock))) == 0 {
				t.Fatal("a single-clock engine passed a scenario written to defeat it")
			}
		})
	}
}

// A confirmed link is not a candidate that scored well: it must win, and it must still win
// after the algorithm changes and after new commits arrive. The replay scenario deliberately
// puts a better-scoring commit in the way of the one the human confirmed.
func TestAConfirmedLinkSurvivesAnAlgorithmChange(t *testing.T) {
	s, ok := Get("replay-after-algorithm-change")
	if !ok {
		t.Fatal("the replay scenario is missing")
	}
	f := buildOrSkip(t, &s)

	if violations := Check(&s, &f, nearestCommit(&f, s.Sessions)); len(violations) == 0 {
		t.Fatal("an engine that recomputed over the human's correction was accepted")
	}

	// Whatever an engine's ranking does, replaying it must land on the confirmed commit.
	confirmed := f.Confirmed["s1"]
	if !slices.Contains(f.hashesFor([]string{"c1"}), confirmed) {
		t.Fatalf("the fixture's correction resolved to %q, not to c1", confirmed)
	}
	for _, engine := range []Links{honestEngine(&s, &f), {"s1": {Commits: []string{confirmed}}}} {
		if violations := Check(&s, &f, engine); len(violations) != 0 {
			t.Fatalf("a replay honouring the correction was rejected:\n%s", Report(violations))
		}
	}
}
