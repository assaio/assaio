package cli

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/attribution"
	"github.com/assaio/assaio/internal/event"
	"github.com/assaio/assaio/internal/github"
	"github.com/assaio/assaio/internal/vcs"
	"github.com/assaio/assaio/internal/version"
)

// ghRunner is the gh `evidence --github` executes; a test hands it a stand-in.
var ghRunner github.Runner = github.Exec

// readPullRequests reads the repository's pull requests through the user's gh and the listed
// commits this clone made, and returns the pull-request data session-commit/v3 links with plus the
// commits to match: head's, and the listed ones read here.
func readPullRequests(cmd *cobra.Command, root, project string, since, now time.Time, head []event.Event) (*attribution.PullRequests, []event.Event, int, error) {
	ctx := cmd.Context()
	got, err := github.ReadPullRequests(ctx, ghRunner, root, project, since, now, version.Version)
	if err != nil {
		return nil, nil, 0, err
	}
	cmd.PrintErrln("note: pull requests read through gh from " + got.Repository.String())
	if got.Repository.IsFork {
		cmd.PrintErrln("note: gh reads this fork's own pull requests; ones opened against its parent are not read (gh repo set-default)")
	}
	if len(got.Requests) == 0 {
		cmd.PrintErrln("note: gh reported no pull request updated in the window, so every candidate is in no commit list read")
	}
	onHead := make(map[string]bool, len(head))
	for i := range head {
		onHead[head[i].ID] = true
	}
	var listed []string
	seen := map[string]bool{}
	for i := range got.Listings {
		hash := got.Listings[i].Payload.(event.PullRequestCommit).Commit
		if !onHead[hash] && !seen[hash] {
			seen[hash] = true
			listed = append(listed, hash)
		}
	}
	read, err := vcs.CollectListed(ctx, root, project, listed, since, now, version.Version)
	if err != nil {
		return nil, nil, 0, err
	}
	if !read.EveryWorktree {
		cmd.PrintErrln("note: git could not list this repository's worktrees; only this one's reflog was read")
	}
	if len(got.Requests) > 0 && !touchesClone(&got, onHead, &read) {
		return nil, nil, 0, fmt.Errorf("none of the %d pull request(s) gh read from %s lists or has as its merge commit a commit "+
			"this clone has: gh may have chosen another repository (gh repo set-default), or this clone has not fetched them",
			len(got.Requests), got.Repository)
	}
	commits := append(slices.Clone(head), read.Events...)
	lines, err := rebaseLines(cmd, root, &got, onHead)
	if err != nil {
		return nil, nil, 0, err
	}
	return &attribution.PullRequests{
		Requests: got.Requests, Listings: got.Listings, Lines: lines, MadeHere: len(read.Events),
		NotInReflog: read.NotInReflog, NotLocal: read.NotLocal, Unreadable: read.Unreadable, BeforeWindow: read.BeforeWindow,
		ObservedAt: now, ReadBackTo: got.BackTo,
	}, commits, read.Skipped, nil
}

// touchesClone reports whether any pull request read lists or merged a commit this clone has: when
// none does, the repository gh read is not this clone's history, and every candidate would wrongly
// read as in no pull request.
func touchesClone(got *github.Read, onHead map[string]bool, read *vcs.Listed) bool {
	for i := range got.Requests {
		if onHead[got.Requests[i].Payload.(event.PullRequest).MergeCommit] {
			return true
		}
	}
	return len(read.Events) > 0 || len(read.NotInReflog) > 0 || len(read.Unreadable) > 0 || len(read.BeforeWindow) > 0 ||
		slices.ContainsFunc(got.Listings, func(e event.Event) bool { return onHead[e.Payload.(event.PullRequestCommit).Commit] })
}

// rebaseLines walks back from each merge commit on HEAD as far as its pull request's commit count,
// so the join can find every commit a rebase merge replayed.
func rebaseLines(cmd *cobra.Command, root string, got *github.Read, onHead map[string]bool) (map[string][]string, error) {
	lines := map[string][]string{}
	for i := range got.Requests {
		pr := got.Requests[i].Payload.(event.PullRequest)
		if pr.MergeCommit == "" || !onHead[pr.MergeCommit] || pr.Commits < 2 {
			continue
		}
		line, err := vcs.FirstParents(cmd.Context(), root, pr.MergeCommit, int(pr.Commits))
		if err != nil {
			return nil, errors.Join(fmt.Errorf("walking back from pull request #%d's merge", pr.Number), err)
		}
		lines[pr.MergeCommit] = line
	}
	return lines, nil
}
