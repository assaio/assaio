package attribution

import (
	"slices"

	"github.com/assaio/assaio/internal/event"
	"github.com/assaio/assaio/internal/layer"
)

// ReviewDismissalObservation is a source transition, separate from current review snapshots.
type ReviewDismissalObservation struct {
	ForgeObservation
	Payload event.ReviewDismissal `json:"payload"`
}

// PRReviewDismissals keeps the filtered timeline's coverage and independently observed events.
// Unknown coverage does not erase a valid event, or imply an empty timeline.
type PRReviewDismissals struct {
	ReadState    string                       `json:"readState"`
	Layer        layer.Layer                  `json:"layer"`
	Coverage     *event.TimelinePopulation    `json:"coverage"`
	Observations []ReviewDismissalObservation `json:"observations"`
}

func reviewDismissalsByPR(prs *PullRequests) map[int64]*PRReviewDismissals {
	out := make(map[int64]*PRReviewDismissals, len(prs.Requests))
	for i := range prs.Requests {
		pr, ok := prs.Requests[i].Payload.(event.PullRequest)
		if !ok {
			continue
		}
		d := &PRReviewDismissals{ReadState: "unavailable", Layer: layer.Outcome}
		if pr.ReviewDismissals != nil {
			d.ReadState, d.Coverage = "read", pr.ReviewDismissals
			d.Observations = []ReviewDismissalObservation{}
		}
		out[pr.Number] = d
	}
	for i := range prs.ReviewDismissals {
		e := &prs.ReviewDismissals[i]
		p, ok := e.Payload.(event.ReviewDismissal)
		if !ok {
			continue
		}
		if d := out[p.Number]; d != nil {
			d.Observations = append(d.Observations, ReviewDismissalObservation{forgeObservation(e), p})
		}
	}
	for _, d := range out {
		slices.SortFunc(d.Observations, func(a, b ReviewDismissalObservation) int { return compareStrings(a.ID, b.ID) })
	}
	return out
}
