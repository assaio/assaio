package cli

import (
	"fmt"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/parser/agy"
	"github.com/assaio/assaio/internal/parser/claude"
	"github.com/assaio/assaio/internal/parser/cline"
	"github.com/assaio/assaio/internal/parser/codex"
	"github.com/assaio/assaio/internal/parser/copilot"
	"github.com/assaio/assaio/internal/parser/gemini"
	"github.com/assaio/assaio/internal/paths"
)

// sourceScan is one built-in source's discovery result: how many inputs sit on disk under
// the roots actually in effect, the label doctor prints for them, and where those roots
// came from. Discovery runs once per invocation, so a printed line and the --strict gate
// can never disagree about what was found.
type sourceScan struct {
	tool       string
	activity   string
	found      int
	configured []string
	roots      []string
	// err is the discovery failure this scan hit, if any. It is carried rather than dropped
	// because "nothing is there" and "we could not look" are different answers, and ingest
	// reports the second as an error while doctor used to print it as the first.
	err error
}

// scanSources counts every built-in source's inputs under its effective roots.
func scanSources(home string, s *config.Sources) []sourceScan {
	return []sourceScan{
		scanClaude(s.Claude, paths.ClaudeRoot(home)),
		scanSource("codex", "file", s.Codex, codex.Discover, paths.CodexRoots(home)...),
		scanSource("gemini-cli", "file", s.Gemini, gemini.Discover, paths.GeminiRoot(home)),
		scanSource("cline", "task", s.Cline, cline.Discover, paths.ClineRoots(home)...),
		scanSource("copilot-cli", "session", s.Copilot, copilot.Discover, paths.CopilotRoot(home)),
		scanSource("agy", "conversation", s.Agy, agy.Discover, paths.AgyRoot(home)),
	}
}

// scanSource counts one tool's inputs across the roots in effect.
func scanSource(tool, noun string, configured []string, discover func(string) ([]string, error), defaults ...string) sourceScan {
	roots := paths.Resolve(configured, defaults...)
	found := 0
	var firstErr error
	for _, root := range roots {
		f, err := discover(root)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		found += len(f)
	}
	return sourceScan{
		tool: tool, activity: toolActivityLabel(found, noun), found: found,
		configured: configured, roots: roots, err: firstErr,
	}
}

// scanClaude counts Claude's two kinds of transcript separately. ingest reads top-level
// sessions and the sub-agent transcripts beneath them, so a count of only the former hides
// the whole sub-agent surface -- thousands of files on a real machine, and exactly the
// usage v0.3.0 made exact.
func scanClaude(configured []string, defaults ...string) sourceScan {
	roots := paths.Resolve(configured, defaults...)
	var main, sub int
	var firstErr error
	for _, root := range roots {
		m, err := claude.Discover(root)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		s, err := claude.DiscoverSubagents(root)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		main += len(m)
		sub += len(s)
	}
	activity := toolActivityLabel(main, "file")
	if sub > 0 {
		activity += fmt.Sprintf(" + %d sub-agent transcript(s)", sub)
	}
	return sourceScan{
		tool: "claude-code", activity: activity, found: main + sub,
		configured: configured, roots: roots, err: firstErr,
	}
}

// toolActivityLabel renders a tool's detected count, so a zero count reads
// unambiguously as "not detected" rather than a bare, ambiguous zero.
func toolActivityLabel(n int, noun string) string {
	if n == 0 {
		return fmt.Sprintf("0 %s(s) — not detected", noun)
	}
	return fmt.Sprintf("%d %s(s)", n, noun)
}
