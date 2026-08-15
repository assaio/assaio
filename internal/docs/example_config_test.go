package docs_test

import (
	"os"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/cli"
	"github.com/assaio/assaio/internal/docs"
)

// TestExampleConfigNamesEveryTopLevelKey holds the gap that let `trace.horizon_days` was the one top-level
// key absent from the file README.md calls "a documented starting point" -- and it is the knob
// governing 102.0 MB of a measured 170.1 MB store. It was in the generated reference and in
// docs/reference.json, so nothing mechanical noticed; two reviewers found it independently.
//
// A key may appear commented out: the file's convention is that an optional section ships as a
// worked example behind "#". What it may not do is go unmentioned.
func TestExampleConfigNamesEveryTopLevelKey(t *testing.T) {
	body, err := os.ReadFile(repoRoot + "/config.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	seen := map[string]bool{}
	for _, k := range docs.Export(cli.NewRootCmd()).Config {
		top := k.Key
		if i := strings.IndexAny(top, ".["); i >= 0 {
			top = top[:i]
		}
		if seen[top] {
			continue
		}
		seen[top] = true
		if !strings.Contains(text, "\n"+top+":") && !strings.Contains(text, "\n# "+top+":") {
			t.Errorf("config.example.yaml never names the top-level key %q; every key it omits is one a reader cannot discover from the file the README sends them to", top)
		}
	}
	if len(seen) == 0 {
		t.Fatal("no config keys came back from the register, so the assertion above never ran")
	}
}
