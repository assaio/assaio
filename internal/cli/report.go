package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

func newReportCmd() *cobra.Command {
	var since, format, by string
	c := &cobra.Command{
		Use:   "report",
		Short: "Print a token/cost report from stored usage",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runReport(cmd, &since, &format, by)
		},
	}
	c.Flags().StringVar(&since, "since", "30d", "time window, e.g. 7d")
	c.Flags().StringVar(&format, "format", "table", "output format: table|json|csv")
	c.Flags().StringVar(&by, "by", "day", "group by: day|project|tool|model|entrypoint|task|outcome|difficulty")
	c.Flags().Bool("compare", false, "show period-over-period top movers vs the previous equal window (renders a movers table, not --format)")
	addDBFlag(c)
	return c
}

func runReport(cmd *cobra.Command, since, format *string, by string) error {
	if err := resolveReportFlags(cmd, since, format); err != nil {
		return err
	}
	start, err := parseSinceAt(*since, time.Now())
	if err != nil {
		return err
	}
	st, err := openReportStore(cmd)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	if compare, _ := cmd.Flags().GetBool("compare"); compare {
		if err := compareFormatConflict(cmd); err != nil {
			return err
		}
		return runCompare(cmd, st, *since, by)
	}
	built, err := buildReport(cmd, st, start, by)
	if err != nil {
		return err
	}
	return renderReport(cmd, built, *format, by)
}

// resolveReportFlags fills since and format from config when the caller did not
// override them on the command line.
func resolveReportFlags(cmd *cobra.Command, since, format *string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	if !cmd.Flags().Changed("since") {
		*since = cfg.Since
	}
	if !cmd.Flags().Changed("format") {
		*format = cfg.Format
	}
	return nil
}

func buildReport(cmd *cobra.Command, st *store.Store, start time.Time, by string) ([]report.Row, error) {
	rows, err := usageForDim(cmd, st, start, by)
	if err != nil {
		return nil, err
	}
	table, err := pricing.Load()
	if err != nil {
		return nil, err
	}
	built := report.Build(rows, table)
	return report.Aggregate(built, by)
}

func renderReport(cmd *cobra.Command, built []report.Row, format, by string) error {
	switch format {
	case "table":
		if len(built) == 0 {
			cmd.Println(emptyStoreHint(cmd, "No usage found."))
			return nil
		}
		return report.RenderTable(cmd.OutOrStdout(), built, by)
	case "json":
		return report.RenderJSON(cmd.OutOrStdout(), built)
	case "csv":
		return report.RenderCSV(cmd.OutOrStdout(), built)
	default:
		return fmt.Errorf("unknown format %q (want table|json|csv)", format)
	}
}
