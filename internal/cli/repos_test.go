package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

func TestReposNamesEachDirectoryAndNeverItsKey(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	first := filepath.Join(t.TempDir(), "api")
	second := filepath.Join(t.TempDir(), "api")
	unseen := filepath.Join(t.TempDir(), "api")
	plain := t.TempDir()
	for _, dir := range []string{first, second, unseen} {
		if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	dbPath := evidenceDBPath(t)
	firstKey, secondKey := repoKeyIn(t, dbPath, first), repoKeyIn(t, dbPath, second)
	at := time.Now().UTC().Add(-time.Hour)
	rec := func(key, id string) usage.Record {
		return usage.Record{
			Tool: "claude-code", SessionID: id, DedupeKey: id, Timestamp: at, Model: "m",
			Project: "api", RepoKey: key, Granularity: "turn",
		}
	}
	seedStoreAt(t, dbPath, []usage.Record{rec(firstKey, "a"), rec(secondKey, "b"), rec("", "c")})

	t.Chdir(second)
	list, err := runCLI(t, "repos")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"api (2)", "← this directory", "api (?)", "Project names shared by more than one repository: api.", "67% of sessions (2 of 3) and 67% of usage rows record their repository", "UNRESOLVED"} {
		if !strings.Contains(list, want) {
			t.Errorf("repos output missing %q:\n%s", want, list)
		}
	}
	if strings.Count(list, "← this directory") != 1 || !strings.Contains(list, "api (2)                                  1") {
		t.Errorf("the working directory should mark api (2) alone:\n%s", list)
	}

	at1, err := runCLI(t, "repos", first, second, unseen, plain)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		first + "  api\n", second + "  api (2)\n",
		unseen + "  no stored usage resolved to this repository; another repository has usage under \"api\"",
		plain + "  (not in a git repository; its usage is shown under " + filepath.Base(plain) + ")",
	} {
		if !strings.Contains(at1, want) {
			t.Errorf("repos <dirs> missing %q:\n%s", want, at1)
		}
	}
	for _, out := range []string{list, at1} {
		for _, key := range []string{firstKey, secondKey, "v1:"} {
			if strings.Contains(out, key) {
				t.Fatalf("repos printed a repository key:\n%s", out)
			}
		}
	}
}
