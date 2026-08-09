package codex

import "testing"

// TestDiffLineCountsSeparatesHeadersByPosition holds the unified-diff grammar: the file
// headers precede the first hunk header, and everything after one is a body line. Matching
// "--- " or "+++ " anywhere instead swallowed real body lines whose own content starts that
// way -- SQL, Lua, Haskell and Ada comments all begin `-- `, so removing one was silently not
// a removal. Unobserved on the audited corpus (0 of 349 real diffs); this is the rule, not a
// correction to a figure.
func TestDiffLineCountsSeparatesHeadersByPosition(t *testing.T) {
	tests := []struct {
		name        string
		diff        string
		wantAdded   int64
		wantRemoved int64
	}{
		{
			name:      "hunk-only diff, the shape codex emits",
			diff:      "@@ -1,2 +1,3 @@\n line1\n+added\n-removed\n",
			wantAdded: 1, wantRemoved: 1,
		},
		{
			name:      "file headers before the first hunk are not body lines",
			diff:      "--- a/x.go\n+++ b/x.go\n@@ -1,1 +1,1 @@\n-old\n+new\n",
			wantAdded: 1, wantRemoved: 1,
		},
		{
			name:      "a removed SQL comment is a removed line, not a header",
			diff:      "@@ -1,3 +1,1 @@\n-- drop this comment\n--- and this one\n line\n",
			wantAdded: 0, wantRemoved: 2,
		},
		{
			name:      "an added line beginning +++ counts",
			diff:      "@@ -1,1 +1,2 @@\n line\n+++ added marker\n",
			wantAdded: 1, wantRemoved: 0,
		},
		{
			name:      "empty diff",
			diff:      "",
			wantAdded: 0, wantRemoved: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			added, removed := diffLineCounts(tt.diff)
			if added != tt.wantAdded || removed != tt.wantRemoved {
				t.Errorf("diffLineCounts = (%d, %d), want (%d, %d)", added, removed, tt.wantAdded, tt.wantRemoved)
			}
		})
	}
}
