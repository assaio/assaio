package attribution

import (
	"slices"

	"github.com/assaio/assaio/internal/event"
	"github.com/assaio/assaio/internal/layer"
)

// PRRate counts PRs once within a declared population. A nil numerator or value withholds
// the calculation; Eligible never silently replaces the full Denominator.
type PRRate struct {
	ID          string          `json:"id"`
	Scope       string          `json:"scope"`
	Layer       layer.Layer     `json:"layer"`
	Population  int             `json:"population"`
	Eligible    int             `json:"eligible"`
	Numerator   *int            `json:"numerator"`
	Denominator int             `json:"denominator"`
	Value       *float64        `json:"value"`
	Provenance  string          `json:"provenance"`
	Confidence  string          `json:"confidence"`
	Reason      string          `json:"reason,omitempty"`
	Gaps        []PopulationGap `json:"gaps"`
}

// PopulationGap states how many PRs lacked one requirement of a rate's population.
type PopulationGap struct {
	Reason       string `json:"reason"`
	PullRequests int    `json:"pullRequests"`
}

func withheldCount(reason string) *DerivedCount {
	return &DerivedCount{Layer: layer.Outcome, Provenance: event.Derived, Confidence: confidenceInsufficient, Reason: reason}
}

func requestedChangesRevisions(pr *event.PullRequest, reviews []ReviewObservation) *DerivedCount {
	if reason := reviewPopulationGap(pr, reviews); reason != "" {
		return withheldCount(reason)
	}
	revisions := map[string]bool{}
	for i := range reviews {
		if reviews[i].Payload.State == "changes_requested" {
			revisions[reviews[i].Payload.Commit] = true
		}
	}
	count := len(revisions)
	return &DerivedCount{Layer: layer.Outcome, Value: &count, Provenance: event.Derived, Confidence: confidenceHigh}
}

func reviewPopulationGap(pr *event.PullRequest, reviews []ReviewObservation) string {
	switch {
	case !pr.ReviewsAvailable:
		return "reviews-unavailable"
	case pr.ReviewsListed != pr.Reviews:
		return "reviews-incomplete"
	case int64(len(reviews)) != pr.ReviewsListed:
		return "review-observation-count-mismatch"
	}
	seen := map[string]bool{}
	for i := range reviews {
		r := &reviews[i]
		if r.ID == "" {
			return "review-id-unavailable"
		}
		if seen[r.ID] {
			return "duplicate-review-observations"
		}
		seen[r.ID] = true
	}
	for i := range reviews {
		r := &reviews[i]
		switch r.Payload.State {
		case "dismissed":
			return "dismissed-review-state-history-unavailable"
		case "pending":
			return "pending-review-not-submitted"
		case "commented", "approved", "changes_requested":
		default:
			return "review-state-unusable"
		}
		if r.Payload.SubmittedAt.IsZero() {
			return "review-submission-time-unavailable"
		}
		if r.Payload.Commit == "" {
			return "reviewed-commit-unavailable"
		}
	}
	return ""
}

func mergeMethod(pr *event.PullRequest, source event.Source) *ObservedMethod {
	m := &ObservedMethod{Layer: layer.Outcome, Source: source, Provenance: event.Derived, Confidence: confidenceInsufficient}
	switch {
	case pr.State != event.ChangeMerged:
		m.Reason = "pull-request-not-merged"
	case pr.MergeCommit == "":
		m.Reason = "merge-commit-unavailable"
	case pr.MergeParents == nil:
		m.Reason = "merge-parent-count-unavailable"
	case *pr.MergeParents > 1:
		value := "merge"
		m.Value, m.Confidence = &value, confidenceHigh
	case *pr.MergeParents == 1:
		m.Reason = "squash-versus-rebase-undetermined"
	default:
		m.Reason = "merge-parent-count-unusable"
	}
	return m
}

func deliveryRates(linked []LinkedPullRequest) []PRRate {
	review := PRRate{
		ID: "requested-changes-presence", Scope: "named-merged-pull-requests-in-read", Layer: layer.Outcome,
		Provenance: event.Derived, Confidence: confidenceInsufficient, Gaps: []PopulationGap{},
	}
	numerator := 0
	for i := range linked {
		pr := &linked[i]
		if pr.State != event.ChangeMerged {
			continue
		}
		review.Population++
		if reason := reviewRateGap(pr); reason != "" {
			countGap(&review, reason)
			continue
		}
		review.Eligible++
		if *pr.RequestedChangesRevisions.Value > 0 {
			numerator++
		}
	}
	review.Denominator = review.Population
	ci := review
	ci.ID, ci.Eligible, ci.Gaps = "failed-check-run-presence", 0, []PopulationGap{}
	ci.Reason = "comparable-pr-pipeline-history-unavailable"
	switch {
	case review.Population == 0:
		review.Reason, ci.Reason = "no-named-merged-pull-requests", "no-named-merged-pull-requests"
	case review.Eligible != review.Population:
		review.Reason = "ineligible-review-populations"
		if len(review.Gaps) == 1 {
			review.Reason = review.Gaps[0].Reason
		}
	default:
		value := float64(numerator) / float64(review.Denominator)
		review.Numerator, review.Value, review.Confidence = &numerator, &value, confidenceHigh
	}
	return []PRRate{review, ci}
}

func reviewRateGap(pr *LinkedPullRequest) string {
	if pr.RequestedChangesRevisions == nil {
		return "reviews-unavailable"
	}
	if pr.RequestedChangesRevisions.Value == nil {
		return pr.RequestedChangesRevisions.Reason
	}
	if pr.Reviews == nil || pr.Reviews.Total == 0 {
		return "no-submitted-review-observations"
	}
	return ""
}

func countGap(rate *PRRate, reason string) {
	for i := range rate.Gaps {
		if rate.Gaps[i].Reason == reason {
			rate.Gaps[i].PullRequests++
			return
		}
	}
	rate.Gaps = append(rate.Gaps, PopulationGap{Reason: reason, PullRequests: 1})
	slices.SortFunc(rate.Gaps, func(a, b PopulationGap) int { return compareStrings(a.Reason, b.Reason) })
}
