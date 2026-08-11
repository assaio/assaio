package docs

import (
	"bytes"
	"fmt"
	"path"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

// GitHubBlob is where a document that is not published on the site still lives.
const GitHubBlob = "https://github.com/assaio/assaio/blob/main/"

// Site is where each document ends up. Published maps a repository-relative Markdown path to
// its URL on the site; Exists reports whether the repository holds a path at all, which is what
// separates "lives on GitHub" from "this link is broken".
type Site struct {
	Published map[string]string
	Exists    func(repoPath string) bool
}

// Markdown renders one document to an HTML fragment, rewriting every relative link to where
// that target actually lives once published. A link this cannot resolve is an error rather than
// a silently emitted 404: the site is the copy a reader trusts, and a dead link there is the
// same class of defect as a stale number.
func Markdown(source, doc string, s Site) (string, []error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.Table, extension.Strikethrough, extension.Linkify),
		// Heading ids come from the context generator below; without this option goldmark
		// assigns none at all and every in-page anchor written in docs/ lands at the top.
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		// Every document rendered here is written in this repository and reviewed in a pull
		// request, and the site guards read the generated file: a page that fetched anything
		// or carried an embed fails `selfcontained`. Without this, goldmark writes a visible
		// "raw HTML omitted" marker where a document has so much as a comment.
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)
	reader := text.NewReader([]byte(doc))
	ctx := parser.NewContext(parser.WithIDs(newGitHubIDs()))
	root := md.Parser().Parse(reader, parser.WithContext(ctx))

	errs := rewriteLinks(root, path.Dir(source), s)

	var out bytes.Buffer
	if err := md.Renderer().Render(&out, []byte(doc), root); err != nil {
		return "", append(errs, err)
	}
	return out.String(), errs
}

func rewriteLinks(root ast.Node, fromDir string, s Site) []error {
	var errs []error
	err := ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		link, ok := n.(*ast.Link)
		if !ok {
			return ast.WalkContinue, nil
		}
		dest, err := resolveLink(fromDir, string(link.Destination), s)
		if err != nil {
			errs = append(errs, err)
			return ast.WalkContinue, nil
		}
		link.Destination = []byte(dest)
		return ast.WalkContinue, nil
	})
	if err != nil {
		errs = append(errs, err)
	}
	return errs
}

// resolveLink maps one Markdown destination to a URL. A published document becomes a site path;
// anything else in the repository becomes a link into the repository, which is honest -- an ADR
// or a Go file is not something the site serves.
func resolveLink(fromDir, dest string, s Site) (string, error) {
	switch {
	case dest == "", strings.HasPrefix(dest, "#"),
		strings.Contains(dest, "://"), strings.HasPrefix(dest, "mailto:"):
		return dest, nil
	}
	target, fragment, _ := strings.Cut(dest, "#")
	if fragment != "" {
		fragment = "#" + fragment
	}
	repoPath := path.Clean(path.Join(fromDir, target))
	if url, ok := s.Published[repoPath]; ok {
		return url + fragment, nil
	}
	// Not published, but in the tree: an ADR, a Go file, a root document. Linking into the
	// repository is the honest answer -- the site does not serve those.
	if s.Exists == nil || s.Exists(repoPath) {
		return GitHubBlob + repoPath + fragment, nil
	}
	return "", fmt.Errorf("links to %q, which resolves to %s -- no such file, and nothing publishes it",
		dest, repoPath)
}
