// Package ingest discovers local session files for each supported tool,
// parses them, and upserts the resulting usage records into the store.
package ingest

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/parser/claude"
	"github.com/assaio/assaio/internal/parser/cline"
	"github.com/assaio/assaio/internal/parser/codex"
	"github.com/assaio/assaio/internal/parser/gemini"
	"github.com/assaio/assaio/internal/paths"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

// Result summarizes one tool's ingest pass: files discovered, records parsed from them,
// records actually inserted (new rows, post-dedupe), lines skipped within otherwise-
// parsed files, and files that failed outright or midway through (Failed). A file that
// fails midway still contributes whatever it yielded before the failure to
// Records/Inserted/Skipped -- skip-and-count, never discard-on-error (see ingestParsed).
type Result struct {
	Tool                                      string
	Files, Records, Inserted, Skipped, Failed int
}

type source struct {
	tool  string
	files []string
	parse func(io.Reader) ([]usage.Record, int, error)
}

// Run discovers every local session file for each supported tool, parses it, and
// upserts records. Inserts are idempotent, so Run is safe to repeat (backfill). A file
// that fails to open or parse is counted as Failed and does not abort the run; the
// remaining files for that tool, and the other tools, still get processed. sources
// overrides the built-in default log roots per tool (see config.Sources); a tool with
// no override discovers under its internal/paths default, unchanged from today.
// Configured plugins (see internal/plugin) run last, one Result each.
//
//nolint:gocritic // sources is a small value bundle read once per backfill run, not a hot path.
func Run(ctx context.Context, home string, st *store.Store, sources config.Sources, plugins []config.PluginConfig) ([]Result, error) {
	discovered, err := discoverSources(home, sources)
	if err != nil {
		return nil, err
	}
	cache := make(projectCache)
	var results []Result
	for _, s := range discovered {
		res, err := ingestSource(ctx, st, s, cache)
		if err != nil {
			return results, err
		}
		results = append(results, res)
	}

	clineDirs, err := discoverClineDirs(home, sources)
	if err != nil {
		return results, err
	}
	clineResult, err := ingestClineDirs(ctx, st, clineDirs, cache)
	if err != nil {
		return results, err
	}
	results = append(results, clineResult)

	pluginResults, err := ingestPlugins(ctx, st, plugins)
	if err != nil {
		return results, err
	}
	results = append(results, pluginResults...)

	return results, nil
}

// discoverSources resolves each tool's roots — sources.<tool> if configured, else the
// internal/paths default — and discovers every session file under them.
//
//nolint:gocritic // sources is a small value bundle read once per backfill run, not a hot path.
func discoverSources(home string, sources config.Sources) ([]source, error) {
	var claudeFiles []string
	for _, root := range paths.Resolve(sources.Claude, paths.ClaudeRoot(home)) {
		found, err := claude.Discover(root)
		if err != nil {
			return nil, err
		}
		claudeFiles = append(claudeFiles, found...)
	}
	var codexFiles []string
	for _, root := range paths.Resolve(sources.Codex, paths.CodexRoots(home)...) {
		found, err := codex.Discover(root)
		if err != nil {
			return nil, err
		}
		codexFiles = append(codexFiles, found...)
	}
	var geminiFiles []string
	for _, root := range paths.Resolve(sources.Gemini, paths.GeminiRoot(home)) {
		found, err := gemini.Discover(root)
		if err != nil {
			return nil, err
		}
		geminiFiles = append(geminiFiles, found...)
	}
	return []source{
		{tool: "claude-code", files: claudeFiles, parse: claude.Parse},
		{tool: "codex", files: codexFiles, parse: codex.Parse},
		{tool: "gemini-cli", files: geminiFiles, parse: gemini.Parse},
	}, nil
}

// discoverClineDirs resolves the Cline roots — sources.cline if configured, else the
// internal/paths default — and discovers every task directory under them.
//
//nolint:gocritic // sources is a small value bundle read once per backfill run, not a hot path.
func discoverClineDirs(home string, sources config.Sources) ([]string, error) {
	var dirs []string
	for _, root := range paths.Resolve(sources.Cline, paths.ClineRoots(home)...) {
		found, err := cline.Discover(root)
		if err != nil {
			return nil, err
		}
		dirs = append(dirs, found...)
	}
	return dirs, nil
}

// ingestSource parses and inserts every file for one tool source, counting failed
// files without aborting the rest. cache memoizes project resolution across files.
func ingestSource(ctx context.Context, st *store.Store, s source, cache projectCache) (Result, error) {
	res := Result{Tool: s.tool, Files: len(s.files)}
	for _, path := range s.files {
		recs, skipped, err := parseFile(path, s.parse)
		if insErr := ingestParsed(ctx, st, cache, &res, recs, skipped, err); insErr != nil {
			return res, insErr
		}
	}
	return res, nil
}

// ingestClineDirs parses and inserts every cline task directory, counting failed
// directories without aborting the rest. cache memoizes project resolution across dirs.
func ingestClineDirs(ctx context.Context, st *store.Store, dirs []string, cache projectCache) (Result, error) {
	res := Result{Tool: "cline", Files: len(dirs)}
	for _, dir := range dirs {
		recs, skipped, err := cline.ParseDir(dir)
		if insErr := ingestParsed(ctx, st, cache, &res, recs, skipped, err); insErr != nil {
			return res, insErr
		}
	}
	return res, nil
}

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
	if len(recs) == 0 {
		return nil
	}
	resolveProjects(recs, cache)
	res.Records += len(recs)
	res.Skipped += skipped
	n, err := st.Insert(ctx, recs)
	if err != nil {
		return err
	}
	res.Inserted += n
	return nil
}
