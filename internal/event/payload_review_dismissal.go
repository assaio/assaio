package event

import (
	"errors"
	"fmt"
	"time"
)

// TimelinePopulation separates the whole PR timeline from the selected dismissal population
// and the slice returned by a fixed last:100 read. Nil means unavailable, not empty.
type TimelinePopulation struct {
	TotalCount      int64 `json:"totalCount"`
	FilteredCount   int64 `json:"filteredCount"`
	PageCount       int64 `json:"pageCount"`
	Listed          int64 `json:"listed"`
	HasPreviousPage bool  `json:"hasPreviousPage"`
}

func (p TimelinePopulation) validate() error {
	if err := nonNegative(map[string]int64{
		"totalCount": p.TotalCount, "filteredCount": p.FilteredCount,
		"pageCount": p.PageCount, "listed": p.Listed,
	}); err != nil {
		return err
	}
	if p.Listed != p.PageCount || p.PageCount > p.FilteredCount || p.FilteredCount > p.TotalCount {
		return errors.New("inconsistent dismissal timeline counts")
	}
	if p.PageCount != min(p.FilteredCount, 100) || p.HasPreviousPage != (p.FilteredCount > p.PageCount) {
		return errors.New("dismissal timeline does not match the last:100 bound")
	}
	return nil
}

// ReviewDismissal records an actual timeline transition. A nullable source review join
// leaves ReviewID, Commit and SubmittedAt absent; PreviousState never restores a snapshot.
type ReviewDismissal struct {
	Number        int64     `json:"number"`
	PreviousState string    `json:"previousState"`
	ReviewID      string    `json:"reviewId,omitempty"`
	Commit        string    `json:"commit,omitempty"`
	SubmittedAt   time.Time `json:"submittedAt,omitzero"`
	DismissedAt   time.Time `json:"dismissedAt"`
}

func (ReviewDismissal) eventType() string { return TypeReviewDismissal }

//nolint:gocritic // the Payload interface is satisfied by values; validated once per source observation.
func (p ReviewDismissal) validate() error {
	if p.Number <= 0 {
		return fmt.Errorf("pull request number %d is not positive", p.Number)
	}
	if !valid(reviewStates, p.PreviousState) {
		return fmt.Errorf("unknown previous review state %q", p.PreviousState)
	}
	if p.DismissedAt.IsZero() {
		return errors.New("review dismissal has no actual dismissal time")
	}
	if p.ReviewID == "" {
		if p.Commit != "" || !p.SubmittedAt.IsZero() {
			return errors.New("review dismissal has review metadata without a review id")
		}
	} else if !isForgeID(p.ReviewID) {
		return errors.New("dismissed review has no usable id")
	}
	if p.Commit != "" && !isObjectID(p.Commit) {
		return errors.New("dismissed review commit is not a commit hash")
	}
	return nil
}
