package codex

import (
	"encoding/json"
	"strings"
)

// patchFileChange is one file's entry in a patch_apply_end's changes map. Codex writes it
// as a type-discriminated union: an `update` carries a unified diff, while an `add` (and
// symmetrically a `delete`) carries the file's whole body instead. The map key (an
// absolute file path) is used only to scope rework tracking in memory and is never copied
// onto a usage.Record (PRIVACY.md).
type patchFileChange struct {
	Type        string `json:"type"`
	UnifiedDiff string `json:"unified_diff"`
	Content     string `json:"content"`
}

// changeLineCounts returns one file change's added/removed lines. A malformed entry (not
// even a valid patchFileChange) contributes 0 rather than aborting the whole event.
func changeLineCounts(raw json.RawMessage) (added, removed int64) {
	var c patchFileChange
	if err := json.Unmarshal(raw, &c); err != nil {
		return 0, 0
	}
	switch c.Type {
	case "add":
		return contentLines(c.Content), 0
	case "delete":
		return 0, contentLines(c.Content)
	}
	return diffLineCounts(c.UnifiedDiff)
}

// contentLines counts the lines of a whole file body. A trailing newline terminates the
// last line rather than starting an empty one, so this is newlines plus an unterminated
// remainder -- what a diff of the same creation would have reported.
func contentLines(s string) int64 {
	if s == "" {
		return 0
	}
	n := int64(strings.Count(s, "\n"))
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}

// diffLineCounts counts a unified diff's "+"/"-" prefixed body lines. File headers
// ("--- a/x", "+++ b/x") and the hunk header ("@@ ... @@") share the "+"/"-" markers but are
// not body lines, so they must not count as added/removed lines. The grammar places the file
// headers before the first hunk header and every line after it in a hunk body, and that
// position is what separates them -- matching "--- " anywhere swallowed a real removed line
// of SQL, Lua, Haskell or Ada, whose comments begin exactly that way.
func diffLineCounts(diff string) (added, removed int64) {
	inHunk := false
	for _, ln := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(ln, "@@"):
			inHunk = true
		case !inHunk && (strings.HasPrefix(ln, "+++ ") || strings.HasPrefix(ln, "--- ")):
		case strings.HasPrefix(ln, "+"):
			added++
		case strings.HasPrefix(ln, "-"):
			removed++
		}
	}
	return added, removed
}
