package cli

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

// statusRecentWindow marks a project or tool "recent" for the Hot/GoingStale/
// DormantTools sections of the dashboard.
const statusRecentWindow = 7 * 24 * time.Hour

// statusTopN caps the Hot and GoingStale lists shown in the dashboard.
const statusTopN = 5

func newStatusCmd() *cobra.Command {
	var since string
	c := &cobra.Command{
		Use:   "status",
		Short: "Show a local usage dashboard: hot projects, going-stale activity, dormant tools",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runStatus(cmd, &since)
		},
	}
	c.Flags().StringVar(&since, "since", "30d", "time window, e.g. 7d")
	addDBFlag(c)
	return c
}

func runStatus(cmd *cobra.Command, since *string) error {
	if err := resolveSince(cmd, since); err != nil {
		return err
	}
	dbPath, err := resolveDBPath(cmd)
	if err != nil {
		return err
	}
	if err := ensureParent(dbPath); err != nil {
		return err
	}
	st, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	n, err := st.Count(cmd.Context())
	if err != nil {
		return err
	}
	cmd.Printf("store: %s\n%d record(s)\n\n", dbPath, n)
	if n == 0 {
		return report.RenderEmptyStatusHint(cmd.OutOrStdout())
	}

	start, err := parseSinceAt(*since, time.Now())
	if err != nil {
		return err
	}
	usageRows, err := st.Usage(cmd.Context(), start)
	if err != nil {
		return err
	}
	sessionRows, err := st.Sessions(cmd.Context(), start)
	if err != nil {
		return err
	}
	table, err := pricing.Load()
	if err != nil {
		return err
	}

	insights := report.BuildInsights(usageRows, table, time.Now(), statusRecentWindow, statusTopN)
	if err := report.RenderStatusSummary(cmd.OutOrStdout(), &insights); err != nil {
		return err
	}
	if err := report.RenderChurnLine(cmd.OutOrStdout(), report.BuildChurn(usageRows)); err != nil {
		return err
	}
	return report.RenderSessionsBlock(cmd.OutOrStdout(), report.BuildSessionStats(sessionRows, time.Now()))
}
