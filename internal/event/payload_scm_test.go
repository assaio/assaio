package event

import (
	"strings"
	"testing"
	"time"
)

var mergeCommit = strings.Repeat("ab", 20)

func TestPullRequestValidate(t *testing.T) {
	merged := time.Date(2026, 9, 30, 7, 41, 0, 0, time.UTC)
	tests := []struct {
		name    string
		pr      PullRequest
		wantErr string
	}{
		{"an open pull request", PullRequest{Number: 98, State: ChangeOpen, Commits: 2, Listed: 2}, ""},
		{"a merged one names its merge", PullRequest{
			Number: 98, State: ChangeMerged, MergedAt: merged,
			MergeCommit: mergeCommit, Commits: 1, Listed: 1,
		}, ""},
		{"a merge the read could not name is still merged", PullRequest{
			Number: 98, State: ChangeMerged,
			MergedAt: merged, Commits: 1, Listed: 1,
		}, ""},
		{"a list cut short says so", PullRequest{Number: 98, State: ChangeOpen, Commits: 250, Listed: 100}, ""},
		{"a SHA-256 repository's hash", PullRequest{
			Number: 1, State: ChangeMerged,
			MergeCommit: strings.Repeat("0f", 32),
		}, ""},
		{"no number", PullRequest{State: ChangeOpen}, "not positive"},
		{"a state the forge does not report", PullRequest{Number: 1, State: "draft"}, "unknown pull request state"},
		{"more listed than counted", PullRequest{Number: 1, State: ChangeOpen, Commits: 1, Listed: 2}, "lists 2 commit(s) of 1"},
		{"negative count", PullRequest{Number: 1, State: ChangeOpen, Commits: -1}, "commits is negative"},
		{"a closed one with a merge commit", PullRequest{Number: 1, State: ChangeClosed, MergeCommit: mergeCommit}, "carries a merge"},
		{"an open one with a merge time", PullRequest{Number: 1, State: ChangeOpen, MergedAt: merged}, "carries a merge"},
		{"a merge commit that is text", PullRequest{Number: 1, State: ChangeMerged, MergeCommit: "fix the login bug"}, "not a commit hash"},
		{"an abbreviated hash", PullRequest{Number: 1, State: ChangeMerged, MergeCommit: "876c1ff"}, "not a commit hash"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertErr(t, tc.pr.validate(), tc.wantErr)
		})
	}
}

func TestPullRequestCommitValidate(t *testing.T) {
	tests := []struct {
		name    string
		listed  PullRequestCommit
		wantErr string
	}{
		{"a listed commit", PullRequestCommit{Number: 98, Commit: mergeCommit}, ""},
		{"no number", PullRequestCommit{Commit: mergeCommit}, "not positive"},
		{"upper-case hex is not what git prints", PullRequestCommit{Number: 1, Commit: strings.ToUpper(mergeCommit)}, "not a commit hash"},
		{"a branch name", PullRequestCommit{Number: 1, Commit: "feature/login"}, "not a commit hash"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertErr(t, tc.listed.validate(), tc.wantErr)
		})
	}
}

// A pull request is not a commit: counting both in one unit would let a consumer average a change
// with a commit. The membership is about one commit, so it keeps the commit grain.
func TestPullRequestObservationsCarryTheirGrains(t *testing.T) {
	for _, tc := range []struct {
		typ, grain string
		payload    Payload
	}{
		{TypePullRequest, GrainChange, PullRequest{Number: 98, State: ChangeOpen}},
		{TypePullRequestCommit, GrainCommit, PullRequestCommit{Number: 98, Commit: mergeCommit}},
	} {
		e := validEvent()
		e.Type, e.ID, e.Grain, e.Privacy, e.Payload = tc.typ, "PR_node", tc.grain, LocalOnly, tc.payload
		e.Source = Source{Name: "gh", Build: "test"}
		if err := e.Validate(); err != nil {
			t.Errorf("%s observation rejected: %v", tc.typ, err)
		}
	}
}
