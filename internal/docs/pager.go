package docs

import "strings"

// readingSequence is every page in the order a reader is led through them: the guides as the
// landing orders them, then the reference. Fragment entries are the standalone reference's own
// sections, not pages, so a list of only those is no sequence at all.
func readingSequence(guides []Guide) []Guide {
	var seq []Guide
	for _, g := range guides {
		if !strings.HasPrefix(g.URL, "#") {
			seq = append(seq, g)
		}
	}
	if len(seq) == 0 {
		return nil
	}
	return append(seq, Guide{URL: ReferenceURL, Title: "Reference"})
}

// neighbours returns the pages before and after current in the reading sequence; either is nil
// at an end, and both are when current is not in it.
func neighbours(guides []Guide, current string) (prev, next *Guide) {
	seq := readingSequence(guides)
	for i := range seq {
		if seq[i].URL != current {
			continue
		}
		if i > 0 {
			prev = &seq[i-1]
		}
		if i+1 < len(seq) {
			next = &seq[i+1]
		}
		return prev, next
	}
	return nil, nil
}

// pagerHTML links the previous and next pages, absolute like the sidebar so a page opened from
// disk still leads somewhere.
func pagerHTML(guides []Guide, current string) string {
	prev, next := neighbours(guides, current)
	if prev == nil && next == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<nav class="pager" aria-label="Previous and next">`)
	if prev != nil {
		b.WriteString(`<a class="prev" href="` + Host + prev.URL + `"><span class="dir">Previous</span> ` +
			esc(prev.Title) + `</a>`)
	}
	if next != nil {
		b.WriteString(`<a class="next" href="` + Host + next.URL + `"><span class="dir">Next</span> ` +
			esc(next.Title) + `</a>`)
	}
	b.WriteString(`</nav>`)
	return b.String()
}
