package projectid

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) (cwd, wantRoot, wantSubpath string)
	}{
		{"repo root itself", setupAtRepoRoot},
		{"nested subdirectory", setupNestedSubdir},
		{"deep nesting", setupDeepNesting},
		{"worktree checkout directory rolls up to main repo", setupWorktreeDir},
		{"subdirectory under a worktree rolls up too", setupWorktreeSubdir},
		{"submodule-shaped git file stays put", setupSubmoduleShape},
		{"malformed gitdir pointer falls back to its own dir", setupMalformedPointer},
		{"directory with no git anywhere", setupNoGit},
		{"nonexistent path", setupNonexistent},
		{"empty cwd", func(t *testing.T) (string, string, string) { return "", "", "" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cwd, wantRoot, wantSubpath := tt.setup(t)
			gotRoot, gotSubpath := Resolve(cwd)
			if gotRoot != wantRoot {
				t.Errorf("Resolve(%q) root = %q, want %q", cwd, gotRoot, wantRoot)
			}
			if gotSubpath != wantSubpath {
				t.Errorf("Resolve(%q) subpath = %q, want %q", cwd, gotSubpath, wantSubpath)
			}
		})
	}
}

// TestResolveDeterministic guards against any hidden dependency on iteration or read
// order: the same cwd must resolve identically every time.
func TestResolveDeterministic(t *testing.T) {
	_, sub := setupNestedSubdirPair(t)
	wantRoot, wantSubpath := Resolve(sub)
	for range 5 {
		gotRoot, gotSubpath := Resolve(sub)
		if gotRoot != wantRoot || gotSubpath != wantSubpath {
			t.Fatalf("Resolve(%q) not deterministic: got (%q,%q), want (%q,%q)", sub, gotRoot, gotSubpath, wantRoot, wantSubpath)
		}
	}
}

// TestRealFilesystemMonorepo is an opt-in, real-filesystem smoke test: point
// ASSAIO_TEST_MONOREPO at any local git checkout to exercise Resolve against a real
// working tree. It skips when the variable is unset, so the suite stays hermetic in CI
// and on machines without a checkout to point at.
func TestRealFilesystemMonorepo(t *testing.T) {
	repo := os.Getenv("ASSAIO_TEST_MONOREPO")
	if repo == "" {
		t.Skip("ASSAIO_TEST_MONOREPO not set; export it to smoke-test Resolve against a real checkout")
	}
	//nolint:gosec // repo is the developer's own opt-in ASSAIO_TEST_MONOREPO path, read only
	if _, err := os.Stat(filepath.Join(repo, ".git")); err != nil {
		t.Fatalf("ASSAIO_TEST_MONOREPO %q is not a git checkout: %v", repo, err)
	}

	sub := filepath.Join(repo, "apps", "mobile")
	if !dirExists(sub) {
		sub = repo
	}
	root, subpath := Resolve(sub)
	t.Logf("Resolve(%q) = (root: %q, subpath: %q)", sub, root, subpath)
	if root != repo {
		t.Errorf("Resolve(%q) root = %q, want %q", sub, root, repo)
	}
}

func setupAtRepoRoot(t *testing.T) (cwd, wantRoot, wantSubpath string) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, ".git"))
	return root, root, ""
}

func setupNestedSubdir(t *testing.T) (cwd, wantRoot, wantSubpath string) {
	root, sub := setupNestedSubdirPair(t)
	return sub, root, filepath.Join("apps", "mobile")
}

// setupNestedSubdirPair builds <root>/.git and <root>/apps/mobile; shared with the
// determinism and Project tests so they exercise the same fixture shape.
func setupNestedSubdirPair(t *testing.T) (root, sub string) {
	t.Helper()
	root = t.TempDir()
	mustMkdirAll(t, filepath.Join(root, ".git"))
	sub = filepath.Join(root, "apps", "mobile")
	mustMkdirAll(t, sub)
	return root, sub
}

func setupDeepNesting(t *testing.T) (cwd, wantRoot, wantSubpath string) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, ".git"))
	sub := filepath.Join(root, "a", "b", "c")
	mustMkdirAll(t, sub)
	return sub, root, filepath.Join("a", "b", "c")
}

func setupWorktreeDir(t *testing.T) (cwd, wantRoot, wantSubpath string) {
	root := writeWorktreeFixture(t)
	return filepath.Join(root, ".claude", "worktrees", "wt"), root, filepath.Join(".claude", "worktrees", "wt")
}

func setupWorktreeSubdir(t *testing.T) (cwd, wantRoot, wantSubpath string) {
	root := writeWorktreeFixture(t)
	sub := filepath.Join(root, ".claude", "worktrees", "wt", "src")
	mustMkdirAll(t, sub)
	return sub, root, filepath.Join(".claude", "worktrees", "wt", "src")
}

// writeWorktreeFixture builds <root>/.git (main repo dir) and
// <root>/.claude/worktrees/wt/.git (a worktree pointer file), mirroring the shape
// `git worktree add` creates on disk.
func writeWorktreeFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, ".git"))
	wt := filepath.Join(root, ".claude", "worktrees", "wt")
	mustMkdirAll(t, wt)
	gitdir := "gitdir: " + filepath.Join(root, ".git", "worktrees", "wt") + "\n"
	mustWriteFile(t, filepath.Join(wt, ".git"), gitdir)
	return root
}

// setupSubmoduleShape covers a ".git" file whose gitdir points elsewhere but never
// through a "/.git/worktrees/" segment — the resolver must not roll it up.
func setupSubmoduleShape(t *testing.T) (cwd, wantRoot, wantSubpath string) {
	root := t.TempDir()
	sub := filepath.Join(root, "vendor", "lib")
	mustMkdirAll(t, sub)
	mustWriteFile(t, filepath.Join(sub, ".git"), "gitdir: ../../.git/modules/vendor/lib\n")
	return sub, sub, ""
}

func setupMalformedPointer(t *testing.T) (cwd, wantRoot, wantSubpath string) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, ".git"), "not a gitdir line\n")
	return root, root, ""
}

func setupNoGit(t *testing.T) (cwd, wantRoot, wantSubpath string) {
	dir := t.TempDir()
	return dir, dir, ""
}

func setupNonexistent(t *testing.T) (cwd, wantRoot, wantSubpath string) {
	dir := filepath.Join(t.TempDir(), "does", "not", "exist")
	return dir, dir, ""
}

func dirExists(path string) bool {
	info, err := os.Stat(path) //nolint:gosec // test helper; paths come from t.TempDir or the developer's own env var
	return err == nil && info.IsDir()
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o750); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
