// Tests for two rework honesty fixes: an unconfirmed rejection rate (no tool calls) must
// not be folded into Read/Purity as a fabricated favorable zero, and a true 0/0 rework
// rate (no lines added) must render "—" like every other undefined ratio in this Result,
// not a formatted "0%" (see internal/analyze/rework.go).
package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// TestReworkUnconfirmedRejectionIsNotFabricatedFavorable reproduces the bug: a window
// with real, low churn (10 of 100 lines reworked, well under the ceiling) but zero
// recorded tool calls used to default rejectionRate to 0 and fold it straight into the
// low/Purity computation as if rejection had been confirmed low. Before the fix this read
// [LOW] with Purity 0.95; after the fix, rejection is excluded (not fabricated as 0), so
// the pair can't be certified low on a signal that was never measured, and Purity reflects
// only the real, known rework rate.
func TestReworkUnconfirmedRejectionIsNotFabricatedFavorable(t *testing.T) {
	usage := []store.UsageRow{
		{
			Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web",
			In: 10, Out: 10, LinesAdded: 100, ReworkLines: 10,
		},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	v, ok := Get(reworkName)
	if !ok {
		t.Fatal("rework not registered")
	}
	got := v.Analyze(in)

	if got.Read.Label == "LOW" {
		t.Fatalf("Read = %+v, want NOT LOW: rejection was never measured (0 tool calls), so the pair can't be confirmed low", got.Read)
	}
	// Nor WATCH: flagging a window because half of it could not be measured is the same
	// fabrication in the other direction, and it read that way on every Codex window forever.
	if got.Read.Label == "WATCH" {
		t.Fatalf("Read = %+v, want the verdict withheld rather than flagged on an unmeasured half", got.Read)
	}
	if got.Purity != neutralPurity {
		t.Fatalf("Purity = %v, want the neutral %v: a withheld verdict draws no gauge in either direction", got.Purity, neutralPurity)
	}
	joined := figureValues(got.Figures)
	if !strings.Contains(joined, "rejection rate: — ") {
		t.Fatalf("Figures = %q, want rejection rate to still show — (0 of 0 tool calls)", joined)
	}
	found := false
	for _, c := range got.Caveats {
		if strings.Contains(c, "unconfirmed") {
			found = true
		}
	}
	if !found {
		t.Fatalf("Caveats = %+v, want a caveat flagging the unconfirmed rejection rate", got.Caveats)
	}
}

// TestReworkKnownRejectionCaveatOmitted asserts the "unconfirmed rejection" caveat only
// appears when there really were zero tool calls -- a window with real tool-call data
// must not carry a caveat that doesn't apply to it.
func TestReworkKnownRejectionCaveatOmitted(t *testing.T) {
	got := reworkValidator{}.Analyze(favorableInput())
	for _, c := range got.Caveats {
		if strings.Contains(c, "unconfirmed") {
			t.Fatalf("Caveats = %+v, want no unconfirmed-rejection caveat when tool calls were recorded", got.Caveats)
		}
	}
}

// TestReworkZeroLinesAddedShowsDashNotZeroPercent is the display-convention fix: churn's
// ReworkRate defaults to 0 on a true 0/0 (no lines added this window), exactly the same
// zero-denominator shape as the rejection rate. The rework Figure must show "—" like
// rejection rate already does, never a formatted "0%" that reads as a measured zero.
func TestReworkZeroLinesAddedShowsDashNotZeroPercent(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", In: 10, Out: 10, ToolCalls: 5},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	v, _ := Get(reworkName)
	got := v.Analyze(in)

	joined := figureValues(got.Figures)
	if !strings.Contains(joined, "rework: — (") {
		t.Fatalf("Figures = %q, want the rework figure to show — (undefined ratio) when no lines were added, not a fabricated 0%%", joined)
	}
	if strings.Contains(joined, "rework: 0%") {
		t.Fatalf("Figures = %q, a 0/0 rework ratio must never render as 0%%", joined)
	}
}

// A source recording changed lines but no undone one used to dilute the churn denominator
// with output nobody watched, which is ADR 0011's failure mode in the one validator the
// decision did not reach: the verdict reads LOW because a silent source is large.
func TestReworkExcludesSourcesThatRecordNoUndoneLine(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", In: 10, Out: 10, LinesAdded: 100, ReworkLines: 40, ToolCalls: 20},
		{Day: "2026-07-10", Tool: "copilot-cli", Model: "claude-sonnet-4-5", Project: "web", In: 10, Out: 10, LinesAdded: 900},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := Evaluate(mustGet(t, reworkName), &in)

	joined := figureValues(got.Figures)
	if !strings.Contains(joined, "rework: 40%") {
		t.Fatalf("Figures = %q, want 40%% over the capable source, not the 4%% the silent one produces", joined)
	}
	if got.Confidence.Signal == nil || *got.Confidence.Signal >= 1 {
		t.Fatalf("signal coverage = %v, want the reach of the rework-capable subset declared", got.Confidence.Signal)
	}
	if !hasCaveatContaining(got.Caveats, "absent from the rate, not churn-free") {
		t.Fatalf("Caveats = %+v, want the excluded sources named", got.Caveats)
	}
}

// No source recording an undone line means no churn verdict, rather than a window
// certified low-friction because nothing could contradict it.
func TestReworkWithholdsVerdictWhenNoSourceRecordsChurn(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-10", Tool: "copilot-cli", Model: "claude-sonnet-4-5", Project: "web", In: 10, Out: 10, LinesAdded: 900},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := Evaluate(mustGet(t, reworkName), &in)

	if got.Read.Key != "neutral" {
		t.Fatalf("Read = %+v, want the verdict withheld where churn cannot be measured", got.Read)
	}
	if !strings.Contains(figureValues(got.Figures), "rework: —") {
		t.Fatalf("Figures = %q, want the rate withheld", figureValues(got.Figures))
	}
}

// One rejection rate, two validators: the same 102-of-65098 refusals must not read "0%" on
// one surface and "0.2%" on the other, which is what a bare whole-number share produced.
func TestReworkAndFrictionAgreeOnTheRejectionRate(t *testing.T) {
	usage := []store.UsageRow{
		{
			Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web",
			In: 10, Out: 10, LinesAdded: 100, ReworkLines: 5, ToolCalls: 65098, Rejected: 102,
		},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	rework := figureFor(Evaluate(mustGet(t, reworkName), &in).Figures, "rejection rate")
	friction := figureFor(Evaluate(mustGet(t, frictionName), &in).Figures, "rejection rate")

	if rework.Value != friction.Value {
		t.Fatalf("rework says %q, friction says %q -- one measurement, two answers", rework.Value, friction.Value)
	}
	if rework.Value == "0%" {
		t.Fatalf("rejection rate = %q, but 102 refusals were recorded: a real signal must never read as absent", rework.Value)
	}
}

func hasCaveatContaining(caveats []string, want string) bool {
	for _, c := range caveats {
		if strings.Contains(c, want) {
			return true
		}
	}
	return false
}

// TestReworkWithholdsARateAboveItsWhole is B132's class at the other end: the parser's
// per-transcript budget cannot stop a *window* from opening between an addition and its
// removal, and ChurnStat's own doc has always said so. Nothing clamped or disclosed it at the
// render, so `--since 1d` over a day that removed 400 of yesterday's lines printed
// "rework: 8000%" -- a share of a whole this window never held.
func TestReworkWithholdsARateAboveItsWhole(t *testing.T) {
	usage := []store.UsageRow{
		{
			Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web",
			In: 100, Out: 100, LinesAdded: 5, ReworkLines: 400, ToolCalls: 20,
		},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	v, _ := Get(reworkName)
	got := v.Analyze(in)

	joined := figureValues(got.Figures)
	if !strings.Contains(joined, "rework: — (") {
		t.Fatalf("Figures = %q, want the rework rate withheld when removals outnumber the additions", joined)
	}
	if strings.Contains(joined, "8000%") {
		t.Fatalf("Figures = %q, a share may never exceed its own whole", joined)
	}
	if !hasCaveatAbout(got.Caveats, "never counted as added") {
		t.Fatalf("Caveats = %q, want the withheld rate to say why", got.Caveats)
	}
	// The churn half is unmeasurable, so the pair is withheld -- and a withheld verdict draws
	// the neutral gauge. Folding the 80.0 ratio in instead would clamp it to 0 and paint
	// friction nobody measured.
	if got.Read != noDataRead {
		t.Fatalf("Read = %+v, want the verdict withheld when the churn half has no share to state", got.Read)
	}
	if got.Purity != neutralPurity {
		t.Fatalf("Purity = %v, want the neutral %v", got.Purity, neutralPurity)
	}
}

func hasCaveatAbout(caveats []string, want string) bool {
	for _, c := range caveats {
		if strings.Contains(c, want) {
			return true
		}
	}
	return false
}
