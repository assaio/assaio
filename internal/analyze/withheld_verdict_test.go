package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// TestReworkGaugeIsNeutralWhenNothingWasMeasured: a window whose only source records neither
// an undone line nor a declined call has two structural silences and no verdict. Both used to
// enter the purity average as zeros, so the faceplate drew a full 1.00 bar -- the strongest
// possible "all clear" -- next to a withheld "—". A gauge with nothing behind it sits at half.
func TestReworkGaugeIsNeutralWhenNothingWasMeasured(t *testing.T) {
	usage := []store.UsageRow{
		{
			Day: "2026-07-10", Tool: "gemini-cli", Model: "gemini-2.5-pro", Project: "web",
			In: 1000, Out: 1000,
		},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	v, ok := Get(reworkName)
	if !ok {
		t.Fatal("rework not registered")
	}
	got := v.Analyze(in)

	if got.Read != noDataRead {
		t.Fatalf("Read = %+v, want the no-verdict read: neither half of this metric is observable here", got.Read)
	}
	if got.Purity != neutralPurity {
		t.Fatalf("Purity = %v, want %v -- a full gauge beside a withheld verdict reads as a passing grade",
			got.Purity, neutralPurity)
	}
}

// TestSkillFigureAndShareShareOneDimension: the "largest single share" is computed within
// whichever of skills/sub-agents has something to compare, and the token total printed above
// it claims to be "in the dimension below". Reading that total off the *larger* dimension put
// two numbers side by side that could not both be true -- an 80% share of a figure 80% was
// never taken from.
func TestSkillFigureAndShareShareOneDimension(t *testing.T) {
	in := BuildInput([]store.UsageRow{{
		Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web",
		In: 1_000_000, Out: 1_000_000,
	}}, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	// One skill (no share to take) beside two sub-agents holding far fewer tokens: the share
	// can only come from the agents, so the total beside it has to as well.
	in.Skills = []store.AttributionRow{{Name: "review", Tokens: 900_000}}
	in.Agents = []store.AttributionRow{
		{Name: "general-purpose", Tokens: 120_000},
		{Name: "explore", Tokens: 80_000},
	}

	v, ok := Get(skillName)
	if !ok {
		t.Fatal("skill-economics not registered")
	}
	got := v.Analyze(in)
	joined := figureValues(got.Figures)
	if !strings.Contains(joined, "attributed tokens: 200.0K") {
		t.Fatalf("Figures = %q, want the sub-agent dimension's own 200.0K total, not the skill dimension's 900K", joined)
	}
	if !strings.Contains(joined, "largest single share: 60%") {
		t.Fatalf("Figures = %q, want the share taken from that same 200.0K", joined)
	}
}
