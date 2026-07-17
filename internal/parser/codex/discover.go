package codex

import "path/filepath"

// Discover returns rollout files under one Codex sessions root (sessions or
// archived_sessions), which are date-partitioned YYYY/MM/DD directories.
func Discover(root string) ([]string, error) {
	return filepath.Glob(filepath.Join(root, "*", "*", "*", "rollout-*.jsonl"))
}
