package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	declared, violations, err := describeMetric(ctx, cfg)
	if err != nil {
		return analyze.Result{}, violations, err
	}
	projection := negotiate(declared, cfg.Allow)
	envelope := buildMetricInput(in, projection)
	stdin, err := envelope.marshal()
	if err != nil {
		return analyze.Result{}, nil, fmt.Errorf("metric plugin %s: encoding input: %w", cfg.Name, err)
	}
	doc, err := metricProtocol.run(ctx, cfg, stdin)
	if err != nil {
		return analyze.Result{}, nil, err
	}
	res, violations, err := parseMetricResult(doc, cfg.Name)
	if err == nil && len(projection.Withheld) > 0 {
		// Deliberately a caveat and not Result.Withheld: that field means "this window could
		// not supply what the analyzer declared it reads", and this is the opposite -- the
		// window had it and the reader's own config refused to hand it over. Putting the second
		// into the first would make one JSON field carry two contradictory meanings.
		//
		// The cost of that choice is that the denial is prose and nothing else: a consumer
		// gating on Withheld (recommend.enough) cannot tell this verdict from one computed on
		// everything the plugin asked for.
		res.Caveats = append(res.Caveats, deniedCaveat(cfg.Name, projection.Withheld))
	}
	if err != nil {
		return analyze.Result{}, violations, fmt.Errorf("metric plugin %s: %w%s", cfg.Name, err, violationSuffix(violations))
	}
	return res, nil, nil
}

// deniedCaveat states what this install refused a plugin that declared it reads it. Without it a
// reader cannot tell a metric that read the timeline from one whose local config line denied it,
// and the two verdicts rest on different evidence.
func deniedCaveat(name string, withheld []analyze.Capability) string {
	names := make([]string, len(withheld))
	for i, c := range withheld {
		names[i] = string(c)
	}
	return "Prov.: the metric plugin " + name + " declares that it reads " + strings.Join(names, ", ") +
		", which this install's `needs:` entry for it does not allow. This verdict rests on less than the plugin asked for."
}
