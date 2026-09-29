package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/assaio/assaio/internal/attribution"
	"github.com/assaio/assaio/internal/projectid"
	"github.com/assaio/assaio/internal/store"
)

// repoKeyIn is the key ingest would give root in the store at dbPath.
func repoKeyIn(t *testing.T, dbPath, root string) string {
	t.Helper()
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	salt, err := st.RepositorySalt(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return projectid.Key(salt, root)
}

// TestEvidenceJoinsOnlyTheRepositoryItStandsIn: another checkout with the same name, and
// sessions whose repository never resolved, stay out of the join -- counted, not matched -- and
// a worktree reaches the same sessions as its main checkout.
func TestEvidenceJoinsOnlyTheRepositoryItStandsIn(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repo, now := evidenceRepo(t)
	other := filepath.Join(t.TempDir(), filepath.Base(repo))
	if err := os.MkdirAll(filepath.Join(other, ".git"), 0o750); err != nil {
		t.Fatal(err)
	}
	dbPath := evidenceDBPath(t)
	label := filepath.Base(repo)
	records := evidenceSessionRecords(label, repoKeyIn(t, dbPath, repo), now, "")
	records = append(records, evidenceSessionRecords(label, repoKeyIn(t, dbPath, other), now, "-other")...)
	records = append(records, evidenceSessionRecords(label, "", now, "-unresolved")...)
	seedStoreAt(t, dbPath, records)

	worktree := filepath.Join(t.TempDir(), "feature")
	gitIn(t, repo, "worktree", "add", "-q", worktree)
	for _, dir := range []string{repo, worktree} {
		var doc attribution.Document
		if err := json.Unmarshal([]byte(runEvidenceCommand(t, dir, "json")), &doc); err != nil {
			t.Fatal(err)
		}
		if doc.Project != label || doc.Summary.Population != 3 || doc.Summary.Matched != 1 ||
			doc.OtherProjects != 3 || doc.IdentityUnresolved != 3 {
			t.Fatalf("evidence --repo %s: project %q population %d matched %d other %d unresolved %d; "+
				"want %q 3 1 3 3", dir, doc.Project, doc.Summary.Population, doc.Summary.Matched,
				doc.OtherProjects, doc.IdentityUnresolved, label)
		}
	}
}

func runEvidenceCommand(t *testing.T, repo, format string) string {
	t.Helper()
	root := NewRootCmd()
	var output, notes bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&notes)
	root.SetArgs([]string{"evidence", "--repo", repo, "--since", "30d", "--format", format})
	if err := root.Execute(); err != nil {
		t.Fatalf("evidence: %v", err)
	}
	return output.String()
}

// TestEvidenceNamesNoProjectForARepositoryWithoutStoredUsage: when every stored session under a
// name belongs to another checkout, this repository has no name in the store, and the document
// says so rather than carrying the other repository's name.
func TestEvidenceNamesNoProjectForARepositoryWithoutStoredUsage(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stored, now := evidenceRepo(t)
	unstored, _ := evidenceRepo(t)
	dbPath := evidenceDBPath(t)
	seedStoreAt(t, dbPath, evidenceSessionRecords(filepath.Base(stored), repoKeyIn(t, dbPath, stored), now, ""))

	var doc attribution.Document
	if err := json.Unmarshal([]byte(runEvidenceCommand(t, unstored, "json")), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Project != "" || doc.Summary.Population != 0 || doc.OtherProjects != 3 || doc.IdentityUnresolved != 0 {
		t.Fatalf("project %q population %d other %d unresolved %d; want \"\" 0 3 0",
			doc.Project, doc.Summary.Population, doc.OtherProjects, doc.IdentityUnresolved)
	}
}

// TestEvidenceNeverMatchesAnUnresolvedSessionUnderItsOwnName: when no other repository has the
// name, sessions whose repository never resolved are shown under it -- and still counted apart,
// because nothing says they ran here.
func TestEvidenceNeverMatchesAnUnresolvedSessionUnderItsOwnName(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	repo, now := evidenceRepo(t)
	dbPath := evidenceDBPath(t)
	label := filepath.Base(repo)
	records := evidenceSessionRecords(label, repoKeyIn(t, dbPath, repo), now, "")
	records = append(records, evidenceSessionRecords(label, "", now, "-unresolved")...)
	seedStoreAt(t, dbPath, records)

	var doc attribution.Document
	if err := json.Unmarshal([]byte(runEvidenceCommand(t, repo, "json")), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Project != label || doc.Summary.Population != 3 || doc.IdentityUnresolved != 3 || doc.OtherProjects != 0 {
		t.Fatalf("project %q population %d unresolved %d other %d; want %q 3 3 0",
			doc.Project, doc.Summary.Population, doc.IdentityUnresolved, doc.OtherProjects, label)
	}
}
