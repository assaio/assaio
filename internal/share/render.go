package share

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"html/template"
	"io"
)

//go:embed share.html.tmpl
var templateSource string

var tmpl = template.Must(template.New("share").Funcs(template.FuncMap{
	"payload": payload,
}).Parse(templateSource))

// RenderHTML writes a as one self-contained page: inline CSS and one inline script, no
// external font, image or network request of any kind. The preview draws every frame on a
// canvas rather than in the DOM, so what the recorder captures is the same pixels the
// person approved -- a DOM preview and a canvas export would be two renderers free to
// disagree.
func RenderHTML(w io.Writer, a *Assay) error {
	return tmpl.Execute(w, a)
}

// payload serializes the assay for the page's own renderer. It is marshalled rather than
// interpolated field by field so the drawing code reads one shape, and json.Marshal's
// escaping keeps a model name -- the only vendor-supplied string that reaches here -- from
// closing the script element.
func payload(a *Assay) (template.JS, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(a); err != nil {
		return "", err
	}
	return template.JS(buf.String()), nil //nolint:gosec // json.Encoder with SetEscapeHTML escapes <, > and & into \u form, so no marshalled string can close the script element.
}

// Text renders the terminal form: the same assay as a block a person can paste into a
// pull request, a Slack thread or a README. It is the one format that survives SSH, and
// the only one that needs no browser at all.
func Text(a *Assay) string {
	var b bytes.Buffer
	// A bytes.Buffer never fails a write, so the errWriter's verdict has nothing to report
	// here; the error path exists for the io.Writer contract writeText keeps for callers.
	_ = writeText(&b, a)
	return b.String()
}
