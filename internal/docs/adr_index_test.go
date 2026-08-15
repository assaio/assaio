package docs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestADRIndexNamesEveryRecord guards the map in docs/README.md, which listed ADRs 0001-0005 of
// twelve. AGENTS.md and site/llms.txt both send a reader there for where architecture
// decisions live, and 0012 -- the table that is now the largest thing in the store -- was not
// on it. An index that silently stops being an index is worse than no index.
func TestADRIndexNamesEveryRecord(t *testing.T) {
	index, err := os.ReadFile(repoRoot + "/docs/README.md")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(repoRoot + "/docs/adr")
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		found++
		if !strings.Contains(string(index), "adr/"+e.Name()) {
			t.Errorf("docs/README.md never links %s", e.Name())
		}
	}
	if found == 0 {
		t.Fatal("no ADRs found, so the assertion above never ran")
	}
}
