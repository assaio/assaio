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
	pruned := false
	var restated restateTotals
	for i := range results {
		r := &results[i]
		restated.add(r)
		cmd.Printf("%-12s  files=%d", r.Tool, r.Files)
		if r.Unchanged != 0 {
			cmd.Printf("  unchanged=%d", r.Unchanged)
		}
		cmd.Printf("  records=%d  inserted=%d", r.Records, r.Inserted)
		if r.Steps != 0 {
			cmd.Printf("  steps=%d", r.Steps)
		}
		if r.HorizonSteps != 0 {
			cmd.Printf("  steps-past-horizon=%d", r.HorizonSteps)
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
		if r.Identity.Rows != 0 {
			cmd.Printf("  identity-changed=%d", r.Identity.Rows)
		}
		if r.Identity.Kept != 0 {
			cmd.Printf("  identity-kept=%d", r.Identity.Kept)
		}
		if r.StepsChanged != 0 {
			cmd.Printf("  steps-changed=%d", r.StepsChanged)
		}
		if r.Failed != 0 {
			cmd.Printf("  failed=%d", r.Failed)
		}
		cmd.Println()
	}
	// Deleting rows frees pages inside the file without shrinking it.
	if pruned {
		cmd.Println("steps-pruned is the whole run's, not that source's: the horizon is applied to the store in one pass, which cannot say whose history went")
		cmd.Println("pruned steps free pages inside the store without shrinking it — run 'assaio-agent compact' to reclaim them")
	}
	restated.print(cmd)
}
