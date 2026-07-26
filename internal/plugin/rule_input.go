package plugin

import (
	"encoding/json"

	"github.com/assaio/assaio/internal/analyze"
)

// ruleInputVersion is the stdin envelope's protocol version, carried in the
// assaio_rule_input key.
const ruleInputVersion = 1

// ruleInput is the wire envelope a rule plugin reads on stdin: the window's validator
// verdicts and nothing else. A rule reasons over verdicts, never over raw usage -- the
// data a rule sees is exactly what `analyze --format json` already prints, so the
// protocol adds no new exposure of a user's usage (ADR 0005).
type ruleInput struct {
	Version  int              `json:"assaio_rule_input"`
	Verdicts []analyze.Result `json:"verdicts"`
}

// buildRuleInput copies the verdicts with Bars stripped. Bars rank by project, skill, and
// sub-agent -- names a person chose, which nothing on this path pseudonymizes -- and a rule
// gates on a verdict, never on which repository produced it. PRIVACY.md promises a rule
// plugin sees titles, verdict labels, figures, and takeaways; this is what keeps that true.
func buildRuleInput(verdicts []analyze.Result) ruleInput {
	stripped := make([]analyze.Result, len(verdicts))
	for i := range verdicts {
		stripped[i] = verdicts[i]
		stripped[i].Bars = nil
		stripped[i].BarsPseudonym = ""
	}
	return ruleInput{Version: ruleInputVersion, Verdicts: stripped}
}

func (ri *ruleInput) marshal() ([]byte, error) { return json.Marshal(ri) }
