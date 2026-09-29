package projectid

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// canonical returns root as the filesystem stores it: symlinks resolved, then every component
// spelled the way its parent directory lists it. On a case- or normalization-insensitive
// filesystem (APFS by default) a directory opens under any case and under either Unicode form,
// and a shell's $PWD keeps whatever was typed, while a tool's getcwd returns the stored
// spelling; hashing both as given would make one repository two. ok is false when root cannot
// be resolved at all.
func canonical(root string) (string, bool) {
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", false
	}
	vol := filepath.VolumeName(real)
	out := vol + string(filepath.Separator)
	for _, part := range strings.Split(real[len(vol):], string(filepath.Separator)) {
		if part == "" {
			continue
		}
		out = filepath.Join(out, storedName(out, part))
	}
	return out, true
}

// storedName is name as dir lists it: the exact entry when there is one, else the only entry
// equal to it ignoring case and Unicode normalization, else name unchanged.
func storedName(dir, name string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return name
	}
	want := foldKey(name)
	match, matches := "", 0
	for _, e := range entries {
		if e.Name() == name {
			return name
		}
		if foldKey(e.Name()) == want {
			match = e.Name()
			matches++
		}
	}
	if matches == 1 {
		return match
	}
	return name
}

func foldKey(s string) string { return strings.ToLower(norm.NFC.String(s)) }
