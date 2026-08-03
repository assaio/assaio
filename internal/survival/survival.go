// Package survival correlates a project's AI activity with how much of its git history
// survives in HEAD -- a directional, local outcome signal. It never attributes specific
// lines to AI (assaio stores counts, not code), so it reports repo-wide survival of the
// window's commits beside the AI lines the store recorded, a correlation to read rather
// than a per-line AI-survival number. It is the local stepping stone toward the
// server-stage git/issue correlation (BACKLOG B18, ROADMAP "Outcome & quality").
//
// The git reading itself is internal/vcs's: this package consumes commit observations and
// blames the working tree, so what a commit changed and what survived are one dataset.
package survival

import (
	"context"

	"github.com/assaio/assaio/internal/event"
	"github.com/assaio/assaio/internal/vcs"
)

// Result is the directional survival picture for one repository over a window.
type Result struct {
	Project   string
	AILines   int64 // AI lines added in the window, from the assaio store (counts only)
	GitAdded  int64 // lines the window's commits added
	Surviving int64 // window-commit lines still present in HEAD (git blame)
	// SurvivalRate is Surviving/GitAdded in 0..1, or -1 when GitAdded is 0 (undefined).
	SurvivalRate float64
	Commits      int // commits in the window
	Files        int // window-touched files successfully blamed
	// Changed is what those commits touched, by category rather than by path: a window that
	// only moved documentation is a different fact from one that rewrote source, and the
	// rate alone cannot tell them apart.
	Changed event.FileCategories
	// Reverts is how many of the window's commits git itself labelled a revert.
	Reverts int
}

// Analyze reads the survival picture from the window's commit observations and the paths
// they touched, given the AI lines the store recorded for the same window. files is
// local-only by construction (see vcs.TouchedFiles) and never reaches the Result.
func Analyze(ctx context.Context, root string, commits []event.Event, files []string, aiLines int64) (Result, error) {
	res := Result{Project: vcs.Project(root), AILines: aiLines}
	inWindow := make(map[string]struct{}, len(commits))
	for i := range commits {
		c, ok := commits[i].Payload.(event.Commit)
		if !ok {
			continue // a mixed observation stream is what the contract is for; read only ours
		}
		res.Commits++
		inWindow[commits[i].ID] = struct{}{}
		res.GitAdded += c.LinesAdded
		res.Changed = add(res.Changed, c.Files)
		if c.Revert {
			res.Reverts++
		}
	}

	surviving, blamed, err := survivingLines(ctx, root, files, inWindow)
	if err != nil {
		return Result{}, err
	}
	res.Surviving, res.Files = surviving, blamed
	res.SurvivalRate = rate(surviving, res.GitAdded)
	return res, nil
}

// survivingLines blames each still-present touched file at HEAD and counts the lines whose
// commit is in the window -- window-authored lines still in the tree. A file git cannot
// blame (deleted or renamed away) is skipped; its lines legitimately did not survive.
func survivingLines(ctx context.Context, root string, files []string, inWindow map[string]struct{}) (surviving int64, blamed int, err error) {
	for _, f := range files {
		hashes, blameErr := vcs.Blame(ctx, root, f)
		if blameErr != nil {
			if ctx.Err() != nil {
				return surviving, blamed, ctx.Err()
			}
			continue // a deleted, renamed-away or binary path cannot be blamed: it did not survive
		}
		blamed++
		for _, h := range hashes {
			if _, ok := inWindow[h]; ok {
				surviving++
			}
		}
	}
	return surviving, blamed, nil
}

// rate is Surviving/GitAdded, or -1 when nothing was added -- an undefined rate rather than a
// zero one, since a window that added nothing neither survived nor failed to.
func rate(surviving, added int64) float64 {
	if added <= 0 {
		return -1
	}
	if surviving > added {
		return 1
	}
	return float64(surviving) / float64(added)
}

func add(a, b event.FileCategories) event.FileCategories {
	return event.FileCategories{
		Test: a.Test + b.Test, Source: a.Source + b.Source, Docs: a.Docs + b.Docs,
		Config: a.Config + b.Config, Generated: a.Generated + b.Generated, Other: a.Other + b.Other,
	}
}
