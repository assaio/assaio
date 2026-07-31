package ingest

import (
	"context"

	"github.com/assaio/assaio/internal/drift"
	"github.com/assaio/assaio/internal/store"
)

// pathSet collects everything one source discovered this pass, the set its per-input state
// is trimmed back to.
func pathSet(groups ...[]string) map[string]bool {
	out := make(map[string]bool)
	for _, g := range groups {
		for _, p := range g {
			out[p] = true
		}
	}
	return out
}

// pruneVanished drops per-input state for files that are no longer on disk. Every vendor
// rotates its own logs, so without this the table would hold a row for every transcript
// ever seen and grow with install age rather than with what is actually there.
//
// A source whose discovery canary fired is skipped. "The files are gone" and "we stopped
// being able to find them" are indistinguishable from here, and discarding the state
// during exactly the failure being diagnosed would destroy the evidence and force a full
// re-parse once the root comes back.
func pruneVanished(ctx context.Context, st *store.Store, seen map[string]map[string]bool) error {
	history, err := st.SourceHistory(ctx)
	if err != nil {
		return err
	}
	for tool, keep := range seen {
		if lostDiscovery(history[tool]) {
			continue
		}
		if _, err := st.PruneIngestState(ctx, tool, keep); err != nil {
			return err
		}
	}
	return nil
}

// lostDiscovery reports whether this source's most recent pass found substantially fewer
// inputs than its history says it should.
func lostDiscovery(runs []store.SourceRun) bool {
	for _, w := range drift.Evaluate(runs) {
		if w.Canary == drift.Discovery {
			return true
		}
	}
	return false
}
