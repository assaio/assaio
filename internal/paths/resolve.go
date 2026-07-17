package paths

import "os"

// Resolve returns configured if it is non-empty, replacing defaults entirely rather
// than merging with them — a partial override never silently combines with a stale
// built-in path. An empty configured (the common case) falls back to defaults.
func Resolve(configured []string, defaults ...string) []string {
	if len(configured) > 0 {
		return configured
	}
	return defaults
}

// Missing returns the subset of roots that do not exist on disk, so a caller can warn
// about a configured path that points nowhere without flagging an absent built-in
// default as an error — the tool may simply not be installed on this machine.
func Missing(roots []string) []string {
	var missing []string
	for _, root := range roots {
		if _, err := os.Stat(root); err != nil {
			missing = append(missing, root)
		}
	}
	return missing
}
