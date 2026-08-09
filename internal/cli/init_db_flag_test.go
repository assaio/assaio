package cli

import "testing"

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
