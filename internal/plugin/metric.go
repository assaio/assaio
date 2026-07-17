package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/assaio/assaio/internal/analyze"
)

// maxMetricStdout bounds a metric plugin's stdout: one handshake line plus one Result
// document -- far below the parser protocol's cap, since a metric emits a verdict, not
// a record stream.
const maxMetricStdout = 1 << 20

// metricHandshake is line 1 of a metric plugin's stdout:
// {"assaio_metric":1,"name":"<configured name>"}.
type metricHandshake struct {
	Protocol int    `json:"assaio_metric"`
	Name     string `json:"name"`
}

func parseMetricHandshake(line []byte, wantName string) error {
	var h metricHandshake
	if err := json.Unmarshal(line, &h); err != nil {
		return fmt.Errorf("invalid handshake: %w", err)
	}
	if h.Protocol != metricInputVersion {
		return fmt.Errorf("handshake protocol %d unsupported (want %d)", h.Protocol, metricInputVersion)
	}
	if h.Name != wantName {
		return fmt.Errorf("handshake name %q does not match configured name %q", h.Name, wantName)
	}
	return nil
}

// RunMetric invokes cfg's metric plugin over in and returns its validated Result, with
// Name stamped plugin:<name>. Any protocol or contract failure is an error -- the metric
// is rejected whole, never rendered partially sanitized (see parseMetricResult).
func RunMetric(ctx context.Context, cfg Config, in *analyze.Input) (analyze.Result, error) {
	res, _, err := runMetric(ctx, cfg, in)
	return res, err
}

// VerifyMetric invokes cfg's metric plugin like RunMetric, but also returns the
// per-check contract violations for the `metrics verify` conformance report.
func VerifyMetric(ctx context.Context, cfg Config, in *analyze.Input) (analyze.Result, []string, error) {
	return runMetric(ctx, cfg, in)
}

func runMetric(ctx context.Context, cfg Config, in *analyze.Input) (analyze.Result, []string, error) {
	envelope := buildMetricInput(in)
	stdin, err := envelope.marshal()
	if err != nil {
		return analyze.Result{}, nil, fmt.Errorf("metric plugin %s: encoding input: %w", cfg.Name, err)
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	stdout := &cappedBuffer{cap: maxMetricStdout}
	stderr := newPrefixWriter(os.Stderr, "[metric/"+cfg.Name+"] ")
	//nolint:gosec // the command is the user's own opt-in config entry, resolved via LookPath
	cmd := exec.CommandContext(ctx, cfg.Command, "analyze")
	cmd.Env = append(os.Environ(), "ASSAIO_METRIC_PROTOCOL=1")
	cmd.WaitDelay = killGrace
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	runErr := cmd.Run()
	stderr.Flush()

	if ctx.Err() != nil {
		return analyze.Result{}, nil, fmt.Errorf("metric plugin %s: timed out after %s", cfg.Name, cfg.Timeout)
	}
	if stdout.exceeded {
		return analyze.Result{}, nil, fmt.Errorf("metric plugin %s: stdout exceeded %d bytes", cfg.Name, maxMetricStdout)
	}
	if runErr != nil {
		return analyze.Result{}, nil, fmt.Errorf("metric plugin %s: %w", cfg.Name, runErr)
	}

	handshake, doc := splitHandshakeLine(stdout.buf.Bytes())
	if len(bytes.TrimSpace(handshake)) == 0 {
		return analyze.Result{}, nil, fmt.Errorf("metric plugin %s: no handshake line: plugin produced no output", cfg.Name)
	}
	if err := parseMetricHandshake(handshake, cfg.Name); err != nil {
		return analyze.Result{}, nil, fmt.Errorf("metric plugin %s: %w", cfg.Name, err)
	}
	res, violations, err := parseMetricResult(doc, cfg.Name)
	if err != nil {
		return analyze.Result{}, violations, fmt.Errorf("metric plugin %s: %w%s", cfg.Name, err, violationSuffix(violations))
	}
	return res, nil, nil
}

// splitHandshakeLine returns stdout's first line and everything after it; a plugin that
// wrote only a handshake yields an empty rest, which parseMetricResult reports as a
// missing result document.
func splitHandshakeLine(out []byte) (line, rest []byte) {
	if idx := bytes.IndexByte(out, '\n'); idx >= 0 {
		return out[:idx], out[idx+1:]
	}
	return out, nil
}

// violationSuffix folds the first violation into the error text so a bare RunMetric
// failure names the offending check without needing the verify report.
func violationSuffix(violations []string) string {
	if len(violations) == 0 {
		return ""
	}
	return ": " + violations[0]
}
