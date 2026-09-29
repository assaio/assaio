// Package ingest discovers local session files for each supported tool,
// parses them, and upserts the resulting usage records into the store.
package ingest

import (
	"context"
	"io"
	"time"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

type source struct {
	tool  string
	files []string
	// parse returns both readings of one file. A source with no sequence reading returns no
	// steps; recordsOnly wraps the parsers that have one reading to give.
	parse func(io.Reader) ([]usage.Record, []usage.Step, int, error)
}

// recordsOnly adapts a parser that reads usage alone to the two-reading shape. It exists so the
// ingest path has one signature rather than two branches, and so a parser that later gains a
// sequence reading is wired by dropping this wrapper.
func recordsOnly(parse func(io.Reader) ([]usage.Record, int, error)) func(io.Reader) ([]usage.Record, []usage.Step, int, error) {
	return func(r io.Reader) ([]usage.Record, []usage.Step, int, error) {
		recs, skipped, err := parse(r)
		return recs, nil, skipped, err
	}
}

// dirSource is a source whose unit of work is a directory rather than a file: a Cline task
// holds two files, and an Antigravity conversation keeps its id in the directory name and
// nowhere inside the transcript.
type dirSource struct {
	tool  string
	dirs  []string
	parse func(dir string) ([]usage.Record, int, error)
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
	salt, err := st.RepositorySalt(ctx)
	if err != nil {
		return nil, err
	}
	cache := newProjectCache(salt)
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
	horizon := traceHorizon(opts, time.Now())
	pruned, err := pruneTrace(ctx, st, horizon)
	if err != nil {
		return nil, err
	}
	claudeResult, err := ingestClaude(ctx, st, sk, claudeMain, claudeSub, cache, horizon)
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
		res, err := ingestSource(ctx, st, sk, s, cache, horizon)
		if err != nil {
			return results, err
		}
		results = append(results, res)
		seen[s.tool] = pathSet(s.files)
	}

	dirSources, err := discoverDirSources(home, sources)
	if err != nil {
		return results, err
	}
	for _, s := range dirSources {
		res, err := ingestDirs(ctx, st, sk, s, cache, horizon)
		if err != nil {
			return results, err
		}
		results = append(results, res)
		seen[s.tool] = pathSet(s.dirs)
	}

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
	if pruned > 0 {
		// The prune is one delete over the whole store, so it belongs to the run rather than to
		// any one source. It is reported on the first result because that is where backfill
		// prints it; attributing it to that source would be a claim about which tool's history
		// went, which this figure does not know.
		results[0].PrunedSteps = pruned
	}

	return results, nil
}

// ingestInput skips one input this build already parsed unchanged, and otherwise parses
// and inserts it. Only a clean parse is recorded, so a file that fails keeps being retried
// and keeps being counted in Failed rather than disappearing from it on the next run.
func ingestInput(ctx context.Context, st *store.Store, sk *skipper, cache *projectCache, res *Result,
	in input, parse func() (parsed, error),
) error {
	if sk.skip(in) {
		res.Unchanged++
		return nil
	}
	p, err := parse()
	if insErr := ingestParsed(ctx, st, cache, res, p.Records, p.Skipped, err); insErr != nil {
		return insErr
	}
	if insErr := ingestSteps(ctx, st, res, p.Steps); insErr != nil {
		return insErr
	}
	if err == nil {
		sk.record(in)
	}
	return nil
}

// ingestSource parses and inserts every file for one tool source, counting failed
// files without aborting the rest. cache memoizes project resolution across files.
func ingestSource(ctx context.Context, st *store.Store, sk *skipper, s source, cache *projectCache, horizon time.Time) (Result, error) {
	res := Result{Tool: s.tool, Files: len(s.files), horizon: horizon}
	for _, path := range s.files {
		err := ingestInput(ctx, st, sk, cache, &res, fileInput(path, s.tool), func() (parsed, error) {
			recs, steps, skipped, err := parseFile(path, s.parse)
			return parsed{Records: recs, Steps: steps, Skipped: skipped}, err
		})
		if err != nil {
			return res, err
		}
	}
	return res, nil
}

// ingestDirs parses and inserts every directory-shaped input for one tool, counting failed
// directories without aborting the rest. cache memoizes project resolution across dirs.
func ingestDirs(ctx context.Context, st *store.Store, sk *skipper, s dirSource, cache *projectCache, horizon time.Time) (Result, error) {
	res := Result{Tool: s.tool, Files: len(s.dirs), horizon: horizon}
	for _, dir := range s.dirs {
		err := ingestInput(ctx, st, sk, cache, &res, dirInput(dir, res.Tool), func() (parsed, error) {
			recs, skipped, err := s.parse(dir)
			return parsed{Records: recs, Skipped: skipped}, err
		})
		if err != nil {
			return res, err
		}
	}
	return res, nil
}
