// Package docs_test holds the checks that read the repository's own published files. It is an
// external test package because it needs the real command tree, and internal/cli imports the
// package under test.
package docs_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/cli"
	"github.com/assaio/assaio/internal/docs"
)

var update = flag.Bool("update", false, "update generated files")

const (
	repoRoot      = "../.."
	referenceJSON = "../../docs/reference.json"
	referenceHTML = "../../" + docs.ReferenceFile
	sitePage      = "../../site/index.html"
	llmsTxt       = "../../site/llms.txt"
	extendingDocs = "../../docs/extending"
)

func reference() *docs.Reference { return docs.Export(cli.NewRootCmd()) }

// The committed reference is what the website and the docs are checked against, so it is only
// evidence while it still equals what the binary would print. Regenerate with `make docs`.
func TestCommittedReferenceMatchesTheBinary(t *testing.T) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(reference()); err != nil {
		t.Fatal(err)
	}
	assertGenerated(t, referenceJSON, buf.Bytes())
}

func TestCommittedReferencePageMatchesTheBinary(t *testing.T) {
	assertGenerated(t, referenceHTML, docs.HTML(reference(), guides(t)))
}

// The reference was published at /reference before it moved inside the documentation, and a
// static host cannot redirect. The page left behind is what keeps a link shared in that window
// working.
func TestCommittedRedirectMatches(t *testing.T) {
	assertGenerated(t, filepath.Join(repoRoot, docs.RedirectFile), docs.Redirect())
}

// A document under docs/ is published or excused, and never neither: a guide added to the tree
// and forgotten is exactly the drift this section exists to remove. Guides() enforces it, and
// this is what runs the enforcement without generating anything.
func TestEveryDocumentIsPublishedOrExcused(t *testing.T) {
	if _, err := docs.Guides(repoRoot); err != nil {
		t.Error(err)
	}
}

// Every published page is generated from the Markdown it is written in, so the two cannot say
// different things. A link the renderer cannot resolve fails here rather than reaching the site
// as a 404 -- the published copy is the one a reader trusts.
func TestCommittedGuidePagesMatchTheirSource(t *testing.T) {
	all := guides(t)
	for i := range all {
		g := &all[i]
		page, errs := docs.GuidePage(repoRoot, g, all)
		for _, err := range errs {
			t.Error(err)
		}
		if len(errs) == 0 {
			assertGenerated(t, filepath.Join(repoRoot, g.File), page)
		}
	}
}

