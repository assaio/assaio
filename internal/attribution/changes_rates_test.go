package attribution

import (
	"reflect"
	"slices"
	"testing"

	"github.com/assaio/assaio/internal/event"
	"github.com/assaio/assaio/internal/layer"
)

func TestReviewShareUsesWholeNamedMergedPRPopulation(t *testing.T) {
	for _, tt := range []struct {
		name       string
		adjust     func(*PullRequests, []Result) []Result
		population int
		eligible   int
		numerator  *int
		value      *float64
		reason     string
	}{
		{"one of two", nil, 2, 2, pointerTo(1), pointerTo(0.5), ""},
		{"shared PR across sessions", func(_ *PullRequests, _ []Result) []Result {
			return append(namesPR(1), namesPR(1)...)
		}, 1, 1, pointerTo(1), pointerTo(float64(1)), ""},
		{"named only by alternative", func(_ *PullRequests, _ []Result) []Result {
			return []Result{{Alternatives: namesPR(1)[0].Candidates}}
		}, 1, 1, pointerTo(1), pointerTo(float64(1)), ""},
		{"known zero", func(_ *PullRequests, _ []Result) []Result { return namesPR(2) }, 1, 1, pointerTo(0), pointerTo(float64(0)), ""},
		{"no named PR", func(_ *PullRequests, _ []Result) []Result { return nil }, 0, 0, nil, nil, "no-named-merged-pull-requests"},
		{"no merged PR", func(prs *PullRequests, results []Result) []Result {
			for i := range prs.Requests {
				p := prs.Requests[i].Payload.(event.PullRequest)
				p.State = event.ChangeOpen
				prs.Requests[i].Payload = p
			}
			return results
		}, 0, 0, nil, nil, "no-named-merged-pull-requests"},
		{"one incomplete PR", func(prs *PullRequests, results []Result) []Result {
			p := prs.Requests[1].Payload.(event.PullRequest)
			p.Reviews = 101
			prs.Requests[1].Payload = p
			return results
		}, 2, 1, nil, nil, "reviews-incomplete"},
		{"one missing connection", func(prs *PullRequests, results []Result) []Result {
			p := prs.Requests[1].Payload.(event.PullRequest)
			p.ReviewsAvailable, p.Reviews, p.ReviewsListed = false, 0, 0
			prs.Requests[1].Payload = p
			return results
		}, 2, 1, nil, nil, "reviews-unavailable"},
		{"one source-empty population", func(prs *PullRequests, results []Result) []Result {
			p := prs.Requests[1].Payload.(event.PullRequest)
			p.Reviews, p.ReviewsListed = 0, 0
			prs.Requests[1].Payload = p
			prs.Reviews = prs.Reviews[:1]
			return results
		}, 2, 1, nil, nil, "no-submitted-review-observations"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prs := twoReviewedPRs()
			results := namesPR(1, 2)
			if tt.adjust != nil {
				results = tt.adjust(prs, results)
			}
			c := ChangesOf(prs, results, "gh")
			got := c.Rates[0]
			if got.Population != tt.population || got.Denominator != tt.population || got.Eligible != tt.eligible || got.Reason != tt.reason {
				t.Fatalf("review share population = %+v", got)
			}
			if !reflect.DeepEqual(got.Numerator, tt.numerator) || !reflect.DeepEqual(got.Value, tt.value) {
				t.Fatalf("share numerator/value = %v/%v, want %v/%v", got.Numerator, got.Value, tt.numerator, tt.value)
			}
			if got.Layer != layer.Outcome || got.Provenance != event.Derived || got.Scope != "named-merged-pull-requests-in-read" {
				t.Fatalf("share lost its measurement contract: %+v", got)
			}
			ci := c.Rates[1]
			if ci.Population != tt.population || ci.Denominator != tt.population || ci.Eligible != 0 || ci.Value != nil || ci.Numerator != nil || ci.Reason == "" {
				t.Fatalf("commit checks became a pipeline rate: %+v", ci)
			}
		})
	}
}

func twoReviewedPRs() *PullRequests {
	prs := reviewedPR(1, reviewEvent(1, "PRR_1", "changes_requested", hashOf("a"), epoch))
	other := reviewedPR(2, reviewEvent(2, "PRR_2", "approved", hashOf("b"), epoch))
	prs.Requests = append(prs.Requests, other.Requests...)
	prs.Reviews = append(prs.Reviews, other.Reviews...)
	return prs
}

func TestUnknownReviewPopulationsReportTheirCountsWithoutReducingTheDenominator(t *testing.T) {
	prs := twoReviewedPRs()
	other := reviewedPR(3, reviewEvent(3, "PRR_3", "approved", hashOf("c"), epoch))
	prs.Requests, prs.Reviews = append(prs.Requests, other.Requests...), append(prs.Reviews, other.Reviews...)
	for i := range prs.Requests {
		p := prs.Requests[i].Payload.(event.PullRequest)
		if i < 2 {
			p.ReviewsAvailable, p.Reviews, p.ReviewsListed = false, 0, 0
		} else {
			p.Reviews = 101
		}
		prs.Requests[i].Payload = p
	}
	got := ChangesOf(prs, namesPR(1, 2, 3), "gh").Rates[0]
	want := []PopulationGap{{Reason: "reviews-incomplete", PullRequests: 1}, {Reason: "reviews-unavailable", PullRequests: 2}}
	if got.Population != 3 || got.Denominator != 3 || got.Eligible != 0 || got.Value != nil || got.Numerator != nil || !reflect.DeepEqual(got.Gaps, want) {
		t.Fatalf("population gap counts = %+v, want %+v", got, want)
	}
}

func TestReviewRatesRemainSnapshotsWhenAReviewWasSubmittedAfterMerge(t *testing.T) {
	prs := reviewedPR(1, reviewEvent(1, "PRR_1", "changes_requested", hashOf("a"), epoch.Add(4*hour)))
	got := ChangesOf(prs, namesPR(1), "gh").Rates[0]
	if got.Value == nil || *got.Value != 1 || got.Eligible != 1 {
		t.Fatalf("a current review snapshot acquired a pre-merge filter: %+v", got)
	}
}

func TestDeliverySummaryOrderAndPRFactsCannotChangeSessionResults(t *testing.T) {
	prs := twoReviewedPRs()
	prs.ObservedAt, prs.ReadBackTo = epoch.Add(10*day), epoch
	results := namesPR(1, 2)
	results[0].Status, results[0].Confidence, results[0].Ambiguous = statusAmbiguous, confidenceInsufficient, true
	before := slices.Clone(results)
	summary := Summarize(results)
	first := ChangesOf(prs, results, "gh")
	slices.Reverse(prs.Requests)
	slices.Reverse(prs.Reviews)
	second := ChangesOf(prs, results, "gh")
	if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(results, before) || Summarize(results) != summary {
		t.Fatal("delivery summary order or forge facts changed a session result")
	}
}
