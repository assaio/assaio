package vcs

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func gitText(t *testing.T, dir string, args ...string) string {
	t.Helper()
	//nolint:gosec // test-only git driver over a t.TempDir() path, not user input
	cmd := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

// listedRepo holds one commit of each kind a pull request can list: made on a branch here, made in
// another worktree here, written by the forge on a branch checked out here, a teammate's fetched
// and then checked out for review, and one this clone never had.
func listedRepo(t *testing.T) (dir string, mine, worktree, forged, teammate, absent string) {
	t.Helper()
	dir = repo(t)
	runGit(t, dir, "checkout", "-b", "feat")
	write(t, dir, "feat.go", "package feat\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "feat")
	mine = gitText(t, dir, "rev-parse", "HEAD")
	write(t, dir, "suggested.go", "package feat\n")
	runGit(t, dir, "add", ".")
	now := time.Now().UTC().Truncate(time.Second)
	commitAs(t, dir, now, now, "GitHub", "noreply@github.com", "applied suggestion")
	forged = gitText(t, dir, "rev-parse", "HEAD")
	runGit(t, dir, "checkout", "main")

	tree := filepath.Join(t.TempDir(), "wt")
	runGit(t, dir, "worktree", "add", "-b", "other", tree)
	write(t, tree, "other.go", "package other\n")
	runGit(t, tree, "add", ".")
	runGit(t, tree, "commit", "-m", "other")
	worktree = gitText(t, tree, "rev-parse", "HEAD")

	teammate = gitText(t, dir, "commit-tree", gitText(t, dir, "rev-parse", "HEAD^{tree}"), "-p", "HEAD", "-m", "theirs")
	runGit(t, dir, "update-ref", "refs/remotes/origin/theirs", teammate)
	runGit(t, dir, "checkout", "--detach", teammate)
	runGit(t, dir, "checkout", "-b", "review", "origin/theirs")
	runGit(t, dir, "checkout", "main")
	return dir, mine, worktree, forged, teammate, strings.Repeat("0a", 20)
}

// TestOnlyCommitsMadeHereAreObserved: presence is not evidence the work was done here -- a fetch
// brings every teammate's branch, and a checkout for review moves HEAD onto one -- so only a commit
// a HEAD reflog records as made here is read, and the rest are counted by why.
func TestOnlyCommitsMadeHereAreObserved(t *testing.T) {
	dir, mine, worktree, forged, teammate, absent := listedRepo(t)
	got, err := CollectListed(context.Background(), dir, "repo",
		[]string{mine, worktree, forged, teammate, absent}, allTime, observedAt, "test")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for i := range got.Events {
		ids = append(ids, got.Events[i].ID)
		if got.Events[i].Subject.Project != "repo" {
			t.Errorf("%s is attributed to %q, want the caller's project", got.Events[i].ID, got.Events[i].Subject.Project)
		}
		if c := commitOf(t, &got.Events[i]); c.CommittedByForge != (got.Events[i].ID == forged) {
			t.Errorf("%s forge flag = %v", got.Events[i].ID, c.CommittedByForge)
		}
	}
	slices.Sort(ids)
	want := []string{mine, worktree, forged}
	slices.Sort(want)
	if !slices.Equal(ids, want) {
		t.Errorf("observed %v, want the three made or checked out here %v", ids, want)
	}
	if !slices.Equal(got.NotInReflog, []string{teammate}) || !slices.Equal(got.NotLocal, []string{absent}) ||
		len(got.Unreadable) != 0 || len(got.BeforeWindow) != 0 || !got.EveryWorktree {
		t.Errorf("fetched-only %v, absent %v, unreadable %v; want [%s], [%s], none",
			got.NotInReflog, got.NotLocal, got.Unreadable, teammate, absent)
	}
}

// TestACommitMadeBeforeTheWindowIsCountedNotLost: it was made here, so it is neither absent nor
// unreadable, and saying nothing about it would hide a clone that did the work.
func TestACommitMadeBeforeTheWindowIsCountedNotLost(t *testing.T) {
	dir, mine, _, _, _, _ := listedRepo(t)
	got, err := CollectListed(context.Background(), dir, "repo", []string{mine}, time.Now().Add(time.Hour), observedAt, "test")
	if err != nil || len(got.Events) != 0 || !slices.Equal(got.BeforeWindow, []string{mine}) {
		t.Fatalf("observed %d, before the window %v, err %v; want %s counted before the window", len(got.Events), got.BeforeWindow, err, mine)
	}
}

// TestAWorktreeWithNoCommitDoesNotHideTheOthers: the main worktree can sit on an unborn branch
// while the work happens in a linked one.
func TestAWorktreeWithNoCommitDoesNotHideTheOthers(t *testing.T) {
	dir, _, worktree, _, _, _ := listedRepo(t)
	runGit(t, dir, "checkout", "--orphan", "scratch")
	made, every, err := MadeHere(context.Background(), dir)
	if err != nil || !every || !made[worktree] {
		t.Fatalf("made here = %d commit(s), every %v, err %v; want the linked worktree's %s", len(made), every, err, worktree)
	}
}

// TestOneUnreadableCommitDoesNotCostTheRest: git fails a whole batch on one bad object, as it does
// on a commit whose blobs a partial clone never fetched.
func TestOneUnreadableCommitDoesNotCostTheRest(t *testing.T) {
	dir, mine, _, _, _, absent := listedRepo(t)
	events, _, unreadable := CollectHashes(context.Background(), dir, "repo", []string{mine, absent}, allTime, observedAt, "test")
	if len(events) != 1 || events[0].ID != mine || !slices.Equal(unreadable, []string{absent}) {
		t.Fatalf("read %d event(s), unreadable %v; want %s read and %s unreadable", len(events), unreadable, mine, absent)
	}
}

func TestFirstParentsFollowTheBranchLine(t *testing.T) {
	dir := repo(t)
	head := gitText(t, dir, "rev-parse", "HEAD")
	got, err := FirstParents(context.Background(), dir, head, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{head, gitText(t, dir, "rev-parse", "HEAD^1")}
	if !slices.Equal(got, want) {
		t.Fatalf("first parents = %v, want %v", got, want)
	}
}

func TestHonoursNoLazyFetch(t *testing.T) {
	for version, want := range map[string]bool{
		"git version 2.54.0 (Apple Git-157)": true,
		"git version 2.44.0":                 true,
		"git version 2.43.5":                 false,
		"git version 3.0.0":                  true,
		"git version 2.windows":              false,
		"":                                   false,
	} {
		if got := honoursNoLazyFetch(version); got != want {
			t.Errorf("honoursNoLazyFetch(%q) = %v, want %v", version, got, want)
		}
	}
}

func TestAFullCloneIsSafeToRead(t *testing.T) {
	unsafe, err := lazyFetchUnsafe(context.Background(), repo(t))
	if err != nil || unsafe {
		t.Fatalf("lazyFetchUnsafe = %v, %v on a full clone", unsafe, err)
	}
}
