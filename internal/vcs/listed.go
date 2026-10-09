package vcs

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/assaio/assaio/internal/event"
)

// Listed is what reading the commits a forge lists for its pull requests found: an observation for
// each one made here within the window, and every other one by why it is not one.
type Listed struct {
	Events []event.Event
	// NotInReflog were in the object store but no HEAD reflog records making them here: fetched or
	// only checked out, the way a teammate's pushed branch arrives, or made in a worktree since
	// removed, or recorded by a reflog entry git has expired.
	NotInReflog []string
	// NotLocal were absent from the object store.
	NotLocal []string
	// Unreadable were recorded as made here but git could not read them.
	Unreadable []string
	// BeforeWindow were made here but committed before the window.
	BeforeWindow []string
	// Skipped had a header this build could not read, as in Collect.
	Skipped int
	// EveryWorktree is false when git could not list the worktrees and only root's reflog was read.
	EveryWorktree bool
}

// ErrLazyFetch refuses a read that could reach the network: a partial clone on a git that ignores
// GIT_NO_LAZY_FETCH would fetch a listed commit it lacks with the user's credentials.
var ErrLazyFetch = errors.New("this partial clone needs git 2.44 or later: an older git would fetch a missing listed commit from the remote")

// CollectListed observes the listed hashes a HEAD reflog of this clone records as made here, in the
// shape Collect gives commits reachable from HEAD, within the same window. The caller passes only
// hashes Collect did not already observe: a listed commit on HEAD's history is a candidate either
// way.
func CollectListed(ctx context.Context, root, project string, hashes []string, since, observedAt time.Time, build string) (Listed, error) {
	got := Listed{EveryWorktree: true}
	if len(hashes) == 0 {
		return got, nil
	}
	unsafe, err := lazyFetchUnsafe(ctx, root)
	switch {
	case err != nil:
		return got, err
	case unsafe:
		return got, ErrLazyFetch
	}
	made, every, err := MadeHere(ctx, root)
	if err != nil {
		return got, err
	}
	got.EveryWorktree = every
	var read, rest []string
	for _, hash := range hashes {
		if made[hash] {
			read = append(read, hash)
		} else {
			rest = append(rest, hash)
		}
	}
	got.NotInReflog, got.NotLocal, err = splitPresent(ctx, root, rest)
	if err != nil {
		return got, err
	}
	got.Events, got.Skipped, got.Unreadable = CollectHashes(ctx, root, project, read, since, observedAt, build)
	got.BeforeWindow = unobserved(read, got.Events, got.Unreadable)
	return got, nil
}

// CollectHashes observes exactly the hashes given, in the shape Collect gives commits reachable
// from HEAD, and returns those git could not read apart.
func CollectHashes(ctx context.Context, root, project string, hashes []string, since, observedAt time.Time, build string) (events []event.Event, skipped int, unreadable []string) {
	if len(hashes) == 0 {
		return nil, 0, nil
	}
	events, skipped, err := readCommits(ctx, root, project, since, observedAt, build, hashes)
	if err != nil {
		return readEach(ctx, root, project, since, observedAt, build, hashes)
	}
	return events, skipped, nil
}

// readEach reads one hash at a time after a batch failed: in a partial clone a commit can be
// present while the blobs its numstat needs are not, and one such commit must not cost the rest.
func readEach(ctx context.Context, root, project string, since, observedAt time.Time, build string, hashes []string) (events []event.Event, skipped int, unreadable []string) {
	for _, hash := range hashes {
		one, s, err := readCommits(ctx, root, project, since, observedAt, build, []string{hash})
		if err != nil {
			unreadable = append(unreadable, hash)
			continue
		}
		events, skipped = append(events, one...), skipped+s
	}
	return events, skipped, unreadable
}

// unobserved is the hashes read that produced no observation and were not unreadable: the window
// left them out, or git printed a header this build could not read.
func unobserved(read []string, events []event.Event, unreadable []string) []string {
	seen := make(map[string]bool, len(events)+len(unreadable))
	for i := range events {
		seen[events[i].ID] = true
	}
	for _, hash := range unreadable {
		seen[hash] = true
	}
	var out []string
	for _, hash := range read {
		if !seen[hash] {
			out = append(out, hash)
		}
	}
	return out
}

// splitPresent parts hashes into those whose commit is in the object store and those that are
// not, without fetching one. cat-file answers every input line in order, "missing" for an absent
// object.
func splitPresent(ctx context.Context, root string, hashes []string) (present, absent []string, err error) {
	if len(hashes) == 0 {
		return nil, nil, nil
	}
	out, err := gitOutputFrom(ctx, root, revisions(hashes), "cat-file", "--batch-check=%(objecttype)")
	if err != nil {
		return nil, nil, err
	}
	answers := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
	for i, hash := range hashes {
		if i < len(answers) && strings.HasSuffix(answers[i], "commit") {
			present = append(present, hash)
		} else {
			absent = append(absent, hash)
		}
	}
	return present, absent, nil
}

// scope turns a walk from HEAD into a read of exactly the listed hashes, handed over on stdin.
func scope(hashes []string, args ...string) []string {
	if len(hashes) == 0 {
		return args
	}
	return append(args, "--no-walk=unsorted", "--stdin")
}

func revisions(hashes []string) io.Reader {
	if len(hashes) == 0 {
		return nil
	}
	return strings.NewReader(strings.Join(hashes, "\n") + "\n")
}
