package docs

import (
	"reflect"
	"strings"
	"testing"
)

var landingFound = map[string]string{
	"docs/index.md":       "Docs",
	"docs/a.md":           "A",
	"docs/b.md":           "B",
	"docs/recipes/c.md":   "C",
	"docs/extending/d.md": "D",
}

func TestReadPaths(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		want []readingPath
		err  string
	}{
		{
			name: "happy path, fragment stripped, prose links ignored",
			doc: "# Docs\n\nStart with [a](a.md) or [the reference](https://assaio.dev/docs/reference).\n\n" +
				"## First\n\nSee [b](b.md) first.\n\n- [A](a.md#setup)\n- [D](extending/d.md)\n\n" +
				"## Second\n\n- [C](recipes/c.md)\n- [B](./b.md)\n\nClosing words, [a](a.md) again.\n",
			want: []readingPath{
				{Title: "First", Sources: []string{"docs/a.md", "docs/extending/d.md"}},
				{Title: "Second", Sources: []string{"docs/recipes/c.md", "docs/b.md"}},
			},
		},
		{
			name: "a nested list is commentary, not membership",
			doc:  "# Docs\n\n## One\n\n- [A](a.md)\n  - see also [B](b.md)\n",
			want: []readingPath{{Title: "One", Sources: []string{"docs/a.md"}}},
		},
		{
			name: "a list under a ### is commentary, not membership",
			doc:  "# Docs\n\n## One\n\n- [A](a.md)\n\n### More\n\n- [B](b.md)\n",
			want: []readingPath{{Title: "One", Sources: []string{"docs/a.md"}}},
		},
		{
			name: "code in a path heading reads as plain text",
			doc:  "# Docs\n\n## Use `check`\n\n- [A](a.md)\n",
			want: []readingPath{{Title: "Use check", Sources: []string{"docs/a.md"}}},
		},
		{
			name: "a doc in two paths",
			doc:  "# Docs\n\n## One\n\n- [A](a.md)\n\n## Two\n\n- [A again](a.md#x)\n",
			err:  `docs/a.md is listed under "One" and again under "Two"`,
		},
		{
			name: "a link to a missing file",
			doc:  "# Docs\n\n## One\n\n- [Gone](gone.md)\n",
			err:  "no publishable document docs/gone.md exists",
		},
		{
			name: "a link to an excused file",
			doc:  "# Docs\n\n## One\n\n- [Map](README.md)\n",
			err:  "docs/README.md is excused in the unpublished list",
		},
		{
			name: "an empty path",
			doc:  "# Docs\n\n## One\n\nOnly a sentence, [a](a.md) in it.\n\n## Two\n\n- [A](a.md)\n",
			err:  `the path "One" lists no guide`,
		},
		{
			name: "a non-docs link in a list",
			doc:  "# Docs\n\n## One\n\n- [A](a.md)\n- [Site](https://assaio.dev/docs/reference)\n",
			err:  "is not a document under docs/",
		},
		{
			name: "a fragment-only link in a list",
			doc:  "# Docs\n\n## One\n\n- [A](a.md)\n- [Here](#one)\n",
			err:  "is not a document under docs/",
		},
		{
			name: "the landing linking itself",
			doc:  "# Docs\n\n## One\n\n- [A](a.md)\n- [Home](index.md)\n",
			err:  "which is this page",
		},
		{
			name: "a path named like a fixed group",
			doc:  "# Docs\n\n## Reference\n\n- [A](a.md)\n",
			err:  `the path "Reference" shares its name`,
		},
		{
			name: "a path with no title",
			doc:  "# Docs\n\n## \n\n- [A](a.md)\n",
			err:  "has a `##` with no title",
		},
		{
			name: "no path at all",
			doc:  "# Docs\n\n- [A](a.md)\n",
			err:  "has no `##` reading path",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readPaths(tt.doc, landingFound)
			if tt.err != "" {
				if err == nil || !strings.Contains(err.Error(), tt.err) || !strings.Contains(err.Error(), LandingSource) {
					t.Fatalf("err = %v, want one naming %s and containing %q", err, LandingSource, tt.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readPaths = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLandingPathsRenderAsCards(t *testing.T) {
	doc := "# Docs\n\nLede.\n\n## One\n\nWhy.\n\n- [A](a.md)\n\nAfter.\n"
	body, _, errs := Markdown(LandingSource, doc, Site{Published: map[string]string{"docs/a.md": "/docs/a"}})
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	want := "<p>Lede.</p>\n<div class=\"path\">\n<h2 id=\"one\">One</h2>\n<p>Why.</p>\n<ul>\n" +
		"<li><a href=\"/docs/a\">A</a></li>\n</ul>\n</div>\n<p>After.</p>"
	if !strings.Contains(body, want) {
		t.Errorf("landing body =\n%s\nwant it to contain\n%s", body, want)
	}
	if other, _, _ := Markdown("docs/a.md", doc, Site{Published: map[string]string{"docs/a.md": "/docs/a"}}); strings.Contains(other, `class="path"`) {
		t.Errorf("a guide other than the landing was wrapped in path cards:\n%s", other)
	}
}
