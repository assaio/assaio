package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/store"
)

// growthMinDays is the span below which a per-day rate is arithmetic rather than evidence: a
// store two days old projects a year from two days, and the answer says more about the two days
// than about the year.
const growthMinDays = 7

// growthLine projects what this store costs to keep, from its own measured span rather than
// from a rate somebody else's machine produced. A central store inherits every member's growth
// at once, which is exactly the operator who cannot see it today (B173, B174).
//
// It is a projection and says so: usage is not uniform, a horizon-bounded table stops growing
// and an unbounded one does not, and a year of the same behaviour is an assumption rather than
// a measurement.
func growthLine(ctx context.Context, st *store.Store, now time.Time) string {
	size, err := st.Size(ctx)
	if err != nil {
		return ""
	}
	oldest, err := st.HistoryStart(ctx, "")
	if err != nil || oldest.IsZero() {
		return ""
	}
	days := now.Sub(oldest).Hours() / 24
	if days < growthMinDays {
		return fmt.Sprintf("this store holds %.0f day(s), too few to project a year from. "+
			"Ask again after %d.", days, growthMinDays)
	}
	perDay := float64(size.Bytes) / days
	return fmt.Sprintf("%s over %.0f day(s) = %s/day, which projects to %s/year at this rate.\n"+
		"              A projection, not a measurement: usage is not uniform, the step timeline stops\n"+
		"              growing at its horizon and the usage table does not, and a year of the same\n"+
		"              behaviour is the assumption doing the work here.",
		humanize.Bytes(size.Bytes), days, humanize.Bytes(int64(perDay)), humanize.Bytes(int64(perDay*365)))
}
