package attribution

import (
	"strconv"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/event"
	"github.com/assaio/assaio/internal/layer"
)

func pointerTo[T any](value T) *T { return &value }

func reviewEvent(number int64, id, state, commit string, submitted time.Time) event.Event {
	return forgeEvent(event.TypeReview, id, epoch, event.TimeStated, event.GrainReview,
		event.Review{Number: number, State: state, Commit: commit, SubmittedAt: submitted})
}

func reviewedPR(number int64, reviews ...event.Event) *PullRequests {
	pr := pullRequest(number, event.ChangeMerged, epoch.Add(2*hour), hashOf("f"), 1, 1)
	p := pr.Payload.(event.PullRequest)
	p.ReviewsAvailable, p.Reviews, p.ReviewsListed = true, int64(len(reviews)), int64(len(reviews))
	pr.Payload = p
	return &PullRequests{Requests: []event.Event{pr}, Reviews: reviews}
}

func namesPR(numbers ...int64) []Result {
	candidate := Candidate{}
	for _, number := range numbers {
		candidate.Changes = append(candidate.Changes, ChangeLink{Number: number})
	}
	return []Result{{Candidates: []Candidate{candidate}}}
}

func TestRequestedChangeRevisionCountsWithholdIncompleteOrUnknownReviewPopulations(t *testing.T) {
	for _, tt := range []struct {
		name   string
		adjust func(*event.PullRequest, []event.Event) []event.Event
		want   *int
		reason string
	}{
		{"known request", nil, pointerTo(1), ""},
		{"known absence", func(_ *event.PullRequest, rs []event.Event) []event.Event {
			p := rs[0].Payload.(event.Review)
			p.State = "approved"
			rs[0].Payload = p
			return rs
		}, pointerTo(0), ""},
		{"source empty", func(p *event.PullRequest, _ []event.Event) []event.Event {
			p.Reviews, p.ReviewsListed = 0, 0
			return []event.Event{}
		}, pointerTo(0), ""},
		{"missing connection", func(p *event.PullRequest, _ []event.Event) []event.Event {
			p.ReviewsAvailable, p.Reviews, p.ReviewsListed = false, 0, 0
			return nil
		}, nil, "reviews-unavailable"},
		{"cut connection", func(p *event.PullRequest, rs []event.Event) []event.Event { p.Reviews = 101; return rs }, nil, "reviews-incomplete"},
		{"missing node", func(p *event.PullRequest, rs []event.Event) []event.Event {
			p.Reviews, p.ReviewsListed = 2, 2
			return rs
		}, nil, "review-observation-count-mismatch"},
		{"dismissed", changeReview("dismissed", hashOf("a"), epoch), nil, "dismissed-review-state-history-unavailable"},
		{"pending", changeReview("pending", hashOf("a"), epoch), nil, "pending-review-not-submitted"},
		{"missing commit", changeReview("approved", "", epoch), nil, "reviewed-commit-unavailable"},
		{"submission missing despite event clock", changeReview("changes_requested", hashOf("a"), time.Time{}), nil, "review-submission-time-unavailable"},
		{"unknown state", changeReview("unexpected", hashOf("a"), epoch), nil, "review-state-unusable"},
		{"missing id", func(_ *event.PullRequest, rs []event.Event) []event.Event { rs[0].ID = ""; return rs }, nil, "review-id-unavailable"},
		{"duplicate id", func(p *event.PullRequest, rs []event.Event) []event.Event {
			p.Reviews, p.ReviewsListed = 2, 2
			return append(rs, rs[0])
		}, nil, "duplicate-review-observations"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prs := reviewedPR(9, reviewEvent(9, "PRR_9", "changes_requested", hashOf("a"), epoch))
			p := prs.Requests[0].Payload.(event.PullRequest)
			if tt.adjust != nil {
				prs.Reviews = tt.adjust(&p, prs.Reviews)
				prs.Requests[0].Payload = p
			}
			pr := ChangesOf(prs, namesPR(9), "gh").Linked[0]
			got := pr.RequestedChangesRevisions
			if got == nil || got.Reason != tt.reason || got.Layer != layer.Outcome || got.Provenance != event.Derived {
				t.Fatalf("requested revisions = %+v", got)
			}
			if tt.want == nil {
				if got.Value != nil || got.Confidence != confidenceInsufficient {
					t.Fatalf("unsupported population became a figure: %+v", got)
				}
			} else if got.Value == nil || *got.Value != *tt.want || got.Confidence != confidenceHigh {
				t.Fatalf("requested revisions = %+v, want %d", got, *tt.want)
			}
			if pr.ReviewRounds == nil || pr.ReviewRounds.Value != nil || pr.ReviewRounds.Reason != "review-states-do-not-define-round-boundaries" {
				t.Fatalf("snapshot became review rounds: %+v", pr.ReviewRounds)
			}
		})
	}
}

