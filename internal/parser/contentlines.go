package parser

import "strings"

// ContentLines counts the lines of a whole file body. A trailing newline terminates the
// last line rather than starting an empty one, so this is newlines plus an unterminated
// remainder -- what a diff of the same creation would have reported.
//
// It is shared because two sources describe a creation the same way and a diff-only reader
// misses both: Codex writes `{"type":"add","content":…}` beside its unified diffs, and
// Claude Code writes `{"type":"create","content":…}` with an empty structuredPatch. Counting
// one and not the other is how the same defect shipped twice.
func ContentLines(s string) int64 {
	if s == "" {
		return 0
	}
	n := int64(strings.Count(s, "\n"))
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}
