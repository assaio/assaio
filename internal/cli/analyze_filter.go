package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/label"
	"github.com/assaio/assaio/internal/store"
)

// addLabelFlags registers the annotation filters that let every metric be read per kind of
// work. See internal/label for the vocabularies and 'assaio-agent mark' for how a session
// gets one.
func addLabelFlags(c *cobra.Command) {
	c.Flags().String(label.Task, "", "only sessions labeled with this task class: "+label.Values(label.Task))
	c.Flags().String(label.Outcome, "", "only sessions labeled with this outcome: "+label.Values(label.Outcome))
	c.Flags().String(label.Difficulty, "", "only sessions labeled with this difficulty: "+label.Values(label.Difficulty))
}

// labelFilterFrom builds the store filter from those flags, rejecting a value outside its
// vocabulary rather than silently matching nothing -- a typo'd --task must not read as "you
// have no sessions of that kind".
func labelFilterFrom(cmd *cobra.Command) (store.LabelFilter, error) {
	var f store.LabelFilter
	for _, axis := range label.Names {
		v, _ := cmd.Flags().GetString(axis)
		if v == "" {
			continue
		}
		if !label.Valid(axis, v) {
			return store.LabelFilter{}, fmt.Errorf("unknown --%s %q (want %s)", axis, v, label.Values(axis))
		}
		switch axis {
		case label.Task:
			f.Task = v
		case label.Outcome:
			f.Outcome = v
		case label.Difficulty:
			f.Difficulty = v
		}
	}
	return f, nil
}

// narrowableValidators drops the ones whose answer belongs to the whole window and cannot
// be re-stated over a subset of it -- a flat monthly plan price, attribution pooled across
// every session, per-model turn counts. This is the same property analyze.WindowScoped
// already marks for the dashboard's per-project drill: comparing a window-wide constant
// against one slice of the window prints a verdict that contradicts the unfiltered one.
func narrowableValidators() (kept []analyze.Validator, skipped []string) {
	for _, v := range analyze.Validators() {
		if analyze.ProjectScoped(v) {
			kept = append(kept, v)
			continue
		}
		skipped = append(skipped, v.Name())
	}
	return kept, skipped
}

// describeFilter renders the active axes as "task=refactor outcome=done".
func describeFilter(f store.LabelFilter) string {
	return describeLabel(store.SessionLabel{Task: f.Task, Outcome: f.Outcome, Difficulty: f.Difficulty})
}

// renderNarrowing states what a filtered analysis is actually about before any verdict is
// printed: which labels, how much of the window they selected, and what could not be
// narrowed. Unselected sessions are excluded from this view, never from the data -- the
// unfiltered run still counts every one of them.
func renderNarrowing(cmd *cobra.Command, f store.LabelFilter, matched, total int, skipped []string) error {
	lw := &lineWriter{w: cmd.OutOrStdout()}
	lw.printf("filtered to %s · %d of %d sessions in this window\n", describeFilter(f), matched, total)
	if matched == 0 {
		lw.println("  nothing is labeled that way yet -- 'assaio-agent mark --list' shows what is.")
	}
	lw.println("  unlabeled sessions are excluded from this view, not from your data.")
	if len(skipped) > 0 {
		lw.printf("  not shown, because they describe the whole window and not a slice of it: %s\n",
			strings.Join(skipped, ", "))
	}
	lw.println("")
	return lw.err
}
