package docs

import (
	"bufio"
	"bytes"
	stdhtml "html"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer/html"
)

// Heading is one h2 of a rendered document: the id its element carries and its text.
type Heading struct {
	ID, Text string
}

// outline reads the h2 headings out of a parsed document. The id is the one the parser already
// assigned: githubIDs suffixes a repeated heading, so generating it again would not match.
func outline(root ast.Node, src []byte) []Heading {
	var out []Heading
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		h, ok := n.(*ast.Heading)
		if !entering || !ok || h.Level != 2 {
			return ast.WalkContinue, nil
		}
		v, ok := h.AttributeString("id")
		if id, isBytes := v.([]byte); ok && isBytes {
			out = append(out, Heading{ID: string(id), Text: plainText(h, src)})
		}
		return ast.WalkSkipChildren, nil
	})
	return out
}

// plainText is a heading's text as the page shows it, with its inline markup dropped: a link
// inside a heading must not become a link nested in the outline's own. The result is unescaped;
// the caller escapes it once on output.
func plainText(n ast.Node, src []byte) string {
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := c.(type) {
		case *ast.Text:
			writeText(w, t.Value(src), t.IsRaw() || t.Parent().Kind() == ast.KindCodeSpan)
			if t.SoftLineBreak() {
				_ = w.WriteByte(' ')
			}
		case *ast.String:
			if t.IsCode() {
				_, _ = w.Write(t.Value)
			} else {
				writeText(w, t.Value, t.IsRaw())
			}
		}
		return ast.WalkContinue, nil
	})
	_ = w.Flush()
	return strings.TrimSpace(stdhtml.UnescapeString(b.String()))
}

// writeText writes a span the way the renderer does, so backslash escapes and entity
// references resolve exactly as they do in the heading itself; raw spans, code among them,
// resolve neither.
func writeText(w *bufio.Writer, value []byte, raw bool) {
	if raw {
		html.DefaultWriter.RawWrite(w, value)
		return
	}
	html.DefaultWriter.Write(w, value)
}

// tocHTML is the "On this page" list, or nothing for a page with fewer than two sections, where
// an outline would only repeat the one heading the reader can already see.
func tocHTML(headings []Heading) string {
	if len(headings) < 2 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<nav class="toc" aria-label="On this page"><p class="sidehead">On this page</p><ul>`)
	for _, h := range headings {
		b.WriteString(`<li><a href="#` + esc(h.ID) + `">` + esc(h.Text) + `</a></li>`)
	}
	b.WriteString(`</ul></nav>`)
	return b.String()
}
