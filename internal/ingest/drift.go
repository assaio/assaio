package ingest

import (
	"time"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

// sourceRuns maps this pass's per-tool results onto the history the drift canaries read.
// fileResults come from discovered files; pluginResults do not, and the split matters:
// see pluginRun.
func sourceRuns(fileResults, pluginResults []Result, at time.Time) []store.SourceRun {
	runs := make([]store.SourceRun, 0, len(fileResults)+len(pluginResults))
	for i := range fileResults {
		runs = append(runs, fileRun(&fileResults[i], at))
	}
	for i := range pluginResults {
		runs = append(runs, pluginRun(&pluginResults[i], at))
	}
	return runs
}

// fileRun records what a file-backed source found and read. Parsed excludes the inputs
// this build already knew, so an incremental pass that read four files stays comparable
// to the full one that read six thousand.
func fileRun(r *Result, at time.Time) store.SourceRun {
	return store.SourceRun{
		Tool: r.Tool, RanAt: at,
		Discovered: r.Files, Parsed: r.Files - r.Unchanged,
		Records: r.Records, Skipped: r.Skipped, ZeroToken: r.ZeroToken,
	}
}

// pluginRun records a plugin's pass with no file counts at all. A plugin discovers and
// reads nothing: it emits records directly, and how many depends on what its tool did
// that day, so a records-per-file baseline would flag ordinary variation as drift. The
// shares that need no baseline -- skipped lines, zero-token records -- still judge it.
func pluginRun(r *Result, at time.Time) store.SourceRun {
	return store.SourceRun{
		Tool: r.Tool, RanAt: at,
		Records: r.Records, Skipped: r.Skipped, ZeroToken: r.ZeroToken,
	}
}

// zeroTokenCount counts records that carry no tokens at all. A vendor that renames or
// moves a token field leaves lines that still parse and still produce records, so this
// share is the only signal that separates silent under-reporting from a quiet week.
func zeroTokenCount(recs []usage.Record) int {
	n := 0
	for i := range recs {
		r := &recs[i]
		if r.InputTokens+r.OutputTokens+r.CacheReadTokens+r.CacheWriteTokens == 0 {
			n++
		}
	}
	return n
}
