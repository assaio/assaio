package vcs

import "testing"

// TestNumstatPath locks the rename-path unwrapping so a renamed file is classified and
// blamed at its current name -- otherwise its lines count as added but never as surviving.
func TestNumstatPath(t *testing.T) {
	cases := map[string]string{
		"foo.go":                "foo.go",
		"old.go => new.go":      "new.go",
		"src/{old => new}/f.go": "src/new/f.go",
		"{a => b}":              "b",
	}
	for in, want := range cases {
		if got := numstatPath(in); got != want {
			t.Errorf("numstatPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsHash(t *testing.T) {
	cases := map[string]bool{
		"0dd2ba2c480515d8791021233aa7c706ea9a6f98": true,
		"author":  false,
		"":        false,
		"0dd2ba2": false,
		"0dd2ba2c480515d8791021233aa7c706ea9a6f9Z": false,
	}
	for in, want := range cases {
		if got := isHash(in); got != want {
			t.Errorf("isHash(%q) = %v, want %v", in, got, want)
		}
	}
}
