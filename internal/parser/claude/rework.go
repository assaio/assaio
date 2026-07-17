package claude

import (
	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/usage"
)

// reworkTracker counts AI-added lines per file path within one transcript, so a later
// edit's removals in the same file can be recognized as undoing the AI's own earlier
// additions -- rework/thrash, not just churn. Keyed by file path in memory only for the
// lifetime of one Parse call: never copied onto a usage.Record and never stored
// (PRIVACY.md promises no paths are persisted).
type reworkTracker map[string]int64

// attribute folds one edit's added/removed line counts for file into out[lastAssistant],
// plus the ReworkLines portion computed by parser.Rework (shared with the Codex parser).
// Dropped entirely, like any edit, when no assistant record has been emitted yet.
func (t reworkTracker) attribute(out []usage.Record, lastAssistant int, file string, added, removed int64) {
	if lastAssistant < 0 {
		return
	}
	rework := parser.Rework(t, file, added, removed)
	out[lastAssistant].LinesAdded += added
	out[lastAssistant].LinesRemoved += removed
	out[lastAssistant].ReworkLines += rework
}
