package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/paths"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

func TestEffectivenessByProjectCSV(t *testing.T) {
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
			LinesAdded: 50, Edits: 2, ToolCalls: 3,
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
	root.SetArgs([]string{"effectiveness", "--by", "project", "--format", "csv"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if !strings.Contains(lines[0], "cost_per_100_lines") {
		t.Fatalf("csv header missing cost_per_100_lines: %q", lines[0])
	}
	if len(lines) != 3 {
		t.Fatalf("want header + 2 project rows, got %d lines: %q", len(lines), out.String())
	}
}

func TestEffectivenessEmptyStoreHint(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"effectiveness"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "No usage found") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestEffectivenessByBogusDimErrors(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"effectiveness", "--by", "bogus"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for unknown --by dimension")
	}
}

func TestEffectivenessTableDefaultsByProject(t *testing.T) {
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
			InputTokens: 100, OutputTokens: 200, Project: "web", DedupeKey: "1", LinesAdded: 50,
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
	root.SetArgs([]string{"effectiveness"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "PROJECT") {
		t.Fatalf("effectiveness with no --by must default to project grouping: %q", s)
	}
	if !strings.Contains(s, "web") {
		t.Fatalf("table missing project name: %q", s)
	}
	if !strings.Contains(s, "directional") {
		t.Fatalf("table missing honesty caveat: %q", s)
	}
}
