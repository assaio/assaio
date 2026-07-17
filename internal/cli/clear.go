package cli

import (
	"errors"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/paths"
	"github.com/assaio/assaio/internal/store"
)

func newClearCmd() *cobra.Command {
	var all, yes bool
	var olderThan, tool string
	c := &cobra.Command{
		Use:   "clear",
		Short: "Delete stored usage data (all, older-than, or per-tool)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runClear(cmd, all, yes, olderThan, tool)
		},
	}
	c.Flags().BoolVar(&all, "all", false, "delete all records")
	c.Flags().StringVar(&olderThan, "older-than", "", "delete records older than e.g. 90d; 0d means everything up to now")
	c.Flags().StringVar(&tool, "tool", "", "restrict to one tool (claude-code|codex|gemini-cli|cline)")
	c.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	return c
}

func runClear(cmd *cobra.Command, all, yes bool, olderThan, tool string) error {
	if !all && olderThan == "" && tool == "" {
		return errors.New("specify --all, --older-than, or --tool")
	}
	if !yes {
		return errors.New("refusing to delete without --yes")
	}
	before, err := clearCutoff(olderThan)
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
	n, err := st.Clear(cmd.Context(), before, tool)
	if err != nil {
		return err
	}
	cmd.Printf("deleted %d record(s)\n", n)
	return nil
}

// clearCutoff resolves --older-than to a cutoff time, or the zero time when unset.
func clearCutoff(olderThan string) (time.Time, error) {
	if olderThan == "" {
		return time.Time{}, nil
	}
	return parseSinceAt(olderThan, time.Now())
}
