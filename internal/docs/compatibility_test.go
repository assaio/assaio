package docs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// compatibilityPolicy is the one file allowed to state what v1.0 freezes.
const compatibilityPolicy = "docs/compatibility.md"

// retiredPromise is a sentence a published file used to carry and no longer may, with the
// answer that replaced it. Three documents each stated a different version of the v1.0
// contract for several releases -- the roadmap said SQLite would not be frozen in one section
// and would be in another, the release guide promised a stable schema, and the extension docs
// promised an in-process Go API -- and every existing consistency check passed the whole time,
// because each file was internally consistent.
var retiredPromises = []struct{ phrase, because string }{
	{"SQLite schema become stable", "the schema is an implementation detail with migration and export guarantees"},
	{"freeze of the plugin\nprotocols and the SQLite schema", "only the plugin protocols freeze"},
	{"arriving toward v1.0", "the in-process Go API is deferred and is not a v1 contract"},
}

// publishedContractFiles are the files that speak about compatibility to someone outside this
// repository. A file may link to the policy; it may not restate it.
var publishedContractFiles = []string{
	"README.md", "ROADMAP.md", "RELEASING.md", "BACKLOG.md",
	"docs/README.md", "docs/extending.md", "docs/extending/team-server.md",
}

func TestNoPublishedFileRestatesARetiredCompatibilityPromise(t *testing.T) {
	for _, name := range publishedContractFiles {
		body := readRepoFile(t, name)
		for _, retired := range retiredPromises {
			if strings.Contains(body, retired.phrase) {
				t.Errorf("%s still says %q; %s -- see %s", name, retired.phrase, retired.because, compatibilityPolicy)
			}
		}
	}
}

// TestTheCompatibilityPolicyIsReachable keeps the single answer findable: a policy nothing
// links to is a fourth version of the promise rather than the end of the other three.
func TestTheCompatibilityPolicyIsReachable(t *testing.T) {
	policy := readRepoFile(t, compatibilityPolicy)
	for _, want := range []string{"Frozen at v1.0", "Not frozen, and guaranteed instead", "Deferred, and not a v1 contract"} {
		if !strings.Contains(policy, want) {
			t.Fatalf("%s is missing its %q section", compatibilityPolicy, want)
		}
	}
	for _, name := range []string{"ROADMAP.md", "RELEASING.md", "docs/extending.md"} {
		if !strings.Contains(readRepoFile(t, name), "compatibility.md") {
			t.Errorf("%s does not link to %s, so it is free to grow its own answer again", name, compatibilityPolicy)
		}
	}
}

func readRepoFile(t *testing.T, name string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(repoRoot, name)) //nolint:gosec // a path built from this repo's own tree
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(body)
}
