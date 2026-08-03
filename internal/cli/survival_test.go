package cli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	//nolint:gosec // test-only git driver over a t.TempDir() path, not user input
	cmd := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// The window's commits now reach the report as observations, so what they changed travels
// with the survival rate instead of the rate standing alone.
func TestSurvivalReportsWhatTheWindowChanged(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	dir := t.TempDir()
	gitIn(t, dir, "init")
	gitIn(t, dir, "config", "user.email", "t@t.test")
	gitIn(t, dir, "config", "user.name", "Test")
	for _, name := range []string{"app.go", "app_test.go", "README.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "-m", "c1")

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"survival", "--repo", dir, "--since", "30d"})
	if err := root.Execute(); err != nil {
		t.Fatalf("survival: %v", err)
	}
	got := out.String()
	for _, want := range []string{"1 commits in window", "source 1", "test 1", "docs 1"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}