func guides(t *testing.T) []docs.Guide {
	t.Helper()
	all, err := docs.Guides(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	return all
}

func assertGenerated(t *testing.T, path string, want []byte) {
	t.Helper()
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, want, 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	got, err := os.ReadFile(path) //nolint:gosec // a generated file at a path this test file names
	if err != nil {
		t.Fatalf("%v -- run `make docs`", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s is not what this binary generates. Run `make docs` and commit the result.", path)
	}
}

// The website may not describe a capability the binary lacks, and -- the direction that caught
// `digest` a release late -- may not stay silent about one it has. Prose is deliberately out of
// scope: the page declares which of its claims are checkable, and only those are checked.
func TestTheSitePageAgreesWithTheBinary(t *testing.T) {
	ref := reference()
	for _, path := range []string{sitePage, referenceHTML} {
		report(t, docs.CheckClaims(ref, path, read(t, path)))
	}
}

// llms.txt is read by an assistant instead of the page, so a source missing from it is a source
// that assistant will say assaio does not support. It carries no attributes, so it gets the
// weaker rule: every source has to be named somewhere in it.
func TestLLMsFileNamesEverySource(t *testing.T) {
	report(t, docs.CheckMentions(reference(), llmsTxt, read(t, llmsTxt), "sources"))
}

// B155 is the failure this covers: six prepared-input fields reached no metric plugin while
// five shipped validators read them, and the extension documentation described a contract that
// was not the one on the wire. Both halves of that contract are checked, because documenting
// one and not the other is how they came apart. A field named only in prose does not count --
// the English word "skills" is not evidence that the `skills` field is documented.
func TestExtensionDocsNameEveryContractField(t *testing.T) {
	ref := reference()
	report(t, docs.CheckIdentifiers(ref, extendingDocs, extendingText(t), "metricInput", "metricResult"))
	report(t, docs.CheckIdentifiers(ref, extendingDocs+"/metric-validator.md",
		read(t, extendingDocs+"/metric-validator.md"), "validatorInput"))
}

func extendingText(t *testing.T) string {
	t.Helper()
	entries, err := os.ReadDir(extendingDocs)
	if err != nil {
		t.Fatal(err)
	}
	var all []byte
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".md" {
			continue
		}
		all = append(all, read(t, filepath.Join(extendingDocs, e.Name()))...)
	}
	return string(all)
}

// An exemption is a claim about the world -- "no published surface needs to name this" -- and
// it outlives what it excused as silently as a stale comment does.
func TestNoStaleExemptions(t *testing.T) {
	ref := reference()
	for id, reason := range docs.Exemptions() {
		if reason == "" {
			t.Errorf("%s is exempt with no reason given", id)
		}
		if !docs.Addressable(ref, id) {
			t.Errorf("%s is exempt, but the binary no longer has it -- delete the exemption", id)
		}
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path) //nolint:gosec // a published file at a path this test file names
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func report(t *testing.T, problems []docs.Problem) {
	t.Helper()
	for _, p := range problems {
		t.Error(p)
	}
}

// A command line printed in a document is an instruction a reader will paste. Checking them
// against the binary's own command tree is what stops a renamed flag from being discovered by
// the reader instead of the build.
func TestDocumentsPrintNoFlagTheBinaryLacks(t *testing.T) {
	ref := reference()
	for _, g := range guides(t) {
		report(t, docs.CheckInvocations(ref, g.Source, read(t, filepath.Join(repoRoot, g.Source))))
	}
	for _, f := range []string{"README.md", "CONTRIBUTING.md", "AGENTS.md", "RELEASING.md"} {
		report(t, docs.CheckInvocations(ref, f, read(t, filepath.Join(repoRoot, f))))
	}
}

// The guides recommend helpers by name. A rename that left the advice behind is the defect this
// catches -- four pages recommended `shareOrDash` for a divide-by-zero, and no package had one.
func TestRecommendedHelpersExistAndAreRecommended(t *testing.T) {
	var corpus strings.Builder
	for _, g := range guides(t) {
		corpus.WriteString(read(t, filepath.Join(repoRoot, g.Source)))
	}
	report(t, docs.CheckRecommendedHelpers(repoRoot, corpus.String()))
}

// Every link the site makes to itself has to land somewhere, fragment included. A published
// page is the copy a reader trusts, and a dead link on it is the same class of defect as a
// stale number -- with the difference that nothing else would notice.
func TestEveryLinkInsideTheSiteResolves(t *testing.T) {
	pages := map[string]map[string]bool{}
	var files []string
	err := filepath.WalkDir(filepath.Join(repoRoot, "site"), func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(p) != ".html" {
			return err
		}
		files = append(files, p)
		body := read(t, p)
		ids := map[string]bool{}
		for _, m := range regexp.MustCompile(`id="([^"]+)"`).FindAllStringSubmatch(body, -1) {
			ids[m[1]] = true
		}
		pages[servedURL(p)] = ids
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range files {
		for _, m := range regexp.MustCompile(`href="https://assaio\.dev([^"]*)"`).FindAllStringSubmatch(read(t, p), -1) {
			target, fragment, _ := strings.Cut(m[1], "#")
			if target == "" {
				target = "/"
			}
			ids, ok := pages[target]
			if !ok {
				if _, err := os.Stat(filepath.Join(repoRoot, "site", strings.TrimPrefix(target, "/"))); err == nil {
					continue // a served asset rather than a page
				}
				t.Errorf("%s links to %s, which the site does not serve", servedURL(p), target)
				continue
			}
			if fragment != "" && !ids[fragment] {
				t.Errorf("%s links to %s#%s, and that page has no such anchor", servedURL(p), target, fragment)
			}
		}
	}
}

// servedURL is the path Workers Assets serves a file at: the extension is dropped, and
// index.html is the root.
func servedURL(path string) string {
	rel := filepath.ToSlash(strings.TrimPrefix(path, filepath.Join(repoRoot, "site")))
	rel = strings.TrimSuffix(rel, ".html")
	if rel == "/index" {
		return "/"
	}
	return rel
}

// Every recipe is checked by something, and the somethings are not equally strong. This holds
// the classification to the pages in both directions, so "each recipe on these pages is checked"
// stays a statement about what the suite does rather than about what it once did.
func TestEveryRecipeIsClassified(t *testing.T) {
	var found []string
	for _, g := range guides(t) {
		if !strings.HasPrefix(g.Source, "docs/recipes/") {
			continue
		}
		for _, r := range docs.ExtractRecipes(read(t, filepath.Join(repoRoot, g.Source))) {
			found = append(found, r.Name)
		}
	}
	report(t, docs.CheckRecipeCoverage(found))
	report(t, docs.CheckCoverageCounts("docs/extending.md", read(t, repoRoot+"/docs/extending.md")))
	t.Logf("recipe coverage: %s", docs.CoverageSummary())
}
