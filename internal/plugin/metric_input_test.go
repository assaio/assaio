package plugin

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/parser"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

var metricInputTestNow = time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)

func metricTestInput() analyze.Input {
	prices := pricing.Table{
		"m-priced": {Input: 1e-6, Output: 2e-6, CacheWrite: 1.25e-6, CacheRead: 1e-7},
	}
	usage := []store.UsageRow{
		{Day: "2026-07-16", Tool: "claude-code", Model: "m-priced", Project: "web", Entrypoint: "cli", In: 100, Out: 200, CacheRead: 1000, CacheWrite: 50, Reasoning: 10, LinesAdded: 40, LinesRemoved: 5, Edits: 3, ToolCalls: 7, Rejected: 1, Compactions: 1, ReworkLines: 2},
		{Day: "2026-07-15", Tool: "codex", Model: "m-unpriced", Project: "api", In: 10, Out: 20},
	}
	sessions := []store.SessionRow{
		{SessionID: "s1", Project: "web", Tool: "claude-code", Model: "m-priced", FirstTs: metricInputTestNow.Add(-time.Hour), LastTs: metricInputTestNow, Turns: 4, OutputTokens: 200, PeakContextTokens: 1100, Edits: 3, Compactions: 1, ActiveMinutes: 42.5},
	}
	return analyze.BuildInput(usage, sessions, prices, metricInputTestNow, 7*24*time.Hour, analyze.Delegation{Sub: 10, Total: 100})
}

