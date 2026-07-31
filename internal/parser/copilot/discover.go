package copilot

import "path/filepath"

// Discover returns every Copilot CLI session's event log under root. Each session owns a
// directory named for its id, and the accounting lives in that directory's events.jsonl;
// the sibling checkpoints/, files/ and research/ directories hold content assaio does not
// read.
func Discover(root string) ([]string, error) {
	return filepath.Glob(filepath.Join(root, "session-state", "*", "events.jsonl"))
}
