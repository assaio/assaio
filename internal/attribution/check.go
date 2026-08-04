package attribution

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

// Check judges one engine's answer against a scenario and returns every way it broke what
// the scenario defends. An empty result is a pass. There is no error return: a violation is
// a finding to report, not a failure of the check.
func Check(s *Scenario, f *Fixture, got Links) []string {
	var out []string
	for _, id := range sortedKeys(got) {
		if _, known := s.Expect[id]; !known {
			out = append(out, s.violation("linked session %q, which this scenario does not contain", id))
		}
	}
	for _, id := range sortedKeys(s.Expect) {
		out = append(out, s.checkSession(f, id, s.Expect[id], got[id])...)
	}
	return out
}

// checkSession judges one session's link. The three cases are deliberately separate: a
// session no evidence reaches, one a human confirmed, and one the evidence speaks for.
func (s *Scenario) checkSession(f *Fixture, id string, want Expectation, got Link) []string {
	var out []string
	for _, hash := range got.Commits {
		if !f.knows(hash) {
			out = append(out, s.violation("session %q was linked to %q, which this fixture never produced", id, hash))
		}
	}

	switch {
	case want.Confirmed != "":
		if !sameSet(got.Commits, []string{f.Hash(want.Confirmed)}) {
			out = append(out, s.violation(
				"session %q must link to the commit a human confirmed (%s), got %v", id, want.Confirmed, got.Commits,
			))
		}
	case len(want.Candidates) == 0:
		if len(got.Commits) > 0 {
			out = append(out, s.violation("session %q must stay unattributed, got %d link(s)", id, len(got.Commits)))
		}
	case want.Ambiguous:
		out = append(out, s.checkAmbiguous(f, id, f.hashesFor(want.Candidates), got)...)
	default:
		if !sameSet(got.Commits, f.hashesFor(want.Candidates)) {
			out = append(out, s.violation(
				"session %q must link to exactly %v, got %v", id, want.Candidates, got.Commits,
			))
		}
	}
	return out
}

// checkAmbiguous is the assertion the corpus exists for. Two things are required and they
// are not the same one: the alternatives have to survive into the answer, and the answer has
// to be marked ambiguous. Naming one commit under an ambiguity flag still shows a reader one
// commit, and offering every candidate unmarked still claims a certainty nothing supports.
func (s *Scenario) checkAmbiguous(f *Fixture, id string, wanted []string, got Link) []string {
	var out []string
	if len(got.Commits) < 2 {
		out = append(out, s.violation(
			"session %q is ambiguous between %d commits and must keep them all, got %v",
			id, len(wanted), got.Commits,
		))
	}
	if !got.Ambiguous {
		out = append(out, s.violation("session %q must be reported as ambiguous", id))
	}
	for _, hash := range got.Commits {
		if f.knows(hash) && !slices.Contains(wanted, hash) {
			out = append(out, s.violation("session %q was linked to a commit outside its candidates", id))
		}
	}
	return out
}

// violation states what broke and what the scenario was protecting, so a failure reads as a
// property lost rather than only as two values differing.
func (s *Scenario) violation(format string, args ...any) string {
	return s.Name + ": " + fmt.Sprintf(format, args...) + " — " + s.Defends
}

// knows reports whether hash is a commit this fixture built.
func (f *Fixture) knows(hash string) bool {
	for _, h := range f.Hashes {
		if h == hash {
			return true
		}
	}
	return false
}

// hashesFor resolves scenario tags to the hashes git produced for them.
func (f *Fixture) hashesFor(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		out = append(out, f.Hash(tag))
	}
	return out
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x, y := slices.Clone(a), slices.Clone(b)
	sort.Strings(x)
	sort.Strings(y)
	return slices.Equal(x, y)
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Report renders violations as one block, for a caller failing a test with them.
func Report(violations []string) string { return strings.Join(violations, "\n") }
