package docs

import (
	"fmt"
	"maps"
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
	// Group is the sidebar heading it appears under: the reading path in docs/index.md that lists
	// it, GroupStart for the landing itself.
	Group string
}

// The two sidebar groups no reading path supplies: the landing, and the generated reference.
const (
	GroupStart = "Start here"
	GroupRef   = "Reference"
)

// unpublished names each Markdown file under docs/ that the site deliberately does not serve,
// with the reason. An entry here is a decision; a file in neither list is an oversight, and
// TestEveryDocumentIsPublishedOrExcused is what tells them apart.
var unpublished = map[string]string{
	"docs/README.md": "a map of the repository's own files, useful in a checkout and noise on a site",
	"docs/site.md":   "how this site is built and deployed -- for the maintainer, not the reader",
	"docs/architecture.md": "the path through the internal packages, step by step -- addressed to " +
		"someone reading the tree, which is why the ADRs it sits beside are excluded too",
	"docs/threat-model.md": "belongs with SECURITY.md and PRIVACY.md, which the site links into the " +
		"repository rather than serving",
	"docs/corrections.md": "a register of figures that were wrong, quoting the command lines and flags " +
		"of the releases it describes -- which the invocation check would read as claims about " +
		"today's binary",
	"docs/operations.md": "every entry point a maintainer can run and what it writes -- for the " +
		"maintainer and the agent harness, like site.md",
}

// Guides reads the published set from the repository and returns it in reading order: the
// landing, then each reading path's guides as docs/index.md lists them. A publishable document
// no path lists fails the build rather than appearing last or not at all.
func Guides(repoRoot string) ([]Guide, error) {
	found, err := documents(repoRoot)
	if err != nil {
		return nil, err
	}
	landing, ok := found[LandingSource]
	if !ok {
		return nil, fmt.Errorf("%s is missing; it is the landing page and the reading order", LandingSource)
	}
	raw, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(LandingSource))) //nolint:gosec // the landing under the repository root
	if err != nil {
		return nil, err
	}
	paths, err := readPaths(string(raw), found)
	if err != nil {
		return nil, err
	}
	out := []Guide{guide(LandingSource, landing, GroupStart)}
	delete(found, LandingSource)
	for _, p := range paths {
		for _, source := range p.Sources {
			out = append(out, guide(source, found[source], p.Title))
			delete(found, source)
		}
	}
	if len(found) > 0 {
		orphans := slices.Sorted(maps.Keys(found))
		return nil, fmt.Errorf("%s is publishable but no reading path in %s lists it; add it to one "+
			"there, or excuse it in the unpublished list in internal/docs/guides.go",
			strings.Join(orphans, ", "), LandingSource)
	}
	return out, nil
}

func guide(source, title, group string) Guide {
	if source == LandingSource {
		return Guide{Source: source, URL: "/docs", File: "site/docs.html", Title: title, Group: group}
	}
	slug := strings.TrimSuffix(path.Base(source), ".md")
	if strings.HasPrefix(source, "docs/recipes/") {
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
// each one names commitments in terms only the codebase gives meaning to. docs/work/ holds
// tasks in progress and parked plans, addressed to whoever picks them up next.
func documents(repoRoot string) (map[string]string, error) {
	out := map[string]string{}
	root := filepath.Join(repoRoot, "docs")
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir():
			if name := d.Name(); name == "adr" || name == "work" || name == "assets" {
				return filepath.SkipDir
			}
			return nil
		case filepath.Ext(p) != ".md":
			return nil
		}
		// filepath.Rel rather than trimming a prefix: repoRoot is written with slashes and
		// WalkDir yields the platform's separator, so the two do not match on Windows and every
		// document reads as absent.
		rel, err := filepath.Rel(repoRoot, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
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

// GroupsInOrder is how the sidebar lays the set out: each group as the guides first reach it,
// then the reference.
func GroupsInOrder(guides []Guide) []string {
	var out []string
	for _, g := range guides {
		if !slices.Contains(out, g.Group) {
			out = append(out, g.Group)
		}
	}
	if !slices.Contains(out, GroupRef) {
		out = append(out, GroupRef)
	}
	return out
}

// InGroup returns the guides of one group, keeping reading order.
func InGroup(guides []Guide, group string) []Guide {
	return slices.DeleteFunc(slices.Clone(guides), func(g Guide) bool { return g.Group != group })
}
