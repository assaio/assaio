package server

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/assaio/assaio/internal/paths"
)

// fixedPseudonymKey seeds report.Pseudonym's per-install secret so these tests pseudonymize
// deterministically instead of depending on whatever key is on the machine.
var fixedPseudonymKey = bytes.Repeat([]byte{0x5a}, 32)

// TestMain gives every test in this package a hermetic data directory. BuildDashboard and
// the GET / handler pseudonymize member/project labels, and report.Pseudonym persists a
// per-install secret to paths.DataDir(); without this seam the tests would create
// pseudonym.key in the real user's home dir.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "assaio-server-test")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("XDG_DATA_HOME", dir); err != nil {
		panic(err)
	}
	dataDir, err := paths.DataDir()
	if err != nil {
		panic(err)
	}
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		panic(err)
	}
	// Filename must match internal/report/anonymize.go's pseudonymKeyFilename.
	keyPath := filepath.Join(dataDir, "pseudonym.key")
	if err := os.WriteFile(keyPath, fixedPseudonymKey, 0o600); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