func roundTrip(t *testing.T, in *analyze.Input) map[string]any {
	t.Helper()
	envelope := buildMetricInput(in)
	raw, err := envelope.marshal()
	if err != nil {
		t.Fatalf("marshal() err = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("envelope is not valid JSON: %v", err)
	}
	return got
}

func TestBuildMetricInputShapesWire(t *testing.T) {
	in := metricTestInput()
	got := roundTrip(t, &in)

	if v, ok := got["assaio_metric_input"].(float64); !ok || v != 1 {
		t.Fatalf("assaio_metric_input = %v, want 1", got["assaio_metric_input"])
	}
	if now, _ := got["now"].(string); now != "2026-07-17T10:00:00Z" {
		t.Fatalf("now = %v, want RFC3339 wall clock", got["now"])
	}
	if days, _ := got["recentDays"].(float64); days != 7 {
		t.Fatalf("recentDays = %v, want 7", got["recentDays"])
	}

	usage, _ := got["usage"].([]any)
	if len(usage) != 2 {
		t.Fatalf("len(usage) = %d, want 2", len(usage))
	}
	row, _ := usage[0].(map[string]any)
	for key, want := range map[string]any{
		"day": "2026-07-16", "tool": "claude-code", "model": "m-priced", "project": "web",
		"entrypoint": "cli", "member": "", "in": 100.0, "out": 200.0, "cacheRead": 1000.0,
		"cacheWrite": 50.0, "reasoning": 10.0, "linesAdded": 40.0, "linesRemoved": 5.0,
		"edits": 3.0, "toolCalls": 7.0, "rejected": 1.0, "compactions": 1.0, "reworkLines": 2.0,
	} {
		if row[key] != want {
			t.Fatalf("usage[0][%q] = %v, want %v", key, row[key], want)
		}
	}

	sessions, _ := got["sessions"].([]any)
	if len(sessions) != 1 {
		t.Fatalf("len(sessions) = %d, want 1", len(sessions))
	}
	sess, _ := sessions[0].(map[string]any)
	if sess["sessionId"] != "s1" || sess["peakContextTokens"] != 1100.0 || sess["activeMinutes"] != 42.5 {
		t.Fatalf("session wire fields wrong: %v", sess)
	}

	delegation, _ := got["delegation"].(map[string]any)
	if delegation["sub"] != 10.0 || delegation["total"] != 100.0 {
		t.Fatalf("delegation = %v, want sub=10 total=100", delegation)
	}

	byModel, _ := got["byModel"].([]any)
	if len(byModel) != 2 {
		t.Fatalf("len(byModel) = %d, want 2", len(byModel))
	}
	costs := map[string]any{}
	for _, m := range byModel {
		stat, _ := m.(map[string]any)
		costs[stat["model"].(string)] = stat["cost"]
	}
	if costs["m-unpriced"] != nil {
		t.Fatalf("unpriced model cost = %v, want null", costs["m-unpriced"])
	}
	if _, ok := costs["m-priced"].(float64); !ok {
		t.Fatalf("priced model cost = %v, want a number", costs["m-priced"])
	}

	byProject, _ := got["byProject"].([]any)
	if len(byProject) != 2 {
		t.Fatalf("len(byProject) = %d, want 2", len(byProject))
	}

	totals, _ := got["totals"].(map[string]any)
	if totals["tokens"] == nil || totals["cacheEfficiency"] == nil {
		t.Fatalf("totals missing fields: %v", totals)
	}
	if _, ok := totals["cost"].(float64); !ok {
		t.Fatalf("totals.cost = %v, want a number (priced rows still price)", totals["cost"])
	}
	if totals["priced"] != false {
		t.Fatalf("totals.priced = %v, want false: an unpriced model in the window flags the undercount", totals["priced"])
	}

	prices, _ := got["prices"].(map[string]any)
	if len(prices) != 1 {
		t.Fatalf("prices = %v, want exactly the one priced model present in usage", prices)
	}
	price, _ := prices["m-priced"].(map[string]any)
	for key, want := range map[string]float64{"input": 1e-6, "output": 2e-6, "cacheWrite": 1.25e-6, "cacheRead": 1e-7} {
		if price[key] != want {
			t.Fatalf("prices[m-priced][%q] = %v, want %v", key, price[key], want)
		}
	}
}

func TestBuildMetricInputEmpty(t *testing.T) {
	got := roundTrip(t, &analyze.Input{})
	for _, key := range []string{"usage", "sessions", "byModel", "byProject"} {
		arr, ok := got[key].([]any)
		if !ok {
			t.Fatalf("%s = %v (%T), want an empty JSON array, never null", key, got[key], got[key])
		}
		if len(arr) != 0 {
			t.Fatalf("%s = %v, want empty", key, arr)
		}
	}
	if _, ok := got["prices"].(map[string]any); !ok {
		t.Fatalf("prices = %v (%T), want an empty JSON object, never null", got["prices"], got["prices"])
	}
}

// TestMetricInputCarriesGranularity gives a metric plugin the same provenance the core
// reports have: without it, a plugin summing usage rows would blend session aggregates
// into a per-turn figure with no way to notice.
func TestMetricInputCarriesGranularity(t *testing.T) {
	in := analyze.BuildInput([]store.UsageRow{
		{Day: "2026-07-16", Tool: "plugin:acme", Model: "m-priced", Granularity: "session", In: 100, Out: 200},
	}, nil, pricing.Table{}, metricInputTestNow, 7*24*time.Hour, analyze.Delegation{})
	got := roundTrip(t, &in)
	rows, ok := got["usage"].([]any)
	if !ok || len(rows) == 0 {
		t.Fatalf("usage rows missing: %+v", got)
	}
	row, _ := rows[0].(map[string]any)
	if row["granularity"] != "session" {
		t.Fatalf("usage row = %+v, want granularity=session", row)
	}
}

// TestMetricInputCarriesSourceCapability gives an out-of-tree metric the one thing the
// envelope withheld: which of the zeros on a row are measurements and which are silence.
// A plugin that cannot ask this is exposed to exactly the bug ADR 0011 fixed in-tree.
func TestMetricInputCarriesSourceCapability(t *testing.T) {
	in := analyze.BuildInput([]store.UsageRow{
		{Day: "2026-07-16", Tool: "claude-code", Model: "m-priced", Granularity: "turn", In: 100, Out: 200},
		{Day: "2026-07-16", Tool: "copilot-cli", Model: "m-priced", Granularity: "session", In: 10, Out: 20},
	}, nil, pricing.Table{}, metricInputTestNow, 7*24*time.Hour, analyze.Delegation{})
	got := roundTrip(t, &in)

	answers, ok := got["answers"].(map[string]any)
	if !ok {
		t.Fatalf("answers = %v (%T), want a JSON object", got["answers"], got["answers"])
	}
	if len(answers) != 2 {
		t.Fatalf("answers = %v, want one entry per tool in the window", answers)
	}
	if !answersContain(answers, "claude-code", parser.SignalReworkLines) {
		t.Errorf("answers[claude-code] = %v, want the rework signal it records", answers["claude-code"])
	}
	if answersContain(answers, "copilot-cli", parser.SignalReworkLines) {
		t.Errorf("answers[copilot-cli] = %v, want no rework claim for a source that records none", answers["copilot-cli"])
	}
	if !answersContain(answers, "copilot-cli", parser.SignalLinesAdded) {
		t.Errorf("answers[copilot-cli] = %v, want the line signals it does record", answers["copilot-cli"])
	}
}

// An out-of-tree parser's records must carry a capability too, and it is the exec
// protocol's floor rather than a table assaio maintains for someone else's tool.
func TestMetricInputCapabilityForPluginSources(t *testing.T) {
	in := analyze.BuildInput([]store.UsageRow{
		{Day: "2026-07-16", Tool: "plugin:acme", Model: "m-priced", Granularity: "turn", In: 100, Out: 200},
	}, nil, pricing.Table{}, metricInputTestNow, 7*24*time.Hour, analyze.Delegation{})
	got := roundTrip(t, &in)
	answers, _ := got["answers"].(map[string]any)

	if !answersContain(answers, "plugin:acme", parser.SignalTokensTotal) {
		t.Errorf("answers[plugin:acme] = %v, want the exec protocol's token floor", answers["plugin:acme"])
	}
	if answersContain(answers, "plugin:acme", parser.SignalEditsCount) {
		t.Errorf("answers[plugin:acme] = %v, want no activity claim the protocol cannot carry", answers["plugin:acme"])
	}
}

func answersContain(answers map[string]any, tool, id string) bool {
	ids, ok := answers[tool].([]any)
	if !ok {
		return false
	}
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}
