package plugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Alert severities, in ascending urgency. Only SeverityError gates a run; info and warn
// are reported and pass.
const (
	SeverityInfo  = "info"
	SeverityWarn  = "warn"
	SeverityError = "error"
)

// Rule-alert contract limits. The count cap bounds what one plugin can push onto the
// check gate's output; lengths are runes, with prose room for the message.
const (
	maxRuleAlerts       = 50
	maxRuleIDLen        = 64
	maxRuleMessageLen   = 400
	maxRuleValidatorLen = 64
)

// Alert is one validated rule-plugin finding. Plugin is stamped at the boundary from the
// configured name, so an alert always carries the provenance of who raised it; Validator
// is the optional verdict the rule read, echoed back for the reader.
type Alert struct {
	Plugin    string
	Rule      string
	Severity  string
	Message   string
	Validator string
}

// wireAlerts is a rule plugin's single stdout document. Alerts is a pointer so a document
// that never mentions the key -- `null`, `{}`, or a filter that stopped matching -- is
// distinguishable from an explicit empty set and can be rejected instead of read as
// "this rule found nothing".
type wireAlerts struct {
	Alerts *[]wireAlert `json:"alerts"`
}

type wireAlert struct {
	Rule      string `json:"rule"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
	Validator string `json:"validator,omitempty"`
}

// parseRuleAlerts decodes a rule plugin's alerts document and enforces the boundary
// contract: whitelisted severity, required rule id and message, count and length caps,
// no control characters. On any violation the document is rejected whole -- assaio never
// applies a partially-sanitized alert set, since a silently dropped alert would weaken a
// gate the user believes is running.
func parseRuleAlerts(doc []byte, pluginName string) ([]Alert, []string, error) {
	var w wireAlerts
	dec := json.NewDecoder(bytes.NewReader(doc))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&w); err != nil {
		return nil, nil, fmt.Errorf("decoding alerts: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, nil, errors.New("trailing data after the alerts document")
	}
	if w.Alerts == nil {
		return nil, nil, errors.New(`no "alerts" key: emit {"alerts":[]} to report nothing`)
	}

	violations := validateRuleAlerts(*w.Alerts)
	if len(violations) > 0 {
		return nil, violations, fmt.Errorf("alerts failed %d contract check(s)", len(violations))
	}

	alerts := make([]Alert, 0, len(*w.Alerts))
	for _, a := range *w.Alerts {
		alerts = append(alerts, Alert{
			Plugin: pluginName, Rule: a.Rule, Severity: a.Severity,
			Message: a.Message, Validator: a.Validator,
		})
	}
	return alerts, nil, nil
}

func validateRuleAlerts(alerts []wireAlert) []string {
	var vs []string
	if len(alerts) > maxRuleAlerts {
		vs = append(vs, fmt.Sprintf("alerts exceeds %d entries", maxRuleAlerts))
	}
	for i, a := range alerts {
		switch a.Severity {
		case SeverityInfo, SeverityWarn, SeverityError:
		default:
			vs = append(vs, fmt.Sprintf("alerts[%d].severity %q is not one of info|warn|error", i, a.Severity))
		}
		vs = checkText(vs, fmt.Sprintf("alerts[%d].rule", i), a.Rule, maxRuleIDLen, true)
		vs = checkText(vs, fmt.Sprintf("alerts[%d].message", i), a.Message, maxRuleMessageLen, true)
		vs = checkText(vs, fmt.Sprintf("alerts[%d].validator", i), a.Validator, maxRuleValidatorLen, false)
	}
	return vs
}
