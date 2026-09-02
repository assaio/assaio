package recommend

import (
	"sort"

	"github.com/assaio/assaio/internal/analyze"
)

// Evidence is what a family may reason from: the window assaio read, and the verdicts it
// already computed over it. Families read the verdicts rather than recomputing, which is what
// makes "no rendered prose can claim more than its evidence" true by construction -- a
// recommendation quotes a figure a validator published, on the same window, with the same
// confidence attached.
type Evidence struct {
	Input   *analyze.Input
	Results []analyze.Result
}

// Result finds the verdict named, or nil.
func (e *Evidence) Result(name string) *analyze.Result {
	for i := range e.Results {
		if e.Results[i].Name == name {
			return &e.Results[i]
		}
	}
	return nil
}

// Family is one rule that proposes experiments. One family, one file, the same law the metric
// validators follow -- and the same reason: precision is measured per family, and a family
// that keeps being rejected has to be disableable as a unit.
type Family interface {
	Name() string
	Propose(*Evidence) []Record
}

var registry []Family

// Register adds f to the set From runs. Called from each family file's init().
func Register(f Family) { registry = append(registry, f) }

// Families returns every registered family, sorted by name.
func Families() []Family {
	out := make([]Family, len(registry))
	copy(out, registry)
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// From runs every family over the window and returns the records that survive validation,
// sorted by family then id so two runs of the same window print the same list in the same
// order. A record a family got wrong is dropped rather than rendered: advice missing the parts
// that make it checkable is the thing this package exists to prevent, and shipping it with a
// warning would be shipping it.
func From(e *Evidence) []Record {
	var out []Record
	for _, f := range Families() {
		out = append(out, propose(f, e)...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Family != out[j].Family {
			return familyRank(out[i].Family) < familyRank(out[j].Family)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// FromFamily runs one family over the window, for a command that renders its own advice
// beside its own arithmetic. Same validation as From: a record missing what would make it
// checkable is dropped rather than printed with a warning.
func FromFamily(name string, e *Evidence) []Record {
	for _, f := range Families() {
		if f.Name() == name {
			return propose(f, e)
		}
	}
	return nil
}

// propose runs one family and keeps the records that survive validation, stamping the two
// fields a family is not asked to repeat on every record it builds.
func propose(f Family, e *Evidence) []Record {
	var out []Record
	proposed := f.Propose(e)
	for i := range proposed {
		r := &proposed[i]
		r.Family = f.Name()
		if r.Status == "" {
			r.Status = StatusProposed
		}
		if err := r.Validate(); err != nil {
			continue
		}
		out = append(out, *r)
	}
	return out
}

// familyOrder is the order a reader meets these in, and it is not alphabetical. Advice about
// assaio's own evidence comes before advice about how somebody works: an engine that recommends
// changing a workflow while its own cost basis is missing a fifth of the window has it backwards
// (ADR 0015). A reader acts on what is at the top, so the ordering is the claim. Between the two
// routing families, the one resting on arithmetic over the whole window comes before the one
// resting on a sampled share, which is the only one of the two that can be a sampling artifact.
var familyOrder = []string{"pricing-coverage", FamilyCheaperRoute, "premium-small-turns"}

// familyRank places a family, putting an unlisted one last rather than first -- a new family
// has to earn its position deliberately.
func familyRank(name string) int {
	for i, f := range familyOrder {
		if f == name {
			return i
		}
	}
	return len(familyOrder)
}

// enough reports whether a verdict is strong enough to build advice on. Abstention is the
// default: a recommendation magnifies whatever error is under it, and one derived from a
// window assaio itself calls low-confidence would be a precise action resting on evidence its
// own envelope says is thin. "Insufficient" and "low" both abstain -- the second is the one
// that matters, because it looks like an answer.
func enough(r *analyze.Result) bool {
	if r == nil {
		return false
	}
	switch r.Confidence.Label {
	case analyze.ConfidenceHigh, analyze.ConfidenceMedium:
		return len(r.Withheld) == 0
	default:
		return false
	}
}
