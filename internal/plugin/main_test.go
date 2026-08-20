package plugin

import (
	"os"
	"testing"
)

// TestMain gives every test in this package a hermetic data directory: member
// pseudonymization persists a per-install secret (internal/pseudonym), and without this a
// `go test` run would read and write the real user's data directory.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "assaio-plugin-test")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("XDG_DATA_HOME", dir); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
