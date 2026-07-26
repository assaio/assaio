package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// ruleScript emits a valid handshake plus the given alerts document, so a test can pick
// exactly which severities the gate sees.
func ruleScript(alertsDoc string) string {
	return "#!/bin/sh\ncat >/dev/null\necho '{\"assaio_rule\":1,\"name\":\"demo\"}'\necho '" + alertsDoc + "'\n"
}

// writeRulePluginConfig installs a rule plugin script plus a config.yaml declaring it
// under configHome (the test's XDG_CONFIG_HOME).
func writeRulePluginConfig(t *testing.T, configHome, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("rule plugin test fixtures are POSIX shell scripts")
	}
	dir := filepath.Join(configHome, "assaio")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	scriptPath := filepath.Join(dir, "rule-demo.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil { //nolint:gosec // test fixture must be executable
		t.Fatal(err)
	}
	yaml := "rules:\n  - name: demo\n    command: " + scriptPath + "\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
}

// runCheckWithRule seeds a store, points config at a rule plugin emitting script, and
// runs `check` with no budget so only the rule gate can fail it.
func runCheckWithRule(t *testing.T, script string) (stdout, stderr string, err error) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	db := filepath.Join(t.TempDir(), "u.db")
	seedStoreAt(t, db, []usage.Record{
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: time.Now().UTC(), Model: "claude-opus-4-5",
			InputTokens: 1000, OutputTokens: 2000, Project: "web", DedupeKey: "1", LinesAdded: 100,
		},
	})
	writeRulePluginConfig(t, configHome, script)
	return runRoot(t, "check", "--db", db)
}

func TestCheckExitsNonZeroOnErrorSeverityAlert(t *testing.T) {
	out, _, err := runCheckWithRule(t, ruleScript(`{"alerts":[{"rule":"budget-drift","severity":"error","message":"Spend outran the plan.","validator":"subscription-fit"}]}`))
	if err == nil {
		t.Fatal("check must exit non-zero when a rule plugin raises an error-severity alert")
	}
	if !strings.Contains(err.Error(), "rule gate failed: demo/budget-drift") {
		t.Fatalf("err = %v, want the gating rule named", err)
	}
	for _, want := range []string{"[error]", "demo/budget-drift", "Spend outran the plan.", "(subscription-fit)"} {
		if !strings.Contains(out, want) {
			t.Fatalf("check output missing %q:\n%s", want, out)
		}
	}
}

func TestCheckPassesOnNonGatingAlerts(t *testing.T) {
	out, _, err := runCheckWithRule(t, ruleScript(`{"alerts":[{"rule":"cache-cold","severity":"warn","message":"Cache reuse is low."},{"rule":"fyi","severity":"info","message":"Nothing to do."}]}`))
	if err != nil {
		t.Fatalf("warn and info alerts must not gate: %v\n%s", err, out)
	}
	for _, want := range []string{"[warn ]", "demo/cache-cold", "[info ]", "demo/fyi"} {
		if !strings.Contains(out, want) {
			t.Fatalf("check output missing %q:\n%s", want, out)
		}
	}
}

// TestCheckFailsClosedOnBrokenRule is the honesty rule for a gate: a rule that could not
// be evaluated is reported and fails the run rather than passing silently.
func TestCheckFailsClosedOnBrokenRule(t *testing.T) {
	out, errOut, err := runCheckWithRule(t, ruleScript(`not json at all`))
	if err == nil {
		t.Fatalf("check must exit non-zero when a rule plugin cannot be evaluated:\n%s", out)
	}
	if !strings.Contains(err.Error(), "rule plugin demo did not run") {
		t.Fatalf("err = %v, want the unevaluated rule named", err)
	}
	if !strings.Contains(errOut, "warning: rule plugin demo: decoding alerts") {
		t.Fatalf("stderr = %q, want the protocol failure warned about", errOut)
	}
	if !strings.Contains(out, "demo: not evaluated") {
		t.Fatalf("check output must show the unevaluated rule, not a silent pass:\n%s", out)
	}
}

func TestCheckReportsNoAlerts(t *testing.T) {
	out, _, err := runCheckWithRule(t, ruleScript(`{"alerts":[]}`))
	if err != nil {
		t.Fatalf("a quiet rule must pass: %v", err)
	}
	if !strings.Contains(out, "rules\n  no alerts") {
		t.Fatalf("check output missing the empty rules section:\n%s", out)
	}
}

// TestCheckWithoutRulesSkipsTheSection keeps check's shape unchanged for the users who
// configure no rules at all.
func TestCheckWithoutRulesSkipsTheSection(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	db := filepath.Join(t.TempDir(), "u.db")
	seedStoreAt(t, db, []usage.Record{
		{Tool: "claude-code", SessionID: "s1", Timestamp: time.Now().UTC(), Model: "claude-opus-4-5", InputTokens: 10, OutputTokens: 20, DedupeKey: "1"},
	})
	out, _, err := runRoot(t, "check", "--db", db)
	if err != nil {
		t.Fatalf("check err = %v", err)
	}
	if strings.Contains(out, "rules") {
		t.Fatalf("no rules configured must print no rules section:\n%s", out)
	}
}
