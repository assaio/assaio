package vcs

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/event"
)

var (
	observedAt = time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	allTime    = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
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

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	full := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// repo builds a history with one commit per shape the collector has to get right: an
// ordinary commit across three categories, a revert git itself labelled, and a merge.
func repo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	runGit(t, dir, "init", "--initial-branch=main")
	runGit(t, dir, "config", "user.email", "t@t.test")
	runGit(t, dir, "config", "user.name", "Test")

	write(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	write(t, dir, "README.md", "# demo\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "first")

	write(t, dir, filepath.Join("internal", "app_test.go"), "package app\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add a test")
	runGit(t, dir, "revert", "--no-edit", "HEAD")

	runGit(t, dir, "checkout", "-b", "side")
	write(t, dir, "side.go", "package side\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "on a branch")
	runGit(t, dir, "checkout", "main")
	runGit(t, dir, "merge", "--no-ff", "-m", "merge side", "side")
	return dir
}

func collect(t *testing.T, dir string) []event.Event {
	t.Helper()
	root, err := RepoRoot(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	events, skipped, err := Collect(context.Background(), root, allTime, observedAt, "v0.7.0")
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped %d commits git printed", skipped)
	}
	return events
}

func commitOf(t *testing.T, e *event.Event) event.Commit {
	t.Helper()
	c, ok := e.Payload.(event.Commit)
	if !ok {
		t.Fatalf("payload is %T, want event.Commit", e.Payload)
	}
	return c
}

// Every observation the collector emits has to survive the contract that will carry it to a
// consumer -- otherwise the collector is producing something only it can read.
func TestCollectEmitsValidObservations(t *testing.T) {
	events := collect(t, repo(t))
	if len(events) != 5 {
		t.Fatalf("got %d observations, want one per commit (5)", len(events))
	}
	for i := range events {
		if err := events[i].Validate(); err != nil {
			t.Fatalf("observation %d rejected by the contract: %v", i, err)
		}
		if events[i].Type != event.TypeCommit || events[i].Grain != event.GrainCommit {
			t.Errorf("observation %d = %s at %s grain, want a commit observation",
				i, events[i].Type, events[i].Grain)
		}
		if events[i].Privacy != event.LocalOnly {
			t.Errorf("observation %d is %s: repository evidence stays on the machine until the "+
				"threat model says otherwise", i, events[i].Privacy)
		}
	}
}

// The id is the commit hash, so re-reading a repository produces the same observations
// rather than a second set a consumer would have to de-duplicate.
func TestCollectIsIdempotent(t *testing.T) {
	dir := repo(t)
	if first, second := collect(t, dir), collect(t, dir); !reflect.DeepEqual(first, second) {
		t.Fatal("two reads of the same history produced different observations")
	}
}

func TestCollectCountsFilesByCategory(t *testing.T) {
	events := collect(t, repo(t))
	first := commitOf(t, &events[len(events)-1]) // oldest: the root commit
	if first.Parents != 0 {
		t.Errorf("Parents = %d, want 0 on a root commit", first.Parents)
	}
	want := event.FileCategories{Source: 1, Docs: 1}
	if first.Files != want {
		t.Errorf("Files = %+v, want %+v", first.Files, want)
	}
	if first.FilesChanged != 2 || first.LinesAdded != 4 || first.LinesRemoved != 0 {
		t.Errorf("counts = %d files +%d -%d, want 2 files +4 -0",
			first.FilesChanged, first.LinesAdded, first.LinesRemoved)
	}
}

// A revert is the one indicator that needs the commit's own message, which is read here and
// stored nowhere -- the payload carries the flag, never the words behind it.
func TestCollectMarksReverts(t *testing.T) {
	events := collect(t, repo(t))
	var reverts int
	for i := range events {
		if commitOf(t, &events[i]).Revert {
			reverts++
		}
	}
	if reverts != 1 {
		t.Fatalf("marked %d reverts, want exactly the one git labelled", reverts)
	}
}

// A merge changes nothing on its own, so its zero line count is a fact rather than a gap --
// and Parents is what lets a consumer tell the two apart.
func TestCollectRecordsMergeParents(t *testing.T) {
	events := collect(t, repo(t))
	merge := commitOf(t, &events[0])
	if merge.Parents != 2 {
		t.Fatalf("newest commit has %d parents, want the merge with 2", merge.Parents)
	}
	if merge.FilesChanged != 0 || merge.LinesAdded != 0 {
		t.Errorf("merge = %+v, want no changes of its own", merge)
	}
}

func TestCollectHonoursTheWindow(t *testing.T) {
	dir := repo(t)
	root, err := RepoRoot(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	events, _, err := Collect(context.Background(), root, time.Now().Add(time.Hour), observedAt, "v0.7.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("got %d observations from a window that starts in the future", len(events))
	}
}

func TestCollectRejectsSomethingThatIsNotARepository(t *testing.T) {
	if _, err := RepoRoot(context.Background(), t.TempDir()); err == nil {
		t.Fatal("want an error for a directory that is not a git repository")
	}
}
