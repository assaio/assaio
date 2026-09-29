package projectid

import (
	"path/filepath"
	"testing"
)

// TestStoredName reads the parent directory's listing, so it runs on every filesystem -- unlike a
// whole-path test, which needs one that opens a directory under another spelling.
func TestStoredName(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"Api", "Za\u0301z\u0307o\u0301\u0142c\u0301", "web"} {
		mustMkdirAll(t, filepath.Join(dir, name))
	}
	tests := []struct{ asked, want string }{
		{"Api", "Api"},
		{"api", "Api"},
		{"API", "Api"},
		{"z\u00e1\u017c\u00f3\u0142\u0107", "Za\u0301z\u0307o\u0301\u0142c\u0301"},
		{"web", "web"},
		{"missing", "missing"},
	}
	for _, tt := range tests {
		if got := storedName(dir, tt.asked); got != tt.want {
			t.Errorf("storedName(%q) = %q, want %q", tt.asked, got, tt.want)
		}
	}
}
