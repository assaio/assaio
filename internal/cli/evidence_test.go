package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/attribution"
	"github.com/assaio/assaio/internal/paths"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

func TestEvidenceRendersMatchedAmbiguousAndUnmatched(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repo, now := evidenceRepo(t)
	seedEvidenceStore(t, filepath.Base(repo), now)

	textOutput := runEvidenceCommand(t, repo, "text")
	for _, want := range []string{
		"candidate coverage: 67% (2/3)", "resolved coverage: 33% (1/3)",
		"outcomes: matched 1 · ambiguous 1 · unmatched 1",
		"confidence=medium", "confidence=insufficient",
		"competing-following-commit-candidates", "no-candidate-window-still-open",
		"source=git", "provenance=parsed", "privacy=local-only",
		"do not prove that an AI session caused a commit", "PR, review, CI, merge",
	} {
		if !strings.Contains(textOutput, want) {
			t.Fatalf("text output missing %q:\n%s", want, textOutput)
		}
	}
	for _, secret := range []string{"client-secret.go", "ticket SECRET"} {
		if strings.Contains(textOutput, secret) {
			t.Fatalf("text output exposed repository content %q:\n%s", secret, textOutput)
		}
	}

	jsonOutput := runEvidenceCommand(t, repo, "json")
	var doc attribution.Document
	if err := json.Unmarshal([]byte(jsonOutput), &doc); err != nil {
		t.Fatalf("JSON output: %v\n%s", err, jsonOutput)
	}
	if doc.Algorithm != attribution.Algorithm || doc.Summary.Population != 3 || len(doc.Results) != 3 {
		t.Fatalf("JSON document = %#v", doc)
	}
	if doc.Results[0].Status != "ambiguous" || len(doc.Results[0].Candidates) != 2 {
		t.Fatalf("ambiguous result lost candidates: %#v", doc.Results[0])
	}

	dbPath, err := paths.DBPath()
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	if count, err := st.Count(context.Background()); err != nil || count != 6 {
		t.Fatalf("rerunning evidence changed store: count=%d err=%v", count, err)
	}
}

func TestEvidenceHasNoTeamStoreFlag(t *testing.T) {
	command, _, err := NewRootCmd().Find([]string{"evidence"})
	if err != nil {
		t.Fatal(err)
	}
	if command.Flags().Lookup("db") != nil {
		t.Fatal("evidence exposes --db despite its local-only contract")
	}
}

func TestEvidenceRejectsMemberRows(t *testing.T) {
	rows := []store.SessionRow{{SessionID: "s1", Project: "repo", Member: "member"}}
	if _, _, _, err := evidenceSessions(rows, "repo"); err == nil {
		t.Fatal("member row was accepted by the local-only evidence adapter")
	}
}

func evidenceRepo(t *testing.T) (string, time.Time) {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "target")
	if err := os.Mkdir(repo, 0o750); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "init", "--initial-branch=main")
	gitIn(t, repo, "config", "user.email", "t@t.test")
	gitIn(t, repo, "config", "user.name", "Test")
	now := time.Now().UTC().Truncate(time.Second)
	path := filepath.Join(repo, "client-secret.go")
	for i, at := range []time.Time{now.Add(-30 * time.Minute), now.Add(-29 * time.Minute)} {
		if err := os.WriteFile(path, []byte{byte('a' + i), '\n'}, 0o600); err != nil {
			t.Fatal(err)
		}
		gitIn(t, repo, "add", "client-secret.go")
		gitCommitAt(t, repo, at, "ticket SECRET")
	}
	return repo, now
}

func gitCommitAt(t *testing.T, repo string, at time.Time, message string) {
	t.Helper()
	//nolint:gosec // test-only git process over a t.TempDir repository
	command := exec.CommandContext(context.Background(), "git", "-C", repo, "commit", "-m", message)
	stamp := at.Format(time.RFC3339)
	command.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_DATE="+stamp, "GIT_COMMITTER_DATE="+stamp)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, output)
	}
}

func seedEvidenceStore(t *testing.T, project string, now time.Time) {
	t.Helper()
	dbPath, err := paths.DBPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureParent(dbPath); err != nil {
		t.Fatal(err)
	}
	records := []usage.Record{}
	for _, session := range []struct {
		id         string
		start, end time.Time
	}{
		{"s-ambiguous", now.Add(-2 * time.Hour), now.Add(-time.Hour)},
		{"s-matched", now.Add(-2 * time.Hour), now.Add(-10 * time.Minute)},
		{"s-unmatched", now.Add(-20 * time.Minute), now.Add(-10 * time.Minute)},
	} {
		for turn, timestamp := range []time.Time{session.start, session.end} {
			records = append(records, usage.Record{
				Tool: "claude-code", SessionID: session.id, Timestamp: timestamp,
				Model: "claude-sonnet-4-5", Project: project, Granularity: "turn",
				DedupeKey: session.id + string(rune('0'+turn)),
			})
		}
	}
	seedStoreAt(t, dbPath, records)
}

func runEvidenceCommand(t *testing.T, repo, format string) string {
	t.Helper()
	root := NewRootCmd()
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&output)
	root.SetArgs([]string{"evidence", "--repo", repo, "--since", "30d", "--format", format})
	if err := root.Execute(); err != nil {
		t.Fatalf("evidence: %v", err)
	}
	return output.String()
}
