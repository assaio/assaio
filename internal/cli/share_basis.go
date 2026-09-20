package cli

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/share"
	"github.com/assaio/assaio/internal/store"
)

// shareBasis reads what the card needs to know about the store behind the window that the
// window's own rows cannot say: when its counting sources were last read, and whether one of
// them stopped being found. Both come from the retained ingest runs.
func shareBasis(cmd *cobra.Command, st *store.Store, usage []store.UsageRow, now time.Time) (share.Basis, error) {
	runs, err := st.SourceHistory(cmd.Context())
	if err != nil {
		return share.Basis{}, err
	}
	tools := countingTools(usage)
	return share.Basis{
		ReadThrough:    readThrough(runs, tools, now),
		StoppedReading: stoppedReading(runs, tools),
	}, nil
}

// countingTools is every source in the window that answers a token or a line count, once each.
func countingTools(usage []store.UsageRow) []string {
	var out []string
	seen := map[string]bool{}
	for i := range usage {
		tool := usage[i].Tool
		if seen[tool] || !parser.Answers(tool, parser.SignalTokensTotal) && !parser.Answers(tool, parser.SignalLinesAdded) {
			continue
		}
		seen[tool] = true
		out = append(out, tool)
	}
	return out
}

// readThrough is the moment by which every counting source had last been read: the earliest of
// their latest runs. Zero when any of them has no readable run on record, which the card treats
// as a basis it cannot check. A window with no counting source has nothing left unread, so it
// is read through now and the figures' own dashes stand.
func readThrough(runs map[string][]store.SourceRun, tools []string, now time.Time) time.Time {
	if len(tools) == 0 {
		return now
	}
	var through time.Time
	for _, tool := range tools {
		r := runs[tool]
		if len(r) == 0 || r[len(r)-1].RanAt.IsZero() {
			return time.Time{}
		}
		if last := r[len(r)-1].RanAt; through.IsZero() || last.Before(through) {
			through = last
		}
	}
	return through
}

// stoppedReading reports a counting source whose latest retained run discovered no input while an
// earlier retained one did: a moved log directory or a parser that stopped finding files leaves
// the rows in place and the run recorded, so nothing else tells this apart from the work stopping.
// A source that never discovered anything is not this case -- an exec plugin discovers no files
// by construction.
func stoppedReading(runs map[string][]store.SourceRun, tools []string) bool {
	for _, tool := range tools {
		r := runs[tool]
		if len(r) < 2 || r[len(r)-1].Discovered > 0 {
			continue
		}
		for i := range r[:len(r)-1] {
			if r[i].Discovered > 0 {
				return true
			}
		}
	}
	return false
}
