package attribution

import (
	"fmt"
	"io"
	"time"
)

func renderReviewDismissals(w io.Writer, d *PRReviewDismissals) error {
	if d == nil {
		return nil
	}
	if p := d.Coverage; p != nil {
		if _, err := fmt.Fprintf(w, "      "+renderDismissalCoverage+"\n", p.TotalCount, p.FilteredCount, p.PageCount, p.Listed, p.HasPreviousPage); err != nil {
			return err
		}
	} else if _, err := fmt.Fprintln(w, "      "+renderDismissalUnavailable); err != nil {
		return err
	}
	for i := range d.Observations {
		o := &d.Observations[i]
		p := &o.Payload
		submitted := "—"
		if !p.SubmittedAt.IsZero() {
			submitted = p.SubmittedAt.Format(time.RFC3339)
		}
		if _, err := fmt.Fprintf(w, "      "+renderDismissalObservation+"\n", o.ID, p.DismissedAt.Format(time.RFC3339),
			p.PreviousState, presentForgeValue(p.ReviewID), presentForgeValue(p.Commit), submitted, o.Source.Name); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w, "      "+renderDismissalCaveat)
	return err
}

func presentForgeValue(value string) string {
	if value == "" {
		return "—"
	}
	return value
}
