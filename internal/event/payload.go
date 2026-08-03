package event

import (
	"fmt"
	"sort"
)

// The observation types this build can produce. ADR 0007 commits the whole vocabulary --
// scm.pull_request, scm.review, ci.check, delivery.merge|revert|survival and ai.session --
// and each lands here with the collector that fills it: one constant, one payload struct.
// Declaring a name is a commitment; declaring a struct nothing produces is speculative
// abstraction.
const (
	TypeUsage  = "ai.usage.observed"
	TypeEdit   = "ai.edit.observed"
	TypeCommit = "vcs.commit.observed"
)

var types = []string{TypeUsage, TypeEdit, TypeCommit}

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
