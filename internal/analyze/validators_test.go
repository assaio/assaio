package analyze

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

var validatorsTestNow = time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)

func testPrices() pricing.Table {
	return pricing.Table{
		"claude-sonnet-4-5": {Input: 3e-6, Output: 1.5e-5},
		"claude-opus-4-5":   {Input: 1.5e-5, Output: 7.5e-5},
	}
}

// favorableInput seeds broad, growing, low-friction usage across two projects on a
// cheaper model, so every validator's favorable read should trigger. Three sessions with
// zero compactions clear the context validator's session floor (contextMinSessionsForHealthy).
func favorableInput() Input {
	usage := []store.UsageRow{
		{
			Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", Entrypoint: "cli",
			In: 1000, Out: 2000, LinesAdded: 200, Edits: 10, ToolCalls: 12, ReworkLines: 5,
		},
		{
			Day: "2026-07-11", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "api", Entrypoint: "cli",
			In: 500, Out: 800, LinesAdded: 100, Edits: 5, ToolCalls: 6, ReworkLines: 2,
		},
		{
			Day: "2026-07-02", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", Entrypoint: "cli",
			In: 200, Out: 300, LinesAdded: 50, Edits: 2, ToolCalls: 3,
		},
	}
	sessions := []store.SessionRow{
		{
			SessionID: "s1", Project: "web", Tool: "claude-code", Model: "claude-sonnet-4-5",
			FirstTs: time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC), LastTs: time.Date(2026, 7, 10, 10, 20, 0, 0, time.UTC),
			Turns: 8, OutputTokens: 2000, PeakContextTokens: 50000, Edits: 5, ActiveMinutes: 22,
		},
		{
			SessionID: "s2", Project: "api", Tool: "claude-code", Model: "claude-sonnet-4-5",
			FirstTs: time.Date(2026, 7, 11, 9, 0, 0, 0, time.UTC), LastTs: time.Date(2026, 7, 11, 9, 15, 0, 0, time.UTC),
			Turns: 4, OutputTokens: 800, PeakContextTokens: 20000, Edits: 3, ActiveMinutes: 14,
		},
		{
			SessionID: "s3", Project: "web", Tool: "claude-code", Model: "claude-sonnet-4-5",
			FirstTs: time.Date(2026, 7, 12, 11, 0, 0, 0, time.UTC), LastTs: time.Date(2026, 7, 12, 11, 18, 0, 0, time.UTC),
			Turns: 6, OutputTokens: 1200, PeakContextTokens: 30000, Edits: 4, ActiveMinutes: 18,
		},
	}
	return BuildInput(usage, sessions, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
}

// watchInput seeds narrow, shrinking, high-friction usage concentrated on a premium
// model, so every validator's Watch read should trigger.
func watchInput() Input {
	usage := []store.UsageRow{
		{
			Day: "2026-07-10", Tool: "claude-code", Model: "claude-opus-4-5", Project: "solo",
			In: 5000, Out: 9000, LinesAdded: 10, Edits: 1, ToolCalls: 20, Rejected: 8, ReworkLines: 9,
		},
		{
			Day: "2026-07-02", Tool: "claude-code", Model: "claude-opus-4-5", Project: "solo",
			In: 1000, Out: 2000, LinesAdded: 500, Edits: 20, ToolCalls: 25,
		},
	}
	sessions := []store.SessionRow{
		{
			SessionID: "s1", Project: "solo", Tool: "claude-code", Model: "claude-opus-4-5",
			FirstTs: time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC), LastTs: time.Date(2026, 7, 10, 10, 40, 0, 0, time.UTC),
			Turns: 30, OutputTokens: 9000, PeakContextTokens: 190000, Edits: 1, Compactions: 2, ActiveMinutes: 38,
		},
		{
			SessionID: "s2", Project: "solo", Tool: "claude-code", Model: "claude-opus-4-5",
			FirstTs: time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC), LastTs: time.Date(2026, 7, 2, 10, 30, 0, 0, time.UTC),
			Turns: 25, OutputTokens: 2000, PeakContextTokens: 150000, Edits: 20, Compactions: 1, ActiveMinutes: 28,
		},
	}
	return BuildInput(usage, sessions, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
}

var allValidatorNames = []string{"adoption", "model-fit", "context", "throughput", "rework"}

// gradingValidators are the ones whose line is derived from the window or cited from
// somewhere, and which therefore still hold a good/watch verdict. Every other validator in
// allValidatorNames reports its figure and refuses the call (see unsourced.go); which list a
// validator belongs to is a decision, so it is written down here rather than discovered.
var gradingValidators = map[string]string{"adoption": "STRONG"}

