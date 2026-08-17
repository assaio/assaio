package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInitHasNoDBFlag: init imports through backfill, which only ever writes this machine's
// own store, while its report half resolved --db. Accepting the flag therefore promised a
// choice the command could not honour -- it filled one database and then reported an empty
// first run from another. The flag is gone rather than threaded through, because a first-run
// wizard has no reason to point at somebody else's store.
func TestInitHasNoDBFlag(t *testing.T) {
	if f := newInitCmd().Flags().Lookup("db"); f != nil {
		t.Fatal("init must not define --db: it cannot import into anything but the local store")
	}
}

// TestBackfillHasNoDBFlag is the same contract from the other side, and the reason above:
// if backfill ever gains one, init's own removal has to be revisited with it.
func TestBackfillHasNoDBFlag(t *testing.T) {
	if f := newBackfillCmd().Flags().Lookup("db"); f != nil {
		t.Fatal("backfill must not define --db: ingest writes paths.DBPath() unconditionally")
	}
}

// TestNoCommandBothImportsAndTakesDBFlag generalizes the two above, because naming commands
// one at a time is what let `share` reintroduce the defect: it called runBackfill and
// registered addDBFlag, and both tests above stayed green. The pairing is the bug whoever
// writes the next importing command will reach for, so the test looks for the pairing rather
// than for a command it already knows about.
//
// Why the pairing is worse than a wrong report: ingest prunes trace steps past the horizon
// before it parses anything, against the local store, and a source that has already deleted
// its transcripts cannot refill them -- so the flag turns "read somewhere else" into an
// unrequested, irreversible delete here.
func TestNoCommandBothImportsAndTakesDBFlag(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatal(err)
		}
		body := string(src)
		// backfill.go declares runBackfill; every other mention is a call.
		if name != "backfill.go" && strings.Contains(body, "runBackfill(") && strings.Contains(body, "addDBFlag(") {
			t.Errorf("%s both imports through runBackfill and registers --db: the import always writes paths.DBPath(), so the flag promises a store it will not fill and prunes one it was not pointed at", name)
		}
	}
}