func changeReview(state, commit string, submitted time.Time) func(*event.PullRequest, []event.Event) []event.Event {
	return func(_ *event.PullRequest, rs []event.Event) []event.Event {
		p := rs[0].Payload.(event.Review)
		p.State, p.Commit, p.SubmittedAt = state, commit, submitted
		rs[0].Payload = p
		return rs
	}
}

func TestRequestedChangeReviewsOnOneRevisionAreNotSeparateRounds(t *testing.T) {
	prs := reviewedPR(9,
		reviewEvent(9, "PRR_3", "changes_requested", hashOf("b"), epoch.Add(hour)),
		reviewEvent(9, "PRR_2", "changes_requested", hashOf("a"), epoch),
		reviewEvent(9, "PRR_1", "changes_requested", hashOf("a"), epoch),
	)
	pr := ChangesOf(prs, namesPR(9), "gh").Linked[0]
	if *pr.RequestedChangesRevisions.Value != 2 || pr.Reviews.States[0].Count != 3 || pr.ReviewRounds.Value != nil {
		t.Fatalf("three reviews at two revisions became rounds: %+v", pr)
	}
}

func TestMergeMethodUsesOnlyKnownMergeTopology(t *testing.T) {
	for _, tt := range []struct {
		state   string
		merge   string
		parents *int64
		value   string
		reason  string
	}{
		{event.ChangeMerged, hashOf("f"), pointerTo(int64(2)), "merge", ""},
		{event.ChangeMerged, hashOf("f"), pointerTo(int64(1)), "", "squash-versus-rebase-undetermined"},
		{event.ChangeMerged, hashOf("f"), nil, "", "merge-parent-count-unavailable"},
		{event.ChangeMerged, hashOf("f"), pointerTo(int64(0)), "", "merge-parent-count-unusable"},
		{event.ChangeMerged, "", nil, "", "merge-commit-unavailable"},
		{event.ChangeOpen, "", nil, "", "pull-request-not-merged"},
	} {
		t.Run(tt.state+"/"+tt.reason+"/"+strconv.FormatBool(tt.parents != nil), func(t *testing.T) {
			prs := reviewedPR(9)
			p := prs.Requests[0].Payload.(event.PullRequest)
			p.State, p.MergeCommit, p.MergeParents = tt.state, tt.merge, tt.parents
			prs.Requests[0].Payload = p
			m := ChangesOf(prs, namesPR(9), "gh").Linked[0].MergeMethod
			if m == nil || m.Reason != tt.reason || m.Source.Name != "gh" || m.Provenance != event.Derived || m.Layer != layer.Outcome {
				t.Fatalf("merge method = %+v", m)
			}
			if tt.value == "" {
				if m.Value != nil || m.Confidence != confidenceInsufficient {
					t.Fatalf("unsupported method became known: %+v", m)
				}
			} else if m.Value == nil || *m.Value != tt.value || m.Confidence != confidenceHigh {
				t.Fatalf("merge topology did not retain method: %+v", m)
			}
		})
	}
}
