package docs_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// usesLine matches a workflow step's action reference, e.g. "- uses: actions/checkout@<ref>".
var usesLine = regexp.MustCompile(`(?m)^\s*-?\s*uses:\s*(\S+)`)

// pinnedRef matches a 40-character commit SHA, the only reference SECURITY.md's supply-chain
// section allows.
var pinnedRef = regexp.MustCompile(`@[0-9a-f]{40}$`)

// TestEveryWorkflowActionIsPinnedBySHA holds SECURITY.md to what the workflows actually do.
// The policy said every action was pinned while `actions/checkout@v4` sat in five jobs across
// two workflows for several releases -- a security claim stronger than its implementation,
// which is worse than no claim. A tag is mutable: whoever can move it can change what runs in
// CI without a diff anyone reviews.
func TestEveryWorkflowActionIsPinnedBySHA(t *testing.T) {
	dir := filepath.Join(repoRoot, ".github", "workflows")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read workflows: %v", err)
	}
	checked := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yml") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name())) //nolint:gosec // a path this test built from the repo tree
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		for _, m := range usesLine.FindAllStringSubmatch(string(body), -1) {
			ref := strings.Trim(m[1], `"'`)
			// A local composite action is referenced by path and has no ref to pin.
			if strings.HasPrefix(ref, "./") {
				continue
			}
			checked++
			if !pinnedRef.MatchString(ref) {
				t.Errorf("%s: %q is not pinned to a commit SHA; SECURITY.md says every action is", e.Name(), ref)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no action references found: this check would pass on an empty workflow directory")
	}
}
