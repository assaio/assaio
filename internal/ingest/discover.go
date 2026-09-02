package ingest

import (
	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/parser/agy"
	"github.com/assaio/assaio/internal/parser/claude"
	"github.com/assaio/assaio/internal/parser/cline"
	"github.com/assaio/assaio/internal/parser/codex"
	"github.com/assaio/assaio/internal/parser/copilot"
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
	var copilotFiles []string
	for _, root := range paths.Resolve(sources.Copilot, paths.CopilotRoot(home)) {
		found, err := copilot.Discover(root)
		if err != nil {
			return nil, err
		}
		copilotFiles = append(copilotFiles, found...)
	}
	return []source{
		{tool: "codex", files: codexFiles, parse: codex.Parse},
		{tool: "gemini-cli", files: geminiFiles, parse: gemini.Parse},
		{tool: "copilot-cli", files: copilotFiles, parse: copilot.Parse},
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

// discoverDirSources resolves the roots of every directory-shaped source — sources.<tool> if
// configured, else the internal/paths default — and discovers the directories under them.
//
//nolint:gocritic // sources is a small value bundle read once per backfill run, not a hot path.
func discoverDirSources(home string, sources config.Sources) ([]dirSource, error) {
	clineDirs, err := discoverDirs(cline.Discover, paths.Resolve(sources.Cline, paths.ClineRoots(home)...))
	if err != nil {
		return nil, err
	}
	agyDirs, err := discoverDirs(agy.Discover, paths.Resolve(sources.Agy, paths.AgyRoot(home)))
	if err != nil {
		return nil, err
	}
	return []dirSource{
		{tool: "cline", dirs: clineDirs, parse: cline.ParseDir},
		{tool: "agy", dirs: agyDirs, parse: agy.ParseDir},
	}, nil
}

func discoverDirs(discover func(string) ([]string, error), roots []string) ([]string, error) {
	var dirs []string
	for _, root := range roots {
		found, err := discover(root)
		if err != nil {
			return nil, err
		}
		dirs = append(dirs, found...)
	}
	return dirs, nil
}
