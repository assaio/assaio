package vcs

import (
	"context"
	"strings"
)

// madeHereEntries matches the reflog entries git writes when it creates a commit in this clone --
// a commit, an amend, a replayed pick in a rebase, a cherry-pick, a revert, an applied patch, or a
// merge commit git made -- and not the ones where HEAD only moved to a commit made elsewhere: a
// checkout, a pull, a reset, a fast-forward. Git matches them; this process sees hashes only, never
// the entries' messages.
const madeHereEntries = `^(commit|cherry-pick|revert|am)( \([a-z]+\))?: |` +
	`^rebase( -i)? \((pick|reword|edit|squash|fixup)\): |^merge [^:]+: Merge made by`

// MadeHere returns every commit a HEAD reflog of root's repository records as made in this clone,
// in any of its worktrees. Git expires reflog entries (gc.reflogExpire, and sooner
// gc.reflogExpireUnreachable for a commit no ref reaches any more), and a removed worktree takes
// its reflog with it, so the set covers only what the reflogs still hold. every is false when git
// could not list the worktrees and only root's was read.
func MadeHere(ctx context.Context, root string) (made map[string]bool, every bool, err error) {
	trees, every := worktrees(ctx, root)
	made = map[string]bool{}
	read := false
	var last error
	for _, tree := range trees {
		out, err := gitOutput(ctx, tree, "log", "-g", "-E", "--grep-reflog="+madeHereEntries, "--format=%H", "HEAD")
		if err != nil {
			last = err // a worktree with no commit yet, or whose directory is gone
			continue
		}
		read = true
		for _, hash := range strings.Fields(string(out)) {
			made[hash] = true
		}
	}
	if !read {
		return nil, every, last
	}
	return made, every, nil
}

// worktrees lists the working trees of root's repository, the main one first, and whether git
// listed them. The paths are read to run git in them and are never kept. A git too old to list
// them with -z reads root alone.
func worktrees(ctx context.Context, root string) ([]string, bool) {
	out, err := gitOutput(ctx, root, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return []string{root}, false
	}
	var trees []string
	for _, field := range strings.Split(string(out), "\x00") {
		if path, ok := strings.CutPrefix(field, "worktree "); ok {
			trees = append(trees, path)
		}
	}
	if len(trees) == 0 {
		return []string{root}, false
	}
	return trees, true
}
