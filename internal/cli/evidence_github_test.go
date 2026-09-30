package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/attribution"
	"github.com/assaio/assaio/internal/event"
	"github.com/assaio/assaio/internal/github"
	"github.com/assaio/assaio/internal/vcs"
)

func gitOutIn(t *testing.T, dir string, env []string, args ...string) string {
	t.Helper()
	//nolint:gosec // test-only git driver over a t.TempDir() path, not user input
	cmd := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null"), env...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

// squashedRepo adds to the evidence repository a branch commit written inside two sessions and
// the squash GitHub made of it on main, after the last session ended.
func squashedRepo(t *testing.T) (repo string, now time.Time, branch, squash string) {
	t.Helper()
	repo, now = evidenceRepo(t)
	gitIn(t, repo, "checkout", "-b", "feat")
	if err := os.WriteFile(filepath.Join(repo, "feat.go"), []byte("package feat\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "add", "feat.go")
	gitCommitAt(t, repo, now.Add(-100*time.Minute), "feat")
	branch = gitOutIn(t, repo, nil, "rev-parse", "HEAD")
	gitIn(t, repo, "checkout", "main")
	stamp := now.Add(-5 * time.Minute).Format(time.RFC3339)
	squash = gitOutIn(t, repo, []string{
		"GIT_AUTHOR_NAME=Dev", "GIT_AUTHOR_EMAIL=dev@example.test", "GIT_AUTHOR_DATE=" + stamp,
		"GIT_COMMITTER_NAME=GitHub", "GIT_COMMITTER_EMAIL=noreply@github.com", "GIT_COMMITTER_DATE=" + stamp,
	}, "commit-tree", branch+"^{tree}", "-p", "main", "-m", "feat (#7)")
	gitIn(t, repo, "update-ref", "refs/heads/main", squash)
	gitIn(t, repo, "checkout", "-f", "main")
	return repo, now, branch, squash
}

// fakeGh answers as gh would for a repository whose one pull request, #7, lists listed and was
// merged as merge.
func fakeGh(t *testing.T, now time.Time, merge string, listed ...string) {
	t.Helper()
	commits := []map[string]any{}
	for _, oid := range listed {
		commits = append(commits, map[string]any{"commit": map[string]any{"oid": oid}})
	}
	page, err := json.Marshal(map[string]any{"data": map[string]any{"repository": map[string]any{"pullRequests": map[string]any{
		"pageInfo": map[string]any{"hasNextPage": false, "endCursor": "c"},
		"nodes": []map[string]any{{
			"id": "PR_kwDOtest7", "number": 7, "state": "MERGED", "updatedAt": now.Add(-4 * time.Minute),
			"mergedAt": now.Add(-5 * time.Minute), "mergeCommit": map[string]any{"oid": merge},
			"commits": map[string]any{"totalCount": len(listed), "nodes": commits},
		}},
	}}}})
	if err != nil {
		t.Fatal(err)
	}
	previous := ghRunner
	ghRunner = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if args[0] == "repo" {
			return []byte(`{"nameWithOwner":"acme/target","url":"https://github.com/acme/target","isFork":false}`), nil
		}
		return page, nil
	}
	t.Cleanup(func() { ghRunner = previous })
}

func runEvidenceArgs(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	root := NewRootCmd()
	var output, notes bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&notes)
	root.SetArgs(append([]string{"evidence", "--since", "30d"}, args...))
	err := root.Execute()
	return output.String(), notes.String(), err
}

func TestEvidenceWithGitHubLinksTheBranchCommitASquashHid(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repo, now, branch, squash := squashedRepo(t)
	seedEvidenceStore(t, repo, now)
	fakeGh(t, now, squash, branch)

	out, notes, err := runEvidenceArgs(t, "--repo", repo, "--format", "json", "--github")
	if err != nil {
		t.Fatal(err)
	}
	var doc attribution.Document
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Algorithm != attribution.AlgorithmChanges || doc.Changes == nil || doc.Changes.PullRequestsRead != 1 {
		t.Fatalf("document = %s with changes %+v", doc.Algorithm, doc.Changes)
	}
	byID := map[string]attribution.Result{}
	for _, r := range doc.Results {
		byID[r.Session.ID] = r
	}
	if r := byID["s-ambiguous"]; r.Change != 7 || len(r.Candidates) != 1 || r.Candidates[0].CommitID != branch ||
		len(r.Candidates[0].Changes) != 1 || r.Candidates[0].Changes[0].Via != "listed" {
		t.Fatalf("s-ambiguous = %+v, want the branch commit, listed by #7", r)
	}
	if r := byID["s-unmatched"]; r.Change != 7 || r.Candidates[0].CommitID != squash || r.Candidates[0].Changes[0].Via != "merged-as" {
		t.Fatalf("s-unmatched = %+v, want the squash #7 merged as", r)
	}
	if !strings.Contains(notes, "pull requests read through gh from github.com/acme/target") {
		t.Fatalf("notes do not name the repository read:\n%s", notes)
	}
	if strings.Contains(out, "acme") {
		t.Fatal("the document carries the repository's owner, a free string from the forge")
	}
}

func TestEvidenceWithoutGitHubStaysSessionCommitV2(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repo, now, _, _ := squashedRepo(t)
	seedEvidenceStore(t, repo, now)
	previous := ghRunner
	ghRunner = func(context.Context, string, ...string) ([]byte, error) {
		t.Fatal("gh ran without --github")
		return nil, nil
	}
	t.Cleanup(func() { ghRunner = previous })
	out, _, err := runEvidenceArgs(t, "--repo", repo, "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"algorithm": "session-commit/v2"`) || strings.Contains(out, `"changes"`) {
		t.Fatalf("offline document changed shape:\n%s", out)
	}
}

func TestEvidenceRefusesPullRequestsThatAreNotThisClone(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repo, now, _, _ := squashedRepo(t)
	seedEvidenceStore(t, repo, now)
	fakeGh(t, now, strings.Repeat("0b", 20), strings.Repeat("0c", 20))
	_, _, err := runEvidenceArgs(t, "--repo", repo, "--github")
	if err == nil || !strings.Contains(err.Error(), "gh repo set-default") {
		t.Fatalf("err = %v, want the wrong-repository refusal", err)
	}
}

func TestEvidenceReportsWhyGhFailed(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repo, now := evidenceRepo(t)
	seedEvidenceStore(t, repo, now)
	previous := ghRunner
	ghRunner = func(context.Context, string, ...string) ([]byte, error) {
		return nil, errors.New("gh repo view: exit status 4: gh is not logged in; run gh auth login")
	}
	t.Cleanup(func() { ghRunner = previous })
	out, _, err := runEvidenceArgs(t, "--repo", repo, "--github")
	if err == nil || !strings.Contains(err.Error(), "gh auth login") || out != "" {
		t.Fatalf("err = %v, output %q; want gh's reason and no partial document", err, out)
	}
}

// TestAPullRequestWithOnlyOlderCommitsIsStillThisClone: a pull request updated in the window can
// list commits made here before it; they prove the clone as well as a commit in the window does.
func TestAPullRequestWithOnlyOlderCommitsIsStillThisClone(t *testing.T) {
	old := strings.Repeat("0d", 20)
	got := &github.Read{
		Requests: []event.Event{{Payload: event.PullRequest{Number: 1, State: event.ChangeOpen}}},
		Listings: []event.Event{{Payload: event.PullRequestCommit{Number: 1, Commit: old}}},
	}
	if !touchesClone(got, map[string]bool{}, &vcs.Listed{BeforeWindow: []string{old}}) {
		t.Fatal("a listed commit made here before the window was taken for another repository's")
	}
	if touchesClone(got, map[string]bool{}, &vcs.Listed{NotLocal: []string{old}}) {
		t.Fatal("a pull request listing only commits this clone lacks was taken for this clone's")
	}
}
