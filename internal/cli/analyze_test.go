package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/paths"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

func TestAnalyzeList(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"analyze", "--list"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{"adoption", "model-fit", "context", "throughput", "rework"} {
		if !strings.Contains(s, want) {
			t.Fatalf("analyze --list missing %q: %s", want, s)
		}
	}
	if got := strings.Count(s, "\n"); got != 5 {
		t.Fatalf("analyze --list must print exactly 5 lines, got %d: %s", got, s)
	}
}

func seedAnalyzeStore(t *testing.T) {
	t.Helper()
	dbPath, err := paths.DBPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureParent(dbPath); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	ts := time.Now().UTC()
	_, err = st.Insert(context.Background(), []usage.Record{
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "claude-opus-4-5",
			InputTokens: 1000, OutputTokens: 2000, Project: "web", DedupeKey: "1",
			LinesAdded: 100, Edits: 5, ToolCalls: 6,
		},
		{
			Tool: "claude-code", SessionID: "s2", Timestamp: ts, Model: "claude-sonnet-4-5",
			InputTokens: 200, OutputTokens: 300, Project: "api", DedupeKey: "2",
			LinesAdded: 40, Edits: 2, ToolCalls: 3,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestAnalyzeModelFitOnSeededStore(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	seedAnalyzeStore(t)

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"analyze", "model-fit"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{"MODEL-FIT", "Model Fit", "premium", "cheaper", "Takeaway:"} {
		if !strings.Contains(s, want) {
			t.Fatalf("analyze model-fit missing %q: %s", want, s)
		}
	}
	if strings.Contains(s, "ADOPTION") || strings.Contains(s, "REWORK") {
		t.Fatalf("analyze model-fit must only print the requested validator: %s", s)
	}
}

func TestAnalyzeAllValidatorsOnSeededStore(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	seedAnalyzeStore(t)

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"analyze"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{"ADOPTION", "MODEL-FIT", "CONTEXT", "THROUGHPUT", "REWORK"} {
		if !strings.Contains(s, want) {
			t.Fatalf("analyze (all) missing block %q: %s", want, s)
		}
	}
}

// TestAnalyzeListWithExtraArgErrors is the finding-6 regression: --list previously
// silently ignored any positional arguments instead of erroring, unlike the rest of the
// CLI (e.g. an unknown validator name, or an unknown --format/--by value).
func TestAnalyzeListWithExtraArgErrors(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"analyze", "--list", "somename"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected an error for --list with an extra positional argument")
	}
}

func TestAnalyzeBogusNameErrors(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"analyze", "bogus"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for unknown validator name")
	}
	if !strings.Contains(err.Error(), "bogus") || !strings.Contains(err.Error(), "adoption") {
		t.Fatalf("error must name the bad input and list valid names: %v", err)
	}
}

func TestAnalyzeEmptyStoreHint(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"analyze"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "No usage yet") || !strings.Contains(out.String(), "backfill") {
		t.Fatalf("empty store must show the rich backfill hint: %q", out.String())
	}
}

func TestAnalyzeFormatJSONIsValidJSON(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	seedAnalyzeStore(t)

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"analyze", "--format", "json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	var decoded []analyze.Result
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("analyze --format json produced invalid JSON: %v\n%s", err, out.String())
	}
	if len(decoded) != 5 {
		t.Fatalf("json output has %d entries, want 5: %s", len(decoded), out.String())
	}
	gotNames := make(map[string]bool, len(decoded))
	for _, r := range decoded {
		gotNames[r.Name] = true
	}
	for _, name := range []string{"adoption", "model-fit", "context", "throughput", "rework"} {
		if !gotNames[name] {
			t.Fatalf("json output missing entry named %q: %s", name, out.String())
		}
	}
}

