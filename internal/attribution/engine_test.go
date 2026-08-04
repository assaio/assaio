package attribution

import (
	"slices"
	"sort"
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

// honestEngine exists to prove the corpus is satisfiable, not to attribute anything: a
// corpus no answer can pass tells an implementer nothing. It reads its candidates from the
// fixture, but takes the ambiguity flag from the scenario rather than deciding it -- deciding
// it is the engine's job (B85), and a stand-in that guessed would be asserting its own
// heuristic here instead of the property the scenario states.
func honestEngine(s *Scenario, f *Fixture) Links {
	out := Links{}
	for i := range s.Sessions {
		id := s.Sessions[i].ID
		if hash, corrected := f.Confirmed[id]; corrected {
			out[id] = Link{Commits: []string{hash}}
			continue
		}
		reachable := commitsAfter(f, epoch.Add(s.Sessions[i].Start))
		if len(reachable) == 0 {
			continue
		}
		out[id] = Link{Commits: reachable, Ambiguous: s.Expect[id].Ambiguous}
	}
	return out
}

// commitsAfter is every observed commit at or after start, oldest first -- the whole set an
// engine may consider before it starts ruling candidates out.
func commitsAfter(f *Fixture, start time.Time) []string {
	var out []event.Event
	for i := range f.Commits {
		if !f.Commits[i].OccurredAt.Before(start) {
			out = append(out, f.Commits[i])
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OccurredAt.Before(out[j].OccurredAt) })
	hashes := make([]string, 0, len(out))
	for i := range out {
		hashes = append(hashes, out[i].ID)
	}
	return hashes
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
