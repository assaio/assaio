package ingest

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/assaio/assaio/internal/store"
)

// threeSessions writes three Claude transcripts under a fresh home and returns their paths.
func threeSessions(t *testing.T) (home string, sessions []string) {
	t.Helper()
	home = t.TempDir()
	for _, name := range []string{"s1.jsonl", "s2.jsonl", "s3.jsonl"} {
		p := filepath.Join(home, ".claude", "projects", "-home-dev-app", name)
		write(t, p, claudeTurn)
		sessions = append(sessions, p)
	}
	return home, sessions
}

// knownPaths returns the inputs this build still has ingest state for.
func knownPaths(t *testing.T, st *store.Store) map[string]bool {
	t.Helper()
	state, err := st.IngestState(context.Background(), buildIdentity())
	if err != nil {
		t.Fatal(err)
	}
	out := make(map[string]bool, len(state))
	for path := range state {
		out[path] = true
	}
	return out
}

// TestBackfillForgetsFilesThatVanished is the size discipline: the vendor rotates its own
// logs, so per-input state must track what is on disk rather than everything ever seen.
func TestBackfillForgetsFilesThatVanished(t *testing.T) {
	home, sessions := threeSessions(t)
	st := openStore(t)
	runClaude(t, st, home, Options{})
	if got := knownPaths(t, st); len(got) != 3 {
		t.Fatalf("want 3 tracked inputs after the first pass, got %d", len(got))
	}
	if err := os.Remove(sessions[2]); err != nil {
		t.Fatal(err)
	}
	runClaude(t, st, home, Options{})
	got := knownPaths(t, st)
	if len(got) != 2 {
		t.Fatalf("want 2 tracked inputs, got %d", len(got))
	}
	if got[sessions[2]] {
		t.Error("a transcript that is no longer on disk must not keep its row")
	}
}

// TestBackfillKeepsStateWhenDiscoveryDrifted is the interlock. "The files are gone" and
// "we stopped being able to find them" look identical from here, so when the discovery
// canary fires the state is frozen: throwing it away during exactly the failure being
// diagnosed would destroy the evidence and force a full re-parse once the root returns.
func TestBackfillKeepsStateWhenDiscoveryDrifted(t *testing.T) {
	home, sessions := threeSessions(t)
	st := openStore(t)
	runClaude(t, st, home, Options{})
	for _, p := range sessions {
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
	}
	runClaude(t, st, home, Options{})
	if got := knownPaths(t, st); len(got) != 3 {
		t.Fatalf("want state frozen at 3 inputs while drift is suspected, got %d", len(got))
	}
}
