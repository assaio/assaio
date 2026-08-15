package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/drift"
	"github.com/assaio/assaio/internal/store"
)

// driftWarnings evaluates every source's recorded ingest history and returns the canaries
// that fired, ordered by tool so output is stable from run to run. backfill and doctor
// both read it: one mechanism, so a warning can never appear in one and not the other.
func driftWarnings(ctx context.Context, st *store.Store) ([]drift.Warning, error) {
	history, err := st.SourceHistory(ctx)
	if err != nil {
		return nil, err
	}
	tools := make([]string, 0, len(history))
	for tool := range history {
		tools = append(tools, tool)
	}
	sort.Strings(tools)
	var out []drift.Warning
	for _, tool := range tools {
		out = append(out, drift.Evaluate(history[tool])...)
	}
	return out, nil
}

// printDriftWarnings writes the warnings to stderr, after backfill's per-source summary
// has already printed the shrunken numbers this is the only line to argue with.
func printDriftWarnings(cmd *cobra.Command, ws []drift.Warning) {
	for _, w := range ws {
		cmd.PrintErrf("warning: %s in %s (%s): %s\n", driftLead(w.Canary), w.Tool, w.Canary, w.Detail)
	}
}

// driftLead names what the canary observed. Four of them compare a source against its own
// history, where drift is the likely cause; barren reports a condition, and calling that
// "possible format drift" would name a cause its evidence cannot support -- these files may
// never have been the token source at all.
func driftLead(canary string) string {
	if canary == drift.Barren {
		return "nothing read from a detected source"
	}
	return "possible format drift"
}

// doctorDriftSection renders doctor's drift line and one indented line per fired canary.
func doctorDriftSection(ws []drift.Warning) string {
	if len(ws) == 0 {
		return "drift:        no canary fired\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "drift:        %d warning(s) — figures below may be under-reporting\n", len(ws))
	for _, w := range ws {
		fmt.Fprintf(&b, "  %s (%s): %s\n", w.Tool, w.Canary, w.Detail)
	}
	return b.String()
}

// strictFailures lists what makes doctor --strict exit non-zero: a fired canary, and a
// source the user explicitly configured that finds nothing. The second is the one drift no
// canary can catch -- a path that never worked has no history to have shrunk from -- so a
// typo in sources: would otherwise pass a cron job silently forever.
func strictFailures(ws []drift.Warning, scans []sourceScan) []string {
	var out []string
	for _, w := range ws {
		out = append(out, fmt.Sprintf("%s: %s (%s)", w.Tool, driftLead(w.Canary), w.Canary))
	}
	for _, sc := range scans {
		if len(sc.configured) > 0 && sc.found == 0 {
			out = append(out, fmt.Sprintf("%s: configured source found no inputs under %v", sc.tool, sc.roots))
		}
	}
	return out
}
