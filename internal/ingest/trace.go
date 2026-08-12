// Step-timeline side of ingestion: the horizon that bounds it and the insert that respects it.
package ingest

import (
	"context"
	"time"

	"github.com/assaio/assaio/internal/config"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

// parsed is one input's whole reading: the usage records, the step sequence behind them, and
// the lines neither could read. One struct because one pass produces both -- a source with no
// sequence simply leaves Steps empty.
type parsed struct {
	Records []usage.Record
	Steps   []usage.Step
	Skipped int
}

// ingestSteps stores one input's step sequence, dropping anything already past the horizon
// rather than inserting it for pruneTrace to delete moments later: insert-then-delete grew the
// SQLite file permanently and made backfill print a step count for rows that no longer existed
// when the command returned. A step the store rejects at its vocabulary boundary is counted as
// skipped, because silent loss is the one thing skip-and-count exists to prevent.
func ingestSteps(ctx context.Context, st *store.Store, res *Result, steps []usage.Step) error {
	if len(steps) == 0 {
		return nil
	}
	kept := steps[:0]
	for i := range steps {
		// A step with no timestamp is invisible to every window that reads it and would still
		// occupy a row, which is the same trade dated() refuses for records.
		if steps[i].Timestamp.IsZero() {
			res.Skipped++
			continue
		}
		if !res.horizon.IsZero() && steps[i].Timestamp.Before(res.horizon) {
			continue
		}
		kept = append(kept, steps[i])
	}
	n, rejected, err := st.InsertSteps(ctx, kept)
	if err != nil {
		return err
	}
	res.Steps += n
	res.Skipped += rejected
	return nil
}

// traceHorizon is the oldest step this run keeps, or the zero time when the configuration asks
// for no horizon at all. It is applied on every ingest rather than by a maintenance command,
// because the horizon is a size bound -- the timeline and its indexes occupy roughly twice what
// the usage table does, and a bound nobody applies is not a bound.
func traceHorizon(opts Options, at time.Time) time.Time {
	switch {
	case opts.TraceHorizonDays == 0:
		return time.Time{}
	case opts.TraceHorizonDays < 0:
		// Validate rejects this, but a lenient load only warns and carries on, and "the value
		// was ignored" must not turn into the most permissive setting there is.
		return at.AddDate(0, 0, -config.DefaultTraceHorizonDays)
	}
	return at.AddDate(0, 0, -opts.TraceHorizonDays)
}

// pruneTrace drops stored steps past the horizon and reports how many went, so a run that
// deletes history says so. A zero horizon means the configuration asked for none.
func pruneTrace(ctx context.Context, st *store.Store, horizon time.Time) (int64, error) {
	if horizon.IsZero() {
		return 0, nil
	}
	return st.PruneSteps(ctx, horizon)
}
