package parser

// Rework returns the AI-added lines of file that removed undoes, capped at what
// addedSoFar already recorded as added for file -- so removing pre-existing (human or
// prior-session) code is never counted as rework -- then folds added into
// addedSoFar[file] for later calls. Shared by every parser that tracks per-file
// add-then-remove thrash within one transcript; the map is expected to live only for the
// duration of one Parse call, and file is used only as its key, never stored.
func Rework(addedSoFar map[string]int64, file string, added, removed int64) int64 {
	rework := min(removed, addedSoFar[file])
	addedSoFar[file] += added
	return rework
}
