package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// goodMetricScript emits a valid handshake + Result and drops a <name>.ran sentinel, so
// tests can assert whether a command actually executed the plugin.
const goodMetricScript = `#!/bin/sh
cat >/dev/null
touch "${0%.sh}.ran"
echo '{"assaio_metric":1,"name":"demo"}'
echo '{"title":"Demo Metric","read":{"key":"watch","label":"WATCH"},"purity":0.4,"howToRead":"Directional demo.","figures":[{"label":"x","value":"1"}],"takeaway":"Demo takeaway."}'
`

const failingMetricScript = `#!/bin/sh
cat >/dev/null
exit 3
`

// writeMetricPluginConfig installs a metric plugin script plus a config.yaml declaring
// it under configHome (the test's XDG_CONFIG_HOME), returning the script path.
func writeMetricPluginConfig(t *testing.T, configHome, script string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("metric plugin test fixtures are POSIX shell scripts")
	}
	dir := filepath.Join(configHome, "assaio")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	scriptPath := filepath.Join(dir, "metric-demo.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil { //nolint:gosec // test fixture must be executable
		t.Fatal(err)
	}
	yaml := "metrics:\n  - name: demo\n    command: " + scriptPath + "\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	return scriptPath
}

func metricRanSentinel(scriptPath string) string {
	return scriptPath[:len(scriptPath)-len(".sh")] + ".ran"
}

func runRoot(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := NewRootCmd()
	var out, errBuf bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errBuf)
	root.SetArgs(args)
	err = root.Execute()
	return out.String(), errBuf.String(), err
}

func TestDashboardIncludesMetricPlugin(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	seedAnalyzeStore(t)
	writeMetricPluginConfig(t, configHome, goodMetricScript)

	outPath := filepath.Join(t.TempDir(), "assay.html")
	if _, _, err := runRoot(t, "dashboard", "--output", outPath); err != nil {
		t.Fatal(err)
	}
	html, err := os.ReadFile(outPath) //nolint:gosec // outPath is a t.TempDir()-based test path
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(html, []byte("Demo Metric")) {
		t.Fatal("dashboard HTML missing the metric plugin's verdict section")
	}
}
