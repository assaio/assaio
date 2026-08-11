package docs

import "fmt"

// A published page cannot be checked against a binary in general -- prose is prose. What it can
// do is declare which of its claims are checkable, and be held to those. A file marks a claim
// with data-claim="<id>" and a set it enumerates with data-covers="<set>"; everything else on
// the page is deliberately outside this check, and saying so is the point. A guard that implied
// it read the prose would be the false green this project exists to refuse.

// Problem is one disagreement between a published file and the binary, phrased as the edit that
// would fix it.
type Problem struct {
	File string
	Text string
}

func (p Problem) String() string { return p.File + ": " + p.Text }

// CheckClaims holds one annotated file to the registries. It reports a claim with no member
// behind it, a count that is not the count, and -- the direction that actually catches a shipped
// capability nobody published -- a member of a covered set that the file never names.
func CheckClaims(ref *Reference, file, content string) []Problem {
	ix := newIndex(ref)
	content = stripComments(content)
	var problems []Problem

	// A page states the same claim in more than one sentence; check reads every occurrence, so
	// the second one here would only repeat the same problem.
	seen := map[string]bool{}
	for _, m := range claimAttr.FindAllStringSubmatch(content, -1) {
		id := m[1]
		if seen[id] {
			continue
		}
		seen[id] = true
		if text, ok := ix.check(id, content); !ok {
			problems = append(problems, Problem{File: file, Text: text})
		}
	}

	for _, loc := range coversAttr.FindAllStringSubmatchIndex(content, -1) {
		set := content[loc[2]:loc[3]]
		members, ok := ix.sets[set]
		if !ok {
			problems = append(problems, Problem{
				File: file,
				Text: fmt.Sprintf("declares data-covers=%q, which is not a set. %s", set, ix.setNames()),
			})
			continue
		}
		// Only claims inside the element making the promise count. A claim elsewhere on the
		// page does not make this list enumerate anything.
		inside := map[string]bool{}
		for _, m := range claimAttr.FindAllStringSubmatch(enclosure(content, loc[0]), -1) {
			inside[m[1]] = true
		}
		for _, member := range members {
			if inside[member.id] || exempt[member.id] != "" {
				continue
			}
			problems = append(problems, Problem{File: file, Text: fmt.Sprintf(
				"covers %q but never claims %q inside that element -- add it, or exempt it in internal/docs with the reason",
				set, member.id,
			)})
		}
	}

	return problems
}

// exempt lists members a published surface need not carry, each with the reason its absence
// cannot go stale. It is in Go rather than in an attribute on purpose: an exemption written on
// the page would let the page silence the guard by editing itself.
var exempt = map[string]string{
	"command.version": "cobra prints the build's own version; there is nothing about it to describe",
}
