package ingest

import (
	"context"

	"github.com/assaio/assaio/internal/store"
)

// claudeTool is the Tool label the Claude Code parser stamps, named once because ingest uses
// it both as a Result label and as the scope of the superseded-aggregate cleanup.
const claudeTool = "claude-code"

// dropSupersededAggregates deletes any stored parent aggregate whose sub-agent transcript is
// now on disk. SuppressCovered keeps a new one out of this parse; it cannot reach a row an
// earlier parse already wrote, and that row summarizes the very turns the transcript
// contributes in full -- so both were counted until here. Idempotent, and a no-op on a store
// that never held one.
func dropSupersededAggregates(ctx context.Context, st *store.Store, covered map[string]struct{}) error {
	superseded, err := st.SupersededAggregates(ctx, claudeTool, covered)
	if err != nil {
		return err
	}
	_, err = st.DeleteDedupeKeys(ctx, claudeTool, superseded)
	return err
}
