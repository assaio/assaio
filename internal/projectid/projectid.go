// Package projectid resolves a session's working directory to its canonical git
// repository root, so a monorepo subdirectory (e.g. apps/mobile) attributes to one
// project instead of fragmenting by leaf directory name.
package projectid

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

// worktreeSegment marks a git-worktree pointer file's gitdir path: the text before it
// is the worktree's main repository root.
const worktreeSegment = "/.git/worktrees/"

// Resolve walks up from cwd to the nearest directory containing a .git entry and
// returns it as repoRoot, rolling a git worktree up to its main repository. subpath is
// cwd relative to repoRoot, or "" when cwd is the root itself. If no .git is found —
// including when cwd does not exist — repoRoot is cwd and subpath is "". Resolve never
// errors or panics: a stat or read failure just falls back to the next directory up.
func Resolve(cwd string) (repoRoot, subpath string) {
	if cwd == "" {
		return "", ""
	}
	cwd = filepath.Clean(cwd)
	repoRoot = findRepoRoot(cwd)
	if repoRoot == "" {
		return cwd, ""
	}
	if rel, err := filepath.Rel(repoRoot, cwd); err == nil && rel != "." {
		subpath = rel
	}
	return repoRoot, subpath
}

// findRepoRoot walks dir and its ancestors looking for a .git entry, stopping at the
// filesystem root. It returns "" if none is found.
func findRepoRoot(dir string) string {
	for {
		if root, ok := repoRootAt(dir); ok {
			return root
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// repoRootAt reports the repository root implied by a .git entry directly inside dir,
// and whether one was found there at all.
func repoRootAt(dir string) (root string, ok bool) {
	gitPath := filepath.Join(dir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return "", false
	}
	if info.IsDir() {
		return dir, true
	}
	if main, ok := worktreeMainRoot(gitPath); ok {
		return main, true
	}
	return dir, true
}

// worktreeMainRoot reads a ".git" file shaped as a git-worktree pointer
// ("gitdir: <root>/.git/worktrees/<name>") and returns the main repository root, the
// text before "/.git/". ok is false for a submodule's ".git" file or anything else
// that does not match the worktree shape, so the caller falls back to treating the
// pointer file's own directory as the root.
func worktreeMainRoot(gitFile string) (root string, ok bool) {
	data, err := os.ReadFile(gitFile) //nolint:gosec // gitFile is built from the directory being walked, not external input
	if err != nil {
		return "", false
	}
	gitdir, ok := gitdirValue(data)
	if !ok {
		return "", false
	}
	gitdir = filepath.ToSlash(gitdir)
	idx := strings.Index(gitdir, worktreeSegment)
	if idx < 0 {
		return "", false
	}
	return filepath.FromSlash(gitdir[:idx]), true
}

// gitdirValue extracts the path after "gitdir:" from a git pointer file's first
// matching line.
func gitdirValue(data []byte) (string, bool) {
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if rest, found := strings.CutPrefix(line, "gitdir:"); found {
			return strings.TrimSpace(rest), true
		}
	}
	return "", false
}
