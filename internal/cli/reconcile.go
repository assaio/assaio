package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/reconcile"
	"github.com/assaio/assaio/internal/report"
)

func newReconcileCmd() *cobra.Command {
	var since, format, mapping string
	c := &cobra.Command{
		Use:   "reconcile <export.csv|export.json>",
		Short: "Compare a vendor's own billing export against assaio's estimate, offline",
		Long: `Reads a usage or billing export you downloaded yourself and reports where its
total and assaio's estimate disagree, what part of that has evidence behind it, and what is
left unexplained.

No credential and no network: the export is a local file. Nothing is adjusted to make the
two sides agree -- a delta assaio cannot explain is the result, not an error.`,
		Example: `  assaio-agent reconcile usage.csv --since 30d
  assaio-agent reconcile usage.csv --map cost=amount,day=usage_date
  assaio-agent reconcile usage.json --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReconcile(cmd, args[0], &since, &format, mapping)
		},
	}
	c.Flags().StringVar(&since, "since", "30d", "time window, e.g. 7d")
	c.Flags().StringVar(&format, "format", "text", "output format: text|json")
	c.Flags().StringVar(&mapping, "map", "", "bind export columns explicitly, e.g. cost=amount,day=usage_date")
	addDBFlag(c)
	return c
}

func runReconcile(cmd *cobra.Command, path string, since, format *string, mapping string) error {
	overrides, err := reconcile.ParseMap(mapping)
	if err != nil {
		return err
	}
	now := time.Now()
	start, err := parseSinceAt(*since, now)
	if err != nil {
		return err
	}
	exp, err := reconcile.ReadFile(path, overrides)
	if err != nil {
		return err
	}
	rows, err := estimateRows(cmd, start)
	if err != nil {
		return err
	}
	res, err := reconcile.Reconcile(exp, rows, start, now)
	if err != nil {
		return err
	}
	switch *format {
	case "text":
		return reconcile.RenderText(cmd.OutOrStdout(), res)
	case "json":
		return reconcile.RenderJSON(cmd.OutOrStdout(), res)
	default:
		return fmt.Errorf("unknown format %q (want text|json)", *format)
	}
}

// estimateRows loads the window's local usage priced at the vendored table -- the same
// build every other cost surface reports from, so a reconciliation checks the shipped
// estimate rather than one computed specially for it.
func estimateRows(cmd *cobra.Command, start time.Time) ([]report.Row, error) {
	st, err := openReportStore(cmd)
	if err != nil {
		return nil, err
	}
	defer func() { _ = st.Close() }()
	usage, err := st.Usage(cmd.Context(), start)
	if err != nil {
		return nil, err
	}
	table, err := pricing.Load()
	if err != nil {
		return nil, err
	}
	return report.Build(usage, table), nil
}
