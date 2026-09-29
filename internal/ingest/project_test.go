package ingest

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

func TestResolveProjectsRollsUpToRepoRoot(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, ".git"))
	sub := filepath.Join(root, "apps", "x")
	mustMkdirAll(t, sub)

	recs := []usage.Record{{Cwd: sub, Project: "x"}}
	resolveProjects(recs, newProjectCache(nil))

	wantProject, wantSubpath := filepath.Base(root), filepath.Join("apps", "x")
	if recs[0].Project != wantProject || recs[0].Subpath != wantSubpath {
		t.Fatalf("Project=%q Subpath=%q, want %q/%q", recs[0].Project, recs[0].Subpath, wantProject, wantSubpath)
	}
}

func TestResolveProjectsLeavesFallbackWhenCwdEmpty(t *testing.T) {
	recs := []usage.Record{{Cwd: "", Project: "fallback-leaf"}}
	resolveProjects(recs, newProjectCache(nil))

	if recs[0].Project != "fallback-leaf" || recs[0].Subpath != "" {
		t.Fatalf("record = %+v, want Project=fallback-leaf Subpath=\"\" untouched", recs[0])
	}
}

func TestResolveProjectsNoRepoFallsBackToLeafBasename(t *testing.T) {
	dir := t.TempDir()
	recs := []usage.Record{{Cwd: dir, Project: "stale-fallback"}}
	resolveProjects(recs, newProjectCache(nil))

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

	cache := newProjectCache(nil)
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
	if len(cache.byCwd) != 1 {
		t.Fatalf("cache has %d entries after two records shared one Cwd, want 1", len(cache.byCwd))
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
// is a guess, and a guess names no repository. A directory that still exists is not a guess, and
// has a repository only when a .git was found above it.
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
		wantKey     bool
	}{
		{"a vanished directory outside any repository", filepath.Join(t.TempDir(), "wt-feature"), "wt-feature", "", true, false},
		{"an existing directory outside any repository", plain, filepath.Base(plain), "", false, false},
		{"a vanished directory inside an existing repository", filepath.Join(repo, "apps", "gone"), filepath.Base(repo), filepath.Join("apps", "gone"), true, false},
		{"a vanished clone nested in an existing repository", filepath.Join(repo, "vendor", "acme"), filepath.Base(repo), filepath.Join("vendor", "acme"), true, false},
		{"an existing directory inside a repository", repo, filepath.Base(repo), "", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recs := []usage.Record{{Cwd: tt.cwd}}
			resolveProjects(recs, newProjectCache(testSalt))
			r := recs[0]
			if r.Project != tt.wantProject || r.Subpath != tt.wantSubpath || r.ProjectGuessed != tt.wantGuessed {
				t.Fatalf("got (%q, %q, guessed=%v), want (%q, %q, guessed=%v)",
					r.Project, r.Subpath, r.ProjectGuessed, tt.wantProject, tt.wantSubpath, tt.wantGuessed)
			}
			if (r.RepoKey != "") != tt.wantKey {
				t.Fatalf("RepoKey = %q, want a key: %v", r.RepoKey, tt.wantKey)
			}
		})
	}
}

var testSalt = []byte("0123456789abcdef0123456789abcdef")

// TestTwoRepositoriesWithOneNameStayApart: two checkouts both named api, one of them reached from
// a subdirectory and a worktree, ingested into one store.
func TestTwoRepositoriesWithOneNameStayApart(t *testing.T) {
	ctx := context.Background()
	base := t.TempDir()
	work := filepath.Join(base, "work", "api")
	scratch := filepath.Join(base, "scratch", "api")
	mustMkdirAll(t, filepath.Join(work, ".git", "worktrees", "wt"))
	mustMkdirAll(t, filepath.Join(scratch, ".git"))
	mustMkdirAll(t, filepath.Join(scratch, "apps", "mobile"))
	wt := filepath.Join(base, "elsewhere", "wt")
	mustMkdirAll(t, wt)
	if err := os.WriteFile(filepath.Join(wt, ".git"), []byte("gitdir: "+filepath.Join(work, ".git", "worktrees", "wt")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	salt, err := st.RepositorySalt(ctx)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Now().UTC().Add(-time.Hour)
	rec := func(cwd, key string) usage.Record {
		return usage.Record{
			Tool: "claude-code", SessionID: key, Timestamp: at, Model: "m",
			DedupeKey: key, Granularity: "turn", InputTokens: 10, Cwd: cwd,
		}
	}
	var res Result
	recs := []usage.Record{rec(work, "1"), rec(wt, "2"), rec(filepath.Join(scratch, "apps", "mobile"), "3")}
	if err := ingestParsed(ctx, st, newProjectCache(salt), &res, recs, 0, nil); err != nil {
		t.Fatal(err)
	}
	rows, err := st.Usage(ctx, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int64{}
	for i := range rows {
		got[rows[i].Project] += rows[i].In
	}
	if want := map[string]int64{"api": 20, "api (2)": 10}; !reflect.DeepEqual(got, want) {
		t.Fatalf("shown = %v, want %v: the worktree joins its main checkout, the other checkout stays apart", got, want)
	}
}
