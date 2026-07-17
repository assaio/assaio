package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackfillReportsCounts(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".claude", "projects", "-x")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	body := `{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":10,"output_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "s.jsonl"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", home)        // Unix
	t.Setenv("USERPROFILE", home) // Windows
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"backfill"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "claude-code") || !strings.Contains(out.String(), "inserted") {
		t.Fatalf("backfill output = %q", out.String())
	}
}
