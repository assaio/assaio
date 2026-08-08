package store

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// uriHostilePathCases are directory names carrying a character that changes what a SQLite
// URI means. '?' is reserved in a Windows filename, so a path holding one cannot exist there
// to be opened wrongly in the first place.
func uriHostilePathCases() []string {
	if runtime.GOOS == "windows" {
		return []string{"plain", "has#hash", "has%percent"}
	}
	return []string{"plain", "has#hash", "has?question", "has%percent"}
}

// TestOpenUsesTheExactPathGiven is B120: the DSN was "file:" + path with nothing escaped,
// so SQLite's own URI parser read a '#' in the path as the start of a fragment and a '?'
// as the start of the query -- opening a database somewhere else and still returning
// success. Reachable from any XDG_DATA_HOME or home directory containing one of them.
func TestOpenUsesTheExactPathGiven(t *testing.T) {
	for _, dir := range uriHostilePathCases() {
		t.Run(dir, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), dir)
			if err := os.MkdirAll(root, 0o750); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "assaio.db")
			s, err := Open(path)
			if err != nil {
				t.Fatalf("Open(%q) = %v", path, err)
			}
			ctx := context.Background()
			if _, err := s.Insert(ctx, []usage.Record{{
				Tool: "codex", SessionID: "s1", Timestamp: time.Now().UTC(),
				Model: "m", InputTokens: 1, DedupeKey: "k1",
			}}); err != nil {
				t.Fatal(err)
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("no database at the path Open was given: %v", err)
			}
			reopened, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = reopened.Close() }()
			n, err := reopened.Count(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if n != 1 {
				t.Fatalf("Count after reopen = %d, want 1 (the row went to a different file)", n)
			}
		})
	}
}

// TestOpenAppliesItsPragmasWhateverThePathContains covers the second half: a '?' in the
// path truncated the DSN *and* took WAL, busy_timeout and foreign_keys with it, so the
// store silently ran without the durability and concurrency settings it declares.
func TestOpenAppliesItsPragmasWhateverThePathContains(t *testing.T) {
	for _, dir := range uriHostilePathCases() {
		t.Run(dir, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), dir)
			if err := os.MkdirAll(root, 0o750); err != nil {
				t.Fatal(err)
			}
			s, err := Open(filepath.Join(root, "assaio.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = s.Close() }()
			var mode string
			if err := s.db.QueryRowContext(context.Background(), `PRAGMA journal_mode`).Scan(&mode); err != nil {
				t.Fatal(err)
			}
			if mode != "wal" {
				t.Fatalf("journal_mode = %q, want wal", mode)
			}
			var fk int
			if err := s.db.QueryRowContext(context.Background(), `PRAGMA foreign_keys`).Scan(&fk); err != nil {
				t.Fatal(err)
			}
			if fk != 1 {
				t.Fatalf("foreign_keys = %d, want 1", fk)
			}
		})
	}
}
