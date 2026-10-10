package github

import (
	"os"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/event"
)

func capturedPage(t *testing.T, name string) page {
	t.Helper()
	//nolint:gosec // only the fixed capture names in this test reach this helper
	body, err := os.ReadFile("testdata/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	p, err := parsePage(body)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCurrentGitHubCapturesConformToTheObservationContract(t *testing.T) {
	for _, tt := range []struct {
		file                     string
		wantRequested, wantMerge bool
	}{
		{"assaio-base", false, false},
		{"cli-base", true, true},
	} {
		t.Run(tt.file, func(t *testing.T) {
			p := capturedPage(t, tt.file)
			if len(p.Nodes) != 50 || p.TotalCount == nil {
				t.Fatal("capture must retain both the repository and read populations")
			}
			requested, merged := false, false
			for i := range p.Nodes {
				n := &p.Nodes[i]
				pr, listed := observations(n, event.Source{Name: sourceName}, "p", observedAt)
				reviews, checks, err := deliveryObservations(n, event.Source{Name: sourceName}, "p", observedAt)
				if err != nil {
					t.Fatal(err)
				}
				all := append([]event.Event{pr}, listed...)
				all = append(all, reviews...)
				all = append(all, checks...)
				for j := range all {
					if err := all[j].Validate(); err != nil {
						t.Fatalf("PR %d: %v", n.Number, err)
					}
				}
				for j := range reviews {
					r := reviews[j].Payload.(event.Review)
					if r.State == "changes_requested" {
						if r.Commit == "" || r.SubmittedAt.IsZero() {
							t.Fatal("the positive capture must state reviewed hash and actual submission time")
						}
						requested = true
					}
				}
				payload := pr.Payload.(event.PullRequest)
				merged = merged || payload.State == event.ChangeMerged && payload.MergeParents != nil && *payload.MergeParents > 1
			}
			if requested != tt.wantRequested || merged != tt.wantMerge {
				t.Fatalf("positive source cases requested/merge = %v/%v", requested, merged)
			}
		})
	}
}

func TestHistoricalCaptureRetainsSuitesRunsAndFailure(t *testing.T) {
	p := capturedPage(t, "assaio-history")
	w := walk{nodes: map[string]node{}}
	for i := range p.Nodes {
		n := &p.Nodes[i]
		w.nodes[n.ID] = *n
	}
	at := time.Date(2026, 10, 9, 22, 5, 0, 0, time.UTC)
	if err := validateHistory(&w, "p", at); err != nil {
		t.Fatal(err)
	}
	var commits, suites, runs, failures int
	for i := range p.Nodes {
		n := &p.Nodes[i]
		commits += len(n.History.Nodes)
		s, r, err := historyObservations(n, event.Source{Name: sourceName}, "p", at)
		if err != nil {
			t.Fatal(err)
		}
		suites += len(s)
		runs += len(r)
		for _, c := range n.History.Nodes {
			for _, raw := range c.Commit.CheckSuites.Nodes {
				found := false
				for j := range s {
					if s[j].ID == n.ID+":"+raw.ID {
						payload := s[j].Payload.(event.CheckSuite)
						if !payload.CreatedAt.Equal(raw.CreatedAt) || !payload.UpdatedAt.Equal(raw.UpdatedAt) || payload.CreatedAt.IsZero() || payload.UpdatedAt.IsZero() {
							t.Fatal("the real suite lost a source timestamp")
						}
						found = true
					}
				}
				if !found {
					t.Fatal("the real suite has no observation")
				}
			}
		}
		for i := range r {
			if r[i].Payload.(event.Check).Conclusion == "failure" {
				failures++
			}
		}
	}
	if len(p.Nodes) != 5 || commits != 7 || suites != 49 || runs != 112 || failures != 1 {
		t.Fatalf("capture = %d PRs, %d commits, %d suites, %d runs, %d failures", len(p.Nodes), commits, suites, runs, failures)
	}
}

func TestARealResourceLimitedResponseIsNotAHistory(t *testing.T) {
	body, err := os.ReadFile("testdata/resource-limit.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parsePage(body); err == nil {
		t.Fatal("a failed real response must not become an empty history")
	}
}
