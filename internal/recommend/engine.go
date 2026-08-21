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
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Family != out[j].Family {
			return out[i].Family < out[j].Family
		}
		return out[i].ID < out[j].ID
	})
	return out
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
