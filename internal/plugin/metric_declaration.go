package plugin

import (
	"slices"

	"github.com/assaio/assaio/internal/analyze"
)

// Declaration is what a metric plugin's `describe` verb states it reads: the capabilities it
// needs, the columns it reads inside them, and the rows it reads at all. The author knows this
// and the person pasting a config entry does not, which is why it moved here in protocol 4 --
// before that the same facts lived in the reader's config.yaml, where omitting one bought a
// withheld section and a hunt through the plugin's own documentation (B168).
type Declaration struct {
	// Needs is the capability vocabulary of internal/analyze, the same one a built-in
	// validator declares. At least one is required: a plugin that reads nothing has nothing
	// to report, and treating an empty list as "everything" would put the pre-4 default back
	// under a name that says the opposite.
	Needs []analyze.Capability `json:"needs"`
	// Fields narrows a section to the columns named, keyed by the section's JSON key. A
	// section absent here is sent whole. A column projected away is *absent* from the
	// document rather than zero; `projection.fields` on the envelope is what tells the two
	// apart on arrival.
	Fields map[string][]string `json:"fields,omitempty"`
	// Where drops rows, keyed `<section>.<column>` with the values that column may hold. It is
	// the plugin's own denominator choice -- `usage.granularity` picks a grain, `trace.scope`
	// picks a population -- so the envelope reports what each predicate excluded.
	Where map[string][]string `json:"where,omitempty"`
}

// Projection is a Declaration after the local config has had its veto: what this run will
// actually carry, and what the config denied. Config constrains, never defines: a capability
// the plugin did not ask for is not granted by a config entry that permits it.
type Projection struct {
	Needs    []analyze.Capability
	Fields   map[string][]string
	Where    map[string][]string
	Withheld []analyze.Capability
}

func (p Projection) grants(c analyze.Capability) bool { return slices.Contains(p.Needs, c) }

// negotiate intersects what the plugin declared with what the config allows. An empty allow
// list is no constraint rather than an empty grant: the overwhelming majority of entries carry
// no `needs:` line, and reading that silence as a denial would break every plugin the moment
// this shipped.
func negotiate(d Declaration, allow []analyze.Capability) Projection {
	p := Projection{Fields: d.Fields, Where: d.Where}
	for _, c := range d.Needs {
		if len(allow) == 0 || slices.Contains(allow, c) {
			p.Needs = append(p.Needs, c)
			continue
		}
		p.Withheld = append(p.Withheld, c)
	}
	if len(p.Withheld) > 0 {
		p.Fields = keptFor(p, d.Fields)
		p.Where = keptFor(p, d.Where)
	}
	return p
}

// keptFor drops the field lists and predicates addressing a section the config denied. Leaving
// them in would echo a projection over a section the document does not carry, which is a
// contradiction a plugin would have to resolve by guessing.
func keptFor(p Projection, keyed map[string][]string) map[string][]string {
	out := make(map[string][]string, len(keyed))
	for key, values := range keyed {
		if p.grants(sections[owningSection(key)].capability) {
			out[key] = values
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// owningSection resolves either address form -- a section name, as `fields` is keyed, or a
// `<section>.<column>` predicate, as `where` is -- to the section it constrains.
func owningSection(key string) string {
	if _, ok := sections[key]; ok {
		return key
	}
	if owner, _, ok := splitPredicate(key); ok {
		return owner
	}
	return key
}
