package docs

import (
	"strings"
	"testing"
)

func TestPagerFollowsReadingOrder(t *testing.T) {
	guides := []Guide{
		{URL: "/docs", Title: "Docs", Group: GroupStart},
		{URL: "/docs/a", Title: "A", Group: "One"},
		{URL: "/docs/b", Title: "B", Group: "Two"},
	}
	tests := []struct {
		name       string
		guides     []Guide
		current    string
		prev, next string
	}{
		{"landing has no previous", guides, "/docs", "", "/docs/a"},
		{"a step across a path boundary", guides, "/docs/a", "/docs", "/docs/b"},
		{"last guide leads to the reference", guides, "/docs/b", "/docs/a", ReferenceURL},
		{"reference has no next", guides, ReferenceURL, "/docs/b", ""},
		{"no guides", nil, ReferenceURL, "", ""},
		{"only the reference's own sections", referenceSections, ReferenceURL, "", ""},
		{"a page outside the sequence", guides, "/docs/elsewhere", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prev, next := neighbours(tt.guides, tt.current)
			if got := urlOf(prev); got != tt.prev {
				t.Errorf("previous = %q, want %q", got, tt.prev)
			}
			if got := urlOf(next); got != tt.next {
				t.Errorf("next = %q, want %q", got, tt.next)
			}
		})
	}
}

func urlOf(g *Guide) string {
	if g == nil {
		return ""
	}
	return g.URL
}

func TestPagerMarkup(t *testing.T) {
	guides := []Guide{{URL: "/docs", Title: "Docs"}, {URL: "/docs/a", Title: "A & B"}}
	tests := []struct {
		name    string
		current string
		want    string
	}{
		{"landing", "/docs", `<nav class="pager" aria-label="Previous and next">` +
			`<a class="next" href="` + Host + `/docs/a"><span class="dir">Next</span> A &amp; B</a></nav>`},
		{"middle", "/docs/a", `<nav class="pager" aria-label="Previous and next">` +
			`<a class="prev" href="` + Host + `/docs"><span class="dir">Previous</span> Docs</a>` +
			`<a class="next" href="` + Host + ReferenceURL + `"><span class="dir">Next</span> Reference</a></nav>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pagerHTML(guides, tt.current); got != tt.want {
				t.Errorf("pagerHTML =\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
	if page := string(HTML(&Reference{}, nil)); strings.Contains(page, `class="pager"`) {
		t.Error("the standalone reference export carries a pager, which leads to pages it does not have")
	}
}
