package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

// seedStoreAt opens (creating) a store at path, inserts recs, and closes it: a standalone
// store distinct from paths.DBPath's default local one, for exercising --db.
func seedStoreAt(t *testing.T, path string, recs []usage.Record) {
	t.Helper()
	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Insert(context.Background(), recs); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestReportDBFlagOpensAlternateStore is the team-CLI proof: --db points report at a
// central store instead of the default local one, and the default local store stays
// untouched -- nothing leaks between the two.
func TestReportDBFlagOpensAlternateStore(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	altDB := filepath.Join(t.TempDir(), "team.db")
	ts := time.Now().UTC()
	seedStoreAt(t, altDB, []usage.Record{
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "claude-opus-4-5",
			InputTokens: 100, OutputTokens: 200, Member: "alice", DedupeKey: "1",
		},
		{
			Tool: "claude-code", SessionID: "s2", Timestamp: ts, Model: "claude-opus-4-5",
			InputTokens: 50, OutputTokens: 10, Member: "bob", DedupeKey: "2",
		},
	})

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"report", "--db", altDB, "--by", "member", "--format", "csv"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "alice") || !strings.Contains(got, "bob") {
		t.Fatalf("report --db must read the alternate store's members: %q", got)
	}

	root2 := NewRootCmd()
	var out2 bytes.Buffer
	root2.SetOut(&out2)
	root2.SetErr(&out2)
	root2.SetArgs([]string{"report"})
	if err := root2.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2.String(), "No usage found") {
		t.Fatalf("default store must stay empty when only --db was written to: %q", out2.String())
	}
}

// TestStatusDBFlagOpensAlternateStore covers the second call path --db threads through:
// status.go used to open paths.DBPath() directly rather than via openReportStore.
func TestStatusDBFlagOpensAlternateStore(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	altDB := filepath.Join(t.TempDir(), "team.db")
	seedStoreAt(t, altDB, []usage.Record{
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: time.Now().UTC(), Model: "claude-opus-4-5",
			InputTokens: 100, Member: "alice", DedupeKey: "1",
		},
	})

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"status", "--db", altDB})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "store: "+altDB) {
		t.Fatalf("status --db must report the alternate store path: %q", got)
	}
	if !strings.Contains(got, "1 record") {
		t.Fatalf("status --db must read the alternate store's records: %q", got)
	}
}
