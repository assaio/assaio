package cli

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/i18n"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

func newStatuslineCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "statusline",
		Short: "Print one ambient line: today's tokens, AI lines, cost basis, and how fresh the data is",
		Long: `Print a single line summarizing the local day so far, for an editor or shell status bar.

The day is the machine's own local day, not a UTC bucket, so work after local midnight
counts where it happened. Figures come from the same aggregation and pricing path as
'report', so the two can never disagree.

The line always ends with the age of the data, because it is only as fresh as the last
ingest -- see docs/automation.md for the hook recipes that keep it current. This command
only reads: it never writes to the store, and any error prints nothing and exits 0 rather
than breaking the prompt it renders into.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runStatusline(cmd)
		},
	}
	addDBFlag(c)
	return c
}

// runStatusline never reports an error: a status line that fails loudly would break the
// prompt it renders into, so every failure path prints nothing and exits 0.
func runStatusline(cmd *cobra.Command) error {
	line, err := statuslineText(cmd, time.Now())
	if err != nil {
		return nil
	}
	cmd.Println(line)
	return nil
}

// statuslineText builds the line for the local day containing now.
func statuslineText(cmd *cobra.Command, now time.Time) (string, error) {
	l := i18n.For("").CLI
	dbPath, err := resolveDBPath(cmd)
	if err != nil {
		return "", err
	}
	// Opening would create the database; a status line must not have that side effect.
	if _, err := os.Stat(dbPath); err != nil {
		return segments(l.StatuslinePrefix, l.StatuslineNoData), nil
	}
	st, err := store.Open(dbPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = st.Close() }()

	n, err := st.Count(cmd.Context())
	if err != nil {
		return "", err
	}
	if n == 0 {
		return segments(l.StatuslinePrefix, l.StatuslineNoData), nil
	}

	cfg, err := loadConfigLenient(cmd)
	if err != nil {
		return "", err
	}
	today, err := statuslineDay(cmd.Context(), st, startOfLocalDay(now))
	if err != nil {
		return "", err
	}
	age := statuslineAge(cmd.Context(), st, now, &l)
	if today.tokens == 0 && today.lines == 0 {
		return segments(l.StatuslinePrefix, l.StatuslineNothingToday, age), nil
	}

	parts := []string{
		l.StatuslinePrefix,
		humanize.Count(today.tokens) + " " + l.StatuslineTokens,
		"+" + humanize.Count(today.lines) + " " + l.StatuslineLines,
	}
	if basis := costBasisSegment(cmd.Context(), st, now, cfg.Pricing, today, &l); basis != "" {
		parts = append(parts, basis)
	}
	return segments(append(parts, age)...), nil
}

// dayTotals is one window rolled up to what the line shows.
type dayTotals struct {
	tokens, lines int64
	cost          float64
	hasUnpriced   bool
}

// statuslineDay rolls up every record at or after start. It prices through report.Build
// and totals through sumCheckTotals -- the same path 'report' and 'check' use -- so the
// status line can never contradict them.
func statuslineDay(ctx context.Context, st *store.Store, start time.Time) (dayTotals, error) {
	rows, err := st.Usage(ctx, start)
	if err != nil {
		return dayTotals{}, err
	}
	table, err := pricing.Load()
	if err != nil {
		return dayTotals{}, err
	}
	totals := sumCheckTotals(report.Build(rows, table))
	out := dayTotals{tokens: totals.Tokens, cost: totals.Cost, hasUnpriced: totals.HasUnpriced}
	for i := range rows {
		out.lines += rows[i].LinesAdded
	}
	return out, nil
}

// costBasisSegment renders the money part, or "" when nothing honest can be said. On a
// flat plan the API-equivalent dollar figure is not spend, so month-to-date is shown
// against the plan price as two raw numbers: a percentage would read as "consumed",
// while a plan only pays off *above* its price.
func costBasisSegment(ctx context.Context, st *store.Store, now time.Time, p config.Pricing, today dayTotals, l *i18n.CLI) string {
	if p.IsSubscription() {
		if p.MonthlySubscriptionCost <= 0 {
			return ""
		}
		month, err := statuslineDay(ctx, st, startOfLocalMonth(now))
		if err != nil || month.cost == 0 {
			return ""
		}
		return "$" + humanize.USD(month.cost) + unpricedMark(month.hasUnpriced) +
			"/$" + humanize.USD(p.MonthlySubscriptionCost) + " " + l.StatuslineMonth
	}
	if today.cost == 0 {
		return ""
	}
	return "~$" + humanize.USD(today.cost) + unpricedMark(today.hasUnpriced) + " " + l.StatuslineAPIEquiv
}

// unpricedMark carries report's convention that a cost excluding unpriced models is a
// floor, not a total.
func unpricedMark(hasUnpriced bool) string {
	if hasUnpriced {
		return "*"
	}
	return ""
}

// statuslineAge reports how long ago the newest ingest ran -- how current the figures are,
// not when AI was last used. A store written before ingest tracking existed reports an
// unknown age rather than a wrong one.
func statuslineAge(ctx context.Context, st *store.Store, now time.Time, l *i18n.CLI) string {
	freshness, err := st.IngestFreshness(ctx)
	if err != nil || len(freshness) == 0 {
		return l.AgeUnknown
	}
	var newest time.Time
	for _, t := range freshness {
		if t.After(newest) {
			newest = t
		}
	}
	return agoLabel(now.Sub(newest), l)
}

func segments(parts ...string) string { return strings.Join(parts, " · ") }

func startOfLocalDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func startOfLocalMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}
