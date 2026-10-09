package github

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/assaio/assaio/internal/event"
)

func deliveryObservations(nd *node, src event.Source, project string, observedAt time.Time) ([]event.Event, []event.Event, error) {
	var reviews, checks []event.Event
	if nd.Reviews != nil {
		for _, r := range nd.Reviews.Nodes {
			if !nodeID(r.ID) {
				return nil, nil, errors.New("review has no usable id")
			}
			at := r.SubmittedAt
			if at.IsZero() {
				at = r.UpdatedAt
			}
			e := envelope(event.TypeReview, r.ID, at, event.TimeStated, event.GrainReview, src, project, observedAt)
			e.Payload = event.Review{Number: nd.Number, State: strings.ToLower(r.State)}
			reviews = append(reviews, e)
		}
	}
	if nd.StatusCheckRollup != nil {
		for i := range nd.StatusCheckRollup.Contexts.Nodes {
			c := &nd.StatusCheckRollup.Contexts.Nodes[i]
			if !nodeID(c.ID) {
				return nil, nil, errors.New("head check has no usable id")
			}
			at, timeSource := observedAt, event.TimeIngestTime
			check := event.Check{Number: nd.Number}
			switch c.Type {
			case "CheckRun":
				check.Kind, check.State, check.Conclusion = "run", strings.ToLower(c.Status), strings.ToLower(c.Conclusion)
				at = c.CompletedAt
				if at.IsZero() {
					at = c.StartedAt
				}
			case "StatusContext":
				check.Kind, check.State = "status", strings.ToLower(c.State)
				at = c.UpdatedAt
				if at.IsZero() {
					at = c.CreatedAt
				}
			default:
				return nil, nil, fmt.Errorf("unknown head-check type %q", c.Type)
			}
			if !at.IsZero() {
				timeSource = event.TimeStated
			} else {
				at = observedAt
			}
			e := envelope(event.TypeCheck, nd.ID+":"+c.ID, at, timeSource, event.GrainCheck, src, project, observedAt)
			e.Payload = check
			checks = append(checks, e)
		}
	}
	return reviews, checks, nil
}
