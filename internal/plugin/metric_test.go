package plugin

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
)

func runMetricScript(t *testing.T, name string, timeout time.Duration) (analyze.Result, []string, error) {
	t.Helper()
	in := metricTestInput()
	cfg := Config{Name: "demo", Command: script(t, name), Timeout: timeout}
	return VerifyMetric(context.Background(), cfg, &in)
}

func TestRunMetricHappyPath(t *testing.T) {
	in := metricTestInput()
	cfg := Config{Name: "demo", Command: script(t, "metric_good.sh"), Timeout: 5 * time.Second}
	got, err := RunMetric(context.Background(), cfg, &in)
	if err != nil {
		t.Fatalf("RunMetric() err = %v", err)
	}
	if got.Name != "plugin:demo" {
		t.Fatalf("Name = %q, want plugin:demo", got.Name)
	}
	if got.Read.Key != "watch" || got.Read.Label != "WATCH" {
		t.Fatalf("Read = %+v, want watch/WATCH", got.Read)
	}
	if len(got.Figures) != 1 || got.Figures[0].Label != "x" {
		t.Fatalf("Figures = %+v, want the emitted figure", got.Figures)
	}
}

func TestRunMetricEmptyInputSafe(t *testing.T) {
	cfg := Config{Name: "demo", Command: script(t, "metric_good.sh"), Timeout: 5 * time.Second}
	if _, err := RunMetric(context.Background(), cfg, &analyze.Input{}); err != nil {
		t.Fatalf("RunMetric(empty Input) err = %v", err)
	}
}

func TestRunMetricFailures(t *testing.T) {
	cases := []struct {
		script  string
		timeout time.Duration
		wantSub string
	}{
		{"metric_bad_handshake.sh", 5 * time.Second, "handshake"},
		{"metric_name_mismatch.sh", 5 * time.Second, "handshake name"},
		{"metric_two_docs.sh", 5 * time.Second, "trailing data"},
		{"metric_oversize.sh", 5 * time.Second, "exceeded"},
		{"metric_timeout.sh", 200 * time.Millisecond, "timed out"},
		{"metric_nonzero.sh", 5 * time.Second, "exit status 3"},
		{"metric_silent.sh", 5 * time.Second, "no handshake"},
	}
	for _, tc := range cases {
		t.Run(tc.script, func(t *testing.T) {
			_, _, err := runMetricScript(t, tc.script, tc.timeout)
			if err == nil {
				t.Fatal("err = nil, want failure")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("err = %v, want substring %q", err, tc.wantSub)
			}
		})
	}
}

func TestVerifyMetricCollectsViolations(t *testing.T) {
	_, violations, err := runMetricScript(t, "metric_invalid_read.sh", 5*time.Second)
	if err == nil {
		t.Fatal("err = nil, want contract violation")
	}
	joined := strings.Join(violations, "; ")
	if !strings.Contains(joined, "read.key") {
		t.Fatalf("violations = %q, want a read.key reason", joined)
	}
}
