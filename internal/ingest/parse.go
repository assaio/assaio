package ingest

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

func parseFile(path string, parse func(io.Reader) ([]usage.Record, int, error)) ([]usage.Record, int, error) {
	//nolint:gosec // paths come from local-home discovery globs
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return parse(f)
}

// ingestParsed folds one file's (or cline directory's) parse outcome into res. parseErr
// only ever marks the file Failed; it never discards recs, since a parser that hits a
// fatal condition partway through (e.g. a scanner error on a corrupt trailing line)
// still returns every record it recovered before that point, and skip-and-count means
// good data is inserted, not thrown away because the rest of the file was not (AGENTS.md).
func ingestParsed(ctx context.Context, st *store.Store, cache projectCache, res *Result, recs []usage.Record, skipped int, parseErr error) error {
	if parseErr != nil {
		res.Failed++
	}
	recs, undated := dated(recs)
	res.Skipped += skipped + undated
	if len(recs) == 0 {
		return nil
	}
	resolveProjects(recs, cache)
	res.Records += len(recs)
	res.ZeroToken += zeroTokenCount(recs)
	n, err := st.InsertLocal(ctx, recs)
	if err != nil {
		return err
	}
	res.Inserted += n
	return nil
}

// dated drops records a log gave no timestamp and reports how many went. Every report,
// every validator and every dashboard window is bounded by `ts >= ?`, so a record stamped
// with the zero time is stored and then invisible to all of them while still counting toward
// the store's size and its row totals -- present in one place, absent in every other. Counted
// as skipped, which is the honest word for evidence that could not be read and the number the
// drift canaries already watch.
func dated(recs []usage.Record) (kept []usage.Record, dropped int) {
	kept = recs[:0]
	for i := range recs {
		if recs[i].Timestamp.IsZero() {
			dropped++
			continue
		}
		kept = append(kept, recs[i])
	}
	return kept, dropped
}
