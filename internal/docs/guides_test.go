package docs

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestGuidesFollowTheLanding(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  []Guide
		err   string
	}{
		{
			name: "landing first, then each path in order",
			files: map[string]string{
				"docs/index.md":      "# Docs\n\n## Extend\n\n- [E](extending.md)\n- [R](recipes/r.md)\n",
				"docs/extending.md":  "# Extending\n",
				"docs/recipes/r.md":  "# Recipe\n",
				"docs/README.md":     "# Map\n",
				"docs/work/plan.md":  "# Plan\n",
				"docs/adr/0001-x.md": "# ADR\n",
			},
			want: []Guide{
				{Source: "docs/index.md", URL: "/docs", File: "site/docs.html", Title: "Docs", Group: GroupStart},
				{
					Source: "docs/extending.md", URL: "/docs/extending", File: "site/docs/extending.html",
					Title: "Extending", Group: "Extend",
				},
				{
					Source: "docs/recipes/r.md", URL: "/docs/recipes/r", File: "site/docs/recipes/r.html",
					Title: "Recipe", Group: "Extend",
				},
			},
		},
		{
			name: "a doc in no path",
			files: map[string]string{
				"docs/index.md": "# Docs\n\n## One\n\n- [A](a.md)\n",
				"docs/a.md":     "# A\n",
				"docs/b.md":     "# B\n",
			},
			err: "docs/b.md is publishable but no reading path in docs/index.md lists it",
		},
		{
			name: "a doc linked only from a nested list is in no path",
			files: map[string]string{
				"docs/index.md": "# Docs\n\n## One\n\n- [A](a.md)\n  - see also [B](b.md)\n",
				"docs/a.md":     "# A\n",
				"docs/b.md":     "# B\n",
			},
			err: "docs/b.md is publishable but no reading path in docs/index.md lists it",
		},
		{
			name: "a doc linked only under a ### is in no path",
			files: map[string]string{
				"docs/index.md": "# Docs\n\n## One\n\n- [A](a.md)\n\n### More\n\n- [B](b.md)\n",
				"docs/a.md":     "# A\n",
				"docs/b.md":     "# B\n",
			},
			err: "docs/b.md is publishable but no reading path in docs/index.md lists it",
		},
		{
			name:  "no landing",
			files: map[string]string{"docs/a.md": "# A\n"},
			err:   "docs/index.md is missing",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			for name, body := range tt.files {
				p := filepath.Join(root, filepath.FromSlash(name))
				if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			got, err := Guides(root)
			if tt.err != "" {
				if err == nil || !strings.Contains(err.Error(), tt.err) {
					t.Fatalf("err = %v, want %q", err, tt.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Guides =\n%+v\nwant\n%+v", got, tt.want)
			}
		})
	}
}
