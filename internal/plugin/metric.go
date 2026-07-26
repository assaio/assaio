package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/assaio/assaio/internal/analyze"
)

// maxMetricStdout bounds a metric plugin's stdout: one handshake line plus one Result
// document -- far below the parser protocol's cap, since a metric emits a verdict, not
// a record stream.
const maxMetricStdout = 1 << 20

// metricProtocol is the ADR 0004 stdio contract: `<command> analyze`, prepared Input on
// stdin, handshake + one Result on stdout.
var metricProtocol = docProtocol{
	kind:      "metric",
	verb:      "analyze",
	env:       "ASSAIO_METRIC_PROTOCOL",
	maxStdout: maxMetricStdout,
	handshake: parseMetricHandshake,
}

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
	doc, err := metricProtocol.run(ctx, cfg, stdin)
	if err != nil {
		return analyze.Result{}, nil, err
	}
	res, violations, err := parseMetricResult(doc, cfg.Name)
	if err != nil {
		return analyze.Result{}, violations, fmt.Errorf("metric plugin %s: %w%s", cfg.Name, err, violationSuffix(violations))
	}
	return res, nil, nil
}
