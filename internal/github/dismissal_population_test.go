package github

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/event"
)

func TestDismissalPopulationKeepsIndependentCountsAndUnavailable(t *testing.T) {
	for _, tt := range []struct {
		name      string
		edit      func(map[string]any)
		available bool
	}{
		{"stated empty despite whole timeline", func(map[string]any) {}, true},
		{"missing total", func(c map[string]any) { delete(c, "totalCount") }, false},
		{"null total", func(c map[string]any) { c["totalCount"] = nil }, false},
		{"missing filtered", func(c map[string]any) { delete(c, "filteredCount") }, false},
		{"null filtered", func(c map[string]any) { c["filteredCount"] = nil }, false},
		{"missing page count", func(c map[string]any) { delete(c, "pageCount") }, false},
		{"null page count", func(c map[string]any) { c["pageCount"] = nil }, false},
		{"null nodes", func(c map[string]any) { c["nodes"] = nil }, false},
		{"missing nodes", func(c map[string]any) { delete(c, "nodes") }, false},
		{"missing pagination", func(c map[string]any) { delete(c, "pageInfo") }, false},
		{"null pagination", func(c map[string]any) { c["pageInfo"] = nil }, false},
		{"missing previous page", func(c map[string]any) { c["pageInfo"] = map[string]any{} }, false},
		{"null previous page", func(c map[string]any) { c["pageInfo"] = map[string]any{"hasPreviousPage": nil} }, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := dismissalSourceConnection(27, 0, false)
			tt.edit(c)
			n := decodeDismissalNode(t, c)
			p, events, err := dismissalObservations(&n, event.Source{Name: sourceName}, "p", observedAt)
			if err != nil || (p != nil) != tt.available || len(events) != 0 {
				t.Fatalf("population = %+v; events %d; error %v", p, len(events), err)
			}
			if p != nil && (p.TotalCount != 27 || p.FilteredCount != 0 || p.PageCount != 0 || p.Listed != 0 || p.HasPreviousPage) {
				t.Fatalf("empty selection lost independent counts: %+v", p)
			}
		})
	}
	n := decodeDismissalNode(t, nil)
	if p, events, err := dismissalObservations(&n, event.Source{Name: sourceName}, "p", observedAt); p != nil || events != nil || err != nil {
		t.Fatalf("null connection = %+v, %+v, %v", p, events, err)
	}
	var nodes []map[string]any
	for i := range 100 {
		nodes = append(nodes, dismissalSource(fmt.Sprintf("RDE_%d", i)))
	}
	n = decodeDismissalNode(t, dismissalSourceConnection(300, 101, true, nodes...))
	p, events, err := dismissalObservations(&n, event.Source{Name: sourceName}, "p", observedAt)
	if err != nil || p == nil || p.TotalCount != 300 || p.FilteredCount != 101 || p.PageCount != 100 || p.Listed != 100 || !p.HasPreviousPage || len(events) != 100 {
		t.Fatalf("bounded slice = %+v, %d events, %v", p, len(events), err)
	}
}

func TestMalformedDismissalCountsFailDocument(t *testing.T) {
	for _, tt := range []struct {
		name string
		edit func(map[string]any)
	}{
		{"negative whole", func(c map[string]any) { c["totalCount"] = -1 }},
		{"negative filtered", func(c map[string]any) { c["filteredCount"] = -1 }},
		{"negative page", func(c map[string]any) { c["pageCount"] = -1 }},
		{"more filtered than whole", func(c map[string]any) { c["totalCount"] = 0 }},
		{"wrong slice count", func(c map[string]any) { c["pageCount"] = 0 }},
		{"short under fixed bound", func(c map[string]any) {
			c["filteredCount"] = 2
			c["pageInfo"] = map[string]any{"hasPreviousPage": true}
		}},
		{"false truncation", func(c map[string]any) { c["pageInfo"] = map[string]any{"hasPreviousPage": true} }},
		{"string count", func(c map[string]any) { c["totalCount"] = "5" }},
		{"fraction count", func(c map[string]any) { c["filteredCount"] = 1.5 }},
		{"negative despite unavailable", func(c map[string]any) { delete(c, "totalCount"); c["pageCount"] = -1 }},
		{"selected zero without page count", func(c map[string]any) { delete(c, "pageCount"); c["filteredCount"] = 0 }},
		{"whole zero without filtered count", func(c map[string]any) { delete(c, "filteredCount"); c["totalCount"] = 0 }},
		{"short slice without page count", func(c map[string]any) { delete(c, "pageCount"); c["filteredCount"] = 2 }},
		{"false truncation without page count", func(c map[string]any) {
			delete(c, "pageCount")
			c["pageInfo"] = map[string]any{"hasPreviousPage": true}
		}},
		{"previous page with missing counts", func(c map[string]any) {
			delete(c, "filteredCount")
			delete(c, "pageCount")
			c["pageInfo"] = map[string]any{"hasPreviousPage": true}
		}},
		{"whole too small for previous page with only whole count", func(c map[string]any) {
			delete(c, "filteredCount")
			delete(c, "pageCount")
			delete(c, "nodes")
			c["pageInfo"] = map[string]any{"hasPreviousPage": true}
		}},
		{"oversized page with missing selection and nodes", func(c map[string]any) {
			delete(c, "filteredCount")
			delete(c, "nodes")
			c["totalCount"], c["pageCount"] = 200, 101
		}},
		{"page exceeds whole without selection or nodes", func(c map[string]any) {
			delete(c, "filteredCount")
			delete(c, "nodes")
			c["totalCount"] = 0
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := dismissalSourceConnection(8, 1, false, dismissalSource("RDE_1"))
			tt.edit(c)
			p := pr("PR_1", 1, observedAt, oid)
			p["reviewDismissals"] = c
			if _, err := parsePage([]byte(pageOf(false, p))); err == nil {
				t.Fatal("invalid dismissal coverage passed")
			}
		})
	}
}

func TestUnavailableDismissalCountsDoNotEraseActualListedEvents(t *testing.T) {
	c := dismissalSourceConnection(8, 1, false, dismissalSource("RDE_1"))
	c["filteredCount"] = nil
	n := decodeDismissalNode(t, c)
	p, events, err := dismissalObservations(&n, event.Source{Name: sourceName}, "p", observedAt)
	if err != nil || p != nil || len(events) != 1 {
		t.Fatalf("available node and unavailable denominator = %+v, %+v, %v", p, events, err)
	}
}

func TestDismissalQueryIsFixedAndIndependentOfReviewSnapshots(t *testing.T) {
	start := strings.Index(pullRequestQuery, "reviewDismissals:")
	end := strings.Index(pullRequestQuery[start:], "statusCheckRollup")
	selection := pullRequestQuery[start : start+end]
	if !strings.Contains(selection, "timelineItems(last: 100, itemTypes: [REVIEW_DISMISSED_EVENT])") {
		t.Fatal("dismissal query lost its fixed selected population")
	}
	for _, forbidden := range []string{"after", "before", "since", "first", "actor", "body", "dismissalMessage", "state", "name"} {
		if regexp.MustCompile(`\b` + forbidden + `\b`).MatchString(selection) {
			t.Errorf("dismissal query contains %q", forbidden)
		}
	}
	if strings.Contains(historyQuery, "reviewDismissals") || strings.Contains(historyQuery, "ReviewDismissedEvent") {
		t.Fatal("independent historical query changed")
	}
}
