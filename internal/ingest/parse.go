package ingest

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

// ingestParsed folds one file's (or cline directory's) parse outcome into res. parseErr
// only ever marks the file Failed; it never discards recs, since a parser that hits a
// fatal condition partway through (e.g. a scanner error on a corrupt trailing line)
// still returns every record it recovered before that point, and skip-and-count means
// good data is inserted, not thrown away because the rest of the file was not (AGENTS.md).
func ingestParsed(ctx context.Context, st *store.Store, cache projectCache, res *Result, recs []usage.Record, skipped int, parseErr error) error {
	if parseErr != nil {
		res.Failed++
	}
	recs, undated := dated(recs, time.Now())
	res.Skipped += skipped + undated
	if len(recs) == 0 {
		return nil
	}
	resolveProjects(recs, cache)
	res.Records += len(recs)
	res.ZeroToken += zeroTokenCount(recs)
	n, w, err := st.InsertLocal(ctx, recs)
	if err != nil {
		return err
	}
	res.Inserted += n
	res.Lowered += w.Lowered
	res.Identity.Add(w.Identity)
	return nil
}

// dated drops records a log gave no plausible timestamp -- none, or one outside the range the
// sync and plugin boundaries already enforce (usage.CheckTimestamp) -- and reports how many
// went. Every report, validator and dashboard window is bounded by `ts >= ?`, so a record
// stamped with the zero time is stored and then invisible to all of them while still counting
// toward the store's size and its row totals; and a Claude Code row keeps the earliest time a
// re-read offers, so an implausibly early one would never be corrected. Counted as skipped,
// which is the honest word for evidence that could not be read and the number the drift
// canaries already watch.
func dated(recs []usage.Record, now time.Time) (kept []usage.Record, dropped int) {
	kept = recs[:0]
	for i := range recs {
		if usage.CheckTimestamp(recs[i].Timestamp, now) != nil {
			dropped++
			continue
		}
		kept = append(kept, recs[i])
	}
	return kept, dropped
}

// parseFile reads one input for both of its readings at once -- the usage records and the step
// sequence -- from a single open and a single scan.
func parseFile(path string, parse func(io.Reader) ([]usage.Record, []usage.Step, int, error)) ([]usage.Record, []usage.Step, int, error) {
	//nolint:gosec // paths come from local-home discovery globs
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return parse(f)
}
