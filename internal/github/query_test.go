package github

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/event"
)

// allowed is every name the query may contain: the connection it walks and the fields ADR 0022
// lists. A title, body, branch, label, author, reviewer, comment or check added to the query fails
// here before it can reach a payload.
var allowed = []string{
	"query", "String", "repository", "pullRequests", "pageInfo", "hasNextPage", "endCursor", "nodes",
	"id", "number", "state", "updatedAt", "mergedAt", "mergeCommit", "oid", "commits", "totalCount", "commit",
	"reviews", "last", "submittedAt", "statusCheckRollup", "contexts", "__typename", "on", "CheckRun", "StatusContext",
	"status", "conclusion", "startedAt", "completedAt", "createdAt",
}

func TestTheQueryAsksForAllowlistedFieldsOnly(t *testing.T) {
	selection := regexp.MustCompile(`\([^)]*\)`).ReplaceAllString(pullRequestQuery, "")
	for _, name := range regexp.MustCompile(`[A-Za-z_]+`).FindAllString(selection, -1) {
		if !slices.Contains(allowed, name) {
			t.Errorf("the query names %q, which is not on the allowlist", name)
		}
	}
}

// TestARealPageBecomesValidObservations reads a page captured from this repository's own pull
// requests with the query above.
func TestARealPageBecomesValidObservations(t *testing.T) {
	body, err := os.ReadFile("testdata/page.json")
	if err != nil {
		t.Fatal(err)
	}
	p, err := parsePage(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Nodes) != 50 || !p.HasNextPage || p.EndCursor == "" {
		t.Fatalf("page = %d nodes, next %v, cursor %q; want a full first page", len(p.Nodes), p.HasNextPage, p.EndCursor)
	}
	observedAt := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	var merged, listings, headChecks int
	for i := range p.Nodes {
		request, listed := observations(&p.Nodes[i], event.Source{Name: sourceName}, "assaio", observedAt)
		all := append([]event.Event{request}, listed...)
		reviews, checks, err := deliveryObservations(&p.Nodes[i], event.Source{Name: sourceName}, "assaio", observedAt)
		if err != nil {
			t.Fatalf("pull request #%d delivery: %v", p.Nodes[i].Number, err)
		}
		all = append(all, reviews...)
		all = append(all, checks...)
		for j := range all {
			if err := all[j].Validate(); err != nil {
				t.Fatalf("pull request #%d: %v", p.Nodes[i].Number, err)
			}
		}
		if request.Payload.(event.PullRequest).State == event.ChangeMerged {
			merged++
		}
		listings += len(listed)
		headChecks += len(checks)
	}
	if merged == 0 || listings < len(p.Nodes) || headChecks < len(p.Nodes) {
		t.Fatalf("%d merged, %d listed commits, %d head checks: the capture should hold merged PRs, commits and checks", merged, listings, headChecks)
	}
}

func TestAPageTheForgeFaultedIsNoPage(t *testing.T) {
	for name, body := range map[string]string{
		"an error beside data":    `{"data":{"repository":{"pullRequests":{"nodes":[]}}},"errors":[{"message":"timeout"}]}`,
		"no such repository":      `{"data":{"repository":null}}`,
		"no data":                 `{}`,
		"not json":                `<html>`,
		"cut off by the size cap": `{"data":{"repository":{"pullRequests":{"nodes":[{"id":"PR_1"`,
	} {
		if _, err := parsePage([]byte(body)); err == nil {
			t.Errorf("%s: parsed without an error", name)
		}
	}
}

func FuzzParse(f *testing.F) {
	body, err := os.ReadFile("testdata/page.json")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(body)
	f.Add([]byte(`{"errors":[{"message":"x"}]}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		p, err := parsePage(b)
		if err != nil {
			return
		}
		for i := range p.Nodes {
			request, listed := observations(&p.Nodes[i], event.Source{Name: sourceName}, "p", time.Unix(1, 0))
			if !nodeID(p.Nodes[i].ID) {
				continue
			}
			all := append([]event.Event{request}, listed...)
			reviews, checks, err := deliveryObservations(&p.Nodes[i], event.Source{Name: sourceName}, "p", time.Unix(1, 0))
			if err != nil {
				continue
			}
			all = append(all, reviews...)
			all = append(all, checks...)
			for j := range all {
				if all[j].Validate() == nil && strings.ContainsAny(all[j].ID, " \n\t/") {
					t.Fatalf("an id with free text passed: %q", all[j].ID)
				}
			}
		}
	})
}
