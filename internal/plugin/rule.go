package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/assaio/assaio/internal/analyze"
)

// maxRuleStdout bounds a rule plugin's stdout: one handshake line plus one alerts
// document, which is smaller than a metric's Result by construction.
const maxRuleStdout = 1 << 20

// ruleProtocol is the ADR 0005 stdio contract: `<command> evaluate`, the window's
// verdicts on stdin, handshake + one alerts document on stdout.
var ruleProtocol = docProtocol{
	kind:      "rule",
	verb:      "evaluate",
	env:       "ASSAIO_RULE_PROTOCOL",
	maxStdout: maxRuleStdout,
	handshake: parseRuleHandshake,
}

// ruleHandshake is line 1 of a rule plugin's stdout:
// {"assaio_rule":1,"name":"<configured name>"}.
type ruleHandshake struct {
	Protocol int    `json:"assaio_rule"`
	Name     string `json:"name"`
}

func parseRuleHandshake(line []byte, wantName string) error {
	var h ruleHandshake
	if err := json.Unmarshal(line, &h); err != nil {
		return fmt.Errorf("invalid handshake: %w", err)
	}
	if h.Protocol != ruleInputVersion {
		return fmt.Errorf("handshake protocol %d unsupported (want %d)", h.Protocol, ruleInputVersion)
	}
	if h.Name != wantName {
		return fmt.Errorf("handshake name %q does not match configured name %q", h.Name, wantName)
	}
	return nil
}

// RunRule invokes cfg's rule plugin over the window's verdicts and returns its validated
// alerts, each stamped with the plugin that raised it. Any protocol or contract failure
// is an error: the plugin's alerts are dropped whole, and the caller must treat a rule
// that did not run as a gate that did not pass (see internal/cli's check wiring).
func RunRule(ctx context.Context, cfg Config, verdicts []analyze.Result) ([]Alert, error) {
	envelope := buildRuleInput(verdicts)
	stdin, err := envelope.marshal()
	if err != nil {
		return nil, fmt.Errorf("rule plugin %s: encoding input: %w", cfg.Name, err)
	}
	doc, err := ruleProtocol.run(ctx, cfg, stdin)
	if err != nil {
		return nil, err
	}
	alerts, violations, err := parseRuleAlerts(doc, cfg.Name)
	if err != nil {
		return nil, fmt.Errorf("rule plugin %s: %w%s", cfg.Name, err, violationSuffix(violations))
	}
	return alerts, nil
}
