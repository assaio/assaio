package ingest

import (
	"context"
	"time"

	"github.com/assaio/assaio/internal/parser/claude"
	"github.com/assaio/assaio/internal/store"
)

// ingestClaude parses and inserts every Claude transcript, both top-level and sub-agent.
// A sub-agent's own file is the authoritative record of its per-turn usage; the parent
// transcript's completed-sub-agent aggregate is only a last-turn summary (and is missing
// entirely for background/async Tasks), so any parent aggregate whose sub-agent has a file
// is suppressed to avoid double-counting. cache memoizes project resolution across files.
func ingestClaude(ctx context.Context, st *store.Store, sk *skipper, mainFiles, subFiles []string, cache projectCache, horizon time.Time) (Result, error) {
	covered := claude.CoveredAgents(subFiles)
	if err := dropSupersededAggregates(ctx, st, covered); err != nil {
		return Result{Tool: claudeTool}, err
	}
	res := Result{Tool: claudeTool, Files: len(mainFiles) + len(subFiles), horizon: horizon}
	files := make([]string, 0, len(mainFiles)+len(subFiles))
	files = append(files, subFiles...)
	files = append(files, mainFiles...)
	for _, path := range files {
		err := ingestInput(ctx, st, sk, cache, &res, fileInput(path, res.Tool), func() (parsed, error) {
			recs, steps, skipped, err := parseFile(path, claude.ParseAll)
			return parsed{Records: claude.SuppressCovered(recs, covered), Steps: steps, Skipped: skipped}, err
		})
		if err != nil {
			return res, err
		}
	}
	return res, nil
}