func TestValidatorsFavorableRead(t *testing.T) {
	in := favorableInput()
	for name, wantRead := range gradingValidators {
		v, ok := Get(name)
		if !ok {
			t.Fatalf("validator %q not registered", name)
		}
		var buf bytes.Buffer
		if err := RenderResultText(&buf, v.Analyze(in)); err != nil {
			t.Fatalf("%s: RenderResultText error: %v", name, err)
		}
		out := buf.String()
		if !strings.Contains(out, "["+wantRead+"]") {
			t.Fatalf("%s favorable output missing [%s]:\n%s", name, wantRead, out)
		}
		if !strings.Contains(out, "Takeaway:") {
			t.Fatalf("%s output missing takeaway line:\n%s", name, out)
		}
	}
}

func TestValidatorsWatchRead(t *testing.T) {
	in := watchInput()
	for name := range gradingValidators {
		v, ok := Get(name)
		if !ok {
			t.Fatalf("validator %q not registered", name)
		}
		var buf bytes.Buffer
		if err := RenderResultText(&buf, v.Analyze(in)); err != nil {
			t.Fatalf("%s: RenderResultText error: %v", name, err)
		}
		out := buf.String()
		if !strings.Contains(out, "[WATCH]") {
			t.Fatalf("%s watch-scenario output missing [WATCH]:\n%s", name, out)
		}
	}
}

// TestUngradedValidatorsReadTheSameBothWays is the whole point of withdrawing an unsourced
// line: the favorable and unfavorable fixtures are meant to be as far apart as the metric can
// see, and a validator that gave up its verdict has to answer both the same way. A colour that
// crept back would show up here rather than on someone's dashboard.
func TestUngradedValidatorsReadTheSameBothWays(t *testing.T) {
	favorable, watch := favorableInput(), watchInput()
	for _, name := range allValidatorNames {
		if _, grades := gradingValidators[name]; grades {
			continue
		}
		t.Run(name, func(t *testing.T) {
			v, ok := Get(name)
			if !ok {
				t.Fatalf("validator %q not registered", name)
			}
			for scenario, in := range map[string]Input{"favorable": favorable, "watch": watch} {
				got := v.Analyze(in)
				if got.Read.Key == "good" || got.Read.Key == "watch" {
					t.Fatalf("%s on the %s fixture reads %+v, want no verdict: its line has no source", name, scenario, got.Read)
				}
				if got.Purity != neutralPurity {
					t.Fatalf("%s on the %s fixture gauges %v, want the neutral gauge", name, scenario, got.Purity)
				}
			}
		})
	}
}

// TestValidatorsEmptyInputSafe asserts every built-in validator handles a zero-value
// Input without panicking and renders an honest "no data" block rather than a
// misleading favorable read computed from nothing.
func TestValidatorsEmptyInputSafe(t *testing.T) {
	for _, name := range allValidatorNames {
		v, ok := Get(name)
		if !ok {
			t.Fatalf("validator %q not registered", name)
		}
		var buf bytes.Buffer
		if err := RenderResultText(&buf, v.Analyze(Input{})); err != nil {
			t.Fatalf("%s: RenderResultText error on empty Input: %v", name, err)
		}
		out := buf.String()
		if !strings.Contains(out, "No usage in this window.") {
			t.Fatalf("%s empty-input output must show the no-data hint, got:\n%s", name, out)
		}
	}
}

// TestModelFitUnrecognizedModelIsNeitherTier asserts a model absent from the price table
// lands in neither the premium nor cheaper tier, rather than silently inflating one side
// of the split.
func TestModelFitUnrecognizedModelIsNeitherTier(t *testing.T) {
	models := []ModelStat{{Model: "gpt-5", Tier: tierUnknown, Tokens: 200}}
	premium, cheaper, other, _, _ := modelTierTotals(models)
	if premium != 0 || cheaper != 0 || other != 200 {
		t.Fatalf("unpriced model must land entirely in other, got premium=%d cheaper=%d other=%d", premium, cheaper, other)
	}
}

// TestAdoptionBroadButFlatTakeaway covers the "broad, not growing" branch: more than one
// project active, but no prior-window data to compare against. Three sessions clear
// adoptionMinSessionsForBroad, so "broad" reads as a confirmed signal, not a trivial one
// (see TestAdoptionTrivialBroadIsNotConfidentlyStrong for the too-thin case).
func TestAdoptionBroadButFlatTakeaway(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", In: 10, Out: 10, LinesAdded: 5},
		{Day: "2026-07-11", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "api", In: 10, Out: 10, LinesAdded: 5},
	}
	sessions := []store.SessionRow{
		{SessionID: "s1", Project: "web", Tool: "claude-code", Model: "claude-sonnet-4-5", FirstTs: validatorsTestNow, Turns: 1},
		{SessionID: "s2", Project: "web", Tool: "claude-code", Model: "claude-sonnet-4-5", FirstTs: validatorsTestNow, Turns: 1},
		{SessionID: "s3", Project: "api", Tool: "claude-code", Model: "claude-sonnet-4-5", FirstTs: validatorsTestNow, Turns: 1},
	}
	in := BuildInput(usage, sessions, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	v, _ := Get("adoption")
	var buf bytes.Buffer
	if err := RenderResultText(&buf, v.Analyze(in)); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "[STRONG]") || !strings.Contains(out, "Usage is broad across projects.") {
		t.Fatalf("broad-but-flat adoption output = %q", out)
	}
}

