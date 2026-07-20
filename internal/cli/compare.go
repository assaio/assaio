package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

// runCompare fetches two adjacent windows of length `since`, diffs them per `by` group, and
// renders the period-over-period top movers. Shared by report and effectiveness's --compare.
// Grouping by "day" is meaningless across two windows (no day is in both), so it falls back
// to "project" -- the natural mover dimension.
func runCompare(cmd *cobra.Command, st *store.Store, since, by string) error {
	if by == "" || by == "day" {
		by = "project"
	}
	days, err := windowDays(since)
	if err != nil {
		return err
	}
	priorStart, cutoff := compareWindow(time.Now(), days)

	rows, err := st.Usage(cmd.Context(), priorStart)
	if err != nil {
		return err
	}
	recent, prior := splitByDay(rows, cutoff)
	table, err := pricing.Load()
	if err != nil {
		return err
	}
	recentEff, err := report.BuildEffectiveness(recent, table, by)
	if err != nil {
		return err
	}
	priorEff, err := report.BuildEffectiveness(prior, table, by)
	if err != nil {
		return err
	}
	movers := report.Movers(recentEff, priorEff)
	if len(movers) == 0 {
		cmd.Println(emptyStoreHint(cmd, "No usage to compare."))
		return nil
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "top movers: last %dd vs the %dd before it, by %s\n", days, days, by); err != nil {
		return err
	}
	return report.RenderMovers(cmd.OutOrStdout(), movers, by)
}

// compareWindow returns the store fetch floor and the recent/prior split cutoff for a
// period-over-period comparison of two adjacent N-day windows ending today (UTC). Both
// windows span exactly N day-buckets -- recent is today-(N-1)..today, prior is
// today-(2N-1)..today-N -- so a mover's Δ is not biased by the recent side silently
// covering one extra bucket. The floor is the midnight of prior's first bucket, so that
// bucket is whole rather than a partial slice from the current time of day.
func compareWindow(now time.Time, days int) (priorStart time.Time, cutoff string) {
	u := now.UTC()
	midnight := time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
	priorStart = midnight.AddDate(0, 0, -(2*days - 1))
	cutoff = midnight.AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	return priorStart, cutoff
}

// splitByDay partitions rows into the recent window (Day >= cutoff) and the prior one. Day
// is an ISO "YYYY-MM-DD" string, so a lexical compare is a chronological one.
func splitByDay(rows []store.UsageRow, cutoff string) (recent, prior []store.UsageRow) {
	for i := range rows {
		if rows[i].Day >= cutoff {
			recent = append(recent, rows[i])
		} else {
			prior = append(prior, rows[i])
		}
	}
	return recent, prior
}

// windowDays extracts the day count from a validated "Nd" window (config.sincePattern and
// parseSinceAt already guarantee the form on the paths that reach here).
func windowDays(window string) (int, error) {
	if !strings.HasSuffix(window, "d") {
		return 0, fmt.Errorf("invalid window %q (want e.g. 7d)", window)
	}
	days, err := strconv.Atoi(strings.TrimSuffix(window, "d"))
	if err != nil || days < 0 {
		return 0, fmt.Errorf("invalid window %q", window)
	}
	return days, nil
}
