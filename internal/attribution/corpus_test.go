package attribution

import (
	"os/exec"
	"slices"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/event"
	"github.com/assaio/assaio/internal/usage"
)

// The corpus is the list B93 asks for. Naming it here means a scenario cannot be quietly
// dropped when it becomes inconvenient to satisfy.
func TestCorpusCoversEveryShapeAttributionHasToSurvive(t *testing.T) {
	want := []string{
		"one-session-one-commit",
		"many-sessions-one-commit",
		"one-session-many-commits",
		"wrong-branch",
		"overlapping-users",
		"delayed-commit",
		"sub-agent",
		"genuinely-ambiguous",
		"manual-correction",
		"replay-after-algorithm-change",
		"squash-landed-after-the-session",
		"squash-landed-in-another-session",
		"rebase-merge",
		"forge-merge-commit",
		"amended-in-a-later-session",
		"local-merge-carries-work",
		"confirmed-forge-merge-survives",
	}
	var got []string
	for _, s := range Corpus() {
		got = append(got, s.Name)
	}
	for _, name := range want {
		if !slices.Contains(got, name) {
			t.Errorf("corpus is missing the %q scenario; it has %v", name, got)
		}
	}
}

func buildOrSkip(t *testing.T, s *Scenario) Fixture {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	f, err := s.Build(t.TempDir())
	if err != nil {
		t.Fatalf("building %s: %v", s.Name, err)
	}
	return f
}

// Every scenario has to build a real repository and real usage records, and every tag its
// expectations name has to exist in what was built. A corpus whose expectations point at
// nothing would pass any engine at all.
func TestEveryScenarioBuildsWhatItsExpectationsName(t *testing.T) {
	for _, s := range Corpus() {
		t.Run(s.Name, func(t *testing.T) {
			f := buildOrSkip(t, &s)
			if len(f.Commits) == 0 {
				t.Fatal("no commit observations were collected")
			}
			if s.Defends == "" {
				t.Error("a scenario must say what it defends; the message is half the assertion")
			}
			for id, want := range s.Expect {
				if !slices.ContainsFunc(f.Sessions, func(r usage.Record) bool { return r.SessionID == id }) {
					t.Errorf("expectation names session %q, which the fixture does not contain", id)
				}
				for _, tag := range append(slices.Clone(want.Candidates), want.Confirmed) {
					if tag != "" && f.Hash(tag) == "" {
						t.Errorf("expectation names commit %q, which the fixture does not contain", tag)
					}
				}
			}
		})
	}
}

// A scenario declaring a parent session has to carry that relationship in the records an
// engine actually receives. Without the marker the store itself uses, "sub-agent" is
// indistinguishable from two unrelated sessions reaching one commit -- the scenario would
// look like many-sessions-one-commit and defend nothing of its own.
func TestTheSubAgentScenarioCarriesTheMarkerAnEngineWillRead(t *testing.T) {
	s, ok := Get("sub-agent")
	if !ok {
		t.Fatal("the sub-agent scenario is missing")
	}
	f := buildOrSkip(t, &s)

	var child, parent []usage.Record
	for _, r := range f.Sessions {
		switch r.SessionID {
		case "s1-sub":
			child = append(child, r)
		case "s1":
			parent = append(parent, r)
		}
	}
	if len(child) == 0 || len(parent) == 0 {
		t.Fatalf("fixture has %d child and %d parent records, want both", len(child), len(parent))
	}
	for _, r := range child {
		if r.Sidechain != 1 {
			t.Errorf("sub-agent record %q has Sidechain = %d, want the marker the store reads", r.DedupeKey, r.Sidechain)
		}
	}
	for _, r := range parent {
		if r.Sidechain != 0 {
			t.Errorf("parent record %q is marked as a sub-agent turn", r.DedupeKey)
		}
	}
}

// The point of the ambiguous scenario is that it is genuinely ambiguous: if one candidate
// were closer in time, or on a different branch, or touched different file categories, an
// engine could separate them honestly and the fixture would be teaching the wrong lesson.
func TestTheAmbiguousScenarioIsActuallyAmbiguous(t *testing.T) {
	s, ok := Get("genuinely-ambiguous")
	if !ok {
		t.Fatal("the genuinely-ambiguous scenario is missing")
	}
	f := buildOrSkip(t, &s)

	want := s.Expect["s1"]
	if len(want.Candidates) < 2 {
		t.Fatalf("candidates = %v, want at least two for ambiguity to mean anything", want.Candidates)
	}
	if !want.Ambiguous {
		t.Fatal("the scenario must require ambiguity to be preserved")
	}
	first := traitsOf(t, &f, want.Candidates[0], "s1")
	for _, tag := range want.Candidates[1:] {
		if other := traitsOf(t, &f, tag, "s1"); other != first {
			t.Errorf("%q and %q differ: %+v vs %+v — an engine could separate them, so this fixture is not ambiguous",
				want.Candidates[0], tag, first, other)
		}
	}
}

