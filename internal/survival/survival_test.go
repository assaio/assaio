package survival

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	//nolint:gosec // test-only git driver over a t.TempDir() path, not user input
	cmd := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestAnalyzeCountsSurvivingWindowLines builds a tiny repo whose three commits add 8 lines
// and leave 6 in HEAD, and checks Analyze reports GitAdded=8, Surviving=6, and the rate --
// the whole point being that removed lines don't survive and the AI count passes through.
func TestAnalyzeCountsSurvivingWindowLines(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "t@t.test")
	runGit(t, dir, "config", "user.name", "Test")

	writeFile(t, dir, "a.txt", "l1\nl2\nl3\nl4\nl5\n")
	runGit(t, dir, "add", "a.txt")
	runGit(t, dir, "commit", "-m", "c1")

	writeFile(t, dir, "a.txt", "l1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\n")
	runGit(t, dir, "commit", "-am", "c2")

	writeFile(t, dir, "a.txt", "l1\nl2\nl3\nl4\nl7\nl8\n")
	runGit(t, dir, "commit", "-am", "c3")

	ctx := context.Background()
	root, err := RepoRoot(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Analyze(ctx, root, time.Now().Add(-time.Hour), 100)
	if err != nil {
		t.Fatal(err)
	}

	if res.Commits != 3 {
		t.Fatalf("Commits = %d, want 3", res.Commits)
	}
	if res.GitAdded != 8 {
		t.Fatalf("GitAdded = %d, want 8 (5+3 added; the removal adds nothing)", res.GitAdded)
	}
	if res.Surviving != 6 {
		t.Fatalf("Surviving = %d, want 6 (the two removed lines don't survive)", res.Surviving)
	}
	if res.AILines != 100 {
		t.Fatalf("AILines = %d, want the 100 passed through", res.AILines)
	}
	if want := 6.0 / 8.0; res.SurvivalRate < want-1e-9 || res.SurvivalRate > want+1e-9 {
		t.Fatalf("SurvivalRate = %v, want %v", res.SurvivalRate, want)
	}
}

// TestNumstatPath locks the rename-path unwrapping so a renamed file is blamed at its
// current name (else its added lines count but never survive, deflating the rate).
func TestNumstatPath(t *testing.T) {
	cases := map[string]string{
		"foo.go":                "foo.go",
		"old.go => new.go":      "new.go",
		"src/{old => new}/f.go": "src/new/f.go",
		"{a => b}":              "b",
	}
	for in, want := range cases {
		if got := numstatPath(in); got != want {
			t.Errorf("numstatPath(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestRepoRootRejectsNonRepo confirms a non-repository path errors rather than being
// treated as an empty repo.
func TestRepoRootRejectsNonRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	if _, err := RepoRoot(context.Background(), t.TempDir()); err == nil {
		t.Fatal("RepoRoot on a non-repo dir: want an error, got nil")
	}
}
