package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writePluginConfig(t *testing.T, script string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.yaml")
	body := "plugins:\n  - name: demo\n    command: " + script + "\n    timeout: 5s\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func pluginScript(t *testing.T, name string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("plugin test fixtures are POSIX shell scripts")
	}
	abs, err := filepath.Abs(filepath.Join("..", "plugin", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func runCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func TestPluginsListShowsConfiguredPlugin(t *testing.T) {
	script := pluginScript(t, "good.sh")
	cfgPath := writePluginConfig(t, script)
	out, err := runCommand(t, "plugins", "list", "--config", cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "demo") || !strings.Contains(out, script) {
		t.Fatalf("plugins list output = %q", out)
	}
}

func TestPluginsListEmpty(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "absent.yaml")
	out, err := runCommand(t, "plugins", "list", "--config", cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "no plugins configured") {
		t.Fatalf("plugins list output = %q", out)
	}
}

func TestPluginsVerifyReportsConformance(t *testing.T) {
	cfgPath := writePluginConfig(t, pluginScript(t, "violations.sh"))
	out, err := runCommand(t, "plugins", "verify", "demo", "--config", cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"handshake OK", "records ok: 1", "skipped:    5", "violations:", "dedupe_key"} {
		if !strings.Contains(out, want) {
			t.Fatalf("plugins verify output missing %q:\n%s", want, out)
		}
	}
}

func TestPluginsVerifyUnknownName(t *testing.T) {
	cfgPath := writePluginConfig(t, pluginScript(t, "good.sh"))
	_, err := runCommand(t, "plugins", "verify", "nope", "--config", cfgPath)
	if err == nil {
		t.Fatal("verify unknown plugin: err = nil, want error")
	}
}

func TestPluginsVerifyFailsOnHandshakeMismatch(t *testing.T) {
	cfgPath := writePluginConfig(t, pluginScript(t, "handshake_mismatch.sh"))
	out, err := runCommand(t, "plugins", "verify", "demo", "--config", cfgPath)
	if err == nil {
		t.Fatalf("verify mismatched plugin: err = nil, output = %q", out)
	}
	if !strings.Contains(out, "FAILED") {
		t.Fatalf("plugins verify output = %q, want FAILED line", out)
	}
}
