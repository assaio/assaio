package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/paths"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

func TestParseSince(t *testing.T) {
	now := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	got, err := parseSinceAt("7d", now)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(now.AddDate(0, 0, -7)) {
		t.Fatalf("parseSince = %v", got)
	}
	if _, err := parseSinceAt("bogus", now); err == nil {
		t.Fatal("expected error on bogus window")
	}
}

func TestReportEmptyCSV(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"report", "--format", "csv"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "day,tool,model,") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestReportByProjectCSV(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	dbPath, err := paths.DBPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureParent(dbPath); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	ts := time.Now().UTC()
	_, err = st.Insert(context.Background(), []usage.Record{
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "claude-opus-4-5",
			InputTokens: 100, OutputTokens: 200, Project: "web", DedupeKey: "1",
		},
		{
			Tool: "codex", SessionID: "s2", Timestamp: ts, Model: "claude-opus-4-5",
			InputTokens: 50, OutputTokens: 10, Project: "api", DedupeKey: "2",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"report", "--format", "csv", "--by", "project"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if !strings.Contains(lines[0], "cache_eff") {
		t.Fatalf("csv header missing cache_eff: %q", lines[0])
	}
	if len(lines) != 3 {
		t.Fatalf("want header + 2 project rows, got %d lines: %q", len(lines), out.String())
	}
}

func TestReportByBogusDimErrors(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"report", "--by", "bogus"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for unknown --by dimension")
	}
}

func TestReportEmptyTableHint(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"report"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "No usage found") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

// TestReportDBFlagMissingPathErrors is the finding-3 regression: a --db path that does
// not exist must be a clear error, never a silently created empty store that then looks
// indistinguishable from "no usage yet".
func TestReportDBFlagMissingPathErrors(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	missing := filepath.Join(t.TempDir(), "does-not-exist.db")
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"report", "--db", missing})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error for --db pointing at a missing path")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Fatalf("error must name the missing path %q: %v", missing, err)
	}
	if _, statErr := os.Stat(missing); statErr == nil {
		t.Fatalf("--db must not silently create the missing store at %q", missing)
	}
}

// TestReportDBFlagEmptyStoreHintDoesNotSuggestBackfill covers the second half of
// finding 3: backfill always writes to the default local store and has no --db flag, so
// the empty-store hint must not send a --db user to run a command that can't fix their
// store.
func TestReportDBFlagEmptyStoreHintDoesNotSuggestBackfill(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	altDB := filepath.Join(t.TempDir(), "team.db")
	seedStoreAt(t, altDB, nil)

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"report", "--db", altDB})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "backfill") {
		t.Fatalf("--db empty-store hint must not suggest backfill, which ignores --db: %q", out.String())
	}
}
