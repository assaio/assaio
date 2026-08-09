package survival

import (
	"context"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/vcs"
)

// TestUnreadableFilesAreCountedNotSilentlyDropped: a path git cannot blame skipped straight
// past a bare `continue`, so the rate was printed as a confident percentage over an unknown
// fraction of the window's files. "It did not survive" and "this build could not read it" are
// different facts, and only the second is a gap in the measurement -- the same skip-and-count
// policy every parser in this repo already follows.
func TestUnreadableFilesAreCountedNotSilentlyDropped(t *testing.T) {
	ctx := context.Background()
	dir := newRepo(t)
	writeFile(t, dir, "kept.go", "a\nb\nc\n")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "one")

	since := time.Now().Add(-time.Hour)
	commits, _, err := vcs.Collect(ctx, dir, since, time.Now(), "test")
	if err != nil {
		t.Fatal(err)
	}
	files, err := vcs.TouchedFiles(ctx, dir, since)
	if err != nil {
		t.Fatal(err)
	}
	// A path the working tree does not have: exactly what a deleted or renamed-away file
	// looks like from here, and exactly what a mis-decoded path looks like too.
	files = append(files, "gone-from-head.go")

	res, err := Analyze(ctx, dir, commits, files, 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.Files != 1 {
		t.Fatalf("Files = %d, want 1 blamed", res.Files)
	}
	if res.Unreadable != 1 {
		t.Fatalf("Unreadable = %d, want 1 -- the rate must say how many files it could not read", res.Unreadable)
	}
}
