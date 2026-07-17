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

func TestStatusEmptyStore(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"status"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "0 record") {
		t.Fatalf("status output = %q", s)
	}
	if !strings.Contains(s, "No usage yet") || !strings.Contains(s, "backfill") {
		t.Fatalf("empty store must show the rich backfill hint: %q", s)
	}
}

func TestStatusDashboardWithData(t *testing.T) {
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
	now := time.Now().UTC()
	_, err = st.Insert(context.Background(), []usage.Record{
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: now, Model: "claude-opus-4-5",
			InputTokens: 1000, OutputTokens: 500, Project: "web", DedupeKey: "1", LinesAdded: 100,
		},
		{
			Tool: "codex", SessionID: "s2", Timestamp: now.AddDate(0, 0, -20), Model: "claude-opus-4-5",
			InputTokens: 300, OutputTokens: 150, Project: "infra", DedupeKey: "2", LinesAdded: 30,
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
	root.SetArgs([]string{"status"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{
		"2 record", "projects", "days of history",
		"Hot", "web",
		"Going stale", "infra",
		"directional", "never a per-person metric",
		"rework:", "thrash proxy", "within-session",
		"Sessions", "2 sessions", "turns", "active work", "peak context",
		"produced code", "conversational", "active day",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("status dashboard missing %q: %s", want, s)
		}
	}
}
