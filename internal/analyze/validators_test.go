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

func TestValidatorsFavorableRead(t *testing.T) {
	want := map[string]string{
		"adoption":   "STRONG",
		"model-fit":  "HEALTHY",
		"context":    "HEALTHY",
		"throughput": "RAMPING",
		"rework":     "LOW",
	}
	in := favorableInput()
	for _, name := range allValidatorNames {
		v, ok := Get(name)
		if !ok {
			t.Fatalf("validator %q not registered", name)
		}
		var buf bytes.Buffer
		if err := RenderResultText(&buf, v.Analyze(in)); err != nil {
			t.Fatalf("%s: RenderResultText error: %v", name, err)
		}
		out := buf.String()
		if wantRead := want[name]; !strings.Contains(out, "["+wantRead+"]") {
			t.Fatalf("%s favorable output missing [%s]:\n%s", name, wantRead, out)
		}
		if !strings.Contains(out, "Takeaway:") {
			t.Fatalf("%s output missing takeaway line:\n%s", name, out)
		}
	}
}

func TestValidatorsWatchRead(t *testing.T) {
	in := watchInput()
	for _, name := range allValidatorNames {
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

// TestThroughputEmptyBarsWhenNoRecentUsage covers the honest empty state when a window's
// usage never lands in the recent sub-window: Bars must be present-but-empty, rendering
// "(none in this window)" rather than silently showing nothing.
func TestThroughputEmptyBarsWhenNoRecentUsage(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-06-01", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "old", In: 10, Out: 10, LinesAdded: 5},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	v, _ := Get("throughput")
	var buf bytes.Buffer
	if err := RenderResultText(&buf, v.Analyze(in)); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "(none in this window)") {
		t.Fatalf("throughput output with no recent usage = %q", out)
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
	if !strings.Contains(out, "rejection rate: — (0 of 0 tool calls declined)") {
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
