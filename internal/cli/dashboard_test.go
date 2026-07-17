package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/paths"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

func seedDashboardStore(t *testing.T) {
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
			InputTokens: 100, OutputTokens: 200, Project: "web", DedupeKey: "1", LinesAdded: 50,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestDashboardWritesAnonymizedByDefault(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	seedDashboardStore(t)

	out := filepath.Join(t.TempDir(), "dash.html")
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"dashboard", "--output", out})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Wrote dashboard to "+out) || !strings.Contains(buf.String(), "pseudonymized") {
		t.Fatalf("unexpected stdout: %q", buf.String())
	}
	html, err := os.ReadFile(out) //nolint:gosec // out is a t.TempDir()-based test path
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(html), ">web<") {
		t.Fatalf("dashboard must not show real project name by default: %s", html)
	}
	if !strings.Contains(string(html), "project-") {
		t.Fatalf("dashboard must show a pseudonymized project name: %s", html)
	}
}

func TestDashboardNoAnonymizeShowsRealNames(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	seedDashboardStore(t)

	out := filepath.Join(t.TempDir(), "dash.html")
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"dashboard", "--output", out, "--no-anonymize"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "pseudonymized") {
		t.Fatalf("--no-anonymize must not claim pseudonymization: %q", buf.String())
	}
	html, err := os.ReadFile(out) //nolint:gosec // out is a t.TempDir()-based test path
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), ">web<") {
		t.Fatalf("--no-anonymize must show the real project name: %s", html)
	}
}

func TestDashboardConfigDefaultOverridesAnonymize(t *testing.T) {
	xdgConfig := t.TempDir()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)
	seedDashboardStore(t)

	cfgDir := filepath.Join(xdgConfig, "assaio")
	if err := os.MkdirAll(cfgDir, 0o750); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(cfgDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("privacy:\n  anonymize: false\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "dash.html")
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"dashboard", "--output", out})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	html, err := os.ReadFile(out) //nolint:gosec // out is a t.TempDir()-based test path
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), ">web<") {
		t.Fatalf("config privacy.anonymize=false must show real project name when no flag overrides it: %s", html)
	}
}

// TestResolveAnonymize is the finding-1 regression: resolveAnonymize must honor the
// explicit VALUE of --anonymize/--no-anonymize, not just whether the flag was set. Table
// covers every direction, including the explicit "=false" spellings that previously
// leaked real names (--no-anonymize=false) or denied them (--anonymize=false).
func TestResolveAnonymize(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		cfgDefault bool
		want       bool
	}{
		{"neither flag falls back to config true", nil, true, true},
		{"neither flag falls back to config false", nil, false, false},
		{"bare --anonymize", []string{"--anonymize"}, false, true},
		{"--anonymize=true", []string{"--anonymize=true"}, false, true},
		{"--anonymize=false", []string{"--anonymize=false"}, true, false},
		{"bare --no-anonymize", []string{"--no-anonymize"}, true, false},
		{"--no-anonymize=true", []string{"--no-anonymize=true"}, true, false},
		{"--no-anonymize=false", []string{"--no-anonymize=false"}, false, true},
		{"--no-anonymize=false overrides a true config default too", []string{"--no-anonymize=false"}, true, true},
		{"both set: --no-anonymize=false still wins and keeps anonymize on", []string{"--anonymize=true", "--no-anonymize=false"}, false, true},
		{"both set: --no-anonymize=true wins over --anonymize=true", []string{"--anonymize=true", "--no-anonymize=true"}, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cobra.Command{Run: func(*cobra.Command, []string) {}}
			c.Flags().Bool("anonymize", true, "")
			c.Flags().Bool("no-anonymize", false, "")
			if err := c.ParseFlags(tt.args); err != nil {
				t.Fatal(err)
			}
			if got := resolveAnonymize(c, tt.cfgDefault); got != tt.want {
				t.Fatalf("resolveAnonymize(%v, cfgDefault=%v) = %v, want %v", tt.args, tt.cfgDefault, got, tt.want)
			}
		})
	}
}

// TestDashboardNoAnonymizeFalseKeepsAnonymizationOn is the CRITICAL finding-1 case:
// --no-anonymize=false means the caller does NOT want the no-anonymize behavior, i.e.
// they want anonymization ON -- the opposite of the pre-fix behavior, which only checked
// whether the flag was Changed and always turned anonymization off.
func TestDashboardNoAnonymizeFalseKeepsAnonymizationOn(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	seedDashboardStore(t)

	out := filepath.Join(t.TempDir(), "dash.html")
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"dashboard", "--output", out, "--no-anonymize=false"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "pseudonymized") {
		t.Fatalf("--no-anonymize=false must keep anonymization on and say so: %q", buf.String())
	}
	html, err := os.ReadFile(out) //nolint:gosec // out is a t.TempDir()-based test path
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(html), ">web<") {
		t.Fatalf("--no-anonymize=false must not leak the real project name: %s", html)
	}
	if !strings.Contains(string(html), "project-") {
		t.Fatalf("--no-anonymize=false must show a pseudonymized project name: %s", html)
	}
}

// TestDashboardAnonymizeFalseShowsRealNames is the second finding-1 case: --anonymize=false
// asks for real names, so it must behave like --no-anonymize, not like anonymization was
// requested.
func TestDashboardAnonymizeFalseShowsRealNames(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	seedDashboardStore(t)

	out := filepath.Join(t.TempDir(), "dash.html")
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"dashboard", "--output", out, "--anonymize=false"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "pseudonymized") {
		t.Fatalf("--anonymize=false must not claim pseudonymization: %q", buf.String())
	}
	html, err := os.ReadFile(out) //nolint:gosec // out is a t.TempDir()-based test path
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), ">web<") {
		t.Fatalf("--anonymize=false must show the real project name: %s", html)
	}
}

func TestDashboardEmptyStoreStillWritesValidHTML(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	out := filepath.Join(t.TempDir(), "dash.html")
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"dashboard", "--output", out})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No usage yet") {
		t.Fatalf("empty store must print the backfill hint: %q", buf.String())
	}
	html, err := os.ReadFile(out) //nolint:gosec // out is a t.TempDir()-based test path
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "No usage in this window.") {
		t.Fatalf("empty store must still write a valid honest-empty-state Assay report: %s", html)
	}
	if strings.Contains(string(html), `class="drilldown"`) {
		t.Fatalf("empty store has no project to drill into, so the drill-down section must be omitted: %s", html)
	}
}

func TestWindowLabel(t *testing.T) {
	tests := []struct{ since, want string }{
		{"30d", "last 30 days"},
		{"1d", "last 1 day"},
		{"0d", "last 0 days"},
		{"7d", "last 7 days"},
	}
	for _, tt := range tests {
		if got := windowLabel(tt.since); got != tt.want {
			t.Fatalf("windowLabel(%q) = %q, want %q", tt.since, got, tt.want)
		}
	}
}
