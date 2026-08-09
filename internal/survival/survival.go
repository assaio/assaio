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
	// Unreadable is window-touched files git could not blame. A deleted or renamed-away path
	// legitimately did not survive, but so does a path this build failed to name, and the two
	// are indistinguishable from here -- so the count is reported beside the rate rather than
	// folded into it, the same skip-and-count policy every parser in this repo follows.
	Unreadable int
	// Changed is what those commits touched, by category rather than by path: a window that
	// only moved documentation is a different fact from one that rewrote source, and the
	// rate alone cannot tell them apart.
	Changed event.FileCategories
	// Reverts is how many of the window's commits git itself labelled a revert.
	Reverts int
	// Merges is how many of the window's commits have more than one parent, and MergeLines
	// how many lines in HEAD blame attributes to them. Both sit outside the rate: `git log
	// --numstat` prints no diff for a merge, so a conflict resolution is never counted as
	// added, while blame names the merge for every line of it. Counting one side and not the
	// other reported more surviving lines than were ever added.
	Merges     int
	MergeLines int64
}

// Analyze reads the survival picture from the window's commit observations and the paths
// they touched, given the AI lines the store recorded for the same window. files is
// local-only by construction (see vcs.TouchedFiles) and never reaches the Result.
func Analyze(ctx context.Context, root string, commits []event.Event, files []string, aiLines int64) (Result, error) {
	res := Result{Project: vcs.Project(root), AILines: aiLines}
	inWindow := res.tally(commits)

	blame, err := survivingLines(ctx, root, files, inWindow)
	if err != nil {
		return Result{}, err
	}
	res.Surviving, res.MergeLines, res.Files = blame.surviving, blame.merged, blame.files
	res.Unreadable = blame.unreadable
	res.SurvivalRate = rate(res.Surviving, res.GitAdded)
	return res, nil
}

// tally reads the window's observations into res and returns each commit's hash mapped to
// whether git called it a merge. A merge contributes neither added lines nor a category
// split, because `git log --numstat` prints no diff for one -- so it is counted as a commit,
// kept out of the rate, and answered for separately by blame.
func (r *Result) tally(commits []event.Event) map[string]bool {
	inWindow := make(map[string]bool, len(commits))
	for i := range commits {
		c, ok := commits[i].Payload.(event.Commit)
		if !ok {
			continue // a mixed observation stream is what the contract is for; read only ours
		}
		r.Commits++
		if c.Revert {
			r.Reverts++
		}
		if c.Parents > 1 {
			r.Merges++
			inWindow[commits[i].ID] = true
			continue
		}
		inWindow[commits[i].ID] = false
		r.GitAdded += c.LinesAdded
		r.Changed = add(r.Changed, c.Files)
	}
	return inWindow
}

// blameTally is what one pass over the touched files found: lines still in HEAD from the
// window's ordinary commits, lines from its merges, how many files could be blamed, and how
// many could not.
type blameTally struct {
	surviving, merged int64
	files, unreadable int
}

// survivingLines blames each still-present touched file at HEAD and counts the lines whose
// commit is in the window -- window-authored lines still in the tree. A merge's lines are
// counted apart from the rest: numstat never reported them as added, so counting them as
// survivors would divide by a total they were never in. A file git cannot blame is skipped
// and counted: a deleted or renamed-away path legitimately did not survive, but a path this
// build failed to read did not fail to survive, and the rate must not present the second as
// the first without saying how many there were.
func survivingLines(ctx context.Context, root string, files []string, inWindow map[string]bool) (blameTally, error) {
	var t blameTally
	for _, f := range files {
		hashes, blameErr := vcs.Blame(ctx, root, f)
		if blameErr != nil {
			if ctx.Err() != nil {
				return t, ctx.Err()
			}
			t.unreadable++
			continue
		}
		t.files++
		for _, h := range hashes {
			merge, ok := inWindow[h]
			switch {
			case !ok:
			case merge:
				t.merged++
			default:
				t.surviving++
			}
		}
	}
	return t, nil
}

// rate is Surviving/GitAdded, or -1 when nothing was added -- an undefined rate rather than a
// zero one, since a window that added nothing neither survived nor failed to. Both sides
// count the same commits, so the ratio cannot exceed 1; it is deliberately left uncorrected
// rather than clamped, because a clamp would print a wrong figure as a confident 100%.
func rate(surviving, added int64) float64 {
	if added <= 0 {
		return -1
	}
	return float64(surviving) / float64(added)
}

func add(a, b event.FileCategories) event.FileCategories {
	return event.FileCategories{
		Test: a.Test + b.Test, Source: a.Source + b.Source, Docs: a.Docs + b.Docs,
		Config: a.Config + b.Config, Generated: a.Generated + b.Generated, Other: a.Other + b.Other,
	}
}