// TestAnalyzeModelFitDelegationFromRealAgentRows proves the delegation share is read
// from real dedupe_key-tagged sub-agent rows in the store, not approximated -- seeds one
// "agent:"-prefixed record alongside a non-agent one and asserts the rendered share
// matches the real ratio.
func TestAnalyzeModelFitDelegationFromRealAgentRows(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dbPath, err := paths.DBPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureParent(dbPath); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	ts := time.Now().UTC()
	_, err = st.Insert(context.Background(), []usage.Record{
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "claude-sonnet-4-5",
			InputTokens: 900, OutputTokens: 100, Project: "web", DedupeKey: "turn-1",
		},
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "claude-sonnet-4-5",
			InputTokens: 90, OutputTokens: 10, Project: "web", DedupeKey: "agent:sub1",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"analyze", "model-fit"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	// sub=100 (90+10), total=1100 (1000+100) -> 100/1100 = 9.1%.
	if s := out.String(); !strings.Contains(s, "9.1%") {
		t.Fatalf("analyze model-fit missing the real 9.1%% delegation share: %s", s)
	}
}

func TestAnalyzeUnknownFormatErrors(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	seedAnalyzeStore(t)

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"analyze", "--format", "bogus"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for unknown --format value")
	}
}

func TestAnalyzeRunsConfiguredMetricPlugin(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	seedAnalyzeStore(t)
	writeMetricPluginConfig(t, configHome, goodMetricScript)

	out, _, err := runRoot(t, "analyze")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "PLUGIN:DEMO · Demo Metric") || !strings.Contains(out, "[WATCH]") {
		t.Fatalf("analyze output missing the metric plugin verdict:\n%s", out)
	}
	if !strings.Contains(out, "ADOPTION") {
		t.Fatalf("analyze output must keep built-ins alongside plugin metrics:\n%s", out)
	}
}

func TestAnalyzeMetricPluginFailureWarnsAndContinues(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	seedAnalyzeStore(t)
	writeMetricPluginConfig(t, configHome, failingMetricScript)

	out, stderr, err := runRoot(t, "analyze")
	if err != nil {
		t.Fatalf("a failing plugin must not fail the whole run: %v", err)
	}
	if !strings.Contains(stderr, "warning: metric plugin demo") {
		t.Fatalf("stderr missing plugin warning: %q", stderr)
	}
	if !strings.Contains(out, "ADOPTION") {
		t.Fatalf("built-ins must still render:\n%s", out)
	}
}

func TestAnalyzeSelectsMetricPluginByName(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	seedAnalyzeStore(t)
	writeMetricPluginConfig(t, configHome, goodMetricScript)

	out, _, err := runRoot(t, "analyze", "plugin:demo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "PLUGIN:DEMO · Demo Metric") {
		t.Fatalf("named selection missing plugin verdict:\n%s", out)
	}
	if strings.Contains(out, "ADOPTION") {
		t.Fatalf("named selection must not run unselected built-ins:\n%s", out)
	}
}

func TestAnalyzeExplicitMetricPluginFailureErrors(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	seedAnalyzeStore(t)
	writeMetricPluginConfig(t, configHome, failingMetricScript)

	if _, _, err := runRoot(t, "analyze", "plugin:demo"); err == nil {
		t.Fatal("explicitly selecting a failing metric plugin must be a hard error")
	}
}

func TestAnalyzeUnknownNameListsPluginNames(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	seedAnalyzeStore(t)
	writeMetricPluginConfig(t, configHome, goodMetricScript)

	_, _, err := runRoot(t, "analyze", "plugin:missing")
	if err == nil || !strings.Contains(err.Error(), "plugin:demo") {
		t.Fatalf("unknown-name error must list plugin names, got: %v", err)
	}
}

func TestAnalyzeListShowsMetricPluginsWithoutRunning(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	script := writeMetricPluginConfig(t, configHome, goodMetricScript)

	out, _, err := runRoot(t, "analyze", "--list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "plugin:demo") || !strings.Contains(out, "exec metric plugin") {
		t.Fatalf("--list missing configured metric plugin:\n%s", out)
	}
	if got := strings.Count(out, "\n"); got != 6 {
		t.Fatalf("--list must print 5 built-ins + 1 plugin = 6 lines, got %d:\n%s", got, out)
	}
	if _, err := os.Stat(metricRanSentinel(script)); err == nil {
		t.Fatal("--list must never execute metric plugins")
	}
}

func TestAnalyzeJSONIncludesMetricPlugin(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	seedAnalyzeStore(t)
	writeMetricPluginConfig(t, configHome, goodMetricScript)

	out, _, err := runRoot(t, "analyze", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	var results []analyze.Result
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("json output invalid: %v\n%s", err, out)
	}
	var found bool
	for _, r := range results {
		if r.Name == "plugin:demo" && r.Title == "Demo Metric" {
			found = true
		}
	}
	if !found {
		t.Fatalf("json results missing plugin:demo: %s", out)
	}
}
