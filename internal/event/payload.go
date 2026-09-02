package event

import (
	"fmt"
	"sort"
)

// The observation types this build can produce. ADR 0007 committed a vocabulary spanning the AI
// and the non-AI domains alike; ADR 0016 withdrew the AI half, because AI usage already has a
// canonical model in usage_record and a second one is a liability rather than a symmetry. Still
// to land here are scm.pull_request, scm.review, ci.check and delivery.merge|revert|survival --
// each with the collector that fills it: one constant, one payload struct. Declaring a name is a
// commitment; declaring a struct nothing produces is speculative abstraction.
const (
	TypeCommit = "vcs.commit.observed"
)

var types = []string{TypeCommit}

// known reports whether t is an observation this build produces.
func known(t string) bool { return valid(types, t) }

// nonNegative rejects the first negative count, named, in a stable order so the same bad
// payload always reports the same field.
func nonNegative(fields map[string]int64) error {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if fields[name] < 0 {
			return fmt.Errorf("%s is negative (%d)", name, fields[name])
		}
	}
	return nil
}
