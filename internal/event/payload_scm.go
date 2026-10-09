package event

import (
	"errors"
	"fmt"
	"time"
)

// The states a forge reports a pull request in.
const (
	ChangeOpen   = "open"
	ChangeClosed = "closed"
	ChangeMerged = "merged"
)

var changeStates = []string{ChangeOpen, ChangeClosed, ChangeMerged}

// PullRequest is a forge's pull request as one read found it: its number, its state, the commit
// its merge made and how many commits it lists. No title, body, branch, label, author or reviewer:
// the query that fills it cannot ask for them. A pull request keeps changing after it is read, so
// the observation is a snapshot, and of two with one ID the later ObservedAt is current.
type PullRequest struct {
	Number   int64     `json:"number"`
	State    string    `json:"state"`
	MergedAt time.Time `json:"mergedAt"`
	// MergeCommit is the commit the forge wrote when it merged: the squash, the merge commit, or
	// the last commit a rebase merge replayed. Empty until the pull request is merged.
	MergeCommit string `json:"mergeCommit,omitempty"`
	// Commits is how many commits the forge counts in the pull request and Listed how many of them
	// the read returned; Listed below Commits means the list was cut.
	Commits int64 `json:"commits"`
	Listed  int64 `json:"listed"`
	// Reviews and Checks count the source connections; listed counts are separate so a bounded
	// read never presents an incomplete connection as a complete one. A nil check rollup has
	// no stated check population.
	Reviews          int64  `json:"reviews"`
	ReviewsListed    int64  `json:"reviewsListed"`
	ReviewsAvailable bool   `json:"reviewsAvailable"`
	Checks           int64  `json:"checks"`
	ChecksListed     int64  `json:"checksListed"`
	CheckState       string `json:"checkState,omitempty"`
}

// PullRequestCommit says that a pull request lists a commit. It is an observation of its own
// because a payload holds no list, and one commit can be listed by several pull requests.
type PullRequestCommit struct {
	Number int64  `json:"number"`
	Commit string `json:"commit"`
}

func (PullRequest) eventType() string       { return TypePullRequest }
func (PullRequestCommit) eventType() string { return TypePullRequestCommit }

// validate keeps a merge's traces off a pull request that is not merged: a merge commit or a merge
// time on an open or closed one is a read that went wrong, not a state.
//
//nolint:gocritic // the Payload interface is satisfied by values; validated once per pull request.
func (p PullRequest) validate() error {
	if p.Number <= 0 {
		return fmt.Errorf("pull request number %d is not positive", p.Number)
	}
	if !valid(changeStates, p.State) {
		return fmt.Errorf("unknown pull request state %q (want %v)", p.State, changeStates)
	}
	if err := nonNegative(map[string]int64{
		"commits": p.Commits, "listed": p.Listed,
		"reviews": p.Reviews, "reviewsListed": p.ReviewsListed,
		"checks": p.Checks, "checksListed": p.ChecksListed,
	}); err != nil {
		return err
	}
	if p.Listed > p.Commits {
		return fmt.Errorf("lists %d commit(s) of %d", p.Listed, p.Commits)
	}
	if p.ReviewsListed > p.Reviews {
		return fmt.Errorf("lists %d review(s) of %d", p.ReviewsListed, p.Reviews)
	}
	if !p.ReviewsAvailable && (p.Reviews != 0 || p.ReviewsListed != 0) {
		return errors.New("reviews have no available connection")
	}
	if p.ChecksListed > p.Checks {
		return fmt.Errorf("lists %d check(s) of %d", p.ChecksListed, p.Checks)
	}
	if p.CheckState != "" && !valid(checkRollupStates, p.CheckState) {
		return fmt.Errorf("unknown check rollup state %q", p.CheckState)
	}
	if p.CheckState == "" && (p.Checks != 0 || p.ChecksListed != 0) {
		return errors.New("checks have no rollup state")
	}
	if p.State != ChangeMerged && (p.MergeCommit != "" || !p.MergedAt.IsZero()) {
		return fmt.Errorf("a %s pull request carries a merge", p.State)
	}
	if p.MergeCommit != "" && !isObjectID(p.MergeCommit) {
		return errors.New("merge commit is not a commit hash")
	}
	return nil
}

func (p PullRequestCommit) validate() error {
	if p.Number <= 0 {
		return fmt.Errorf("pull request number %d is not positive", p.Number)
	}
	if !isObjectID(p.Commit) {
		return errors.New("listed commit is not a commit hash")
	}
	return nil
}

// isObjectID accepts a git object name in full: 40 lowercase hex digits under SHA-1, 64 under
// SHA-256. Anything else could be a name or a message.
func isObjectID(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}
