package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

func TestClearAllRequiresConfirmation(t *testing.T) {
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
	if _, err := st.Insert(context.Background(), []usage.Record{{
		Tool: "codex", DedupeKey: "1",
		Timestamp: time.Now(), Model: "m",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"clear", "--all", "--yes"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "deleted 1") {
		t.Fatalf("clear output = %q", out.String())
	}
}

func TestClearRequiresScopeFlag(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"clear"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no scope flag given")
	}
	if !strings.Contains(err.Error(), "--all, --older-than, or --tool") {
		t.Fatalf("error = %q", err)
	}
}

func TestClearRequiresYes(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"clear", "--all"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --yes not given")
	}
	if !strings.Contains(err.Error(), "refusing to delete without --yes") {
		t.Fatalf("error = %q", err)
	}
}
