package github

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/event"
)

func historyPage(t *testing.T, fields string) string {
	t.Helper()
	return `{"data":{"repository":{"pullRequests":{"totalCount":9,"pageInfo":{"hasNextPage":false},"nodes":[{"id":"PR_1","number":1,"updatedAt":"2026-09-30T11:00:00Z","history":{"totalCount":1,"nodes":[{"commit":{"oid":"` + oid + `",` + fields + `}}]}}]}}}}`
}

func TestHistoryKeepsEveryNullableConnectionAndItsBound(t *testing.T) {
	for _, tt := range []struct {
		name, fields              string
		wantSuites                *event.Population
		wantSuiteEvents, wantRuns int
		wantRunCoverage           *event.Population
	}{
		{"null suites", `"checkSuites":null`, nil, 0, 0, nil},
		{"missing suite total", `"checkSuites":{"nodes":[]}`, nil, 0, 0, nil},
		{"null suite nodes", `"checkSuites":{"totalCount":0,"nodes":null}`, nil, 0, 0, nil},
		{"stated empty suites", `"checkSuites":{"totalCount":0,"nodes":[]}`, &event.Population{}, 0, 0, nil},
		{"null runs", `"checkSuites":{"totalCount":11,"nodes":[{"id":"CS_1","status":"COMPLETED","conclusion":"FAILURE","checkRuns":null}]}`, &event.Population{Total: 11, Listed: 1}, 1, 0, nil},
		{"stated empty runs", `"checkSuites":{"totalCount":1,"nodes":[{"id":"CS_1","status":"COMPLETED","checkRuns":{"totalCount":0,"nodes":[]}}]}`, &event.Population{Total: 1, Listed: 1}, 1, 0, &event.Population{}},
		{"all-run truncation", `"checkSuites":{"totalCount":1,"nodes":[{"id":"CS_1","status":"COMPLETED","conclusion":"FAILURE","checkRuns":{"totalCount":6,"nodes":[{"id":"CR_1","status":"COMPLETED","conclusion":"FAILURE","startedAt":"2026-09-30T10:00:00Z"}]}}]}`, &event.Population{Total: 1, Listed: 1}, 1, 1, &event.Population{Total: 6, Listed: 1}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parsePage([]byte(historyPage(t, tt.fields)))
			if err != nil {
				t.Fatal(err)
			}
			nd := &p.Nodes[0]
			pop := suitesOf(nd, oid)
			if !samePopulation(pop, tt.wantSuites) {
				t.Fatalf("suite coverage = %+v, want %+v", pop, tt.wantSuites)
			}
			suites, runs, err := historyObservations(nd, event.Source{Name: "gh"}, "p", observedAt)
			if err != nil || len(suites) != tt.wantSuiteEvents || len(runs) != tt.wantRuns {
				t.Fatalf("history = %d suites, %d runs, %v", len(suites), len(runs), err)
			}
			if len(suites) > 0 && !samePopulation(suites[0].Payload.(event.CheckSuite).Runs, tt.wantRunCoverage) {
				t.Fatalf("run coverage = %+v", suites[0].Payload)
			}
			for _, e := range append(suites, runs...) {
				if err := e.Validate(); err != nil {
					t.Fatal(err)
				}
			}
			if len(runs) > 0 && !runs[0].Payload.(event.Check).CompletedAt.IsZero() {
				t.Fatal("a fallback start must not fabricate a completion")
			}
		})
	}
}

func samePopulation(a, b *event.Population) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

func TestHistoryCannotAttachAcrossAChangedCommitPopulation(t *testing.T) {
	basePage := pageOf(false, pr("PR_1", 1, observedAt, oid))
	for _, tt := range []struct {
		name, history, state string
	}{
		{"identical", historyPage(t, `"checkSuites":{"totalCount":0,"nodes":[]}`), "read"},
		{"changed hash", strings.ReplaceAll(historyPage(t, `"checkSuites":null`), oid, strings.Repeat("cd", 20)), "commit-list-changed"},
		{"changed total", strings.Replace(historyPage(t, `"checkSuites":null`), `"totalCount":1`, `"totalCount":2`, 1), "commit-list-changed"},
		{"outside independent read", pageOf(false), "outside-history-read"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var baseCalls, historyCalls int
			run := func(_ context.Context, _ string, args ...string) ([]byte, error) {
				if args[0] == "repo" {
					return []byte(view), nil
				}
				if slices.Contains(args, "query="+historyQuery) {
					historyCalls++
					return []byte(tt.history), nil
				}
				baseCalls++
				return []byte(basePage), nil
			}
			got, err := ReadPullRequests(context.Background(), run, "/repo", "p", since, observedAt, "test")
			if err != nil {
				t.Fatal(err)
			}
			if got.Requests[0].Payload.(event.PullRequest).HistoryState != tt.state || baseCalls != 2 || historyCalls < 1 || historyCalls > 2 {
				t.Fatalf("history state/calls = %+v/%d/%d", got.Requests[0].Payload, baseCalls, historyCalls)
			}
		})
	}
}

func TestHistoryErrorFailsTheWholeRead(t *testing.T) {
	run := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if args[0] == "repo" {
			return []byte(view), nil
		}
		if slices.Contains(args, "query="+historyQuery) {
			return []byte(`{"data":{"repository":{}},"errors":[{"message":"Resource limits exceeded"}]}`), nil
		}
		return []byte(pageOf(false, pr("PR_1", 1, observedAt, oid))), nil
	}
	got, err := ReadPullRequests(context.Background(), run, "/repo", "p", since, observedAt, "test")
	if err == nil || !strings.Contains(err.Error(), "Resource limits") || len(got.Requests) != 0 {
		t.Fatalf("partial document = %+v, error = %v", got, err)
	}
}

func TestHistoryQueryRequestsAllRunsWithinFixedBounds(t *testing.T) {
	for _, want := range []string{"pullRequests(first: 5", "commits(first: 100", "checkSuites(last: 10", "checkRuns(last: 5, filterBy: {checkType: ALL})"} {
		if !strings.Contains(historyQuery, want) {
			t.Fatalf("history query lacks %q", want)
		}
	}
}

func FuzzHistory(f *testing.F) {
	f.Add([]byte(`{"data":{"repository":{"pullRequests":{"nodes":[]}}}}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var r node
		if err := json.Unmarshal(raw, &r); err != nil {
			return
		}
		_, _, _ = historyObservations(&r, event.Source{Name: "gh"}, "p", observedAt)
	})
}
