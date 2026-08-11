// Package docs projects assaio's live registries into one machine-readable reference, so the
// published surfaces are generated or checked rather than transcribed. It owns no facts: every
// block below reads the package that already answers the question, and a fact stated here would
// be the second opinion this package exists to remove. See BACKLOG B161.
package docs

import "github.com/spf13/cobra"

// Version is the reference schema's own version, carried as assaio_docs. It is deliberately
// not the binary's version: version.Version is "dev" on a plain build and a tag on a release
// build, so stamping it would make the committed reference differ on every machine -- and
// site/reference.html is a served file, which may name no version at all (docs/site.md).
const Version = 1

// Reference is everything about assaio that can be enumerated rather than described.
type Reference struct {
	Version    int         `json:"assaio_docs"`
	Signals    []Signal    `json:"signals"`
	Sources    []Source    `json:"sources"`
	Validators []Validator `json:"validators"`
	Commands   []Command   `json:"commands"`
	// GlobalFlags are the root's persistent flags, accepted by every command above.
	GlobalFlags []Flag      `json:"globalFlags"`
	Config      []ConfigKey `json:"config"`
	// ValidatorInput is the in-tree metric contract; MetricWire is the out-of-tree one. Both
	// are published because an extension surface weaker than the core it extends is the
	// failure B155 named, and neither list can be checked against the other by hand.
	ValidatorInput []Field    `json:"validatorInput"`
	MetricWire     MetricWire `json:"metricWire"`
}

// Export builds the reference. The command tree arrives as an argument rather than being
// imported: internal/cli will import this package to wire `docs export`, so reaching back for
// it would be an import cycle -- and a walk over an injected tree is testable without
// constructing the whole CLI.
func Export(root *cobra.Command) *Reference {
	return &Reference{
		Version:        Version,
		Signals:        signals(),
		Sources:        sources(),
		Validators:     validators(),
		Commands:       commands(root),
		GlobalFlags:    globalFlags(root),
		Config:         configKeys(),
		ValidatorInput: validatorInput(),
		MetricWire:     metricWire(),
	}
}
