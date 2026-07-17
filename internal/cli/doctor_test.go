package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorDetectsClaude(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".claude", "projects", "-x")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "s.jsonl"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"doctor"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	if !strings.Contains(s, "claude-code") || !strings.Contains(s, "1 file") {
		t.Fatalf("doctor output = %q", s)
	}
	if !strings.Contains(s, "plugins:      none") {
		t.Fatalf("doctor output missing plugins line: %q", s)
	}
	if !strings.Contains(s, "0 file(s) — not detected") {
		t.Fatalf("doctor output must read an undetected tool as 'not detected', not a bare zero: %q", s)
	}
	if !strings.Contains(s, "inventory:    ") || !strings.Contains(s, "projects") {
		t.Fatalf("doctor output missing inventory line: %q", s)
	}
}

func TestDoctorCountsConfiguredPlugins(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	cfgPath := writePluginConfig(t, pluginScript(t, "good.sh"))

	out, err := runCommand(t, "doctor", "--config", cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "plugins:      1 configured") {
		t.Fatalf("doctor output = %q", out)
	}
}

func TestDoctorShowsDefaultSourceOrigin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	out, err := runCommand(t, "doctor")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "claude-code") || !strings.Contains(out, "(default)") {
		t.Fatalf("doctor output = %q, want claude-code labeled (default)", out)
	}
	if strings.Contains(out, "config-overridden") {
		t.Fatalf("doctor output = %q, want no config-overridden marker with no sources configured", out)
	}
}

func TestDoctorShowsConfigOverriddenSourceAndMissingHint(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	missingRoot := filepath.Join(t.TempDir(), "does-not-exist")
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	body := "sources:\n  claude:\n    - " + missingRoot + "\n"
	if err := os.WriteFile(cfgPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := runCommand(t, "doctor", "--config", cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "config-overridden") {
		t.Fatalf("doctor output = %q, want a config-overridden marker for claude-code", out)
	}
	if !strings.Contains(out, missingRoot) {
		t.Fatalf("doctor output = %q, want the configured path named", out)
	}
	if !strings.Contains(out, "hint: configured path(s) not found") {
		t.Fatalf("doctor output = %q, want a missing-path hint", out)
	}
}

func TestDoctorShowsConfigOverriddenSourceWithoutHintWhenPathExists(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	customRoot := filepath.Join(home, "custom-claude-logs")
	projectDir := filepath.Join(customRoot, "-custom-project")
	if err := os.MkdirAll(projectDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "s.jsonl"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	body := "sources:\n  claude:\n    - " + customRoot + "\n"
	if err := os.WriteFile(cfgPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := runCommand(t, "doctor", "--config", cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "config-overridden") || !strings.Contains(out, customRoot) {
		t.Fatalf("doctor output = %q, want claude-code under the configured root", out)
	}
	if !strings.Contains(out, "1 file") {
		t.Fatalf("doctor output = %q, want the configured root's 1 file counted", out)
	}
	if strings.Contains(out, "hint:") {
		t.Fatalf("doctor output = %q, want no missing-path hint when the configured root exists", out)
	}
}
