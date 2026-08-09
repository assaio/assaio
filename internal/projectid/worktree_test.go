package projectid

import (
	"path/filepath"
	"testing"
)

// TestRelativeGitdirPointerRollsUp covers the pointer file git 2.48 and later writes:
// "gitdir:" relative to the worktree rather than absolute. Read verbatim it matched no
// "/.git/worktrees/" segment, so every worktree session in every repository resolved to the
// pointer file's own directory -- and, once Rel could not relate that to the cwd, to a
// project literally named "..".
func TestRelativeGitdirPointerRollsUp(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, ".git"))
	wt := filepath.Join(root, "wt")
	mustMkdirAll(t, wt)
	mustWriteFile(t, filepath.Join(wt, ".git"), "gitdir: ../.git/worktrees/wt\n")

	gotRoot, gotSubpath := Resolve(wt)
	if gotRoot != root {
		t.Errorf("root = %q, want %q", gotRoot, root)
	}
	if gotSubpath != "wt" {
		t.Errorf("subpath = %q, want %q", gotSubpath, "wt")
	}
}

// TestWorktreeOutsideItsRepoHasNoSubpath holds the PRIVACY.md line: a worktree checked out
// beside its main repository is not inside it, so Rel produces a path that climbs out.
// Storing that would put host directory names in a field documented as a repository-relative
// subpath. The honest answer is that there is no subpath.
func TestWorktreeOutsideItsRepoHasNoSubpath(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "repo")
	mustMkdirAll(t, filepath.Join(root, ".git", "worktrees", "feature"))
	wt := filepath.Join(base, "elsewhere", "feature")
	mustMkdirAll(t, wt)
	mustWriteFile(t, filepath.Join(wt, ".git"),
		"gitdir: "+filepath.Join(root, ".git", "worktrees", "feature")+"\n")

	gotRoot, gotSubpath := Resolve(wt)
	if gotRoot != root {
		t.Errorf("root = %q, want %q", gotRoot, root)
	}
	if gotSubpath != "" {
		t.Errorf("subpath = %q, want \"\" -- a path that escapes the root is not a subpath of it", gotSubpath)
	}
}
