package docs

import (
	"bytes"
	"strconv"

	"github.com/yuin/goldmark/ast"
)

// githubIDs generates heading anchors the way GitHub does, because every cross-reference in
// docs/ was written against GitHub's rendering: "`answers` — which zeros…" is addressed as
// #answers--which-zeros…, with the double hyphen the em dash leaves behind. goldmark's own
// slugger collapses that, so the published page would answer to a different anchor than the
// repository does and every deep link written by hand would land at the top of the page.
type githubIDs struct{ seen map[string]int }

func newGitHubIDs() *githubIDs { return &githubIDs{seen: map[string]int{}} }

func (g *githubIDs) Generate(value []byte, _ ast.NodeKind) []byte {
	slug := slugify(value)
	if slug == "" {
		slug = "section"
	}
	n := g.seen[slug]
	g.seen[slug]++
	if n > 0 {
		return []byte(slug + "-" + strconv.Itoa(n))
	}
	return []byte(slug)
}

func (g *githubIDs) Put(value []byte) { g.seen[string(value)]++ }

// slugify is GitHub's rule: lowercase, drop everything that is not a letter, digit, hyphen,
// underscore or space, then turn each remaining space into a hyphen. Each space individually --
// collapsing a run is what loses the double hyphen an em dash leaves.
func slugify(value []byte) string {
	var b bytes.Buffer
	for _, r := range string(bytes.ToLower(value)) {
		switch {
		case r == ' ':
			b.WriteByte('-')
		case r == '-' || r == '_' || isAlphanumeric(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isAlphanumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') ||
		(r >= 'A' && r <= 'Z') || r > 127 && !isPunct(r)
}

// isPunct keeps non-ASCII punctuation out of a slug the way GitHub's \w class does. Letters
// with diacritics stay, because a heading in another language must still be addressable.
func isPunct(r rune) bool {
	switch r {
	case '—', '–', '“', '”', '‘', '’', '…', '·', '«', '»':
		return true
	}
	return false
}
