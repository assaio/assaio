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
	now := time.Now()
	priorStart := now.AddDate(0, 0, -2*days)
	cutoff := now.AddDate(0, 0, -days).UTC().Format("2006-01-02")

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
