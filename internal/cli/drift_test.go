package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const driftTurn = `{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":10,"output_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n"

// driftHome points every path lookup at a fresh home holding one Claude transcript, and
// returns the transcript's path so a test can make it vanish.
func driftHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	session := filepath.Join(home, ".claude", "projects", "-x", "s1.jsonl")
	if err := os.MkdirAll(filepath.Dir(session), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(session, []byte(driftTurn), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	return session
}

// bulkHome points every path lookup at a home holding one transcript of n turns, enough
// to grow the store past its minimum page allocation.
func bulkHome(t *testing.T, n int) {
	t.Helper()
	home := t.TempDir()
	var b strings.Builder
	for i := range n {
		fmt.Fprintf(&b, `{"type":"assistant","uuid":"a%d","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":10,"output_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`+"\n", i)
	}
	session := filepath.Join(home, ".claude", "projects", "-x", "s1.jsonl")
	if err := os.MkdirAll(filepath.Dir(session), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(session, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

// vanish is the discovery-drift scenario: a first backfill establishes the baseline, then
// the source's logs are no longer where they were.
func vanish(t *testing.T) {
	t.Helper()
	session := driftHome(t)
	if _, err := runCommand(t, "backfill"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(session); err != nil {
		t.Fatal(err)
	}
}

func TestBackfillStaysQuietOnAHealthyRun(t *testing.T) {
	driftHome(t)
	if _, err := runCommand(t, "backfill"); err != nil {
		t.Fatal(err)
	}
	out, err := runCommand(t, "backfill")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "possible format drift") {
		t.Fatalf("healthy backfill must not warn: %q", out)
	}
}

func TestBackfillWarnsWhenASourceStopsBeingFound(t *testing.T) {
	vanish(t)
	out, err := runCommand(t, "backfill")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "warning: possible format drift in claude-code") {
		t.Fatalf("backfill did not warn about drift: %q", out)
	}
	if !strings.Contains(out, "discovery") {
		t.Fatalf("warning must name the canary that fired: %q", out)
	}
}

func TestDoctorReportsNoDriftOnAHealthyStore(t *testing.T) {
	driftHome(t)
	if _, err := runCommand(t, "backfill"); err != nil {
		t.Fatal(err)
	}
	out, err := runCommand(t, "doctor")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "drift:") {
		t.Fatalf("doctor is missing the drift section: %q", out)
	}
	if strings.Contains(out, "possible format drift") {
		t.Fatalf("healthy store must report no drift: %q", out)
	}
}

func TestDoctorShowsAFiredCanary(t *testing.T) {
	vanish(t)
	if _, err := runCommand(t, "backfill"); err != nil {
		t.Fatal(err)
	}
	out, err := runCommand(t, "doctor")
	if err != nil {
		t.Fatalf("doctor must stay diagnostic without --strict: %v", err)
	}
	if !strings.Contains(out, "claude-code") || !strings.Contains(out, "discovery") {
		t.Fatalf("doctor did not report the fired canary: %q", out)
	}
}

func TestDoctorStrictFailsOnAFiredCanary(t *testing.T) {
	vanish(t)
	if _, err := runCommand(t, "backfill"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCommand(t, "doctor", "--strict"); err == nil {
		t.Fatal("doctor --strict must exit non-zero when a canary fired")
	}
}

func TestDoctorStrictPassesOnAHealthyStore(t *testing.T) {
	driftHome(t)
	if _, err := runCommand(t, "backfill"); err != nil {
		t.Fatal(err)
	}
	if _, err := runCommand(t, "doctor", "--strict"); err != nil {
		t.Fatalf("doctor --strict must pass on a healthy store: %v", err)
	}
}

// TestDoctorStrictFailsOnAConfiguredSourceThatFindsNothing covers the case no canary can:
// a typo in sources: has no history to be compared against, so the very first run has to
// catch it.
func TestDoctorStrictFailsOnAConfiguredSourceThatFindsNothing(t *testing.T) {
	driftHome(t)
	cfg := filepath.Join(t.TempDir(), "config.yaml")
	body := "sources:\n  claude:\n    - " + filepath.Join(t.TempDir(), "nowhere") + "\n"
	if err := os.WriteFile(cfg, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := runCommand(t, "doctor", "--strict", "--config", cfg)
	if err == nil {
		t.Fatalf("doctor --strict must fail on a configured source with no files: %q", out)
	}
}
