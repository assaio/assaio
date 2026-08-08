package parser

// Rework returns the AI-added lines of file that removed undoes, drawn from a per-file
// budget of additions not yet undone -- so removing pre-existing (human or prior-session)
// code is never counted as rework, and two removals can never both claim the same added
// lines. The rework it returns is spent from unreworked[file] and added is folded in for
// later calls, which is what keeps a file's total rework at or below its total additions.
// Shared by every parser that tracks per-file add-then-remove thrash within one
// transcript; the map is expected to live only for the duration of one Parse call, and
// file is used only as its key, never stored.
func Rework(unreworked map[string]int64, file string, added, removed int64) int64 {
	rework := min(removed, unreworked[file])
	unreworked[file] += added - rework
	return rework
}
