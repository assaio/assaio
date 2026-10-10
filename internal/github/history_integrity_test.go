package github

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/event"
)

func TestHistoryIDsAreUniqueAcrossThePullRequest(t *testing.T) {
	for _, tt := range []struct {
		name, commits, want string
	}{
		{
			"suite across commits",
			`[{"commit":{"oid":"` + oid + `","checkSuites":{"totalCount":1,"nodes":[{"id":"CS_1","status":"QUEUED"}]}}},{"commit":{"oid":"` + strings.Repeat("cd", 20) + `","checkSuites":{"totalCount":1,"nodes":[{"id":"CS_1","status":"QUEUED"}]}}}]`,
			"duplicate suite id",
		},
		{
			"run across suites",
			`[{"commit":{"oid":"` + oid + `","checkSuites":{"totalCount":2,"nodes":[{"id":"CS_1","status":"QUEUED","checkRuns":{"totalCount":1,"nodes":[{"id":"CR_1","status":"QUEUED"}]}},{"id":"CS_2","status":"QUEUED","checkRuns":{"totalCount":1,"nodes":[{"id":"CR_1","status":"QUEUED"}]}}]}}}]`,
			"duplicate check run id",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parsePage([]byte(`{"data":{"repository":{"pullRequests":{"pageInfo":{"hasNextPage":false},"nodes":[{"id":"PR_1","number":1,"history":{"totalCount":2,"nodes":` + tt.commits + `}}]}}}}`))
			if err != nil {
				t.Fatal(err)
			}
			w := walk{nodes: map[string]node{"PR_1": p.Nodes[0]}}
			if err := validateHistory(&w, "p", observedAt); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("duplicate accepted: %v", err)
			}
		})
	}
}

func TestSuiteOccurrenceDoesNotReplaceItsSourceTimes(t *testing.T) {
	created := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
	updated := created.Add(time.Minute)
	for _, tt := range []struct {
		name, fields                 string
		created, updated, occurrence time.Time
		source                       string
	}{
		{"both", `,"createdAt":"2026-09-30T10:00:00Z","updatedAt":"2026-09-30T10:01:00Z"`, created, updated, updated, event.TimeStated},
		{"creation only", `,"createdAt":"2026-09-30T10:00:00Z"`, created, time.Time{}, created, event.TimeStated},
		{"unavailable", "", time.Time{}, time.Time{}, observedAt, event.TimeIngestTime},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parsePage([]byte(historyPage(t, `"checkSuites":{"totalCount":1,"nodes":[{"id":"CS_1","status":"QUEUED"`+tt.fields+`}]}`)))
			if err != nil {
				t.Fatal(err)
			}
			suites, _, err := historyObservations(&p.Nodes[0], event.Source{Name: sourceName}, "p", observedAt)
			if err != nil || len(suites) != 1 {
				t.Fatalf("suite = %v, %v", suites, err)
			}
			s := suites[0].Payload.(event.CheckSuite)
			if !s.CreatedAt.Equal(tt.created) || !s.UpdatedAt.Equal(tt.updated) || !suites[0].OccurredAt.Equal(tt.occurrence) || suites[0].TimeSource != tt.source {
				t.Fatalf("source times or fallback changed: %+v", suites[0])
			}
		})
	}
}
