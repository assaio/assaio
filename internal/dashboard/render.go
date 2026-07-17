package dashboard

import (
	_ "embed"
	"html/template"
	"io"
	"strconv"

	"github.com/assaio/assaio/internal/analyze"
)

//go:embed dashboard.html.tmpl
var templateSource string

// templateFuncs are the Assay template's only computed helpers: percent formatting, the
// faceplate/ledger numbering, and the locale lookup. Everything else the template
// renders is already a plain, pre-formatted string on Data or analyze.Result -- see each
// validator's own formatting in internal/analyze.
var templateFuncs = template.FuncMap{
	"pct":            pctString,
	"facets":         facets,
	"subpathName":    subpathName,
	"barsApplicable": func(r analyze.Result) bool { return r.Bars != nil },
	"locale":         func() localeStrings { return en },
}

var tmpl = template.Must(template.New("dashboard").Funcs(templateFuncs).Parse(templateSource))

// RenderHTML writes d as a single self-contained HTML page: inline CSS only, no external
// fonts/scripts/network requests, one small inline theme-toggle script. html/template
// auto-escapes every field, including project/subpath names sourced from local session
// logs.
//
//nolint:gocritic // Data is caller-constructed and rendered once per command run; by-value matches Build's own return and keeps this the simple entry point its result flows into directly.
func RenderHTML(w io.Writer, d Data) error {
	return tmpl.Execute(w, d)
}

// pctString renders a 0..1 fraction as a one-decimal percent width, e.g. 0.847 -> "84.7"
// (the template supplies the literal "%").
func pctString(f float64) string {
	return strconv.FormatFloat(f*100, 'f', 1, 64)
}

// facetView pairs a validator Result with its 1-based display position -- the numbering
// the top-level faceplate, the drill's compact faceplate, and the ledger all share.
//
// EXTENSIBILITY SEAM: this is the only place a Result's position in the report is
// decided, and it is generic over analyze.Validators()'s registration order -- a newly
// registered analyze.Validator gets a number and renders on the faceplate and ledger
// with no template change.
type facetView struct {
	N int
	R analyze.Result
}

func facets(results []analyze.Result) []facetView {
	out := make([]facetView, len(results))
	for i := range results {
		r := results[i]
		r.Purity = clampPurity(r.Purity)
		out[i] = facetView{N: i + 1, R: r}
	}
	return out
}

// clampPurity keeps a validator's Purity within the gauge's valid [0,1] range. Purity is
// "set honestly per validator" (see analyze.Result's doc comment), including by a
// third-party, out-of-tree analyze.Validator this package cannot control -- one that
// misbehaves and returns outside [0,1] would otherwise render a broken or overflowing
// gauge width.
func clampPurity(p float64) float64 {
	switch {
	case p < 0:
		return 0
	case p > 1:
		return 1
	default:
		return p
	}
}

// subpathName renders a project-relative subpath for the drill-down table, substituting
// a repository-root label for the empty string.
func subpathName(s string) string {
	if s == "" {
		return ". (root)"
	}
	return s
}
