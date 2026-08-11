package docs

import "strings"

// HTML renders the reference as one page inside the documentation. It names no version: a
// served file may not carry a stamp that has to be updated at every tag (docs/site.md), and the
// schema version belongs in the JSON, where a tool reads it. Every URL it emits drops the
// ".html" -- Workers Assets serves the file at its extensionless path and redirects the other
// form, so a canonical naming the file would point at a redirect.
func HTML(ref *Reference, guides []Guide) []byte {
	var b strings.Builder
	b.WriteString(`<h1>Reference</h1>
<p class="lede">Everything assaio can enumerate about itself, read from the registries inside the
binary rather than transcribed by hand. What a figure <em>means</em> is prose and lives in the
guides; what can be listed is listed here, and a check fails the build when this page and the
binary disagree. The same document is machine-readable through
<code>assaio-agent docs export</code>.</p>`)
	writeSignals(&b, ref.Signals)
	writeSources(&b, ref.Sources)
	writeValidators(&b, ref.Validators)
	writeCommands(&b, ref.Commands, ref.GlobalFlags)
	writeConfig(&b, ref.Config)
	writeWire(&b, ref.ValidatorInput, ref.MetricWire)

	return renderPage(&pageSpec{
		Title: "Reference",
		Description: "Every signal, source, validator, command, flag, configuration key and " +
			"metric-contract field assaio has, generated from the binary's own registries.",
		URL:     ReferenceURL,
		Current: ReferenceURL,
		Body:    b.String(),
		Guides:  guides,
	})
}
