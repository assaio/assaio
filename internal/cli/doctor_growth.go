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

// growthLine reports what this store costs to keep, from its own measured span. A central store
// inherits every member's growth at once, which is exactly the operator who could not see it
// (B173, B174).
//
// Three things it deliberately does not do, each of which the first cut got wrong:
// it measures *live* bytes rather than the file, because free pages are the normal state after
// every step prune and counting them makes deleting history raise the projected year; it calls
// the year an upper bound rather than an estimate, because the step timeline stops growing at
// its horizon while the usage table does not, so a rate averaged over both over-states a year;
// and it names the oldest observation, because a single archived row from an old session owns
// the denominator and one such row moved this figure 13x on a real store.
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
		return fmt.Sprintf("this store spans %.0f day(s), too few to project from. Ask again after %d.",
			days, growthMinDays)
	}
	live := size.Bytes - size.Reclaimable
	perDay := float64(live) / days
	return fmt.Sprintf("%s live over %.0f day(s) since %s = %s/day.\n"+
		"              At most %s/year, and likely less: that rate averages the step timeline, which\n"+
		"              stops growing at its horizon, together with the usage table, which does not.\n"+
		"              An upper bound, not an estimate. One old record sets the span it divides by, and\n"+
		"              `clear --older-than` can raise this figure rather than lower it -- it shortens the\n"+
		"              span faster than it frees bytes, because the timeline was already horizon-bounded.\n"+
		"              The bound holds only for a source older than the span: one ingested more recently\n"+
		"              has contributed over fewer days than this rate divides by, so its own rate is higher\n"+
		"              than its share of this figure suggests.",
		humanize.Bytes(live), days, oldest.UTC().Format("2006-01-02"),
		humanize.Bytes(int64(perDay)), humanize.Bytes(int64(perDay*365)))
}
