package docs

import (
	"html"
	"strings"
)

// table writes one HTML table whose body declares the registry set it covers, so the claim
// checker holds the generated page to the same rule as the hand-written one -- a generator
// that dropped a row fails the same test a forgetful edit would.
type table struct {
	b *strings.Builder
}

func newTable(b *strings.Builder, covers string, headers ...string) *table {
	b.WriteString(`<div class="scroll"><table><thead><tr>`)
	for _, h := range headers {
		b.WriteString("<th>" + esc(h) + "</th>")
	}
	b.WriteString(`</tr></thead><tbody`)
	if covers != "" {
		b.WriteString(` data-covers="` + esc(covers) + `"`)
	}
	b.WriteString(">")
	return &table{b: b}
}

// row writes one row, claiming the registry member it describes. Cells are written as given:
// callers pass cell() output, which escapes.
func (t *table) row(claim string, cells ...string) {
	t.b.WriteString("<tr")
	if claim != "" {
		t.b.WriteString(` data-claim="` + esc(claim) + `"`)
	}
	t.b.WriteString(">")
	for _, c := range cells {
		t.b.WriteString(c)
	}
	t.b.WriteString("</tr>")
}

func (t *table) close() { t.b.WriteString("</tbody></table></div>") }

// cell renders one escaped table cell. class is written verbatim and is never user input.
func cell(class, text string) string {
	open := "<td>"
	if class != "" {
		open = `<td class="` + class + `">`
	}
	return open + esc(text) + "</td>"
}

// rawCell renders a cell whose contents are already-escaped markup built by the caller.
func rawCell(class, markup string) string {
	open := "<td>"
	if class != "" {
		open = `<td class="` + class + `">`
	}
	return open + markup + "</td>"
}

func esc(s string) string { return html.EscapeString(s) }

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "—"
}

func bullets(items []string) string {
	if len(items) == 0 {
		return `<span class="dim">—</span>`
	}
	var b strings.Builder
	b.WriteString(`<ul class="gaps">`)
	for _, it := range items {
		b.WriteString("<li>" + esc(it) + "</li>")
	}
	b.WriteString("</ul>")
	return b.String()
}
