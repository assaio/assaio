package docs

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestOutlineTakesTheRenderedIDs(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		want []Heading
	}{
		{
			name: "a repeated heading keeps the suffix the page carries",
			doc:  "# T\n\n## Setup\n\n## Setup\n\n## Run it\n",
			want: []Heading{{"setup", "Setup"}, {"setup-1", "Setup"}, {"run-it", "Run it"}},
		},
		{
			name: "h2 only",
			doc:  "# T\n\n## One\n\n### Deeper\n\n#### Deepest\n\n## Two\n",
			want: []Heading{{"one", "One"}, {"two", "Two"}},
		},
		{
			name: "inline markup dropped from the text, not from the id",
			doc:  "# T\n\n## `answers` — which *zeros*\n\n## B\n",
			want: []Heading{{"answers--which-zeros", "answers — which zeros"}, {"b", "B"}},
		},
		{
			name: "escapes and entities resolve as the heading renders them",
			doc:  "# T\n\n## A &amp; B \\_x\\_ `a\\_b`\n\n## C &lt;D&gt;\n",
			want: []Heading{{"a-amp-b-_x_-a_b", `A & B _x_ a\_b`}, {"c-ltdgt", "C <D>"}},
		},
		{
			name: "no h2",
			doc:  "# T\n\nText.\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, got, errs := Markdown("docs/t.md", tt.doc, Site{})
			if len(errs) > 0 {
				t.Fatal(errs)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("outline = %+v, want %+v", got, tt.want)
			}
			for _, h := range got {
				if !strings.Contains(body, `<h2 id="`+h.ID+`">`) {
					t.Errorf("outline names #%s, which no rendered h2 carries:\n%s", h.ID, body)
				}
			}
		})
	}
}

func TestTOCNeedsTwoSections(t *testing.T) {
	two := []Heading{{"a", "A & co"}, {"b", "B"}}
	tests := []struct {
		name     string
		headings []Heading
		want     string
	}{
		{"none", nil, ""},
		{"one", two[:1], ""},
		{"two", two, `<nav class="toc" aria-label="On this page"><p class="sidehead">On this page</p><ul>` +
			`<li><a href="#a">A &amp; co</a></li><li><a href="#b">B</a></li></ul></nav>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tocHTML(tt.headings); got != tt.want {
				t.Errorf("tocHTML = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOutlineAppearsOnlyOnGuides(t *testing.T) {
	root := t.TempDir()
	sections := "\n\n## One\n\n- [A](a.md)\n\n## Two\n\n- [B](b.md)\n"
	for name, body := range map[string]string{"index.md": "# Docs" + sections, "a.md": "# A" + sections, "b.md": "# B\n"} {
		if err := os.MkdirAll(filepath.Join(root, "docs"), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "docs", name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	guides, err := Guides(root)
	if err != nil {
		t.Fatal(err)
	}
	page := func(g *Guide) []byte {
		out, errs := GuidePage(root, g, guides)
		if len(errs) > 0 {
			t.Fatal(errs)
		}
		return out
	}
	tests := []struct {
		name string
		page []byte
		want bool
	}{
		{"guide with two sections", page(&guides[1]), true},
		{"guide with none", page(&guides[2]), false},
		{"landing", page(&guides[0]), false},
		{"reference", HTML(&Reference{}, guides), false},
		{"standalone reference", HTML(&Reference{}, nil), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := strings.Contains(string(tt.page), `class="toc"`); got != tt.want {
				t.Errorf("has an outline = %v, want %v", got, tt.want)
			}
		})
	}
}
