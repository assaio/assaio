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
	res.Skipped += skipped
	if len(recs) == 0 {
		return nil
	}
	resolveProjects(recs, cache)
	res.Records += len(recs)
	n, err := st.InsertLocal(ctx, recs)
	if err != nil {
		return err
	}
	res.Inserted += n
	return nil
}
