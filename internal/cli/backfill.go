package cli

import (
	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/ingest"
	"github.com/assaio/assaio/internal/paths"
	"github.com/assaio/assaio/internal/store"
)

func newBackfillCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backfill",
		Short: "Import all historical local session logs into the store",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runBackfill(cmd)
		},
	}
}

func runBackfill(cmd *cobra.Command) error {
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
	results, err := ingest.Run(cmd.Context(), home, st, cfg.Sources, cfg.Plugins)
	if err != nil {
		return err
	}
	printBackfillResults(cmd, results)
	return nil
}

func printBackfillResults(cmd *cobra.Command, results []ingest.Result) {
	for _, r := range results {
		cmd.Printf("%-12s  files=%d  records=%d  inserted=%d", r.Tool, r.Files, r.Records, r.Inserted)
		if r.Skipped != 0 {
			cmd.Printf("  skipped=%d", r.Skipped)
		}
		if r.Failed != 0 {
			cmd.Printf("  failed=%d", r.Failed)
		}
		cmd.Println()
	}
}
