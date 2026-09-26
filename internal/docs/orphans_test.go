package docs_test

import (
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/assaio/assaio/internal/docs"
)

// `make docs` writes pages and never deletes one, and the host serves every file under site/,
// so the page of a removed guide would stay live describing what the binary no longer has.
func TestNoGeneratedPageOutlivesItsSource(t *testing.T) {
	generated := map[string]bool{docs.ReferenceFile: true}
	for _, g := range guides(t) {
		generated[g.File] = true
	}
	err := filepath.WalkDir(filepath.Join(repoRoot, "site", "docs"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".html" {
			return err
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		if !generated[filepath.ToSlash(rel)] {
			t.Errorf("%s has no source left; delete it, since `make docs` never deletes a page", filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
