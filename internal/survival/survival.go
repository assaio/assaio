// Package survival correlates a project's AI activity with how much of its git history
// survives in HEAD -- a directional, local outcome signal. It never attributes specific
// lines to AI (assaio stores counts, not code), so it reports repo-wide survival of the
// window's commits beside the AI lines the store recorded, a correlation to read rather
// than a per-line AI-survival number. It is the local stepping stone toward the
// server-stage git/issue correlation (BACKLOG B18, ROADMAP "Outcome & quality").
package survival

import (
	"context"
	"path/filepath"
	"time"
)

// Result is the directional survival picture for one repository over a window.
type Result struct {
	Project   string
	RepoRoot  string
	Since     time.Time
	AILines   int64 // AI lines added in the window, from the assaio store (counts only)
	GitAdded  int64 // lines the window's commits added (git numstat)
	Surviving int64 // window-commit lines still present in HEAD (git blame)
	// SurvivalRate is Surviving/GitAdded in 0..1, or -1 when GitAdded is 0 (undefined).
	SurvivalRate float64
	Commits      int // commits in the window
	Files        int // window-touched files successfully blamed
}

// RepoRoot resolves repoPath to its git working-tree root (erroring when it is not a repo),
// so a caller can derive the project name and look up its AI lines before Analyze.
func RepoRoot(ctx context.Context, repoPath string) (string, error) {
	return repoRoot(ctx, repoPath)
}

// Project is the project name Analyze reports for root -- its basename, matching how ingest
// resolves a session's project from its git repository root.
func Project(root string) string { return filepath.Base(root) }

// Analyze computes the survival picture for the already-resolved repo root since `since`,
// given the AI lines the store recorded for its project in the same window.
func Analyze(ctx context.Context, root string, since time.Time, aiLines int64) (Result, error) {
	commits, err := windowCommits(ctx, root, since)
	if err != nil {
		return Result{}, err
	}
	added, files, err := addedAndTouched(ctx, root, since)
	if err != nil {
		return Result{}, err
	}
	surviving, blamed, err := survivingLines(ctx, root, files, commits)
	if err != nil {
		return Result{}, err
	}
	rate := -1.0
	if added > 0 {
		rate = float64(surviving) / float64(added)
		if rate > 1 {
			rate = 1
		}
	}
	return Result{
		Project: Project(root), RepoRoot: root, Since: since,
		AILines: aiLines, GitAdded: added, Surviving: surviving, SurvivalRate: rate,
		Commits: len(commits), Files: blamed,
	}, nil
}
