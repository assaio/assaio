package github

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/assaio/assaio/internal/event"
)

type dismissalConnection struct {
	TotalCount    *int64 `json:"totalCount"`
	FilteredCount *int64 `json:"filteredCount"`
	PageCount     *int64 `json:"pageCount"`
	PageInfo      *struct {
		HasPreviousPage *bool `json:"hasPreviousPage"`
	} `json:"pageInfo"`
	Nodes []*dismissalNode `json:"nodes"`
}

type dismissalNode struct {
	Type                string    `json:"__typename"`
	ID                  string    `json:"id"`
	CreatedAt           time.Time `json:"createdAt"`
	PreviousReviewState string    `json:"previousReviewState"`
	Review              *struct {
		ID          string    `json:"id"`
		SubmittedAt time.Time `json:"submittedAt"`
		Commit      *struct {
			OID string `json:"oid"`
		} `json:"commit"`
	} `json:"review"`
}

func dismissalObservations(nd *node, src event.Source, project string, observedAt time.Time) (*event.TimelinePopulation, []event.Event, error) {
	conn := nd.ReviewDismissals
	if conn == nil {
		return nil, nil, nil
	}
	population, err := dismissalPopulation(conn)
	if err != nil {
		return nil, nil, err
	}
	var events []event.Event
	seen := map[string]bool{}
	for _, raw := range conn.Nodes {
		if raw == nil || !nodeID(raw.ID) || seen[raw.ID] {
			return nil, nil, errors.New("dismissal has no usable unique id")
		}
		seen[raw.ID] = true
		if raw.Type != "ReviewDismissedEvent" {
			return nil, nil, fmt.Errorf("unknown dismissal timeline type %q", raw.Type)
		}
		payload := event.ReviewDismissal{
			Number: nd.Number, PreviousState: strings.ToLower(raw.PreviousReviewState), DismissedAt: raw.CreatedAt.UTC(),
		}
		if raw.Review != nil {
			if !nodeID(raw.Review.ID) {
				return nil, nil, errors.New("dismissed review has no usable id")
			}
			payload.ReviewID, payload.SubmittedAt = raw.Review.ID, raw.Review.SubmittedAt.UTC()
			if raw.Review.Commit != nil {
				if raw.Review.Commit.OID == "" {
					return nil, nil, errors.New("dismissed review commit has no usable hash")
				}
				payload.Commit = raw.Review.Commit.OID
			}
		}
		e := envelope(event.TypeReviewDismissal, raw.ID, raw.CreatedAt.UTC(), event.TimeStated, event.GrainReview, src, project, observedAt)
		e.Payload = payload
		if err := e.Validate(); err != nil {
			return nil, nil, err
		}
		events = append(events, e)
	}
	return population, events, nil
}

func dismissalPopulation(conn *dismissalConnection) (*event.TimelinePopulation, error) {
	for _, count := range []*int64{conn.TotalCount, conn.FilteredCount, conn.PageCount} {
		if count != nil && *count < 0 {
			return nil, errors.New("dismissal timeline count is negative")
		}
	}
	listed := int64(len(conn.Nodes))
	if listed > 100 ||
		(conn.TotalCount != nil && listed > *conn.TotalCount) ||
		(conn.TotalCount != nil && conn.FilteredCount != nil && *conn.FilteredCount > *conn.TotalCount) ||
		(conn.PageCount != nil && *conn.PageCount > 100) ||
		(conn.TotalCount != nil && conn.PageCount != nil && *conn.PageCount > *conn.TotalCount) ||
		(conn.FilteredCount != nil && conn.Nodes != nil && listed != min(*conn.FilteredCount, 100)) ||
		(conn.FilteredCount != nil && conn.PageCount != nil && *conn.PageCount != min(*conn.FilteredCount, 100)) ||
		(conn.PageCount != nil && conn.Nodes != nil && *conn.PageCount != listed) {
		return nil, errors.New("inconsistent dismissal timeline population or last:100 bound")
	}
	if conn.PageInfo != nil && conn.PageInfo.HasPreviousPage != nil {
		previous := *conn.PageInfo.HasPreviousPage
		if (conn.FilteredCount != nil && previous != (*conn.FilteredCount > 100)) ||
			(previous && conn.TotalCount != nil && *conn.TotalCount <= 100) ||
			(previous && conn.PageCount != nil && *conn.PageCount != 100) ||
			(previous && conn.Nodes != nil && listed != 100) {
			return nil, errors.New("inconsistent dismissal timeline pagination")
		}
	}
	if conn.TotalCount == nil || conn.FilteredCount == nil || conn.PageCount == nil ||
		conn.PageInfo == nil || conn.PageInfo.HasPreviousPage == nil || conn.Nodes == nil {
		return nil, nil
	}
	p := &event.TimelinePopulation{
		TotalCount: *conn.TotalCount, FilteredCount: *conn.FilteredCount, PageCount: *conn.PageCount,
		Listed: int64(len(conn.Nodes)), HasPreviousPage: *conn.PageInfo.HasPreviousPage,
	}
	if p.Listed != p.PageCount || p.PageCount > p.FilteredCount || p.FilteredCount > p.TotalCount ||
		p.PageCount != min(p.FilteredCount, 100) || p.HasPreviousPage != (p.FilteredCount > p.PageCount) {
		return nil, errors.New("inconsistent dismissal timeline population or last:100 bound")
	}
	return p, nil
}
