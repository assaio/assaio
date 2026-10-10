package github

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/event"
)

func dismissalSource(id string) map[string]any {
	return map[string]any{
		"__typename": "ReviewDismissedEvent", "id": id,
		"createdAt": observedAt.Add(-time.Hour), "previousReviewState": "APPROVED",
		"review": map[string]any{"id": "PRR_1", "submittedAt": observedAt.Add(-2 * time.Hour), "commit": map[string]any{"oid": oid}},
	}
}

func dismissalSourceConnection(total, filtered int, previous bool, nodes ...map[string]any) map[string]any {
	if nodes == nil {
		nodes = []map[string]any{}
	}
	return map[string]any{
		"totalCount": total, "filteredCount": filtered, "pageCount": len(nodes),
		"pageInfo": map[string]any{"hasPreviousPage": previous}, "nodes": nodes,
	}
}

func decodeDismissalNode(t *testing.T, connection any) node {
	t.Helper()
	p := pr("PR_1", 1, observedAt.Add(-time.Minute), oid)
	p["reviewDismissals"] = connection
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var n node
	if err := json.Unmarshal(b, &n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestDismissalsPreserveActualTransitionAndNullableReview(t *testing.T) {
	for _, state := range []string{"APPROVED", "CHANGES_REQUESTED", "PENDING"} {
		for _, joined := range []bool{true, false} {
			raw := dismissalSource("RDE_1")
			raw["previousReviewState"] = state
			if !joined {
				raw["review"] = nil
			}
			n := decodeDismissalNode(t, dismissalSourceConnection(9, 1, false, raw))
			population, events, err := dismissalObservations(&n, event.Source{Name: sourceName, Build: "test"}, "p", observedAt)
			if err != nil || population == nil || population.TotalCount != 9 || population.FilteredCount != 1 || population.PageCount != 1 || population.Listed != 1 || len(events) != 1 {
				t.Fatalf("state %s joined %v: %+v, %+v, %v", state, joined, population, events, err)
			}
			e := events[0]
			p := e.Payload.(event.ReviewDismissal)
			if e.ID != "RDE_1" || e.Type != event.TypeReviewDismissal || e.Grain != event.GrainReview || e.TimeSource != event.TimeStated || e.Privacy != event.LocalOnly || e.Provenance != event.Parsed ||
				e.Source.Name != sourceName || e.Source.Build != "test" || e.Subject.Project != "p" || !e.ObservedAt.Equal(observedAt) || !e.OccurredAt.Equal(observedAt.Add(-time.Hour)) || !p.DismissedAt.Equal(e.OccurredAt) || !strings.EqualFold(p.PreviousState, state) {
				t.Fatalf("transition lost source metadata: %+v / %+v", e, p)
			}
			if joined {
				if p.ReviewID != "PRR_1" || p.Commit != oid || !p.SubmittedAt.Equal(observedAt.Add(-2*time.Hour)) {
					t.Fatalf("joined metadata = %+v", p)
				}
			} else if p.ReviewID != "" || p.Commit != "" || !p.SubmittedAt.IsZero() {
				t.Fatalf("null review invented metadata: %+v", p)
			}
		}
	}
}

func TestMalformedDismissalFailsTheWholeDocument(t *testing.T) {
	for _, tt := range []struct {
		name string
		edit func(map[string]any)
	}{
		{"missing id", func(r map[string]any) { delete(r, "id") }},
		{"text id", func(r map[string]any) { r["id"] = "body text" }},
		{"oversized id", func(r map[string]any) { r["id"] = strings.Repeat("a", 129) }},
		{"unknown type", func(r map[string]any) { r["__typename"] = "IssueComment" }},
		{"missing type", func(r map[string]any) { delete(r, "__typename") }},
		{"unknown state", func(r map[string]any) { r["previousReviewState"] = "UNKNOWN" }},
		{"missing state", func(r map[string]any) { delete(r, "previousReviewState") }},
		{"no actual time", func(r map[string]any) { r["createdAt"] = nil }},
		{"malformed time", func(r map[string]any) { r["createdAt"] = "yesterday" }},
		{"no joined id", func(r map[string]any) { delete(r["review"].(map[string]any), "id") }},
		{"bad joined id", func(r map[string]any) { r["review"].(map[string]any)["id"] = "reviewer name" }},
		{"bad submission", func(r map[string]any) { r["review"].(map[string]any)["submittedAt"] = "later" }},
		{"bad hash", func(r map[string]any) { r["review"].(map[string]any)["commit"] = map[string]any{"oid": "branch"} }},
		{"missing hash", func(r map[string]any) { r["review"].(map[string]any)["commit"] = map[string]any{} }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			raw := dismissalSource("RDE_1")
			tt.edit(raw)
			p := pr("PR_old", 1, since.Add(-time.Hour), oid)
			p["reviewDismissals"] = dismissalSourceConnection(5, 1, false, raw)
			if _, err := parsePage([]byte(pageOf(false, p))); err == nil {
				t.Fatal("malformed dismissal in a PR outside the retained window passed")
			}
		})
	}
	for _, nodes := range [][]map[string]any{{nil}, {dismissalSource("RDE_1"), dismissalSource("RDE_1")}} {
		n := decodeDismissalNode(t, dismissalSourceConnection(5, len(nodes), false, nodes...))
		if _, _, err := dismissalObservations(&n, event.Source{Name: sourceName}, "p", observedAt); err == nil {
			t.Fatal("null or duplicate dismissal node passed")
		}
	}
}

func TestReadAddsDismissalsWithoutRestoringReviewSnapshots(t *testing.T) {
	p := pr("PR_1", 1, observedAt.Add(-time.Minute), oid)
	p["reviewDismissals"] = dismissalSourceConnection(8, 1, false, dismissalSource("RDE_1"))
	p["reviews"] = map[string]any{"totalCount": 1, "nodes": []map[string]any{
		{"id": "PRR_1", "state": "DISMISSED", "submittedAt": observedAt.Add(-2 * time.Hour), "updatedAt": observedAt.Add(-time.Hour), "commit": map[string]any{"oid": oid}},
	}}
	f := &forge{view: view, pages: []string{pageOf(false, p), pageOf(false, p)}}
	got, err := ReadPullRequests(context.Background(), f.run, "/repo", "p", since, observedAt, "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ReviewDismissals) != 1 || len(got.Reviews) != 1 || got.Reviews[0].Payload.(event.Review).State != "dismissed" || got.Requests[0].Payload.(event.PullRequest).ReviewDismissals.FilteredCount != 1 {
		t.Fatalf("independent transition/snapshot = %+v", got)
	}
}
