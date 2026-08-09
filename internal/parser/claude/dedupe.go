package claude

// seen reports whether this uuid has already been folded in, recording it if not. A
// transcript that repeats a line verbatim is a streamed retry, and the guard has to sit ahead
// of every line kind rather than only ahead of the assistant one: an edit result is where the
// counts are, so a repeated one attributed the same patch's added, removed and rework lines
// twice. Measured on 5,597 real transcripts: 329 repeated edit results, carrying 460 added and
// 656 removed lines, plus 5 repeated tool denials. A line with no uuid is always folded in --
// there is nothing to recognise it by.
func (st *parseState) seen(uuid string) bool {
	if uuid == "" {
		return false
	}
	if _, dup := st.seenLine[uuid]; dup {
		return true
	}
	st.seenLine[uuid] = struct{}{}
	return false
}
