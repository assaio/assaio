package attribution

import (
	"slices"

	"github.com/assaio/assaio/internal/event"
	"github.com/assaio/assaio/internal/layer"
)

type deliveryStates struct {
	reviews *ObservedStates
	checks  *ObservedStates
}

func deliveryByPR(prs *PullRequests) map[int64]*deliveryStates {
	states := make(map[int64]*deliveryStates, len(prs.Requests))
	for i := range prs.Requests {
		pr, ok := prs.Requests[i].Payload.(event.PullRequest)
		if !ok {
			continue
		}
		delivery := &deliveryStates{}
		if pr.ReviewsAvailable {
			delivery.reviews = &ObservedStates{Layer: layer.Outcome, Total: pr.Reviews, Listed: pr.ReviewsListed, States: []StateCount{}}
		}
		if pr.CheckState != "" {
			delivery.checks = &ObservedStates{Layer: layer.Outcome, Total: pr.Checks, Listed: pr.ChecksListed, State: pr.CheckState, States: []StateCount{}}
		}
		states[pr.Number] = delivery
	}
	for i := range prs.Reviews {
		if r, ok := prs.Reviews[i].Payload.(event.Review); ok {
			if delivery := states[r.Number]; delivery != nil && delivery.reviews != nil {
				countState(delivery.reviews, r.State)
			}
		}
	}
	for i := range prs.Checks {
		if c, ok := prs.Checks[i].Payload.(event.Check); ok {
			if delivery := states[c.Number]; delivery != nil && delivery.checks != nil {
				state := c.Kind + ":" + c.State
				if c.Conclusion != "" {
					state = c.Kind + ":" + c.Conclusion
				}
				countState(delivery.checks, state)
			}
		}
	}
	return states
}

func countState(s *ObservedStates, state string) {
	for i := range s.States {
		if s.States[i].State == state {
			s.States[i].Count++
			return
		}
	}
	s.States = append(s.States, StateCount{State: state, Count: 1})
	slices.SortFunc(s.States, func(a, b StateCount) int { return compareStrings(a.State, b.State) })
}
