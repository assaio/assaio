package claude

import "path/filepath"

// Discover returns all transcript files under the Claude projects root. The directory
// name is an opaque encoding of the cwd; callers read the real cwd from inside if needed.
func Discover(root string) ([]string, error) {
	return filepath.Glob(filepath.Join(root, "*", "*.jsonl"))
}
