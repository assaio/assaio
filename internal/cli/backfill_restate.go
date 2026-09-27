package cli

import (
	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/ingest"
	"github.com/assaio/assaio/internal/store"
)

// restateTotals sums, across sources, what a backfill's re-reads changed on rows already
// stored. Each is explained once below the per-source lines, because from the store's side a
// corrected rule and a parser regression look the same and only the reader knows which build
// changed what.
type restateTotals struct {
	lowered, stepsChanged int
	identity              store.IdentityChanges
}

func (t *restateTotals) add(r *ingest.Result) {
	t.lowered += r.Lowered
	t.stepsChanged += r.StepsChanged
	t.identity.Add(r.Identity)
}

func (t *restateTotals) print(cmd *cobra.Command) {
	if t.lowered > 0 {
		cmd.Printf("restated-down: %d stored row(s) had a figure lowered by this re-read.\n", t.lowered)
		cmd.Println("  That is what a corrected attribution rule looks like from here, and also what a parser")
		cmd.Println("  regression looks like. If this build did not change how a signal is counted, the drop is")
		cmd.Println("  the thing to explain — the store cannot tell the two apart, which is why it says so.")
	}
	if c := t.identity; c.Rows > 0 {
		cmd.Printf("identity-changed: %d stored row(s) got a different model, timestamp, project, entrypoint,\n", c.Rows)
		cmd.Printf("  branch or label from this re-read (model %d · ts %d · project %d · entrypoint %d · branch %d ·\n",
			c.Model, c.TS, c.Project, c.Entrypoint, c.Branch)
		cmd.Printf("  label %d). A corrected extraction rule and a parser regression look the same from here, and\n", c.Label)
		cmd.Println("  Cline can name a new model for a task that is still running. A model change moves cost —")
		cmd.Println("  'assaio-agent doctor' shows whether the new name has a price; an earlier timestamp moves the day")
		cmd.Println("  a row counts toward.")
	}
	if c := t.identity; c.Kept > 0 {
		cmd.Printf("identity-kept: %d stored row(s) kept a name this re-read no longer reported — what a parser\n", c.Kept)
		cmd.Println("  that stopped reading a field looks like.")
	}
	if t.stepsChanged > 0 {
		cmd.Printf("steps-changed: %d stored step(s) got a different time, kind, position, target or model from\n", t.stepsChanged)
		cmd.Println("  this re-read. A corrected extraction rule and a parser regression look the same from here.")
	}
}
