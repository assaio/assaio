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
	"testing"

	"github.com/assaio/assaio/internal/cli"
	"github.com/assaio/assaio/internal/docs"
)

var update = flag.Bool("update", false, "update generated files")

const (
	referenceJSON = "../../docs/reference.json"
	referenceHTML = "../../site/reference.html"
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
	assertGenerated(t, referenceHTML, docs.HTML(reference()))
}

func assertGenerated(t *testing.T, path string, want []byte) {
	t.Helper()
	if *update {
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
