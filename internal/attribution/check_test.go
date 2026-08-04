package attribution

import (
	"strings"
	"testing"
)

// fixtureOf is a fixture with only the tag-to-hash table Check needs, so these tests judge
// the checker rather than the repository builder.
func fixtureOf(tags ...string) Fixture {
	f := Fixture{Hashes: make(map[string]string, len(tags))}
	for _, tag := range tags {
		f.Hashes[tag] = "hash-" + tag
	}
	return f
}

func ambiguousScenario() Scenario {
	return Scenario{
		Name:    "two-indistinguishable-sessions",
		Defends: "nothing in the fixture separates the candidates, so neither may be chosen",
		Expect: map[string]Expectation{
			"s1": {Candidates: []string{"c1", "c2"}, Ambiguous: true},
		},
	}
}

// The corpus exists to catch one failure above all others: an engine answering confidently
// where the evidence cannot tell two candidates apart. Picking one is not a better answer
// than reporting both, it is a fabricated one.
func TestCheckRejectsAForcedChoiceWhereTheFixtureIsAmbiguous(t *testing.T) {
	s := ambiguousScenario()
	f := fixtureOf("c1", "c2")

	got := Links{"s1": {Commits: []string{f.Hashes["c1"]}, Ambiguous: false}}

	violations := Check(&s, &f, got)
	if len(violations) == 0 {
		t.Fatal("Check accepted one confident link where the fixture cannot distinguish two")
	}
	if !strings.Contains(strings.Join(violations, "; "), s.Defends) {
		t.Errorf("violations = %q, want them to state what the scenario defends", violations)
	}
}

// Keeping both candidates and saying so is the answer the corpus is written to accept.
func TestCheckAcceptsAmbiguityKept(t *testing.T) {
	s := ambiguousScenario()
	f := fixtureOf("c1", "c2")

	got := Links{"s1": {Commits: []string{f.Hashes["c1"], f.Hashes["c2"]}, Ambiguous: true}}

	if violations := Check(&s, &f, got); len(violations) != 0 {
		t.Fatalf("Check = %q, want the ambiguity-preserving answer accepted", violations)
	}
}

// Reporting a link as ambiguous while naming a single commit is the same forced choice with
// a disclaimer attached: a reader still sees one commit.
func TestCheckRejectsAmbiguityClaimedOverOneCommit(t *testing.T) {
	s := ambiguousScenario()
	f := fixtureOf("c1", "c2")

	got := Links{"s1": {Commits: []string{f.Hashes["c1"]}, Ambiguous: true}}

	if violations := Check(&s, &f, got); len(violations) == 0 {
		t.Fatal("Check accepted a single commit flagged ambiguous; the flag is not the point, the alternatives are")
	}
}

// A session no evidence reaches has to come back unattributed. Inventing a link for it is
// how an unattributed share silently becomes zero.
func TestCheckRejectsALinkWhereNoneWasEarned(t *testing.T) {
	s := Scenario{
		Name:    "wrong-branch",
		Defends: "a commit on a branch the session never touched is not a candidate",
		Expect:  map[string]Expectation{"s1": {}},
	}
	f := fixtureOf("c1")

	got := Links{"s1": {Commits: []string{f.Hashes["c1"]}}}

	if violations := Check(&s, &f, got); len(violations) == 0 {
		t.Fatal("Check accepted a link for a session the corpus says must stay unattributed")
	}
}

// A human's confirmed link is not one candidate among others: it wins, and an engine that
// reports anything else has overruled a person with a heuristic.
func TestCheckRequiresAConfirmedLinkToWin(t *testing.T) {
	s := Scenario{
		Name:    "manual-correction",
		Defends: "a human's confirmation outranks the evidence and survives an algorithm change",
		Expect: map[string]Expectation{
			"s1": {Candidates: []string{"c2"}, Confirmed: "c2"},
		},
	}
	f := fixtureOf("c1", "c2")

	got := Links{"s1": {Commits: []string{f.Hashes["c1"]}}}

	if violations := Check(&s, &f, got); len(violations) == 0 {
		t.Fatal("Check accepted an answer that ignored the human's confirmed link")
	}
}

// An engine may not answer for a session the scenario never described: a link keyed to
// something the fixture does not contain is not an answer, it is noise.
func TestCheckRejectsAnUnknownSession(t *testing.T) {
	s := ambiguousScenario()
	f := fixtureOf("c1", "c2")

	got := Links{
		"s1":      {Commits: []string{f.Hashes["c1"], f.Hashes["c2"]}, Ambiguous: true},
		"ghost-1": {Commits: []string{f.Hashes["c1"]}},
	}

	if violations := Check(&s, &f, got); len(violations) == 0 {
		t.Fatal("Check accepted a link for a session the scenario does not contain")
	}
}

// A commit hash the fixture never produced cannot be evidence for anything.
func TestCheckRejectsAnUnknownCommit(t *testing.T) {
	s := ambiguousScenario()
	f := fixtureOf("c1", "c2")

	got := Links{"s1": {Commits: []string{"deadbeef"}, Ambiguous: true}}

	if violations := Check(&s, &f, got); len(violations) == 0 {
		t.Fatal("Check accepted a commit hash that is not in the fixture")
	}
}
