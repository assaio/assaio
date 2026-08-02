package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/paths"
	"github.com/assaio/assaio/internal/store"
)

func newClearCmd() *cobra.Command {
	var all, yes, labels bool
	var olderThan, tool string
	c := &cobra.Command{
		Use:   "clear",
		Short: "Delete stored usage data (all, older-than, or per-tool)",
		Long: `Delete stored usage records. Session labels are never touched by --all, --older-than or
--tool: they are the one thing in the store no re-import can rebuild, since a person typed
them. Delete them deliberately with --labels.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runClear(cmd, clearRequest{all: all, yes: yes, labels: labels, olderThan: olderThan, tool: tool})
		},
	}
	c.Flags().BoolVar(&all, "all", false, "delete all records")
	c.Flags().StringVar(&olderThan, "older-than", "", "delete records older than e.g. 90d; 0d means everything up to now")
	c.Flags().StringVar(&tool, "tool", "", "restrict to one tool (claude-code|codex|gemini-cli|cline)")
	c.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	c.Flags().BoolVar(&labels, "labels", false, "delete the session labels too (never removed by the other flags)")
	return c
}

// clearRequest is what one clear invocation asks for.
type clearRequest struct {
	all, yes, labels bool
	olderThan, tool  string
}

// targetsUsage reports whether any usage-record selector was given; --labels alone deletes
// only annotations.
func (r clearRequest) targetsUsage() bool { return r.all || r.olderThan != "" || r.tool != "" }

func runClear(cmd *cobra.Command, req clearRequest) error {
	if !req.targetsUsage() && !req.labels {
		return errors.New("specify --all, --older-than, --tool, or --labels")
	}
	if req.all && (req.olderThan != "" || req.tool != "") {
		return errors.New("--all cannot be combined with --older-than or --tool")
	}
	if req.tool != "" && !validClearTool(req.tool) {
		return fmt.Errorf("unknown tool %q (want claude-code|codex|gemini-cli|cline|plugin:<name>)", req.tool)
	}
	if !req.yes {
		return errors.New("refusing to delete without --yes")
	}
	before, err := clearCutoff(req.olderThan)
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
	if req.targetsUsage() {
		n, err := st.Clear(cmd.Context(), before, req.tool)
		if err != nil {
			return err
		}
		cmd.Printf("deleted %d record(s)\n", n)
	}
	if err := clearLabels(cmd, st, req); err != nil {
		return err
	}
	// Deleting rows frees pages inside the file without shrinking it, so someone who ran
	// clear to get disk space back has to be told it is not back yet.
	if size, sizeErr := st.Size(cmd.Context()); sizeErr == nil && size.Reclaimable > 0 {
		cmd.Printf("%s still held by the store — run 'assaio-agent compact' to reclaim it\n",
			humanize.Bytes(size.Reclaimable))
	}
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

// clearLabels deletes the annotations when --labels was passed, and otherwise says how many
// survived. Saying so is the point: they are unrecoverable, so a person who cleared the
// store needs to know they are still there rather than discovering it later.
func clearLabels(cmd *cobra.Command, st *store.Store, req clearRequest) error {
	if req.labels {
		n, err := st.DeleteLabels(cmd.Context())
		if err != nil {
			return err
		}
		cmd.Printf("deleted %d session label(s)\n", n)
		return nil
	}
	n, err := st.LabelCount(cmd.Context())
	if err != nil || n == 0 {
		return err
	}
	cmd.Printf("kept %d session label(s) -- no re-import can rebuild them; use --labels to delete them too\n", n)
	return nil
}
