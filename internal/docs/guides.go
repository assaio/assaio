package docs

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// Guide is one document the site publishes.
type Guide struct {
	// Source is its repository-relative Markdown path -- the one place it is written.
	Source string
	// URL is where it is served; File is where the generated page is committed.
	URL, File string
	Title     string
	// Group is the heading it appears under in the docs index.
	Group string
}

// Groups, in the order a reader meets them.
const (
	GroupStart   = "Start here"
	GroupGuides  = "Guides"
	GroupRecipes = "Recipes"
	GroupRef     = "Reference"
)

// order is the one editorial fact about this set: what a reader should meet first. Everything
// else -- which documents exist, what they are called -- is read from the tree, so a new page
// cannot be forgotten. A publishable document missing from this list fails the build rather than
// appearing last or not at all.
var order = []string{
	"docs/extending.md",
	"docs/extending/metric-validator.md",
	"docs/extending/metric-validator-example.md",
	"docs/extending/metric-plugin.md",
	"docs/extending/rule-plugin.md",
	"docs/extending/parser-plugin.md",
	"docs/extending/data-source.md",
	"docs/extending/query-your-data.md",
	"docs/extending/team-server.md",
	"docs/extending/source-fields.md",
	"docs/automation.md",
	"docs/reconcile.md",
	"docs/format-resilience.md",
	"docs/recipes/label-rules.md",
	"docs/recipes/ci-gates.md",
	"docs/recipes/rule-plugins.md",
	"docs/recipes/automation.md",
	"docs/recipes/extensions.md",
}

// unpublished names each Markdown file under docs/ that the site deliberately does not serve,
// with the reason. An entry here is a decision; a file in neither list is an oversight, and
// TestEveryDocumentIsPublishedOrExcused is what tells them apart.
var unpublished = map[string]string{
	"docs/README.md": "a map of the repository's own files, useful in a checkout and noise on a site",
	"docs/site.md":   "how this site is built and deployed -- for the maintainer, not the reader",
}

// Guides reads the published set from the repository and returns it in reading order.
func Guides(repoRoot string) ([]Guide, error) {
	found, err := documents(repoRoot)
	if err != nil {
		return nil, err
	}
	out := make([]Guide, 0, len(found))
	for _, source := range order {
		title, ok := found[source]
		if !ok {
			return nil, fmt.Errorf("%s is listed in reading order but the repository has no such file", source)
		}
		delete(found, source)
		out = append(out, guide(source, title))
	}
	for source := range found {
		return nil, fmt.Errorf("%s is publishable but appears in neither the reading order nor the "+
			"unpublished list in internal/docs/guides.go", source)
	}
	return out, nil
}

func guide(source, title string) Guide {
	slug := strings.TrimSuffix(path.Base(source), ".md")
	group := GroupGuides
	switch {
	case source == "docs/extending.md":
		return Guide{Source: source, URL: "/docs", File: "site/docs.html", Title: title, Group: GroupStart}
	case strings.HasPrefix(source, "docs/recipes/"):
		group = GroupRecipes
		slug = "recipes/" + slug
	}
	return Guide{
		Source: source,
		URL:    "/docs/" + slug,
		File:   "site/docs/" + slug + ".html",
		Title:  title,
		Group:  group,
	}
}

// documents finds every Markdown file under docs/ that could be published, mapped to its title.
// ADRs are excluded as a directory: they are decision records addressed to contributors, and
// each one names commitments in terms only the codebase gives meaning to.
func documents(repoRoot string) (map[string]string, error) {
	out := map[string]string{}
	root := filepath.Join(repoRoot, "docs")
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir():
			if name := d.Name(); name == "adr" || name == "superpowers" || name == "assets" {
				return filepath.SkipDir
			}
			return nil
		case filepath.Ext(p) != ".md":
			return nil
		}
		rel := filepath.ToSlash(strings.TrimPrefix(p, repoRoot+string(filepath.Separator)))
		if _, excused := unpublished[rel]; excused {
			return nil
		}
		title, err := firstHeading(p)
		if err != nil {
			return err
		}
		out[rel] = title
		return nil
	})
	return out, err
}

func firstHeading(p string) (string, error) {
	b, err := os.ReadFile(p) //nolint:gosec // a path this package walked under the repository root
	if err != nil {
		return "", err
	}
	for line := range strings.Lines(string(b)) {
		if after, ok := strings.CutPrefix(line, "# "); ok {
			return strings.TrimSpace(after), nil
		}
	}
	return "", fmt.Errorf("%s has no top-level heading to take a title from", p)
}

// PublishedMap is what the link rewriter needs: source path to served URL, including the
// generated reference, which has no Markdown source of its own.
func PublishedMap(guides []Guide) map[string]string {
	out := map[string]string{ReferenceSource: Host + ReferenceURL}
	for _, g := range guides {
		out[g.Source] = Host + g.URL
	}
	return out
}

// The reference is generated rather than written, so it has no source file -- but a document
// linking to it needs a name to link by, and docs/reference.json is the artifact it is.
const (
	ReferenceSource = "docs/reference.json"
	ReferenceURL    = "/docs/reference"
	ReferenceFile   = "site/docs/reference.html"
)

// GroupsInOrder is how the index lays the set out.
func GroupsInOrder() []string { return []string{GroupStart, GroupGuides, GroupRecipes, GroupRef} }

// InGroup returns the guides of one group, keeping reading order.
func InGroup(guides []Guide, group string) []Guide {
	return slices.DeleteFunc(slices.Clone(guides), func(g Guide) bool { return g.Group != group })
}
