package ingest

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/store"
)

// lastRun returns the tool's most recent recorded ingest pass.
func lastRun(t *testing.T, st *store.Store, tool string) store.SourceRun {
	t.Helper()
	history, err := st.SourceHistory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	runs := history[tool]
	if len(runs) == 0 {
		t.Fatalf("no recorded run for %q, have %v", tool, history)
	}
	return runs[len(runs)-1]
}

func TestRunRecordsWhatEachSourceRead(t *testing.T) {
	home, _ := claudeHome(t)
	st := openStore(t)
	runClaude(t, st, home, Options{})
	got := lastRun(t, st, "claude-code")
	if got.Discovered != 1 || got.Parsed != 1 || got.Records != 1 {
		t.Fatalf("run = %+v, want Discovered=1 Parsed=1 Records=1", got)
	}
}

// TestSecondRunRecordsUnchangedFilesAsUnread pins the distinction the yield canary rests
// on: an incremental pass still discovers every file but reads none of them, and reading
// nothing must never look like getting nothing out of what was read.
func TestSecondRunRecordsUnchangedFilesAsUnread(t *testing.T) {
	home, _ := claudeHome(t)
	st := openStore(t)
	runClaude(t, st, home, Options{})
	runClaude(t, st, home, Options{})
	got := lastRun(t, st, "claude-code")
	if got.Discovered != 1 || got.Parsed != 0 || got.Records != 0 {
		t.Fatalf("run = %+v, want Discovered=1 Parsed=0 Records=0", got)
	}
}

func TestRunCountsRecordsCarryingNoTokens(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, ".claude", "projects", "-home-dev-app", "s1.jsonl"),
		claudeTurn+claudeZeroTokenTurn)
	st := openStore(t)
	runClaude(t, st, home, Options{})
	got := lastRun(t, st, "claude-code")
	if got.Records != 2 || got.ZeroToken != 1 {
		t.Fatalf("run = %+v, want Records=2 ZeroToken=1", got)
	}
}

// TestPluginRunCarriesNoFileCounts keeps the file-shaped canaries away from a source that
// has no files: a plugin's per-run record count tracks whatever its tool did that day, so
// judging it against a records-per-file baseline would report drift on ordinary variation.
func TestPluginRunCarriesNoFileCounts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin test fixtures are POSIX shell scripts")
	}
	pluginScript, err := filepath.Abs(filepath.Join("..", "plugin", "testdata", "good.sh"))
	if err != nil {
		t.Fatal(err)
	}
	st := openStore(t)
	plugins := []config.PluginConfig{{Name: "demo", Command: pluginScript, Timeout: "5s"}}
	if _, err := Run(context.Background(), t.TempDir(), st, config.Sources{}, plugins, Options{}); err != nil {
		t.Fatal(err)
	}
	got := lastRun(t, st, "plugin:demo")
	if got.Discovered != 0 || got.Parsed != 0 {
		t.Fatalf("run = %+v, want Discovered=0 Parsed=0 for a plugin source", got)
	}
	if got.Records == 0 {
		t.Fatalf("run = %+v, want the plugin's records recorded", got)
	}
}
