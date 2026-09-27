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

// TestInitImportsAConfiguredParserPlugin: a machine whose only source is a parser plugin is a
// supported setup, so init must run it rather than print the no-logs hint.
func TestInitImportsAConfiguredParserPlugin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	cfgPath := filepath.Join(cfgDir, "assaio", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o700); err != nil {
		t.Fatal(err)
	}
	body := "plugins:\n  - name: demo\n    command: " + pluginScript(t, "good.sh") + "\n    timeout: 5s\n"
	if err := os.WriteFile(cfgPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	out, err := runCommand(t, "init", "--non-interactive")
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	if strings.Contains(out, "No supported tool's logs were found") {
		t.Fatalf("init ignored the configured plugin: %q", out)
	}
	if !strings.Contains(out, initPluginNotice) {
		t.Fatalf("the privacy line describes assaio's own parsers; with a plugin it must say what it cannot vouch for: %q", out)
	}
	pluginIdx := strings.Index(out, "plugin:demo")
	importIdx := strings.Index(out, "imported")
	if pluginIdx < 0 || importIdx < 0 || pluginIdx > importIdx {
		t.Fatalf("init must name the plugin before running it, then import: %q", out)
	}
	if _, statErr := os.Stat(dashboardDefaultOutput); statErr != nil {
		t.Fatalf("init wrote no report: %v", statErr)
	}
}

// TestInitWithAPluginThatCannotRunExplainsItself: a configured plugin whose command does not
// exist is named with the reason, is not counted as a source, and leaves the no-data hint.
func TestInitWithAPluginThatCannotRunExplainsItself(t *testing.T) {
	for name, command := range map[string]string{
		"not on PATH":           "assaio-parser-not-on-path",
		"absolute path missing": filepath.Join(t.TempDir(), "missing"),
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			t.Chdir(t.TempDir())
			cfgPath := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(cfgPath, []byte("plugins:\n  - name: demo\n    command: "+command+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			out, err := runCommand(t, "init", "--non-interactive", "--config", cfgPath)
			if err != nil {
				t.Fatalf("init: %v\n%s", err, out)
			}
			if !strings.Contains(out, "plugin demo: command") || !strings.Contains(out, initSourceHint) {
				t.Fatalf("init must name the unrunnable plugin and fall back to the hint: %q", out)
			}
		})
	}
}
