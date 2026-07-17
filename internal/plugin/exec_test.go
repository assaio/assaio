package plugin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func script(t *testing.T, name string) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestRunHappyPath(t *testing.T) {
	cfg := Config{Name: "demo", Command: script(t, "good.sh"), Timeout: 5 * time.Second}
	recs, stats, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run() err = %v", err)
	}
	if stats.Records != 2 || stats.Skipped != 0 {
		t.Fatalf("stats = %+v, want Records=2 Skipped=0", stats)
	}
	if len(recs) != 2 {
		t.Fatalf("len(recs) = %d, want 2", len(recs))
	}
	for _, r := range recs {
		if r.Tool != "plugin:demo" {
			t.Fatalf("Tool = %q, want namespaced plugin:demo", r.Tool)
		}
	}
	if recs[0].Project != "myrepo" || recs[0].GitBranch != "main" || recs[0].Entrypoint != "cli" {
		t.Fatalf("optional fields not carried through: %+v", recs[0])
	}
}

func TestRunHandshakeMismatch(t *testing.T) {
	cfg := Config{Name: "demo", Command: script(t, "handshake_mismatch.sh"), Timeout: 5 * time.Second}
	_, _, err := Run(context.Background(), cfg)
	if err == nil {
		t.Fatal("Run() err = nil, want handshake mismatch error")
	}
	if !strings.Contains(err.Error(), "handshake tool") {
		t.Fatalf("err = %v, want handshake tool mismatch message", err)
	}
}

func TestRunInvalidRecordsSkippedAndCounted(t *testing.T) {
	cfg := Config{Name: "demo", Command: script(t, "violations.sh"), Timeout: 5 * time.Second}
	recs, stats, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run() err = %v", err)
	}
	if stats.Records != 1 {
		t.Fatalf("stats.Records = %d, want 1", stats.Records)
	}
	// negative token, empty dedupe_key, bad timestamp, invalid granularity, invalid JSON.
	if stats.Skipped != 5 {
		t.Fatalf("stats.Skipped = %d, want 5", stats.Skipped)
	}
	if len(recs) != 1 {
		t.Fatalf("len(recs) = %d, want 1", len(recs))
	}
}

func TestVerifyCollectsViolationReasons(t *testing.T) {
	cfg := Config{Name: "demo", Command: script(t, "violations.sh"), Timeout: 5 * time.Second}
	_, violations, stats, err := Verify(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Verify() err = %v", err)
	}
	if len(violations) != stats.Skipped {
		t.Fatalf("len(violations) = %d, want %d (== stats.Skipped)", len(violations), stats.Skipped)
	}
	for _, v := range violations {
		if v.Line <= 1 {
			t.Fatalf("violation line = %d, want > 1 (handshake is line 1)", v.Line)
		}
		if v.Reason == "" {
			t.Fatal("violation reason empty")
		}
	}
}

func TestRunTimeoutKillsPlugin(t *testing.T) {
	cfg := Config{Name: "demo", Command: script(t, "timeout.sh"), Timeout: 200 * time.Millisecond}
	start := time.Now()
	_, _, err := Run(context.Background(), cfg)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("Run() err = nil, want timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("err = %v, want timeout message", err)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("Run() took %s, want it to return promptly after the configured timeout", elapsed)
	}
}

func TestRunNonZeroExitFails(t *testing.T) {
	cfg := Config{Name: "demo", Command: script(t, "nonzero_exit.sh"), Timeout: 5 * time.Second}
	_, _, err := Run(context.Background(), cfg)
	if err == nil {
		t.Fatal("Run() err = nil, want non-zero exit error")
	}
}

func TestRunStderrPassthroughPrefixesPluginName(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = origStderr }()

	cfg := Config{Name: "demo", Command: script(t, "good.sh"), Timeout: 5 * time.Second}
	_, _, runErr := Run(context.Background(), cfg)
	if closeErr := w.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	os.Stderr = origStderr
	if runErr != nil {
		t.Fatalf("Run() err = %v", runErr)
	}

	buf := make([]byte, 4096)
	n, readErr := r.Read(buf)
	if readErr != nil && !errors.Is(readErr, os.ErrClosed) {
		t.Fatal(readErr)
	}
	got := string(buf[:n])
	if !strings.Contains(got, "[plugin/demo] diagnostic line on stderr") {
		t.Fatalf("stderr output = %q, want prefixed diagnostic line", got)
	}
}

func TestRunToolNamespacing(t *testing.T) {
	cfg := Config{Name: "demo", Command: script(t, "good.sh"), Timeout: 5 * time.Second}
	recs, _, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range recs {
		if strings.HasPrefix(r.Tool, "plugin:plugin:") {
			t.Fatalf("double-namespaced tool: %q", r.Tool)
		}
		if r.Tool != "plugin:"+cfg.Name {
			t.Fatalf("Tool = %q, want plugin:%s", r.Tool, cfg.Name)
		}
	}
}
