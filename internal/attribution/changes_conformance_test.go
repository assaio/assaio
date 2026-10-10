package attribution

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/event"
	"github.com/assaio/assaio/internal/github"
)

func TestCapturedReviewsAndMergeTopologySupportOnlyTheirDeclaredCounts(t *testing.T) {
	page, err := os.ReadFile("../github/testdata/cli-base.json")
	if err != nil {
		t.Fatal(err)
	}
	baseCalls := 0
	run := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if args[0] == "repo" {
			return []byte(`{"nameWithOwner":"cli/cli","url":"https://github.com/cli/cli","isFork":false}`), nil
		}
		for _, arg := range args {
			if strings.Contains(arg, "history: commits(first: 100)") {
				return []byte(`{"data":{"repository":{"pullRequests":{"nodes":[],"pageInfo":{"hasNextPage":false}}}}}`), nil
			}
		}
		baseCalls++
		// The runner replays one captured page and its reread; the uncaptured tail is empty.
		if baseCalls == 2 {
			return []byte(`{"data":{"repository":{"pullRequests":{"nodes":[],"pageInfo":{"hasNextPage":false}}}}}`), nil
		}
		return page, nil
	}
	at := time.Date(2026, 10, 9, 22, 19, 0, 0, time.UTC)
	got, err := github.ReadPullRequests(context.Background(), run, "/fixture", "p", epoch, at, "test")
	if err != nil {
		t.Fatal(err)
	}
	prs := &PullRequests{
		Requests: got.Requests, Reviews: got.Reviews, Checks: got.Checks,
		Listings: got.Listings, ObservedAt: at, ReadBackTo: got.BackTo,
	}
	var numbers []int64
	for _, e := range got.Requests {
		numbers = append(numbers, e.Payload.(event.PullRequest).Number)
	}
	c := ChangesOf(prs, namesPR(numbers...), "gh")
	var requested, merges, unknownMethods int
	for _, pr := range c.Linked {
		if pr.RequestedChangesRevisions.Value != nil && *pr.RequestedChangesRevisions.Value > 0 {
			requested++
		}
		if pr.MergeMethod.Value != nil && *pr.MergeMethod.Value == "merge" {
			merges++
		}
		if pr.MergeMethod.Reason == "squash-versus-rebase-undetermined" {
			unknownMethods++
		}
		if pr.ReviewRounds.Value != nil || pr.ReviewRounds.Reason == "" {
			t.Fatal("captured review states cannot establish rounds")
		}
	}
	if requested == 0 || merges != 11 || unknownMethods != 3 {
		t.Fatalf("capture supports %d requested-revision PRs, %d merges and %d unknown single-parent methods", requested, merges, unknownMethods)
	}
	if c.Rates[0].Denominator != 14 || c.Rates[0].Eligible != 14 || c.Rates[0].Value == nil ||
		c.Rates[0].Numerator == nil || *c.Rates[0].Numerator != 2 || *c.Rates[0].Value != float64(2)/14 || c.Rates[0].Reason != "" ||
		c.Rates[1].Value != nil || c.Rates[1].Reason == "" {
		t.Fatalf("a source gap was converted into a rate: %+v", c.Rates)
	}
}
