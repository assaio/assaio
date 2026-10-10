package attribution

import (
	"encoding/json"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/event"
	"github.com/assaio/assaio/internal/layer"
)

func dismissalEvent(number int64, id, previous string) event.Event {
	return forgeEvent(event.TypeReviewDismissal, id, epoch.Add(hour), event.TimeStated, event.GrainReview,
		event.ReviewDismissal{
			Number: number, PreviousState: previous, DismissedAt: epoch.Add(hour),
			ReviewID: "PRR_9", Commit: hashOf("a"), SubmittedAt: epoch,
		})
}

func setDismissalCoverage(prs *PullRequests, coverage *event.TimelinePopulation) {
	p := prs.Requests[0].Payload.(event.PullRequest)
	p.ReviewDismissals = coverage
	prs.Requests[0].Payload = p
}

func dismissalEvents(count int) []event.Event {
	out := make([]event.Event, count)
	for i := range out {
		out[i] = dismissalEvent(9, "RDE_"+strconv.Itoa(i), "approved")
	}
	return out
}

func TestDismissalCoverageKeepsUnknownSeparateFromEmptyAndTruncated(t *testing.T) {
	for _, tt := range []struct {
		name     string
		coverage *event.TimelinePopulation
		events   []event.Event
		state    string
		fragment string
	}{
		{"unknown", nil, nil, "unavailable", `"coverage":null,"observations":null`},
		{
			"known empty selection", &event.TimelinePopulation{TotalCount: 23}, nil, "read",
			`"totalCount":23,"filteredCount":0,"pageCount":0,"listed":0,"hasPreviousPage":false`,
		},
		{
			"bounded selection", &event.TimelinePopulation{TotalCount: 140, FilteredCount: 101, PageCount: 100, Listed: 100, HasPreviousPage: true}, dismissalEvents(100), "read",
			`"filteredCount":101,"pageCount":100,"listed":100,"hasPreviousPage":true`,
		},
		{"unknown denominator with source event", nil, []event.Event{dismissalEvent(9, "RDE_9", "approved")}, "unavailable", `"coverage":null,"observations":[`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prs := reviewedPR(9)
			setDismissalCoverage(prs, tt.coverage)
			prs.ReviewDismissals = tt.events
			d := ChangesOf(prs, namesPR(9), "gh").Linked[0].ReviewDismissals
			body, err := json.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			if d.ReadState != tt.state || d.Layer != layer.Outcome || !strings.Contains(string(body), tt.fragment) {
				t.Fatalf("dismissal coverage = %s, want state=%s and %s", body, tt.state, tt.fragment)
			}
			if tt.coverage != nil && len(tt.events) == 0 && d.Observations == nil {
				t.Fatal("known selected population lost its explicit empty observation list")
			}
		})
	}
}

func TestDismissalsPreserveSourceAndDoNotRestoreSnapshotMeasurements(t *testing.T) {
	prs := reviewedPR(9, reviewEvent(9, "PRR_9", "dismissed", hashOf("a"), epoch))
	baseline := ChangesOf(prs, namesPR(9), "gh")
	setDismissalCoverage(prs, &event.TimelinePopulation{TotalCount: 23, FilteredCount: 1, PageCount: 1, Listed: 1})
	prs.ReviewDismissals = []event.Event{dismissalEvent(9, "RDE_9", "changes_requested")}
	got := ChangesOf(prs, namesPR(9), "gh")
	d := got.Linked[0].ReviewDismissals
	if len(d.Observations) != 1 {
		t.Fatalf("dismissal events = %+v", d)
	}
	o := &d.Observations[0]
	if o.Source.Name != "gh" || o.Privacy != event.LocalOnly || o.Provenance != event.Parsed ||
		o.TimeSource != event.TimeStated || !o.OccurredAt.Equal(o.Payload.DismissedAt) ||
		!o.Payload.SubmittedAt.Equal(epoch) || o.Payload.PreviousState != "changes_requested" {
		t.Fatalf("dismissal source envelope = %+v", o)
	}
	got.Linked[0].ReviewDismissals = baseline.Linked[0].ReviewDismissals
	if !reflect.DeepEqual(got, baseline) || got.Linked[0].RequestedChangesRevisions.Value != nil || got.Rates[0].Value != nil {
		t.Fatal("dismissal history changed a current snapshot, round, pipeline or merge measurement")
	}
}

func TestDismissalsAreScopedToNamedPRsAndOrderedWithoutInventingJoinMetadata(t *testing.T) {
	prs := reviewedPR(9)
	setDismissalCoverage(prs, &event.TimelinePopulation{TotalCount: 3, FilteredCount: 2, PageCount: 2, Listed: 2})
	nullReview := dismissalEvent(9, "RDE_A", "approved")
	p := nullReview.Payload.(event.ReviewDismissal)
	p.ReviewID, p.Commit = "", ""
	// The source's nullable review carries neither a reviewed commit nor a submission time.
	p.SubmittedAt = time.Time{}
	nullReview.Payload = p
	prs.ReviewDismissals = []event.Event{dismissalEvent(99, "RDE_OTHER", "approved"), dismissalEvent(9, "RDE_Z", "approved"), nullReview}
	first := ChangesOf(prs, namesPR(9), "gh")
	slices.Reverse(prs.ReviewDismissals)
	second := ChangesOf(prs, namesPR(9), "gh")
	if !reflect.DeepEqual(first, second) || len(first.Linked[0].ReviewDismissals.Observations) != 2 || first.Linked[0].ReviewDismissals.Observations[0].ID != "RDE_A" {
		t.Fatal("dismissals leaked across PR populations or depended on source order")
	}
	if p := first.Linked[0].ReviewDismissals.Observations[0].Payload; p.ReviewID != "" || p.Commit != "" || !p.SubmittedAt.IsZero() {
		t.Fatal("null source review gained invented join metadata")
	}
}
