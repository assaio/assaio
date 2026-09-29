// Tests for the model-fit unknown-tier honesty fix: 100%-unpriced usage must never read
// as a confident favorable HEALTHY (see internal/analyze/model_fit.go).
package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// TestModelFitAllUnknownModelIsNotConfidentlyHealthy reproduces the bug: when 100% of a
// window's spend runs on a model absent from the price table, otherTokens was folded into
// the Purity/Read denominator but never surfaced, so premiumShare computed to 0 and the
// validator read a confident [HEALTHY] with Purity 1 -- the exact opposite of honest,
// since the whole window is invisible to pricing.
func TestModelFitAllUnknownModelIsNotConfidentlyHealthy(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-12", Tool: "claude-code", Model: "totally-unpriced-model", Project: "p", In: 1000, Out: 2000, LinesAdded: 40},
	}
	in := BuildInput(usage, nil, pricing.Table{}, validatorsTestNow, 7*24*time.Hour, Delegation{})
	v, ok := Get(modelFitName)
	if !ok {
		t.Fatal("model-fit not registered")
	}
	got := v.Analyze(in)

	if got.Read.Label == "HEALTHY" {
		t.Fatalf("Read = %+v, want NOT HEALTHY when 100%% of spend is on an unpriced model", got.Read)
	}
	if got.Purity >= 0.9 {
		t.Fatalf("Purity = %v, want a non-confident value when there is no priced-tier signal at all", got.Purity)
	}
	joined := figureValues(got.Figures)
	if !strings.Contains(joined, "unpriced") || !strings.Contains(joined, "100.0%") {
		t.Fatalf("Figures = %q, want the unpriced share surfaced as 100.0%%", joined)
	}
	if len(got.Caveats) == 0 {
		t.Fatal("Caveats empty, want a caveat flagging the unpriced share")
	}
}

// TestModelFitPartiallyUnknownStillReadsKnownTiersHonestly asserts a modest unpriced
// share (below modelFitUnknownWatchCeiling) does not force a watch by itself, and the
// premium/cheaper split is computed over the known tokens, not diluted by the unknown ones.
func TestModelFitPartiallyUnknownStillReadsKnownTiersHonestly(t *testing.T) {
	prices := pricing.Table{"claude-sonnet-4-5": {Output: 15e-6}}
	usage := []store.UsageRow{
		{Day: "2026-07-12", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "p", In: 900, Out: 900},
		{Day: "2026-07-12", Tool: "claude-code", Model: "totally-unpriced-model", Project: "p", In: 100, Out: 100},
	}
	in := BuildInput(usage, nil, prices, validatorsTestNow, 7*24*time.Hour, Delegation{})
	v, _ := Get(modelFitName)
	got := v.Analyze(in)

	if got.Read != reportedRead {
		t.Fatalf("Read = %+v, want the split reported: unpriced share is only 200/2000=10%%, well under the ceiling that withholds it", got.Read)
	}
	joined := figureValues(got.Figures)
	if !strings.Contains(joined, "10.0%") {
		t.Fatalf("Figures = %q, want the unpriced share surfaced as 10.0%%", joined)
	}
}

// TestModelFitNoUnknownTokensOmitsUnpricedFigure keeps the report uncluttered when every
// model this window is priced: no "unpriced" line at all, not a redundant "0.0%" one.
func TestModelFitNoUnknownTokensOmitsUnpricedFigure(t *testing.T) {
	got := modelFitValidator{}.Analyze(favorableInput())
	for _, f := range got.Figures {
		if strings.Contains(f.Label, "unpriced") {
			t.Fatalf("Figures = %+v, want no unpriced figure when every model is priced", got.Figures)
		}
	}
}

// TestLinesPerMTokWithholdsATrivialBase is the artifact the shipped golden printed: 5,298,857
// lines per million tokens off a 3.5K-token base, beside a real 6,386 -- an 830x contrast that
// is a property of the denominator, under a line suggesting the cheaper model is the bargain.
func TestLinesPerMTokWithholdsATrivialBase(t *testing.T) {
	if got := linesPerMTok(18_500, 3_500); got != "—" {
		t.Fatalf("linesPerMTok(18500, 3500) = %q, want a withheld rate off a trivial base", got)
	}
	if got := linesPerMTok(0, 0); got != "—" {
		t.Fatalf("linesPerMTok(0, 0) = %q, want a dash", got)
	}
	if got := linesPerMTok(6_000, 1_000_000); got == "—" {
		t.Fatal("a rate over a million tokens was withheld")
	}
}

// TestModelFitTierLinesComeOnlyFromTheirOwnModel: a tier's lines per 1M tokens divides only
// usage whose source records the lines of the model that ran. A source that records no line
// would dilute the rate, and a Copilot CLI session that ran two models credits every line to one
// of them: counted, this window reads 636.4 for premium and 0.0 for cheaper.
func TestModelFitTierLinesComeOnlyFromTheirOwnModel(t *testing.T) {
	day := validatorsTestNow.Format("2006-01-02")
	usage := []store.UsageRow{
		{Day: day, Tool: "claude-code", Model: "premium-model", Project: "p", In: 1_000_000, LinesAdded: 500},
		{Day: day, Tool: "gemini-cli", Model: "premium-model", Project: "p", In: 1_000_000},
		{Day: day, Tool: "copilot-cli", Model: "premium-model", Project: "p", Granularity: "session", In: 200_000, LinesAdded: 900},
		{Day: day, Tool: "copilot-cli", Model: "cheap-model", Project: "p", Granularity: "session", In: 200_000},
	}
	in := BuildInput(usage, nil, preparedTestPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := modelFitValidator{}.Analyze(in)

	if note := figureFor(got.Figures, premiumTierLabel()).Note; note != "500.0 lines/1M tok" {
		t.Errorf("premium note = %q, want 500.0 lines/1M tok", note)
	}
	if note := figureFor(got.Figures, cheaperTierLabel()).Note; note != "— lines/1M tok" {
		t.Errorf("cheaper note = %q, want a dash: its only usage is a session whose lines went to another model", note)
	}
	if !strings.Contains(strings.Join(got.Caveats, "\n"), "leaves out 58% of tiered tokens") {
		t.Errorf("Caveats = %q, want the left-out tiered tokens stated", got.Caveats)
	}
}
