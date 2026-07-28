package ingest

import (
	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/parser/claude"
	"github.com/assaio/assaio/internal/parser/cline"
	"github.com/assaio/assaio/internal/parser/codex"
	"github.com/assaio/assaio/internal/parser/gemini"
	"github.com/assaio/assaio/internal/paths"
)

// discoverSources resolves each tool's roots — sources.<tool> if configured, else the
// internal/paths default — and discovers every session file under them.
//
//nolint:gocritic // sources is a small value bundle read once per backfill run, not a hot path.
func discoverSources(home string, sources config.Sources) ([]source, error) {
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
		{tool: "codex", files: codexFiles, parse: codex.Parse},
		{tool: "gemini-cli", files: geminiFiles, parse: gemini.Parse},
	}, nil
}

// discoverClaude resolves the Claude roots and returns both the top-level session
// transcripts and the sub-agent transcripts found beneath them. They are discovered
// together so ingestClaude can reconcile the two — a completed sub-agent appears in both
// the parent (as a last-turn summary) and its own file (in full).
//
//nolint:gocritic // sources is a small value bundle read once per backfill run, not a hot path.
func discoverClaude(home string, sources config.Sources) (main, sub []string, err error) {
	for _, root := range paths.Resolve(sources.Claude, paths.ClaudeRoot(home)) {
		m, err := claude.Discover(root)
		if err != nil {
			return nil, nil, err
		}
		main = append(main, m...)
		s, err := claude.DiscoverSubagents(root)
		if err != nil {
			return nil, nil, err
		}
		sub = append(sub, s...)
	}
	return main, sub, nil
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
