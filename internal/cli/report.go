package cli

import (
	"fmt"
	"io"
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
	c.Flags().Bool("identify", false, "export raw member names instead of pseudonyms (team stores only; names individuals)")
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
	id := memberIdentity(cmd)
	built, err := buildReport(cmd, st, start, by, id)
	if err != nil {
		return err
	}
	return renderReport(cmd, built, *format, by, id)
}

// memberIdentity reads --identify. Pseudonymous is what an operator who did not ask for
// anything gets, on every format alike.
func memberIdentity(cmd *cobra.Command) report.MemberIdentity {
	if raw, _ := cmd.Flags().GetBool("identify"); raw {
		return report.MemberIdentified
	}
	return report.MemberPseudonymous
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

func buildReport(cmd *cobra.Command, st *store.Store, start time.Time, by string, id report.MemberIdentity) ([]report.Row, error) {
	rows, err := usageForDim(cmd, st, start, by)
	if err != nil {
		return nil, err
	}
	table, err := pricing.Load()
	if err != nil {
		return nil, err
	}
	built := report.Build(rows, table)
	if id == report.MemberIdentified {
		built = report.BuildIdentified(rows, table)
	}
	return report.Aggregate(built, by)
}

func renderReport(cmd *cobra.Command, built []report.Row, format, by string, id report.MemberIdentity) error {
	switch format {
	case "table":
		if len(built) == 0 {
			cmd.Println(emptyStoreHint(cmd, "No usage found."))
			return nil
		}
		if err := report.RenderTable(cmd.OutOrStdout(), report.CollapseForTable(built, by), by); err != nil {
			return err
		}
		// The disclosure reads the uncollapsed rows: the collapse drops Member, and a team
		// store still has to state which identity the operator asked for.
		return printMemberDisclosure(cmd.OutOrStdout(), built, id)
	case "json":
		if err := report.RenderJSON(cmd.OutOrStdout(), built); err != nil {
			return err
		}
		// The machine formats carry the note on stderr: a line inside the document would
		// corrupt the CSV and the JSON array that every script already parses.
		return printMemberDisclosure(cmd.ErrOrStderr(), built, id)
	case "csv":
		if err := report.RenderCSV(cmd.OutOrStdout(), built); err != nil {
			return err
		}
		return printMemberDisclosure(cmd.ErrOrStderr(), built, id)
	default:
		return fmt.Errorf("unknown format %q (want table|json|csv)", format)
	}
}

// printMemberDisclosure writes which member identity the export carries, and nothing at all
// when no row names one.
func printMemberDisclosure(w io.Writer, built []report.Row, id report.MemberIdentity) error {
	note := report.MemberDisclosure(built, id)
	if note == "" {
		return nil
	}
	_, err := fmt.Fprintln(w, note)
	return err
}
