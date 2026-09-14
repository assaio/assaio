package analyze

import (
	"sort"

	"github.com/assaio/assaio/internal/layer"
)

// registry holds every self-registered Validator. Populated by each validator file's
// init(); never mutated after program start.
var registry []Validator

// Register adds v to the set of validators `assaio analyze` runs. This is the extension
// point for a new metric: add a file under internal/analyze implementing Validator and
// call Register(yourValidator{}) from that file's init(). No other wiring is required --
// the validator appears in `assaio analyze --list` and every analyze run automatically.
//
// A Name is registered once. It is the key the JSON document writes a Result under and the
// argument `assaio analyze <name>` selects on, so two validators sharing one both run while
// Get answers for only the first, and the document then carries the same key twice with no
// reader able to tell which verdict it is holding.
func Register(v Validator) {
	if !layer.Valid(v.Layer()) {
		panic("analyze: validator " + v.Name() + " declares layer " + string(v.Layer()) +
			", which is not one of activity|output|outcome|impact")
	}
	if _, taken := Get(v.Name()); taken {
		panic("analyze: validator " + v.Name() + " is already registered, and one name " +
			"cannot carry two verdicts")
	}
	registry = append(registry, v)
}

// Validators returns every registered Validator, sorted by Name for stable,
// deterministic output across runs.
func Validators() []Validator {
	out := make([]Validator, len(registry))
	copy(out, registry)
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// Get returns the validator registered under name, if any.
func Get(name string) (Validator, bool) {
	for _, v := range registry {
		if v.Name() == name {
			return v, true
		}
	}
	return nil, false
}
