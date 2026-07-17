package gemini

import "path/filepath"

// Discover returns chat log files under the Gemini CLI root. The glob is narrow
// (tmp/*/chats only) since ~/.gemini is shared with other tools (e.g. Antigravity).
func Discover(root string) ([]string, error) {
	return filepath.Glob(filepath.Join(root, "tmp", "*", "chats", "session-*.jsonl"))
}
