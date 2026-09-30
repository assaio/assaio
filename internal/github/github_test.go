package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/event"
)

var (
	since      = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	observedAt = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
)

// forge is a fake gh: it answers repo view, then serves pages of pull requests from pages, one per
// call in order, recording every call.
type forge struct {
	view  string
	pages []string
	calls [][]string
	fail  int // the page call that fails, counting from 1; 0 for none
}

func (f *forge) run(_ context.Context, _ string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, args)
	if args[0] == "repo" {
		return []byte(f.view), nil
	}
	n := len(f.calls) - 1
	if n == f.fail {
		return []byte(`{"errors":[{"message":"API rate limit exceeded"}]}`), errors.New("exit status 1")
	}
	if n > len(f.pages) {
		return nil, fmt.Errorf("page %d was never meant to be read", n)
	}
	return []byte(f.pages[n-1]), nil
}

const view = `{"nameWithOwner":"acme/api","url":"https://github.com/acme/api","isFork":false}`

func pr(id string, number int, updated time.Time, oids ...string) map[string]any {
	commits := make([]map[string]any, 0, len(oids))
	for _, oid := range oids {
		commits = append(commits, map[string]any{"commit": map[string]any{"oid": oid}})
	}
	return map[string]any{
		"id": id, "number": number, "state": "OPEN", "updatedAt": updated, "mergedAt": nil, "mergeCommit": nil,
		"commits": map[string]any{"totalCount": len(oids), "nodes": commits},
	}
}

func pageOf(next bool, nodes ...map[string]any) string {
	b, _ := json.Marshal(map[string]any{"data": map[string]any{"repository": map[string]any{"pullRequests": map[string]any{
		"pageInfo": map[string]any{"hasNextPage": next, "endCursor": fmt.Sprintf("cursor-%d", len(nodes))},
		"nodes":    nodes,
	}}}})
	return string(b)
}

var oid = strings.Repeat("ab", 20)

func numbers(events []event.Event) []int64 {
	var out []int64
	for i := range events {
		out = append(out, events[i].Payload.(event.PullRequest).Number)
	}
	slices.Sort(out)
	return out
}

func TestTheReadStopsAtTheWindowAndRereadsTheTop(t *testing.T) {
	day := 24 * time.Hour
	f := &forge{view: view, pages: []string{
		pageOf(true, pr("PR_3", 3, observedAt.Add(-day), oid), pr("PR_2", 2, observedAt.Add(-2*day))),
		pageOf(true, pr("PR_1", 1, since.Add(time.Hour)), pr("PR_0", 0, since.Add(-time.Hour))),
		// The re-read: #1 moved to the top while the walk was on page 2. The local clock is behind
		// the forge's, so only the walk's own newest update can bound the re-read.
		pageOf(true, pr("PR_1", 1, observedAt.Add(-day+time.Minute), oid), pr("PR_3", 3, observedAt.Add(-day), oid)),
	}}
	got, err := ReadPullRequests(context.Background(), f.run, "/repo", "api", since, observedAt, "test")
	if err != nil {
		t.Fatal(err)
	}
	if n := numbers(got.Requests); !slices.Equal(n, []int64{1, 2, 3}) {
		t.Fatalf("read pull requests %v, want 1-3 and not #0, last updated before the window", n)
	}
	if len(f.calls) != 4 {
		t.Fatalf("%d gh calls, want repo view, two pages and one re-read", len(f.calls))
	}
	for i := range got.Requests {
		if got.Requests[i].ID == "PR_1" && !got.Requests[i].OccurredAt.Equal(observedAt.Add(-day+time.Minute)) {
			t.Errorf("#1 kept its first copy (%s), want the re-read's", got.Requests[i].OccurredAt)
		}
	}
	if len(got.Listings) != 2 || !got.BackTo.Equal(since) || got.Repository.String() != "github.com/acme/api" {
		t.Fatalf("listings %d, back to %s, repository %s", len(got.Listings), got.BackTo, got.Repository)
	}
}

