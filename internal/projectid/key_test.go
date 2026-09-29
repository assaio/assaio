package projectid

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var testSalt = []byte("0123456789abcdef0123456789abcdef")

func TestKey(t *testing.T) {
	base := t.TempDir()
	work := filepath.Join(base, "work", "api")
	scratch := filepath.Join(base, "scratch", "api")
	mustMkdirAll(t, filepath.Join(work, ".git"))
	mustMkdirAll(t, filepath.Join(scratch, ".git"))
	alias := filepath.Join(base, "alias")
	if err := os.Symlink(work, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	tests := []struct {
		name  string
		a, b  string
		saltB []byte
		same  bool
	}{
		{"same root twice", work, work, testSalt, true},
		{"same basename, different roots", work, scratch, testSalt, false},
		{"a symlinked alias shares its target's key", alias, work, testSalt, true},
		{"another store's salt gives another key", work, work, []byte("another salt, another store....."), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, b := Key(testSalt, tt.a), Key(tt.saltB, tt.b)
			if (a == b) != tt.same {
				t.Errorf("Key(%q) = %q, Key(%q) = %q; same = %v, want %v", tt.a, a, tt.b, b, a == b, tt.same)
			}
		})
	}
}

func TestKeyShapeHidesThePath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "payments-api")
	mustMkdirAll(t, filepath.Join(root, ".git"))
	k := Key(testSalt, root)
	if !strings.HasPrefix(k, KeyVersion) || len(k) != len(KeyVersion)+keyHexLen {
		t.Fatalf("Key = %q, want %q followed by %d hex characters", k, KeyVersion, keyHexLen)
	}
	if strings.Contains(k, "payments") {
		t.Fatalf("Key = %q carries part of the path", k)
	}
}

func TestKeyNeedsASaltAndARoot(t *testing.T) {
	if k := Key(nil, "/some/root"); k != "" {
		t.Errorf("Key(nil salt) = %q, want \"\": an unsalted key is a plain hash of a path", k)
	}
	if k := Key(testSalt, ""); k != "" {
		t.Errorf("Key(no root) = %q, want \"\"", k)
	}
}

// TestKeyIgnoresHowThePathWasSpelled covers a case- or normalization-insensitive filesystem,
// where a shell's $PWD keeps the spelling that was typed and a tool's getcwd returns the stored
// one. It skips on a filesystem that tells the spellings apart, where they are different
// directories.
func TestKeyIgnoresHowThePathWasSpelled(t *testing.T) {
	base := t.TempDir()
	// Stored decomposed (NFD) and capitalized; typed composed (NFC) and in lower case.
	storedDir, typedDir := "Za\u0301z\u0307o\u0301\u0142c\u0301", "z\u00e1\u017c\u00f3\u0142\u0107"
	stored := filepath.Join(base, storedDir, "Api")
	typed := filepath.Join(base, typedDir, "api")
	mustMkdirAll(t, filepath.Join(stored, ".git"))
	if _, err := os.Stat(typed); err != nil {
		t.Skip("this filesystem tells case or Unicode forms apart")
	}
	if a, b := Key(testSalt, typed), Key(testSalt, stored); a != b || a == "" {
		t.Fatalf("Key(%q) = %q, Key(%q) = %q; want one non-empty key", typed, a, stored, b)
	}
}

func TestKeyOfAMissingRootIsEmpty(t *testing.T) {
	if k := Key(testSalt, filepath.Join(t.TempDir(), "gone")); k != "" {
		t.Fatalf("Key(missing root) = %q, want \"\"", k)
	}
}
