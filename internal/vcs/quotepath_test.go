package vcs

import "testing"

// TestNumstatPathUnquotes covers git's core.quotePath output, which is on by default: any
// path byte outside ASCII arrives wrapped in quotes with octal escapes. Left as-is the string
// names no file, so `git blame` rejected it (its lines silently left the survival rate) and
// path.Ext read `.go"` (so the file counted as "other" rather than source).
func TestNumstatPathUnquotes(t *testing.T) {
	tests := []struct {
		name  string
		field string
		want  string
	}{
		{"plain ascii path", "internal/vcs/git.go", "internal/vcs/git.go"},
		{"quoted non-ascii path", `"caf\303\251.go"`, "café.go"},
		{"quoted non-ascii in a directory", `"sub/\303\274n\303\257.go"`, "sub/ünï.go"},
		{"rename with a quoted old side", `"caf\303\251.go" => cafe2.go`, "cafe2.go"},
		{"rename into a quoted new side", `plain.go => "dir/caf\303\251.go"`, "dir/café.go"},
		{"rename with both sides quoted", `"a/\303\251.go" => "b/\303\251.go"`, "b/é.go"},
		{"brace rename stays supported", "pre/{old => new}/post.go", "pre/new/post.go"},
		{"escaped quote inside a name", `"say\".go"`, `say".go`},
		{"unterminated quote is left alone", `"broken`, `"broken`},
		{"unparseable escape is left alone rather than mangled", `"bad\qescape"`, `"bad\qescape"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := numstatPath(tt.field); got != tt.want {
				t.Errorf("numstatPath(%q) = %q, want %q", tt.field, got, tt.want)
			}
		})
	}
}

// TestQuotedPathClassifiesBySuffix is the second half of the same defect: the trailing quote
// made every non-ASCII source file land in "other", so a window that rewrote source read as
// one that moved something uncategorized.
func TestQuotedPathClassifiesBySuffix(t *testing.T) {
	if got := classify(numstatPath(`"caf\303\251.go"`)); got != catSource {
		t.Errorf("classify = %q, want %q", got, catSource)
	}
}
