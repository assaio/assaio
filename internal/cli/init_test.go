package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitPrintsWhatItWillReadBeforeReadingIt(t *testing.T) {
	driftHome(t)
	t.Chdir(t.TempDir())
	out, err := runCommand(t, "init", "--non-interactive")
	if err != nil {
		t.Fatal(err)
	}
	readIdx := strings.Index(out, filepath.Join(".claude", "projects"))
	importIdx := strings.Index(out, "imported")
	if readIdx < 0 || importIdx < 0 {
		t.Fatalf("init must name the paths and then report the import: %q", out)
	}
	if readIdx > importIdx {
		t.Fatalf("init must disclose what it will read before reading it: %q", out)
	}
}

func TestInitImportsAndLeavesSomethingToOpen(t *testing.T) {
	driftHome(t)
	// init writes the report into the working directory, so give it one of its own.
	t.Chdir(t.TempDir())
	out, err := runCommand(t, "init", "--non-interactive")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, dashboardDefaultOutput) {
		t.Fatalf("init must end with a report to open: %q", out)
	}
	if _, statErr := os.Stat(dashboardDefaultOutput); statErr != nil {
		t.Fatalf("init pointed at %s but wrote nothing there: %v", dashboardDefaultOutput, statErr)
	}
}

// TestInitOnAMachineWithNoToolsExplainsItself keeps a first run from looking broken when
// the honest answer is "nothing is installed here yet".
func TestInitOnAMachineWithNoToolsExplainsItself(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	out, err := runCommand(t, "init", "--non-interactive")
	if err != nil {
		t.Fatalf("a machine with no AI tools is not an error: %v", err)
	}
	if !strings.Contains(out, "sources:") {
		t.Fatalf("init must say how to point assaio at logs it could not find: %q", out)
	}
}

// TestInitWritesNoConfigWhenDefaultsWork is the "write a config only when needed" half: a
// config that merely restates the built-in defaults is one more thing a user has to
// maintain, and one more place for the two to disagree.
func TestInitWritesNoConfigWhenDefaultsWork(t *testing.T) {
	driftHome(t)
	t.Chdir(t.TempDir())
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	if _, err := runCommand(t, "init", "--non-interactive"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "assaio", "config.yaml")); err == nil {
		t.Fatal("defaults worked, so init must not leave a config file behind")
	}
}
