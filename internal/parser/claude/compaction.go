package claude

// markCompaction attributes a context-compaction event to the last assistant record and
// reports whether the line marked one: isCompactSummary true, or subtype "compact_boundary"
// -- Claude Code's two markers for a context overflow that got auto-summarized. Dropped, not
// attributed, when no assistant record has been emitted yet.
func (st *parseState) markCompaction(isCompactSummary bool, subtype string) bool {
	if !isCompactSummary && subtype != "compact_boundary" {
		return false
	}
	if st.last >= 0 {
		st.out[st.last].Compactions++
	}
	return true
}
