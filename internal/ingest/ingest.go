// Package ingest discovers local session files for each supported tool,
// parses them, and upserts the resulting usage records into the store.
package ingest

import (
	"context"
	"io"
	"time"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/parser/claude"
	"github.com/assaio/assaio/internal/parser/cline"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
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
}

type source struct {
	tool  string
	files []string
	parse func(io.Reader) ([]usage.Record, int, error)
}

// Run discovers every local session file for each supported tool, parses it, and
// upserts records. Inserts are idempotent, so Run is safe to repeat (backfill). An input
// this build already parsed unchanged is skipped and counted as Unchanged; opts.Full
// re-parses everything. A file that fails to open or parse is counted as Failed and does
// not abort the run; the remaining files for that tool, and the other tools, still get
// processed. sources overrides the built-in default log roots per tool (see
// config.Sources); a tool with no override discovers under its internal/paths default,
// unchanged from today. Configured plugins (see internal/plugin) run last, one Result
// each, and hold no file state: they produce records directly rather than being read.
//
//nolint:gocritic // sources is a small value bundle read once per backfill run, not a hot path.
func Run(ctx context.Context, home string, st *store.Store, sources config.Sources, plugins []config.PluginConfig, opts Options) ([]Result, error) {
	cache := make(projectCache)
	var results []Result
	seen := make(map[string]map[string]bool)

	sk, err := newSkipper(ctx, st, opts.Full)
	if err != nil {
		return nil, err
	}

	claudeMain, claudeSub, err := discoverClaude(home, sources)
	if err != nil {
		return nil, err
	}
	claudeResult, err := ingestClaude(ctx, st, sk, claudeMain, claudeSub, cache)
	if err != nil {
		return results, err
	}
	results = append(results, claudeResult)
	seen[claudeResult.Tool] = pathSet(claudeMain, claudeSub)

	discovered, err := discoverSources(home, sources)
	if err != nil {
		return results, err
	}
	for _, s := range discovered {
		res, err := ingestSource(ctx, st, sk, s, cache)
		if err != nil {
			return results, err
		}
		results = append(results, res)
		seen[s.tool] = pathSet(s.files)
	}

	clineDirs, err := discoverClineDirs(home, sources)
	if err != nil {
		return results, err
	}
	clineResult, err := ingestClineDirs(ctx, st, sk, clineDirs, cache)
	if err != nil {
		return results, err
	}
	results = append(results, clineResult)
	seen[clineResult.Tool] = pathSet(clineDirs)

	at := time.Now()
	if err := sk.flush(ctx, st, at); err != nil {
		return results, err
	}

	pluginResults, err := ingestPlugins(ctx, st, plugins)
	if err != nil {
		return results, err
	}
	if err := st.RecordSourceRun(ctx, sourceRuns(results, pluginResults, at)); err != nil {
		return results, err
	}
	if err := pruneVanished(ctx, st, seen); err != nil {
		return results, err
	}
	results = append(results, pluginResults...)

	return results, nil
}

// ingestInput skips one input this build already parsed unchanged, and otherwise parses
// and inserts it. Only a clean parse is recorded, so a file that fails keeps being retried
// and keeps being counted in Failed rather than disappearing from it on the next run.
func ingestInput(ctx context.Context, st *store.Store, sk *skipper, cache projectCache, res *Result,
	in input, parse func() ([]usage.Record, int, error),
) error {
	if sk.skip(in) {
		res.Unchanged++
		return nil
	}
	recs, skipped, err := parse()
	if insErr := ingestParsed(ctx, st, cache, res, recs, skipped, err); insErr != nil {
		return insErr
	}
	if err == nil {
		sk.record(in)
	}
	return nil
}

// ingestClaude parses and inserts every Claude transcript, both top-level and sub-agent.
// A sub-agent's own file is the authoritative record of its per-turn usage; the parent
// transcript's completed-sub-agent aggregate is only a last-turn summary (and is missing
// entirely for background/async Tasks), so any parent aggregate whose sub-agent has a file
// is suppressed to avoid double-counting. cache memoizes project resolution across files.
func ingestClaude(ctx context.Context, st *store.Store, sk *skipper, mainFiles, subFiles []string, cache projectCache) (Result, error) {
	covered := claude.CoveredAgents(subFiles)
	if err := dropSupersededAggregates(ctx, st, covered); err != nil {
		return Result{Tool: claudeTool}, err
	}
	res := Result{Tool: claudeTool, Files: len(mainFiles) + len(subFiles)}
	files := make([]string, 0, len(mainFiles)+len(subFiles))
	files = append(files, subFiles...)
	files = append(files, mainFiles...)
	for _, path := range files {
		err := ingestInput(ctx, st, sk, cache, &res, fileInput(path, res.Tool), func() ([]usage.Record, int, error) {
			recs, skipped, err := parseFile(path, claude.Parse)
			return claude.SuppressCovered(recs, covered), skipped, err
		})
		if err != nil {
			return res, err
		}
	}
	return res, nil
}

// ingestSource parses and inserts every file for one tool source, counting failed
// files without aborting the rest. cache memoizes project resolution across files.
func ingestSource(ctx context.Context, st *store.Store, sk *skipper, s source, cache projectCache) (Result, error) {
	res := Result{Tool: s.tool, Files: len(s.files)}
	for _, path := range s.files {
		err := ingestInput(ctx, st, sk, cache, &res, fileInput(path, s.tool), func() ([]usage.Record, int, error) {
			return parseFile(path, s.parse)
		})
		if err != nil {
			return res, err
		}
	}
	return res, nil
}

// ingestClineDirs parses and inserts every cline task directory, counting failed
// directories without aborting the rest. cache memoizes project resolution across dirs.
func ingestClineDirs(ctx context.Context, st *store.Store, sk *skipper, dirs []string, cache projectCache) (Result, error) {
	res := Result{Tool: "cline", Files: len(dirs)}
	for _, dir := range dirs {
		err := ingestInput(ctx, st, sk, cache, &res, dirInput(dir, res.Tool), func() ([]usage.Record, int, error) {
			return cline.ParseDir(dir)
		})
		if err != nil {
			return res, err
		}
	}
	return res, nil
}
