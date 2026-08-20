package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// skillInput seeds a window with the given skill and sub-agent attribution rows, plus one
// usage row so "no usage at all" and "usage without labels" stay distinguishable.
func skillInput(skills, agents []store.AttributionRow) Input {
	usage := []store.UsageRow{
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", In: 1000, Out: 500},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	in.Skills, in.Agents = skills, agents
	return in
}

// TestSkillEconomicsFlagsOneSkillDominating asserts a single skill taking most of the
// attributed tokens is surfaced -- the "what is quietly eating the budget" case.
func TestSkillEconomicsFlagsOneSkillDominating(t *testing.T) {
	got := mustGet(t, skillName).Analyze(skillInput(
		[]store.AttributionRow{
			{Name: "deep-research", Tokens: 900_000, Lines: 5, Records: 200, Sessions: 12},
			{Name: "code-review", Tokens: 100_000, Lines: 40, Records: 30, Sessions: 4},
		}, nil,
	))

	if got.Read != reportedRead {
		t.Fatalf("Read = %+v, want the share reported: concentration is not by itself a fault", got.Read)
	}
	if !strings.Contains(figureValues(got.Figures), "90%") {
		t.Fatalf("Figures = %q, want the 90%% largest-share figure", figureValues(got.Figures))
	}
	// The name is the Label alone so --anonymize can replace it; the dimension rides in
	// Value, which pseudonymization leaves intact.
	if got.Bars[0].Label != "deep-research" || !strings.HasPrefix(got.Bars[0].Value, "skill · ") {
		t.Fatalf("Bars[0] = %+v, want the bare name as Label and the dimension in Value", got.Bars[0])
	}
	if got.BarsPseudonym != PseudonymSkill {
		t.Fatal("BarsPseudonym must be set: skill and sub-agent names are user-authored")
	}
}

// TestSkillEconomicsSpreadAcrossSkills asserts an even split gets the same descriptive read a
// concentrated one does, with the share left to tell them apart (B177).
func TestSkillEconomicsSpreadAcrossSkills(t *testing.T) {
	got := mustGet(t, skillName).Analyze(skillInput(
		[]store.AttributionRow{
			{Name: "a", Tokens: 300_000, Lines: 30},
			{Name: "b", Tokens: 300_000, Lines: 25},
			{Name: "c", Tokens: 300_000, Lines: 20},
		}, nil,
	))

	if got.Read != reportedRead {
		t.Fatalf("Read = %+v, want the descriptive read for an even three-way split", got.Read)
	}
	if !strings.Contains(figureValues(got.Figures), "33%") {
		t.Fatalf("Figures = %q, want the 33%% largest-share figure", figureValues(got.Figures))
	}
}

// TestSkillEconomicsLabelsBothDimensions asserts skills and sub-agents render side by side
// and stay distinguishable, since a skill and an agent can share a name.
func TestSkillEconomicsLabelsBothDimensions(t *testing.T) {
	got := mustGet(t, skillName).Analyze(skillInput(
		[]store.AttributionRow{{Name: "review", Tokens: 400_000, Lines: 30}, {Name: "other", Tokens: 100_000}},
		[]store.AttributionRow{{Name: "review", Tokens: 200_000, Lines: 10}, {Name: "other", Tokens: 50_000}},
	))

	pairs := make([]string, 0, len(got.Bars))
	for _, b := range got.Bars {
		pairs = append(pairs, b.Label+"="+b.Value)
	}
	joined := strings.Join(pairs, " | ")
	if !strings.Contains(joined, "review=skill · ") || !strings.Contains(joined, "review=agent · ") {
		t.Fatalf("Bars = %q, want the skill and the agent distinguishable by dimension", joined)
	}
}

// TestSkillEconomicsUnlabelledUsageIsNotZero asserts a window with usage but no attribution
// says exactly that, rather than reporting an empty split that looks like a finding.
func TestSkillEconomicsUnlabelledUsageIsNotZero(t *testing.T) {
	got := mustGet(t, skillName).Analyze(skillInput(nil, nil))

	if got.Read != noDataRead {
		t.Fatalf("Read = %+v, want the neutral read with no attribution", got.Read)
	}
	if !strings.Contains(got.Takeaway, "carried a skill or sub-agent label") {
		t.Fatalf("Takeaway = %q, want the unlabelled-usage explanation", got.Takeaway)
	}
}

// TestSkillEconomicsEmptyInputSafe asserts a truly empty window is reported as no usage,
// distinct from usage that simply carried no labels.
func TestSkillEconomicsEmptyInputSafe(t *testing.T) {
	got := mustGet(t, skillName).Analyze(Input{})
	if got.Read != noDataRead || got.Takeaway != "No usage in this window." {
		t.Fatalf("empty Input = %+v, want the no-data read and takeaway", got)
	}
}

// TestSkillEconomicsSingleLabelIsUndefined asserts a window with exactly one label yields
// the neutral read: its share is 100% by construction, so a WATCH there would be arithmetic
// reported as a finding, and anyone using one skill would be flagged forever.
func TestSkillEconomicsSingleLabelIsUndefined(t *testing.T) {
	got := mustGet(t, skillName).Analyze(skillInput(
		[]store.AttributionRow{{Name: "code-review", Tokens: 400_000, Lines: 30}}, nil,
	))

	if got.Read != noDataRead {
		t.Fatalf("Read = %+v, want the neutral read with a single label to compare", got.Read)
	}
	if !strings.Contains(got.Takeaway, "more than one skill or sub-agent") {
		t.Fatalf("Takeaway = %q, want the nothing-to-compare explanation", got.Takeaway)
	}
}
