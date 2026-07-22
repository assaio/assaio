package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

func newEffectivenessCmd() *cobra.Command {
	var since, format, by string
	c := &cobra.Command{
		Use:   "effectiveness",
		Short: "Print AI-line output vs. cost per project, a directional efficiency signal",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runEffectiveness(cmd, &since, &format, by)
		},
	}
	c.Flags().StringVar(&since, "since", "30d", "time window, e.g. 7d")
	c.Flags().StringVar(&format, "format", "table", "output format: table|json|csv")
	c.Flags().StringVar(&by, "by", "project", "group by: project|tool|model|entrypoint|day|member")
	c.Flags().Bool("compare", false, "show period-over-period top movers vs the previous equal window (renders a movers table, not --format)")
	addDBFlag(c)
	return c
}

func runEffectiveness(cmd *cobra.Command, since, format *string, by string) error {
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
	built, err := buildEffectiveness(cmd, st, start, by)
	if err != nil {
		return err
	}
	return renderEffectiveness(cmd, built, *format, by)
}

func buildEffectiveness(cmd *cobra.Command, st *store.Store, start time.Time, by string) ([]report.EffRow, error) {
	rows, err := st.Usage(cmd.Context(), start)
	if err != nil {
		return nil, err
	}
	table, err := pricing.Load()
	if err != nil {
		return nil, err
	}
	return report.BuildEffectiveness(rows, table, by)
}

func renderEffectiveness(cmd *cobra.Command, built []report.EffRow, format, by string) error {
	switch format {
	case "table":
		if len(built) == 0 {
			cmd.Println(emptyStoreHint(cmd, "No usage found."))
			return nil
		}
		return report.RenderEffectivenessTable(cmd.OutOrStdout(), built, by)
	case "json":
		return report.RenderEffectivenessJSON(cmd.OutOrStdout(), built)
	case "csv":
		return report.RenderEffectivenessCSV(cmd.OutOrStdout(), built)
	default:
		return fmt.Errorf("unknown format %q (want table|json|csv)", format)
	}
}
