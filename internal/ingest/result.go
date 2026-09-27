package ingest

import (
	"time"

	"github.com/assaio/assaio/internal/store"
)

// Result summarizes one tool's ingest pass: files discovered, files left unparsed because
// this build already read them unchanged (Unchanged), records parsed from the rest,
// records actually inserted (new rows, post-dedupe), lines skipped within otherwise-
// parsed files, files that failed outright or midway through (Failed), and how many of
// the parsed records carried no tokens at all (ZeroToken, the format-drift canary's
// input). A file that fails midway still contributes whatever it yielded before the
// failure to Records/Inserted/Skipped -- skip-and-count, never discard-on-error (see
// ingestParsed).
type Result struct {
	Tool                                                 string
	Files, Unchanged, Records, Inserted, Skipped, Failed int
	ZeroToken                                            int
	// Steps is how many new step rows this source contributed: the session as a sequence
	// rather than a total. 0 for a source that emits no timeline, which the depth matrix
	// declares rather than leaving as a bare zero.
	Steps int
	// PrunedSteps is how many stored steps this run dropped for being past the horizon. A
	// deletion nobody counts is the silent loss the whole skip-and-count policy exists against,
	// and this one is assaio deleting the user's history on its own initiative.
	PrunedSteps int64
	// HorizonSteps is how many steps this run read and did not store, for being older than the
	// horizon before they ever reached the store. Counted for the same reason PrunedSteps is,
	// one moment earlier: the first run on a source that gains a sequence reading reads a whole
	// history and keeps the horizon's worth of it, and evidence dropped in silence is the thing
	// skip-and-count exists to prevent. It is not Skipped -- nothing failed to be read.
	HorizonSteps int64
	// Lowered is how many stored rows this run restated *downward*. Assigned columns exist so a
	// corrected attribution rule can reach history, which means the store cannot tell a fix
	// landing from a parser regression erasing evidence -- so the count is reported rather than
	// left to be inferred from a figure that quietly shrank.
	Lowered int
	// Identity is what this run's re-reads replaced in stored identity columns, or declined to
	// report, for the same reason Lowered is counted.
	Identity store.IdentityChanges
	// StepsChanged is how many stored steps a re-read gave a different time, kind, position,
	// target or model.
	StepsChanged int
	// horizon is the oldest step this run keeps; steps older are never inserted. Unexported:
	// it is an input to the pass, not part of what the pass reports.
	horizon time.Time
}
