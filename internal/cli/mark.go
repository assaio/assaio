package cli

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/label"
	"github.com/assaio/assaio/internal/store"
)

func newMarkCmd() *cobra.Command {
	var since string
	var last, list, unmark bool
	c := &cobra.Command{
		Use:   "mark [session]",
		Short: "Label a session with what kind of work it was, how it ended, and how hard it was",
		Long: `Attach the one thing the logs cannot contain: what the work was for. A session can be
labeled with a task class, an outcome, and a difficulty, and every metric can then be read
per kind of work -- "did AI-written code hold up" means little averaged over research, a
refactor, and a greenfield feature.

With no session argument this marks the most recent session in the repository you are
standing in, which is the common case: label the work right after finishing it. Pass --last
for the most recent session anywhere, or a session id prefix to name one exactly.

Only the category values listed below are accepted -- never free text, never a prompt. The
labels stay on this machine: they are not sent by 'sync'. Sessions nobody labels stay fully
counted everywhere; an unlabeled session is not a failed one.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMark(cmd, args, markFlags{since: &since, last: last, list: list, unmark: unmark})
		},
	}
	c.Flags().String(label.Task, "", "what kind of work it was: "+label.Values(label.Task))
	c.Flags().String(label.Outcome, "", "how it ended: "+label.Values(label.Outcome))
	c.Flags().String(label.Difficulty, "", "how hard it was: "+label.Values(label.Difficulty))
	c.Flags().BoolVar(&last, "last", false, "target the most recent session in any project")
	c.Flags().BoolVar(&list, "list", false, "list recent sessions and their labels instead of marking")
	c.Flags().BoolVar(&unmark, "unmark", false, "remove the session's labels")
	c.Flags().StringVar(&since, "since", "7d", "window for --list, e.g. 30d")
	addDBFlag(c)
	return c
}

// markFlags carries the mode switches, so runMark's signature stays readable.
type markFlags struct {
	since  *string
	last   bool
	list   bool
	unmark bool
}

func runMark(cmd *cobra.Command, args []string, f markFlags) error {
	st, err := openReportStore(cmd)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	if f.list {
		return runMarkList(cmd, st, *f.since)
	}
	set, changed, err := requestedLabels(cmd)
	if err != nil {
		return err
	}
	if !f.unmark && !changed {
		return fmt.Errorf("nothing to set: pass --%s, --%s, --%s or --unmark (see 'mark --list')",
			label.Task, label.Outcome, label.Difficulty)
	}
	if f.unmark && changed {
		return errors.New("--unmark removes every label and cannot be combined with setting one")
	}
	ref, err := resolveMarkTarget(cmd, st, args, f.last)
	if err != nil {
		return err
	}
	if f.unmark {
		return unmarkSession(cmd, st, ref)
	}
	return markSession(cmd, st, ref, set)
}

// requestedLabels reads the three axis flags, validating each against its vocabulary.
// changed reports whether any was passed at all -- distinct from being passed empty, which
// is how a single axis is cleared without touching the others.
func requestedLabels(cmd *cobra.Command) (set store.SessionLabel, changed bool, err error) {
	for _, axis := range label.Names {
		if !cmd.Flags().Changed(axis) {
			continue
		}
		v, _ := cmd.Flags().GetString(axis)
		if !label.Valid(axis, v) {
			return set, false, fmt.Errorf("unknown --%s %q (want %s)", axis, v, label.Values(axis))
		}
		switch axis {
		case label.Task:
			set.Task = v
		case label.Outcome:
			set.Outcome = v
		case label.Difficulty:
			set.Difficulty = v
		}
		changed = true
	}
	return set, changed, nil
}

// markSession merges the requested axes over what is already stored, so setting an outcome
// later never erases the task set earlier, and writes the result.
func markSession(cmd *cobra.Command, st *store.Store, ref store.SessionRef, set store.SessionLabel) error {
	current, _, err := st.Label(cmd.Context(), ref)
	if err != nil {
		return err
	}
	merged := current
	merged.MarkedAt = time.Now()
	if cmd.Flags().Changed(label.Task) {
		merged.Task = set.Task
	}
	if cmd.Flags().Changed(label.Outcome) {
		merged.Outcome = set.Outcome
	}
	if cmd.Flags().Changed(label.Difficulty) {
		merged.Difficulty = set.Difficulty
	}
	if !merged.Set() {
		// Every axis was cleared, which is an unmark however it was spelled.
		return unmarkSession(cmd, st, ref)
	}
	if err := st.Mark(cmd.Context(), ref, merged); err != nil {
		return err
	}
	cmd.Printf("marked %s\n", describeSession(ref))
	cmd.Printf("  %s\n", describeLabel(merged))
	return nil
}

func unmarkSession(cmd *cobra.Command, st *store.Store, ref store.SessionRef) error {
	removed, err := st.Unmark(cmd.Context(), ref)
	if err != nil {
		return err
	}
	if !removed {
		cmd.Printf("%s carried no labels\n", describeSession(ref))
		return nil
	}
	cmd.Printf("unmarked %s\n", describeSession(ref))
	return nil
}
