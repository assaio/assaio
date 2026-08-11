package docs

import (
	"fmt"
	"sort"
	"strings"
)

// member is one thing a set contains, under the id a page claims it by and the name a page
// writes it as. The two differ because a claim is addressed and prose is read.
type member struct{ id, name string }

// index answers the two questions a check asks: does this id name something, and what does a
// set contain. A set holds only what a published surface is held to *enumerate*; the claim
// space is wider, so a page may address a subcommand without every page having to list it.
type index struct {
	ids    map[string]bool
	counts map[string]int
	sets   map[string][]member
}

func newIndex(ref *Reference) *index {
	ix := &index{ids: map[string]bool{}, counts: map[string]int{}, sets: map[string][]member{}}

	for i := range ref.Signals {
		ix.add("signals", "signal."+ref.Signals[i].ID, ref.Signals[i].ID)
	}
	for _, s := range ref.Sources {
		ix.add("sources", "source."+s.Tool, s.Tool)
	}
	for _, v := range ref.Validators {
		ix.add("validators", "validator."+v.Name, v.Name)
	}
	for _, c := range ref.Commands {
		// A surface has to name every capability, not every mode of one: `signals coverage` is
		// addressable and published, but `signals` is the thing assaio grew.
		if c.TopLevel() {
			ix.add("commands", "command."+c.Path, c.Path)
		} else {
			ix.ids["command."+c.Path] = true
		}
		// Flags are addressable but belong to no covered set: a page that leans on one may pin
		// it, and none is obliged to list them all.
		for _, f := range c.Flags {
			ix.ids["flag."+c.Path+"."+f.Name] = true
		}
	}
	for _, f := range ref.GlobalFlags {
		ix.ids["flag."+f.Name] = true
	}
	for _, k := range ref.Config {
		ix.add("config", "config."+k.Key, k.Key)
	}
	// Only each envelope's own fields: a nested column is named "day" or "tool", and requiring
	// those would pass on any document mentioning a day. The failure these sets are for is
	// B155 -- a whole field of a contract that no documentation learned about. The return path
	// gets the same treatment as the read path, because documenting one and not the other is
	// how the two came apart in the first place.
	for _, f := range ref.MetricWire.Input {
		if topLevelField(f.Path) {
			ix.add("metricInput", "wire.in."+f.Path, f.Path)
		}
	}
	for _, f := range ref.MetricWire.Result {
		if topLevelField(f.Path) {
			ix.add("metricResult", "wire.out."+f.Path, f.Path)
		}
	}
	for _, f := range ref.ValidatorInput {
		ix.add("validatorInput", "input."+f.Name, f.Name)
	}

	ix.counts["signals.count"] = len(ref.Signals)
	ix.counts["sources.count"] = len(ref.Sources)
	ix.counts["validators.count"] = len(ref.Validators)
	return ix
}

func (ix *index) add(set, id, name string) {
	ix.ids[id] = true
	ix.sets[set] = append(ix.sets[set], member{id: id, name: name})
}

// check reports whether one claim holds. A count claim is verified against the text the
// annotated element renders, because that sentence is what a reader believes.
func (ix *index) check(id, content string) (problem string, ok bool) {
	if want, isCount := ix.counts[id]; isCount {
		texts := elementTexts(content, id)
		// A zero-length result is the shape a scanner fails in -- a self-closing tag, an
		// attribute broken across a line. Reporting the count verified without having read a
		// character of it is the false green this whole file exists to avoid.
		if len(texts) == 0 {
			return fmt.Sprintf("claims %s, but no element carrying it states a number", id), false
		}
		for _, text := range texts {
			if !mentionsCount(text, want) {
				return fmt.Sprintf("claims %s, but the registry has %d and the element reads %q",
					id, want, text), false
			}
		}
		return "", true
	}
	if !ix.ids[id] {
		return fmt.Sprintf("claims %q, which the binary has no %s for", id, kindOf(id)), false
	}
	return "", true
}

func (ix *index) setNames() string {
	names := make([]string, 0, len(ix.sets))
	for name := range ix.sets {
		names = append(names, name)
	}
	sort.Strings(names)
	return "Known sets: " + strings.Join(names, ", ") + "."
}

func topLevelField(path string) bool { return !strings.ContainsAny(path, ".[{") }

func kindOf(id string) string {
	if kind, _, found := strings.Cut(id, "."); found {
		return kind
	}
	return "claim"
}
