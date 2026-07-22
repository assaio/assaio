package cli

import (
	"errors"
	"fmt"
	"strings"
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
	if all && (olderThan != "" || tool != "") {
		return errors.New("--all cannot be combined with --older-than or --tool")
	}
	if tool != "" && !validClearTool(tool) {
		return fmt.Errorf("unknown tool %q (want claude-code|codex|gemini-cli|cline|plugin:<name>)", tool)
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

// validClearTool reports whether tool is a stored tool name clear can target: a built-in
// source or an out-of-tree plugin's "plugin:<name>" namespace. Rejecting anything else
// stops a typo (e.g. "claude" for "claude-code") from silently deleting nothing while
// reporting success.
func validClearTool(tool string) bool {
	switch tool {
	case "claude-code", "codex", "gemini-cli", "cline":
		return true
	}
	return strings.HasPrefix(tool, "plugin:") && len(tool) > len("plugin:")
}

// clearCutoff resolves --older-than to a cutoff time, or the zero time when unset.
func clearCutoff(olderThan string) (time.Time, error) {
	if olderThan == "" {
		return time.Time{}, nil
	}
	return parseSinceAt(olderThan, time.Now())
}