// TestThroughputBarsCoverWholeWindowMatchingTotal locks in the window-consistency fix:
// the top-project bars break down the whole-window "AI lines total" figure, so usage
// outside the recent trend sub-window still shows in the bars instead of the total and
// the bars silently covering different windows.
func TestThroughputBarsCoverWholeWindowMatchingTotal(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-06-01", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "old", In: 10, Out: 10, LinesAdded: 5},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	v, _ := Get("throughput")
	got := v.Analyze(in)
	if len(got.Bars) != 1 || got.Bars[0].Label != "old" {
		t.Fatalf("Bars = %+v, want one bar for the in-window project 'old'", got.Bars)
	}
	if !strings.Contains(got.Bars[0].Value, "5") {
		t.Fatalf("Bars[0].Value = %q, want the 5 whole-window lines matching the total", got.Bars[0].Value)
	}
}

// TestReworkDashOnZeroToolCalls asserts the rejection-rate ratio renders "—" rather than
// a fabricated 0% when there are no tool calls to divide by.
func TestReworkDashOnZeroToolCalls(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", In: 10, Out: 10, LinesAdded: 5},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	v, _ := Get("rework")
	var buf bytes.Buffer
	if err := RenderResultText(&buf, v.Analyze(in)); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "rejection rate: — (0 of 0 calls that record a refusal)") {
		t.Fatalf("rework output with zero tool calls = %q", out)
	}
}

// TestValidatorsHowToReadNonEmpty asserts every built-in validator populates
// Result.HowToRead -- on real data, and on the zero-value "no data" Input -- since both
// the CLI's "? " line and the dashboard's ledger helper line render it unconditionally.
func TestValidatorsHowToReadNonEmpty(t *testing.T) {
	inputs := map[string]Input{"favorable": favorableInput(), "watch": watchInput(), "empty": {}}
	for _, name := range allValidatorNames {
		v, ok := Get(name)
		if !ok {
			t.Fatalf("validator %q not registered", name)
		}
		for inputName, in := range inputs {
			if got := v.Analyze(in).HowToRead; got == "" {
				t.Fatalf("%s: HowToRead empty on %s input", name, inputName)
			}
		}
	}
}

// TestAdoptionPerActiveDayDashWhenNoUsageDays covers Sessions present but Usage empty:
// Inventory.Days is 0, so sessions/active-day must render "—", not a divide-by-zero 0.
func TestAdoptionPerActiveDayDashWhenNoUsageDays(t *testing.T) {
	sessions := []store.SessionRow{
		{SessionID: "s1", Project: "web", Tool: "claude-code", Model: "claude-sonnet-4-5", FirstTs: validatorsTestNow, Turns: 1},
	}
	in := BuildInput(nil, sessions, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	v, _ := Get("adoption")
	var buf bytes.Buffer
	if err := RenderResultText(&buf, v.Analyze(in)); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "sessions/active-day: —") {
		t.Fatalf("adoption output with no usage days = %q", out)
	}
}

// TestAdoptionWithholdsBreadthWhenNoSourceNamesAProject: breadth counts projects, and a project
// comes from a session's working directory. Antigravity CLI writes none, so every row is
// unattributed -- and "0 projects · usage is narrow and flat" would be a verdict about how far
// AI use has spread, drawn from a field the tool never kept.
func TestAdoptionWithholdsBreadthWhenNoSourceNamesAProject(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-10", Tool: "agy", ToolCalls: 2},
		{Day: "2026-07-11", Tool: "agy", ToolCalls: 2},
	}
	sessions := []store.SessionRow{
		{SessionID: "s1", Tool: "agy", FirstTs: validatorsTestNow, Turns: 3},
		{SessionID: "s2", Tool: "agy", FirstTs: validatorsTestNow, Turns: 3},
		{SessionID: "s3", Tool: "agy", FirstTs: validatorsTestNow, Turns: 3},
	}
	in := BuildInput(usage, sessions, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	v, _ := Get("adoption")
	got := v.Analyze(in)
	for _, f := range got.Figures {
		if f.Label == "projects" && f.Value != "—" {
			t.Errorf("projects = %q, want — from a window whose source records no working directory", f.Value)
		}
	}
	if strings.Contains(got.Takeaway, "narrow and flat") {
		t.Errorf("Takeaway = %q, want it to name the missing capture rather than grade breadth", got.Takeaway)
	}
	if !strings.Contains(got.Takeaway, "working directory") {
		t.Errorf("Takeaway = %q, want it to say why breadth cannot be read", got.Takeaway)
	}
}
