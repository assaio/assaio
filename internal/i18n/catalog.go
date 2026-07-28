// Package i18n holds every static, data-independent string assaio renders, so a
// translation is a new catalog rather than an edit spread across the codebase.
//
// The boundary is what belongs here, not merely what is visible. Chrome -- labels,
// headers, footers, and the long-form explain pages -- reads the same on every run and
// lives here. Text derived from the queried data -- a validator's Read, Takeaway, and
// HowToRead, figures, project and model names -- is data, flows from analyze.Result, and
// never belongs here; translating it would mean templating interpolated numbers, which is
// a separate piece of work.
//
// This package is a leaf: it imports nothing from internal/, so any package may read it
// without creating a cycle. internal/analyze deliberately does not -- it stays a pure
// measurement package, and the explain pages are looked up by validator name from the CLI.
package i18n

// Catalog is one language's complete set of static strings.
type Catalog struct {
	Dashboard Dashboard
	CLI       CLI
	// Explain maps a validator's name to its long-form page. Absent means the validator
	// has no page yet; internal/cli's completeness test makes that a build failure.
	Explain map[string]string
}

// en is today's -- and only -- locale, assembled from the per-surface files beside this one.
var en = Catalog{Dashboard: enDashboard, CLI: enCLI, Explain: enExplain}

// For returns the catalog for locale, falling back to English for an empty or unknown
// one. Adding a language means adding its Catalog and one case here.
func For(locale string) Catalog {
	switch locale {
	case "en", "":
		return en
	default:
		return en
	}
}

// Explain returns the long-form page for a validator name in the default locale.
func Explain(name string) (string, bool) {
	text, ok := For("").Explain[name]
	return text, ok
}
