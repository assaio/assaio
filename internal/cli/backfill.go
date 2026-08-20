package cli

import (
	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/ingest"
	"github.com/assaio/assaio/internal/paths"
	"github.com/assaio/assaio/internal/store"
)

func newBackfillCmd() *cobra.Command {
	var full bool
	c := &cobra.Command{
		Use:   "backfill",
		Short: "Import all historical local session logs into the store",
		Long: `Import local session logs into the store. Inputs this build already parsed
unchanged are skipped and reported as unchanged=, so a repeat run costs almost nothing.

A new build never trusts the previous one's state and re-reads everything once, which is
how history gains signals an older parser could not extract. Use --full to force that
re-read on the same build -- notably when working on a parser, since a local build keeps a
stable identity across rebuilds.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runBackfill(cmd, ingest.Options{Full: full})
		},
	}
	c.Flags().BoolVar(&full, "full", false, "re-parse every input, ignoring stored ingest state")
	return c
}

func runBackfill(cmd *cobra.Command, opts ingest.Options) error {
	home, err := paths.Home()
	if err != nil {
		return err
	}
	dbPath, err := paths.DBPath()
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
	cfg, err := loadConfigLenient(cmd)
	if err != nil {
		return err
	}
	opts.TraceHorizonDays = cfg.Trace.HorizonDays
	results, err := ingest.Run(cmd.Context(), home, st, cfg.Sources, cfg.Plugins, opts)
	if err != nil {
		return err
	}
	printBackfillResults(cmd, results)
	warnings, err := driftWarnings(cmd.Context(), st)
	if err != nil {
		return err
	}
	printDriftWarnings(cmd, warnings)
	return nil
}

func printBackfillResults(cmd *cobra.Command, results []ingest.Result) {
	pruned, lowered := false, 0
	for _, r := range results {
		lowered += r.Lowered
		cmd.Printf("%-12s  files=%d", r.Tool, r.Files)
		if r.Unchanged != 0 {
			cmd.Printf("  unchanged=%d", r.Unchanged)
		}
		cmd.Printf("  records=%d  inserted=%d", r.Records, r.Inserted)
		if r.Steps != 0 {
			cmd.Printf("  steps=%d", r.Steps)
		}
		if r.PrunedSteps != 0 {
			cmd.Printf("  steps-pruned=%d", r.PrunedSteps)
			pruned = true
		}
		if r.Skipped != 0 {
			cmd.Printf("  skipped=%d", r.Skipped)
		}
		if r.Lowered != 0 {
			cmd.Printf("  restated-down=%d", r.Lowered)
		}
		if r.Failed != 0 {
			cmd.Printf("  failed=%d", r.Failed)
		}
		cmd.Println()
	}
	// Deleting rows frees pages inside the file without shrinking it.
	if pruned {
		cmd.Println("pruned steps free pages inside the store without shrinking it — run 'assaio-agent compact' to reclaim them")
	}
	if lowered > 0 {
		cmd.Printf("restated-down: %d stored row(s) had a figure lowered by this re-read.\n", lowered)
		cmd.Println("  That is what a corrected attribution rule looks like from here, and also what a parser")
		cmd.Println("  regression looks like. If this build did not change how a signal is counted, the drop is")
		cmd.Println("  the thing to explain — the store cannot tell the two apart, which is why it says so.")
	}
}
