package ingest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/assaio/assaio/internal/usage"
)

func TestResolveProjectsRollsUpToRepoRoot(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, ".git"))
	sub := filepath.Join(root, "apps", "x")
	mustMkdirAll(t, sub)

	recs := []usage.Record{{Cwd: sub, Project: "x"}}
	resolveProjects(recs, make(projectCache))

	wantProject, wantSubpath := filepath.Base(root), filepath.Join("apps", "x")
	if recs[0].Project != wantProject || recs[0].Subpath != wantSubpath {
		t.Fatalf("Project=%q Subpath=%q, want %q/%q", recs[0].Project, recs[0].Subpath, wantProject, wantSubpath)
	}
}

func TestResolveProjectsLeavesFallbackWhenCwdEmpty(t *testing.T) {
	recs := []usage.Record{{Cwd: "", Project: "fallback-leaf"}}
	resolveProjects(recs, make(projectCache))

	if recs[0].Project != "fallback-leaf" || recs[0].Subpath != "" {
		t.Fatalf("record = %+v, want Project=fallback-leaf Subpath=\"\" untouched", recs[0])
	}
}

func TestResolveProjectsNoRepoFallsBackToLeafBasename(t *testing.T) {
	dir := t.TempDir()
	recs := []usage.Record{{Cwd: dir, Project: "stale-fallback"}}
	resolveProjects(recs, make(projectCache))

	if want := filepath.Base(dir); recs[0].Project != want || recs[0].Subpath != "" {
		t.Fatalf("Project=%q Subpath=%q, want %q/\"\"", recs[0].Project, recs[0].Subpath, want)
	}
}

// TestResolveProjectsCachesByCwd guards the whole point of the cache: many records
// sharing one Cwd (the common case — every record in a session) must resolve the
// filesystem once, not once per record.
func TestResolveProjectsCachesByCwd(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, ".git"))
	sub := filepath.Join(root, "apps", "x")
	mustMkdirAll(t, sub)

	cache := make(projectCache)
	recs := []usage.Record{
		{Cwd: sub, DedupeKey: "1"},
		{Cwd: sub, DedupeKey: "2"},
	}
	resolveProjects(recs, cache)

	wantProject, wantSubpath := filepath.Base(root), filepath.Join("apps", "x")
	for _, r := range recs {
		if r.Project != wantProject || r.Subpath != wantSubpath {
			t.Fatalf("record %s: Project=%q Subpath=%q, want %q/%q", r.DedupeKey, r.Project, r.Subpath, wantProject, wantSubpath)
		}
	}
	if len(cache) != 1 {
		t.Fatalf("cache has %d entries after two records shared one Cwd, want 1", len(cache))
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o750); err != nil {
		t.Fatal(err)
	}
}

// TestResolveProjectsMarksOnlyAVanishedDirectoryAsGuessed: once a working directory is gone,
// whatever resolves now -- its own name, or an outer repository a removed nested clone sat in --
// is a guess. A directory that still exists is not.
func TestResolveProjectsMarksOnlyAVanishedDirectoryAsGuessed(t *testing.T) {
	repo := t.TempDir()
	mustMkdirAll(t, filepath.Join(repo, ".git"))
	plain := t.TempDir()
	tests := []struct {
		name        string
		cwd         string
		wantProject string
		wantSubpath string
		wantGuessed bool
	}{
		{"a vanished directory outside any repository", filepath.Join(t.TempDir(), "wt-feature"), "wt-feature", "", true},
		{"an existing directory outside any repository", plain, filepath.Base(plain), "", false},
		{"a vanished directory inside an existing repository", filepath.Join(repo, "apps", "gone"), filepath.Base(repo), filepath.Join("apps", "gone"), true},
		{"a vanished clone nested in an existing repository", filepath.Join(repo, "vendor", "acme"), filepath.Base(repo), filepath.Join("vendor", "acme"), true},
		{"an existing directory inside a repository", repo, filepath.Base(repo), "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recs := []usage.Record{{Cwd: tt.cwd}}
			resolveProjects(recs, make(projectCache))
			r := recs[0]
			if r.Project != tt.wantProject || r.Subpath != tt.wantSubpath || r.ProjectGuessed != tt.wantGuessed {
				t.Fatalf("got (%q, %q, guessed=%v), want (%q, %q, guessed=%v)",
					r.Project, r.Subpath, r.ProjectGuessed, tt.wantProject, tt.wantSubpath, tt.wantGuessed)
			}
		})
	}
}