func TestTheReadAsksWithRawStringsOnTheResolvedHost(t *testing.T) {
	f := &forge{
		view:  `{"nameWithOwner":"2048/1234","url":"https://ghe.example.test/2048/1234","isFork":true}`,
		pages: []string{pageOf(false), pageOf(false)},
	}
	got, err := ReadPullRequests(context.Background(), f.run, "/repo", "api", since, observedAt, "test")
	if err != nil {
		t.Fatal(err)
	}
	call := strings.Join(f.calls[1], " ")
	for _, want := range []string{"--hostname ghe.example.test", "-f owner=2048", "-f name=1234"} {
		if !strings.Contains(call, want) {
			t.Errorf("call %q lacks %q", call, want)
		}
	}
	if strings.Contains(call, "-F ") || strings.Contains(call, "cursor=") {
		t.Errorf("call %q types a value or sends a cursor on the first page", call)
	}
	if !got.Repository.IsFork {
		t.Error("a fork was not reported as one")
	}
}

func TestAPageBoundSaysHowFarBackTheReadGot(t *testing.T) {
	var pages []string
	for n := range maxPages {
		pages = append(pages, pageOf(true, pr(fmt.Sprintf("PR_%d", n), n+1, observedAt.Add(-time.Duration(n+1)*time.Hour))))
	}
	pages = append(pages, pageOf(false, pr("PR_0", 1, observedAt.Add(-time.Hour))))
	f := &forge{view: view, pages: pages}
	got, err := ReadPullRequests(context.Background(), f.run, "/repo", "api", since, observedAt, "test")
	if err != nil {
		t.Fatal(err)
	}
	if want := observedAt.Add(-maxPages * time.Hour); !got.BackTo.Equal(want) {
		t.Fatalf("back to %s, want the oldest update read, %s", got.BackTo, want)
	}
}

// TestGhsOwnReasonSurvivesABodyThatIsNotAnAnswer: a rate-limit or gateway page is not a GraphQL
// answer, and "no such repository" would misname what failed.
func TestGhsOwnReasonSurvivesABodyThatIsNotAnAnswer(t *testing.T) {
	run := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if args[0] == "repo" {
			return []byte(view), nil
		}
		return []byte(`{"message":"You have exceeded a secondary rate limit"}`), errors.New("gh api: exit status 1: HTTP 403")
	}
	_, err := ReadPullRequests(context.Background(), run, "/repo", "api", since, observedAt, "test")
	if err == nil || !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("err = %v, want gh's own reason", err)
	}
}

func TestAFailedPageFailsTheRead(t *testing.T) {
	f := &forge{view: view, fail: 2, pages: []string{pageOf(true, pr("PR_1", 1, observedAt.Add(-time.Hour)))}}
	_, err := ReadPullRequests(context.Background(), f.run, "/repo", "api", since, observedAt, "test")
	if err == nil || !strings.Contains(err.Error(), "rate limit") {
		t.Fatalf("err = %v, want the forge's own reason", err)
	}
}

func TestAPullRequestWithoutAUsableIDFailsTheRead(t *testing.T) {
	f := &forge{view: view, pages: []string{pageOf(false, pr("PR 1\nnot an id", 1, observedAt.Add(-time.Hour))), pageOf(false)}}
	if _, err := ReadPullRequests(context.Background(), f.run, "/repo", "api", since, observedAt, "test"); err == nil {
		t.Fatal("a pull request with a free-text id was read")
	}
}

func TestResolveRejectsAnAnswerThatNamesNoRepository(t *testing.T) {
	for _, body := range []string{`{}`, `{"nameWithOwner":"acme","url":"https://github.com/acme"}`, `nope`} {
		f := &forge{view: body}
		if _, err := Resolve(context.Background(), f.run, "/repo"); err == nil {
			t.Errorf("Resolve accepted %s", body)
		}
	}
}
