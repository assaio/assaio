package claude

import "github.com/assaio/assaio/internal/usage"

// markCompaction attributes a context-compaction event to out[lastAssistant] and reports
// whether the line marked one: isCompactSummary true, or subtype "compact_boundary" --
// Claude Code's two markers for a context overflow that got auto-summarized. Dropped, not
// attributed, when no assistant record has been emitted yet.
func markCompaction(out []usage.Record, lastAssistant int, isCompactSummary bool, subtype string) bool {
	if !isCompactSummary && subtype != "compact_boundary" {
		return false
	}
	if lastAssistant >= 0 {
		out[lastAssistant].Compactions++
	}
	return true
}
