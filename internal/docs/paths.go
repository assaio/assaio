package docs

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// LandingSource is the page served at /docs and the one place reading order is written: each
// `##` is a reading path, and the links in the lists under it are that path's guides, in order.
// Links in the lede or in a path's sentence are prose and do not make a guide a member.
const LandingSource = "docs/index.md"

type readingPath struct {
	Title   string
	Sources []string
}

// pathSection is one `##` of the landing with the blocks that belong to it: everything up to
// the next `#` or `##`, cut after its last list, so a closing paragraph stays with the page.
type pathSection struct {
	heading *ast.Heading
	blocks  []ast.Node
}

func pathSections(root ast.Node) []pathSection {
	var out []pathSection
	var cur *pathSection
	flush := func() {
		if cur == nil {
			return
		}
		last := 0
		for i, n := range cur.blocks {
			if n.Kind() == ast.KindList {
				last = i
			}
		}
		cur.blocks = cur.blocks[:last+1]
		out = append(out, *cur)
		cur = nil
	}
	for n := root.FirstChild(); n != nil; n = n.NextSibling() {
		if h, ok := n.(*ast.Heading); ok && h.Level <= 2 {
			flush()
			if h.Level == 2 {
				cur = &pathSection{heading: h, blocks: []ast.Node{h}}
			}
			continue
		}
		if cur != nil {
			cur.blocks = append(cur.blocks, n)
		}
	}
	flush()
	return out
}

// readPaths reads the reading paths out of the landing. found is every publishable document
// mapped to its title; each error names the landing, since that is the file to fix.
func readPaths(doc string, found map[string]string) ([]readingPath, error) {
	src := []byte(doc)
	root := newMarkdown().Parser().Parse(text.NewReader(src))
	var paths []readingPath
	var errs []error
	fail := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(LandingSource+": "+format, args...))
	}
	owner := map[string]string{}
	for _, s := range pathSections(root) {
		p := readingPath{Title: plainText(s.heading, src)}
		if p.Title == "" {
			fail("has a `##` with no title; a path is named by its heading")
		}
		if p.Title == GroupStart || p.Title == GroupRef || slices.ContainsFunc(paths,
			func(q readingPath) bool { return q.Title == p.Title }) {
			fail("the path %q shares its name with another sidebar group; each needs its own", p.Title)
		}
		for _, dest := range listLinks(s.blocks, src) {
			source, err := landingTarget(dest, found)
			if err != nil {
				fail("the path %q lists %q, %v", p.Title, dest, err)
				continue
			}
			if prev, dup := owner[source]; dup {
				fail("%s is listed under %q and again under %q; a guide belongs to one path",
					source, prev, p.Title)
				continue
			}
			owner[source] = p.Title
			p.Sources = append(p.Sources, source)
		}
		if len(p.Sources) == 0 {
			fail("the path %q lists no guide", p.Title)
		}
		paths = append(paths, p)
	}
	if len(paths) == 0 && len(errs) == 0 {
		fail("has no `##` reading path")
	}
	return paths, errors.Join(errs...)
}

// listLinks is every link a path lists as a member: those in the first paragraph of each item
// of the lists directly under the `##`, before any deeper heading. A nested list or a list under
// a `###` is commentary on the path, not part of it.
func listLinks(blocks []ast.Node, src []byte) []string {
	var out []string
	for _, b := range blocks[1:] {
		if b.Kind() == ast.KindHeading {
			break
		}
		if b.Kind() != ast.KindList {
			continue
		}
		for item := b.FirstChild(); item != nil; item = item.NextSibling() {
			if first := item.FirstChild(); first != nil &&
				(first.Kind() == ast.KindParagraph || first.Kind() == ast.KindTextBlock) {
				out = append(out, links(first, src)...)
			}
		}
	}
	return out
}

func links(n ast.Node, src []byte) []string {
	var out []string
	_ = ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch l := n.(type) {
		case *ast.Link:
			if entering {
				out = append(out, string(l.Destination))
			}
		case *ast.AutoLink:
			if entering {
				out = append(out, string(l.URL(src)))
			}
		}
		return ast.WalkContinue, nil
	})
	return out
}

// landingTarget resolves one list link the way resolveLink does, and accepts only a publishable
// document other than the landing itself.
func landingTarget(dest string, found map[string]string) (string, error) {
	if dest == "" || strings.HasPrefix(dest, "#") || strings.Contains(dest, ":") {
		return "", errors.New("which is not a document under docs/; a path lists only guides")
	}
	target, _, _ := strings.Cut(dest, "#")
	source := path.Clean(path.Join(path.Dir(LandingSource), target))
	_, excused := unpublished[source]
	_, ok := found[source]
	switch {
	case source == LandingSource:
		return "", errors.New("which is this page; the landing is not a guide")
	case excused:
		return "", fmt.Errorf("but %s is excused in the unpublished list in internal/docs/guides.go", source)
	case !ok:
		return "", fmt.Errorf("but no publishable document %s exists", source)
	}
	return source, nil
}
