package cli

import (
	"strings"
	"testing"
)

func TestMetricsListEmpty(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	out, _, err := runRoot(t, "metrics", "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "no metric plugins configured") {
		t.Fatalf("empty list output wrong: %q", out)
	}
}

func TestMetricsListShowsConfigured(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	writeMetricPluginConfig(t, configHome, goodMetricScript)

	out, _, err := runRoot(t, "metrics", "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "demo") || !strings.Contains(out, "metric-demo.sh") || !strings.Contains(out, "timeout 1m0s") {
		t.Fatalf("metrics list output wrong: %q", out)
	}
}

func TestMetricsVerifyGood(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	seedAnalyzeStore(t)
	writeMetricPluginConfig(t, configHome, goodMetricScript)

	out, _, err := runRoot(t, "metrics", "verify", "demo")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"demo: handshake OK", "result: VALID", "PLUGIN:DEMO · Demo Metric", "[WATCH]"} {
		if !strings.Contains(out, want) {
			t.Fatalf("verify output missing %q:\n%s", want, out)
		}
	}
}

func TestMetricsVerifyReportsViolations(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	seedAnalyzeStore(t)
	invalidRead := `#!/bin/sh
cat >/dev/null
echo '{"assaio_metric":1,"name":"demo"}'
echo '{"title":"T","read":{"key":"great","label":"GREAT"},"howToRead":"H","takeaway":"K"}'
`
	writeMetricPluginConfig(t, configHome, invalidRead)

	out, _, err := runRoot(t, "metrics", "verify", "demo")
	if err == nil {
		t.Fatal("verify must return an error for a contract-violating plugin")
	}
	if !strings.Contains(out, "read.key") {
		t.Fatalf("verify output missing the violation reason:\n%s", out)
	}
}

func TestMetricsVerifyUnknownName(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, _, err := runRoot(t, "metrics", "verify", "nope"); err == nil {
		t.Fatal("verify of an unconfigured metric must error")
	}
}
