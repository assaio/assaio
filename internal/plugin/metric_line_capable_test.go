package plugin

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// lineCapableWindow mixes the three ways lines and tokens come apart: a source that records
// lines, one that records none, and a Copilot CLI session that ran two models and credited its
// lines to one of them.
func lineCapableWindow() analyze.Input {
	rows := []store.UsageRow{
		{Day: "2026-03-02", Tool: "claude-code", Model: "premium", Project: "p", Granularity: "turn", In: 1000, LinesAdded: 50},
		{Day: "2026-03-02", Tool: "gemini-cli", Model: "premium", Project: "p", Granularity: "turn", In: 3000},
		{Day: "2026-03-02", Tool: "copilot-cli", Model: "premium", Project: "p", Granularity: "session", In: 1000, LinesAdded: 100},
		{Day: "2026-03-02", Tool: "copilot-cli", Model: "cheap", Project: "p", Granularity: "session", In: 1000},
	}
	prices := pricing.Table{"premium": {Input: 1e-5}, "cheap": {Input: 1e-6}}
	return analyze.BuildInput(rows, nil, prices, time.Now(), 7*24*time.Hour, analyze.Delegation{})
}

// TestTheEnvelopeCarriesTheLineCapablePopulation: lines, tokens and cost beside each other span
// every source, so an out-of-tree $/100 lines over them repeats the bug the built-in ratio was
// corrected for. The line-capable fields are what that ratio divides.
func TestTheEnvelopeCarriesTheLineCapablePopulation(t *testing.T) {
	in := lineCapableWindow()
	wire := buildMetricInput(&in, everything())

	type want struct {
		lines, tokens int64
		cost          *float64
	}
	cost := func(v float64) *float64 { return &v }
	check := func(name string, lines, tokens int64, got *float64, w want) {
		t.Helper()
		if lines != w.lines || tokens != w.tokens {
			t.Errorf("%s: lineCapable %d lines over %d tokens, want %d over %d", name, lines, tokens, w.lines, w.tokens)
		}
		switch {
		case (got == nil) != (w.cost == nil):
			t.Errorf("%s: lineCapableCost = %v, want %v", name, got, w.cost)
		case got != nil && math.Abs(*got-*w.cost) > 1e-12:
			t.Errorf("%s: lineCapableCost = %v, want %v", name, *got, *w.cost)
		}
	}

	whole := want{150, 3000, cost(0.021)}
	check("totals", wire.Totals.LineCapableLines, wire.Totals.LineCapableTokens, wire.Totals.LineCapableCost, whole)
	check("byProject", wire.ByProject[0].LineCapableLines, wire.ByProject[0].LineCapableTokens, wire.ByProject[0].LineCapableCost, whole)
	for _, m := range wire.ByModel {
		w := want{50, 1000, cost(0.01)}
		if m.Model == "cheap" {
			w = want{}
		}
		check("byModel "+m.Model, m.LineCapableLines, m.LineCapableTokens, m.LineCapableCost, w)
	}
	if wire.Totals.Lines != 150 || wire.Totals.Tokens != 6000 {
		t.Errorf("totals lines/tokens = %d/%d, want the whole window beside the line-capable part", wire.Totals.Lines, wire.Totals.Tokens)
	}
}

// TestALineCapableColumnCanBeProjected: a field a projection cannot name is one a plugin that
// declares its columns never receives.
func TestALineCapableColumnCanBeProjected(t *testing.T) {
	in := lineCapableWindow()
	p := granting(analyze.CapUsage)
	p.Fields = map[string][]string{"byModel": {"model", "lineCapableLines", "lineCapableTokens", "lineCapableCost"}}
	out, err := envelopeOf(&in, p)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		ByModel []map[string]any `json:"byModel"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	for _, row := range doc.ByModel {
		for _, col := range p.Fields["byModel"] {
			if _, ok := row[col]; !ok {
				t.Errorf("projected byModel row %v lacks %s", row, col)
			}
		}
	}
}