// traits are every signal an attribution engine could separate two candidates by, read off
// the fixture itself: the project the observation belongs to, the categories it touched, whether
// it is a merge, whether the forge committed it, and whether either of its times lands inside the
// session's window. Two candidates that match on all of them are indistinguishable by anything
// the fixture carries -- which is what "ambiguous" has to mean for the assertion to be about
// honesty rather than about a rule assaio happens to apply.
type traits struct {
	project              string
	categories           event.FileCategories
	merge                bool
	forge                bool
	withinWindow         bool
	authoredWithinWindow bool
}

func traitsOf(t *testing.T, f *Fixture, tag, sessionID string) traits {
	t.Helper()
	hash := f.Hash(tag)
	for i := range f.Commits {
		if f.Commits[i].ID != hash {
			continue
		}
		c, ok := f.Commits[i].Payload.(event.Commit)
		if !ok {
			t.Fatalf("observation for %q is not a commit", tag)
		}
		start, end := sessionWindow(t, f, sessionID)
		within := func(at time.Time) bool { return !at.Before(start) && !at.After(end) }
		return traits{
			project:              f.Commits[i].Subject.Project,
			categories:           c.Files,
			merge:                c.Parents > 1,
			forge:                c.CommittedByForge,
			withinWindow:         within(f.Commits[i].OccurredAt),
			authoredWithinWindow: within(c.AuthoredAt),
		}
	}
	t.Fatalf("no observation for %q", tag)
	return traits{}
}

// sessionWindow is the first and last timestamp the session's records carry.
func sessionWindow(t *testing.T, f *Fixture, sessionID string) (start, end time.Time) {
	t.Helper()
	for i := range f.Sessions {
		if f.Sessions[i].SessionID != sessionID {
			continue
		}
		ts := f.Sessions[i].Timestamp
		if start.IsZero() || ts.Before(start) {
			start = ts
		}
		if ts.After(end) {
			end = ts
		}
	}
	if start.IsZero() {
		t.Fatalf("no records for session %q", sessionID)
	}
	return start, end
}

// TestTheLandingFixturesAreWhatTheyClaim: each landing scenario is only a test of the forge rules
// if the collector reads its commits the way the forge would have left them -- the flag, the
// parents, and an author time apart from the commit time where the forge or an amend moved it.
func TestTheLandingFixturesAreWhatTheyClaim(t *testing.T) {
	type shape struct {
		forge         bool
		parents       int64
		writtenBefore bool
	}
	for _, tt := range []struct {
		scenario, tag string
		want          shape
	}{
		{"squash-landed-after-the-session", "sq", shape{forge: true, parents: 1}},
		{"squash-landed-in-another-session", "sq", shape{forge: true, parents: 1}},
		{"rebase-merge", "rb", shape{forge: true, parents: 1, writtenBefore: true}},
		{"forge-merge-commit", "mg", shape{forge: true, parents: 2}},
		{"amended-in-a-later-session", "c1", shape{parents: 1, writtenBefore: true}},
		{"local-merge-carries-work", "mg", shape{parents: 2}},
		{"confirmed-forge-merge-survives", "mg", shape{forge: true, parents: 2}},
	} {
		t.Run(tt.scenario, func(t *testing.T) {
			s, ok := Get(tt.scenario)
			if !ok {
				t.Fatalf("the %s scenario is missing", tt.scenario)
			}
			f := buildOrSkip(t, &s)
			e := observed(t, &f, tt.tag)
			c, _ := e.Payload.(event.Commit)
			got := shape{forge: c.CommittedByForge, parents: c.Parents, writtenBefore: c.AuthoredAt.Before(e.OccurredAt)}
			if got != tt.want {
				t.Fatalf("%s reads as %+v, want %+v", tt.tag, got, tt.want)
			}
			if !tt.want.writtenBefore && !c.AuthoredAt.Equal(e.OccurredAt) {
				t.Fatalf("%s authored %s, committed %s: want one moment", tt.tag, c.AuthoredAt, e.OccurredAt)
			}
		})
	}
}

func observed(t *testing.T, f *Fixture, tag string) *event.Event {
	t.Helper()
	hash := f.Hash(tag)
	for i := range f.Commits {
		if f.Commits[i].ID == hash {
			return &f.Commits[i]
		}
	}
	t.Fatalf("no observation for %q", tag)
	return nil
}
