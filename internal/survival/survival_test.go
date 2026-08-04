package survival

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/event"
	"github.com/assaio/assaio/internal/vcs"
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

// tryGit runs git and reports whether it succeeded, for the one command a test wants to
// fail: a merge that conflicts.
func tryGit(t *testing.T, dir string, args ...string) bool {
	t.Helper()
	//nolint:gosec // test-only git driver over a t.TempDir() path, not user input
	cmd := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	return cmd.Run() == nil
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// newRepo builds an empty repository with a local identity, so the tests below differ only
// in the history they commit into it.
func newRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "t@t.test")
	runGit(t, dir, "config", "user.name", "Test")
	return dir
}

// analyzed drives the whole path a `survival` run takes: collect the window's commit
// observations, list the paths to blame, and read the result off both.
func analyzed(t *testing.T, dir string, aiLines int64) Result {
	t.Helper()
	ctx := context.Background()
	root, err := vcs.RepoRoot(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	since := time.Now().Add(-time.Hour)
	commits, skipped, err := vcs.Collect(ctx, root, since, time.Now(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped %d commits", skipped)
	}
	files, err := vcs.TouchedFiles(ctx, root, since)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Analyze(ctx, root, commits, files, aiLines)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// TestAnalyzeCountsSurvivingWindowLines builds a tiny repo whose three commits add 8 lines
// and leave 6 in HEAD, and checks Analyze reports GitAdded=8, Surviving=6, and the rate --
// the whole point being that removed lines don't survive and the AI count passes through.
func TestAnalyzeCountsSurvivingWindowLines(t *testing.T) {
	dir := newRepo(t)
	writeFile(t, dir, "a.txt", "l1\nl2\nl3\nl4\nl5\n")
	runGit(t, dir, "add", "a.txt")
	runGit(t, dir, "commit", "-m", "c1")

	writeFile(t, dir, "a.txt", "l1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\n")
	runGit(t, dir, "commit", "-am", "c2")

	writeFile(t, dir, "a.txt", "l1\nl2\nl3\nl4\nl7\nl8\n")
	runGit(t, dir, "commit", "-am", "c3")

	res := analyzed(t, dir, 100)
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

// What the window's commits touched now travels with the survival number, from the same
// observations: a window that only changed docs is a different fact from one that rewrote
// source, and a rate alone cannot tell them apart.
func TestAnalyzeReportsWhatTheWindowChanged(t *testing.T) {
	dir := newRepo(t)
	writeFile(t, dir, "app.go", "package app\n")
	writeFile(t, dir, "app_test.go", "package app\n")
	writeFile(t, dir, "README.md", "# x\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "c1")
	runGit(t, dir, "revert", "--no-edit", "HEAD")

	res := analyzed(t, dir, 0)
	if want := (event.FileCategories{Source: 2, Test: 2, Docs: 2}); res.Changed != want {
		t.Errorf("Changed = %+v, want %+v -- the revert touches all three again", res.Changed, want)
	}
	if res.Reverts != 1 {
		t.Errorf("Reverts = %d, want the one git labelled", res.Reverts)
	}
}

// conflictMergeRepo builds the one history where git's own reporting is asymmetric: a merge
// resolved by hand. `git log --numstat` prints no diff for a merge, so its 50 resolved lines
// are never counted as added -- but `git blame` names the merge for every one of them.
func conflictMergeRepo(t *testing.T) string {
	t.Helper()
	dir := newRepo(t)
	writeFile(t, dir, "f.txt", "x\n")
	runGit(t, dir, "add", "f.txt")
	runGit(t, dir, "commit", "-m", "base")

	runGit(t, dir, "checkout", "-b", "side")
	writeFile(t, dir, "f.txt", "S\n")
	runGit(t, dir, "commit", "-am", "side")
	runGit(t, dir, "checkout", "-")
	writeFile(t, dir, "f.txt", "M\n")
	runGit(t, dir, "commit", "-am", "main")

	if tryGit(t, dir, "merge", "side") {
		t.Fatal("merge was expected to conflict, so the resolution below is what git cannot report")
	}
	var resolved strings.Builder
	for i := range 50 {
		resolved.WriteString("resolved-" + strconv.Itoa(i) + "\n")
	}
	writeFile(t, dir, "f.txt", resolved.String())
	runGit(t, dir, "add", "f.txt")
	runGit(t, dir, "commit", "-m", "merge side")
	return dir
}

// A merge's conflict resolution is a hole in git's own reporting, and the rate must not be
// computed across it: counting blamed merge lines as survivors while numstat never counted
// them as added produced 50 surviving out of 3 added, rendered as a flat "100%". Both sides
// now count the same commits, and what the merge holds is reported separately instead.
func TestAnalyzeExcludesMergesFromBothSidesOfTheRate(t *testing.T) {
	res := analyzed(t, conflictMergeRepo(t), 0)

	if res.Merges != 1 {
		t.Fatalf("Merges = %d, want the one merge commit in the window", res.Merges)
	}
	if res.MergeLines != 50 {
		t.Errorf("MergeLines = %d, want the 50 resolved lines blame attributes to the merge", res.MergeLines)
	}
	if res.Surviving > res.GitAdded {
		t.Errorf("Surviving = %d exceeds GitAdded = %d: a line git never reported as added cannot survive",
			res.Surviving, res.GitAdded)
	}
	if res.SurvivalRate > 1 {
		t.Errorf("SurvivalRate = %v, want a rate no clamp had to rescue", res.SurvivalRate)
	}
}

// An empty window is not a zero survival rate: nothing was added, so nothing survived or
// failed to, and there is no rate to report.
func TestAnalyzeReportsNoRateWithoutAddedLines(t *testing.T) {
	dir := newRepo(t)
	runGit(t, dir, "commit", "--allow-empty", "-m", "nothing")

	if res := analyzed(t, dir, 0); res.SurvivalRate != -1 {
		t.Fatalf("SurvivalRate = %v, want -1 for a window that added nothing", res.SurvivalRate)
	}
}
