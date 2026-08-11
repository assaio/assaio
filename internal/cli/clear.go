package cli

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/parser"
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
them. Delete them deliberately with --labels, which follows the same scope as the deletion
it accompanies -- and only ever takes the label of a session the clear removes entirely.

A clear that is not time-scoped also forgets it ever read the inputs, so the next backfill
rebuilds what it deleted. --older-than keeps that memory on purpose: pruning history is a
request to forget records, not to re-import them.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runClear(cmd, clearRequest{all: all, yes: yes, labels: labels, olderThan: olderThan, tool: tool})
		},
	}
	c.Flags().BoolVar(&all, "all", false, "delete all records")
	c.Flags().StringVar(&olderThan, "older-than", "", "delete records older than e.g. 90d; 0d means everything up to now")
	c.Flags().StringVar(&tool, "tool", "", "restrict to one tool ("+clearToolChoices()+")")
	c.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	c.Flags().BoolVar(&labels, "labels", false, "delete the session labels in the same scope too (never removed by the other flags)")
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
		return fmt.Errorf("unknown tool %q (want %s)", req.tool, clearToolChoices())
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
	// Named before anything is deleted, because this command has no --db and cannot be
	// pointed anywhere else: whoever believed they were clearing a copy has exactly one
	// chance to see which file this is, and the deletion is not reversible.
	if n, countErr := st.Count(cmd.Context()); countErr == nil {
		cmd.Printf("clearing %s (%s records)\n", dbPath, humanize.Int(n))
	} else {
		cmd.Printf("clearing %s\n", dbPath)
	}
	// Labels first: their scope is read from the usage records the clear is about to
	// remove, so asking afterwards would find nothing to scope by.
	if err := clearLabels(cmd, st, req, before); err != nil {
		return err
	}
	if req.targetsUsage() {
		n, err := st.Clear(cmd.Context(), before, req.tool)
		if err != nil {
			return err
		}
		cmd.Printf("deleted %d record(s)\n", n)
		cmd.Println(reimportNote(before.IsZero()))
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
	if slices.Contains(parser.Tools(), tool) {
		return true
	}
	return strings.HasPrefix(tool, parser.PluginPrefix) && len(tool) > len(parser.PluginPrefix)
}

// clearToolChoices is what --tool accepts, spelled from the depth matrix so the help text
// and the error can never name a different set than the check does.
func clearToolChoices() string {
	return strings.Join(append(parser.Tools(), parser.PluginPrefix+"<name>"), "|")
}

// reimportNote says whether a plain re-import brings the deleted records back. A clear
// that dropped the ingest watermarks re-reads on the next backfill; a time-scoped one kept
// them on purpose, and only --full reaches past them.
func reimportNote(forgotInputs bool) string {
	if forgotInputs {
		return "the next 'assaio-agent backfill' re-reads those inputs from scratch"
	}
	return "pruned history is not re-imported by a plain 'assaio-agent backfill' -- use --full if you meant to rebuild it"
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
func clearLabels(cmd *cobra.Command, st *store.Store, req clearRequest, before time.Time) error {
	if req.labels {
		n, err := st.DeleteLabels(cmd.Context(), before, req.tool)
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
