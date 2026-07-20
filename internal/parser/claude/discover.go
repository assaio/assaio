package claude

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// Discover returns all top-level transcript files under the Claude projects root. The
// directory name is an opaque encoding of the cwd; callers read the real cwd from inside
// if needed. Sub-agent transcripts (see DiscoverSubagents) live one level deeper and are
// deliberately not returned here.
func Discover(root string) ([]string, error) {
	return filepath.Glob(filepath.Join(root, "*", "*.jsonl"))
}

// DiscoverSubagents returns every sub-agent transcript under root: the agent-<id>.jsonl
// files Claude Code writes beneath a session's subagents/ directory, including nested
// workflow sub-agents. Each carries the sub-agent's own per-turn usage; the parent
// transcript only summarizes a completed sub-agent (its last turn), and omits a
// background/async one entirely, so these files are the authoritative source of sub-agent
// cost. An unreadable directory is skipped rather than aborting discovery.
func DiscoverSubagents(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && isSubagentFile(path) {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// isSubagentFile reports whether path is an agent-<id>.jsonl transcript inside a
// subagents/ subtree (its .meta.json sidecar and any non-agent file are excluded).
func isSubagentFile(path string) bool {
	base := filepath.Base(path)
	if !strings.HasPrefix(base, "agent-") || !strings.HasSuffix(base, ".jsonl") {
		return false
	}
	dir := filepath.ToSlash(filepath.Dir(path))
	return strings.HasSuffix(dir, "/subagents") || strings.Contains(dir, "/subagents/")
}
