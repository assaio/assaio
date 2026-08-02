package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

// seedMarkStore writes two sessions in two projects into a temp store selected by
// XDG_DATA_HOME, and returns the directory holding it.
func seedMarkStore(t *testing.T) {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)
	dbDir := filepath.Join(dataDir, "assaio")
	if err := os.MkdirAll(dbDir, 0o750); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(dbDir, "assaio.db"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := st.Insert(context.Background(), []usage.Record{
		{
			Tool: "claude-code", SessionID: "aaaa1111-old", DedupeKey: "1", Timestamp: now.Add(-2 * time.Hour),
			Model: "m", Project: "web", InputTokens: 10, OutputTokens: 5, Granularity: "turn",
		},
		{
			Tool: "claude-code", SessionID: "bbbb2222-new", DedupeKey: "2", Timestamp: now.Add(-time.Hour),
			Model: "m", Project: "api", InputTokens: 20, OutputTokens: 7, Granularity: "turn",
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
}

// runCLI executes one command line against a fresh root and returns its combined output.
func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func TestMarkByPrefixAndMerge(t *testing.T) {
	seedMarkStore(t)

	out, err := runCLI(t, "mark", "aaaa1111", "--task", "refactor")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "task=refactor") || !strings.Contains(out, "web") {
		t.Fatalf("mark output = %q", out)
	}

	// A later axis must merge, not replace: the task set above has to survive.
	out, err = runCLI(t, "mark", "aaaa1111", "--outcome", "done")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "task=refactor outcome=done") {
		t.Fatalf("second mark output = %q, want the earlier task preserved", out)
	}
}

func TestMarkLastTargetsNewestSession(t *testing.T) {
	seedMarkStore(t)

	out, err := runCLI(t, "mark", "--last", "--task", "bugfix")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "bbbb2222") {
		t.Fatalf("mark --last output = %q, want the newest session", out)
	}
}

func TestMarkRejectsBadInput(t *testing.T) {
	seedMarkStore(t)

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"value outside the vocabulary", []string{"mark", "aaaa1111", "--task", "chore"}, "unknown --task"},
		{"nothing to set", []string{"mark", "aaaa1111"}, "nothing to set"},
		{"no such session", []string{"mark", "zzzz", "--task", "docs"}, "no stored session starts with"},
		{"unmark plus a value", []string{"mark", "aaaa1111", "--unmark", "--task", "docs"}, "cannot be combined"},
		{"id plus --last", []string{"mark", "aaaa1111", "--last", "--task", "docs"}, "cannot be combined"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runCLI(t, tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want one containing %q", err, tc.want)
			}
		})
	}
}

func TestMarkAmbiguousPrefixListsCandidates(t *testing.T) {
	seedMarkStore(t)
	// Both seeded ids share no prefix, so seed a colliding pair through the store directly.
	dbPath := filepath.Join(os.Getenv("XDG_DATA_HOME"), "assaio", "assaio.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Insert(context.Background(), []usage.Record{
		{Tool: "codex", SessionID: "dupe-one", DedupeKey: "3", Timestamp: time.Now(), Model: "m"},
		{Tool: "codex", SessionID: "dupe-two", DedupeKey: "4", Timestamp: time.Now(), Model: "m"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = runCLI(t, "mark", "dupe", "--task", "docs")
	if err == nil || !strings.Contains(err.Error(), "matches 2 sessions") {
		t.Fatalf("error = %v, want an ambiguity report", err)
	}
}

// A short prefix matches dozens of sessions on a real store, so the candidate list is capped:
// a wall of ids is not a suggestion.
func TestMarkAmbiguousPrefixTruncatesTheCandidateList(t *testing.T) {
	seedMarkStore(t)
	dbPath := filepath.Join(os.Getenv("XDG_DATA_HOME"), "assaio", "assaio.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	recs := make([]usage.Record, 0, 10)
	for i := range 10 {
		recs = append(recs, usage.Record{
			Tool: "codex", SessionID: fmt.Sprintf("crowd-%02d", i),
			DedupeKey: fmt.Sprintf("crowd-%02d", i), Timestamp: time.Now(), Model: "m",
		})
	}
	if _, err := st.Insert(context.Background(), recs); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = runCLI(t, "mark", "crowd", "--task", "docs")
	if err == nil {
		t.Fatal("want an ambiguity error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "matches 10 sessions") {
		t.Fatalf("error must state the full count, got %q", msg)
	}
	if !strings.Contains(msg, "and 4 more") {
		t.Fatalf("error must name the ids it did not list, got %q", msg)
	}
	if got := strings.Count(msg, "crowd-"); got != ambiguityShown {
		t.Fatalf("listed %d ids, want %d", got, ambiguityShown)
	}
}

func TestMarkUnmarkAndClearedAxes(t *testing.T) {
	seedMarkStore(t)
	if _, err := runCLI(t, "mark", "aaaa1111", "--task", "docs"); err != nil {
		t.Fatal(err)
	}

	out, err := runCLI(t, "mark", "aaaa1111", "--unmark")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "unmarked") {
		t.Fatalf("unmark output = %q", out)
	}

	// Clearing the last remaining axis is the same thing however it is spelled.
	if _, err := runCLI(t, "mark", "aaaa1111", "--task", "docs"); err != nil {
		t.Fatal(err)
	}
	out, err = runCLI(t, "mark", "aaaa1111", "--task", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "unmarked") {
		t.Fatalf("clearing every axis output = %q, want an unmark", out)
	}
}

func TestMarkListShowsLabelsAndCoverage(t *testing.T) {
	seedMarkStore(t)
	if _, err := runCLI(t, "mark", "aaaa1111", "--task", "refactor", "--difficulty", "high"); err != nil {
		t.Fatal(err)
	}

	out, err := runCLI(t, "mark", "--list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "task=refactor difficulty=high") {
		t.Fatalf("--list output = %q, want the stored labels", out)
	}
	if !strings.Contains(out, "1 of 2 labeled") {
		t.Fatalf("--list output = %q, want the labeled count", out)
	}
	// The unlabeled session must still be listed, with an explicit placeholder.
	if !strings.Contains(out, "—") {
		t.Fatalf("--list output = %q, want unlabeled sessions shown too", out)
	}
}

func TestClearKeepsLabelsUnlessAsked(t *testing.T) {
	seedMarkStore(t)
	if _, err := runCLI(t, "mark", "aaaa1111", "--task", "refactor"); err != nil {
		t.Fatal(err)
	}

	out, err := runCLI(t, "clear", "--all", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "kept 1 session label") {
		t.Fatalf("clear --all output = %q, want the labels reported as kept", out)
	}

	out, err = runCLI(t, "clear", "--labels", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "deleted 1 session label") {
		t.Fatalf("clear --labels output = %q", out)
	}
}
