package claude

// markCompaction attributes a context-compaction event to the last assistant record and
// reports whether the line marked one: isCompactSummary true, or subtype "compact_boundary"
// -- Claude Code's two markers for a context overflow that got auto-summarized. Dropped, not
// attributed, when no assistant record has been emitted yet.
//
// The two markers are one event written as two adjacent lines, so only the first of a run is
// counted. On the maintainer's corpus every one of the 19 real compactions arrives as that
// pair and none arrives alone, so counting per marker reported 38 -- a friction signal at
// exactly twice its true value, with nothing outside the parser to check it against.
func (st *parseState) markCompaction(isCompactSummary bool, subtype string) bool {
	if !isCompactSummary && subtype != "compact_boundary" {
		st.inCompaction = false
		return false
	}
	if st.inCompaction {
		return true
	}
	st.inCompaction = true
	if st.last >= 0 {
		st.out[st.last].Compactions++
	}
	return true
}
