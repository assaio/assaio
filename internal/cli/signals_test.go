package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

func TestSignalsList(t *testing.T) {
	out, err := runCLI(t, "signals", "list")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ai.tokens.total", "ai.lines.added", "signals describe"} {
		if !strings.Contains(out, want) {
			t.Fatalf("list output missing %q:\n%s", want, out)
		}
	}
	// The listing names which sources answer each signal, which is the difference between a
	// catalog and a wish list. Copilot totals a session, so it can never contribute a turn.
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "ai.turns.count") && strings.Contains(line, "copilot") {
			t.Fatalf("a session-total source must not be listed as answering turns: %q", line)
		}
		if strings.HasPrefix(line, "ai.lines.added") && !strings.Contains(line, "copilot") {
			t.Fatalf("copilot does record changed lines and must be listed: %q", line)
		}
	}
}

func TestSignalsDescribe(t *testing.T) {
	out, err := runCLI(t, "signals", "describe", "ai.tokens.reasoning")
	if err != nil {
		t.Fatal(err)
	}
	// The zero line is the reason the catalog exists; it must always be printed.
	if !strings.Contains(out, "a zero means:") {
		t.Fatalf("describe must say what a zero means:\n%s", out)
	}
	if !strings.Contains(out, "does not report reasoning separately") {
		t.Fatalf("describe must carry the signal's own caveat:\n%s", out)
	}
}

func TestSignalsDescribeUnknown(t *testing.T) {
	_, err := runCLI(t, "signals", "describe", "ai.vibes.total")
	if err == nil || !strings.Contains(err.Error(), "unknown signal") {
		t.Fatalf("error = %v, want an unknown-signal error", err)
	}
}

// The point of the command: a window whose sources cannot record edits must say so, rather
// than reporting a zero that reads as "no edits happened".
func TestSignalsCoverageSeparatesUnsupportedFromZero(t *testing.T) {
	seedSignalsStore(t)
	out, err := runCLI(t, "signals", "coverage", "--since", "30d")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "not supported") {
		t.Fatalf("a token-only source must produce an unsupported group:\n%s", out)
	}
	unsupported := sectionAfter(out, "not supported")
	if !strings.Contains(unsupported, "ai.lines.added") {
		t.Fatalf("gemini-cli records no edits, so ai.lines.added must be unsupported:\n%s", out)
	}
	if strings.Contains(unsupported, "ai.tokens.total") {
		t.Fatalf("tokens are supported by every source and must not be listed unsupported:\n%s", out)
	}
}

// A partial verdict rendered as "100%" contradicts itself on its own line. Rounding a 99.85%
// share up is exactly the kind of confident-looking wrong number this project refuses.
func TestSignalsCoveragePartialNeverRoundsToWhole(t *testing.T) {
	seedMixedSignalsStore(t)
	out, err := runCLI(t, "signals", "coverage", "--since", "30d")
	if err != nil {
		t.Fatal(err)
	}
	partial := sectionAfter(out, "partially supported")
	if partial == "" {
		t.Fatalf("a mixed window must produce a partial group:\n%s", out)
	}
	if strings.Contains(partial, "100%") {
		t.Fatalf("partial support rendered as 100%%:\n%s", partial)
	}
	if !strings.Contains(partial, ">99%") {
		t.Fatalf("a 99.9%% share must render as >99%%:\n%s", partial)
	}
}

// sectionAfter returns the text from heading up to the next blank-line-separated group.
func sectionAfter(out, heading string) string {
	i := strings.Index(out, heading)
	if i < 0 {
		return ""
	}
	rest := out[i:]
	if j := strings.Index(rest, "\n\n"); j >= 0 {
		return rest[:j]
	}
	return rest
}

func TestSignalsCoverageEmptyStore(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)
	out, err := runCLI(t, "signals", "coverage")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "backfill") {
		t.Fatalf("an empty store must point at backfill:\n%s", out)
	}
}

// seedMixedSignalsStore writes a window overwhelmingly from a deep source with a sliver from
// a token-only one, so the activity signals land just under full support.
func seedMixedSignalsStore(t *testing.T) {
	t.Helper()
	dbDir := newSignalsStoreDir(t)
	st, err := store.Open(filepath.Join(dbDir, "assaio.db"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := st.Insert(context.Background(), []usage.Record{
		{
			Tool: "claude-code", SessionID: "c1", DedupeKey: "c1", Timestamp: now.Add(-time.Hour),
			Model: "m", Project: "web", InputTokens: 999_000, OutputTokens: 1000, Granularity: "turn",
		},
		{
			Tool: "gemini-cli", SessionID: "g1", DedupeKey: "g1", Timestamp: now.Add(-time.Hour),
			Model: "gemini-2.5-pro", Project: "web", InputTokens: 300, OutputTokens: 100, Granularity: "turn",
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
}

// seedSignalsStore writes a window from a token-only source alone, which is what makes the
// unsupported verdict reachable.
func seedSignalsStore(t *testing.T) {
	t.Helper()
	dbDir := newSignalsStoreDir(t)
	st, err := store.Open(filepath.Join(dbDir, "assaio.db"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := st.Insert(context.Background(), []usage.Record{
		{
			Tool: "gemini-cli", SessionID: "g1", DedupeKey: "g1", Timestamp: now.Add(-time.Hour),
			Model: "gemini-2.5-pro", Project: "web", InputTokens: 100, OutputTokens: 50, Granularity: "turn",
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
}

// newSignalsStoreDir points XDG_DATA_HOME at a fresh temp store and returns its directory.
func newSignalsStoreDir(t *testing.T) string {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)
	dbDir := filepath.Join(dataDir, "assaio")
	if err := os.MkdirAll(dbDir, 0o750); err != nil {
		t.Fatal(err)
	}
	return dbDir
}
