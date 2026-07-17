package plugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/assaio/assaio/internal/analyze"
)

// Metric-result contract limits. Counts and lengths bound what a plugin can push onto
// the CLI and dashboard; lengths are runes, prose fields (howToRead, takeaway, caveats)
// get more room than labels.
const (
	maxMetricFigures  = 12
	maxMetricBars     = 30
	maxMetricCaveats  = 8
	maxMetricLabelLen = 16
	maxMetricTitleLen = 80
	maxMetricDescLen  = 200
	maxMetricShortLen = 120
	maxMetricProseLen = 400
)

// parseMetricResult decodes a metric plugin's single Result document and enforces the
// boundary contract: whitelisted read.key, required prose, count/length caps, no control
// characters, purity/frac clamped. On any violation the metric is rejected whole --
// assaio never renders a partially-sanitized or fabricated verdict -- with every reason
// returned for `metrics verify`. Name is always stamped plugin:<name> so a plugin cannot
// shadow a built-in validator.
func parseMetricResult(doc []byte, pluginName string) (analyze.Result, []string, error) {
	var r analyze.Result
	dec := json.NewDecoder(bytes.NewReader(doc))
	if err := dec.Decode(&r); err != nil {
		return analyze.Result{}, nil, fmt.Errorf("decoding result: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return analyze.Result{}, nil, errors.New("trailing data after the result document")
	}

	violations := validateMetricResult(&r)
	if len(violations) > 0 {
		return analyze.Result{}, violations, fmt.Errorf("result failed %d contract check(s)", len(violations))
	}

	r.Name = "plugin:" + pluginName
	r.Purity = clampFrac(r.Purity)
	for i := range r.Bars {
		r.Bars[i].Frac = clampFrac(r.Bars[i].Frac)
	}
	return r, nil, nil
}

func validateMetricResult(r *analyze.Result) []string {
	var vs []string
	switch r.Read.Key {
	case "good", "watch", "neutral":
	default:
		vs = append(vs, fmt.Sprintf("read.key %q is not one of good|watch|neutral", r.Read.Key))
	}
	vs = checkText(vs, "read.label", r.Read.Label, maxMetricLabelLen, true)
	vs = checkText(vs, "title", r.Title, maxMetricTitleLen, true)
	vs = checkText(vs, "describe", r.Describe, maxMetricDescLen, false)
	vs = checkText(vs, "howToRead", r.HowToRead, maxMetricProseLen, true)
	vs = checkText(vs, "takeaway", r.Takeaway, maxMetricProseLen, true)

	if len(r.Figures) > maxMetricFigures {
		vs = append(vs, fmt.Sprintf("figures exceeds %d entries", maxMetricFigures))
	}
	for i, f := range r.Figures {
		vs = checkText(vs, fmt.Sprintf("figures[%d].label", i), f.Label, maxMetricShortLen, true)
		vs = checkText(vs, fmt.Sprintf("figures[%d].value", i), f.Value, maxMetricShortLen, true)
		vs = checkText(vs, fmt.Sprintf("figures[%d].note", i), f.Note, maxMetricShortLen, false)
	}

	if len(r.Bars) > maxMetricBars {
		vs = append(vs, fmt.Sprintf("bars exceeds %d entries", maxMetricBars))
	}
	for i, b := range r.Bars {
		vs = checkText(vs, fmt.Sprintf("bars[%d].label", i), b.Label, maxMetricShortLen, true)
		vs = checkText(vs, fmt.Sprintf("bars[%d].value", i), b.Value, maxMetricShortLen, true)
	}

	if len(r.Caveats) > maxMetricCaveats {
		vs = append(vs, fmt.Sprintf("caveats exceeds %d entries", maxMetricCaveats))
	}
	for i, c := range r.Caveats {
		vs = checkText(vs, fmt.Sprintf("caveats[%d]", i), c, maxMetricProseLen, true)
	}
	return vs
}

// checkText appends a violation for a missing required field, a rune count over max, or
// any control character (single-line fields by contract; also a terminal-escape guard --
// HTML is already covered by html/template escaping).
func checkText(vs []string, field, s string, maxRunes int, required bool) []string {
	if s == "" {
		if required {
			vs = append(vs, field+" is required")
		}
		return vs
	}
	if utf8.RuneCountInString(s) > maxRunes {
		vs = append(vs, fmt.Sprintf("%s exceeds %d chars", field, maxRunes))
	}
	if strings.ContainsFunc(s, unicode.IsControl) {
		vs = append(vs, field+" contains a control character")
	}
	return vs
}

func clampFrac(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}
