package analyze

import (
	"strconv"

	"github.com/assaio/assaio/internal/store"
)

const (
	skillName     = "skill-economics"
	skillTitle    = "Skill & Agent Economics"
	skillDescribe = "Which skills and sub-agents the tokens went to, and how much code each produced."
	// skillHowToRead is Result.HowToRead for this validator -- see its doc comment.
	skillHowToRead = "Shared skills and sub-agents are where a team's AI spend quietly concentrates. A skill burning a large share of the tokens is not a fault -- research and review legitimately read a lot without writing -- but it should be a deliberate choice rather than a surprise."
	// skillConcentrationCeiling is the token share above which a single skill or sub-agent
	// dominates the window enough to name.
	skillConcentrationCeiling = 0.5
	// skillMinTokens is the attributed-token floor below which the split is too thin to read.
	skillMinTokens = 100_000
	// skillMinEntries is the per-dimension floor for a concentration verdict: one label
	// always holds 100% of its own dimension.
	skillMinEntries = 2
	skillTopN       = 5
)

func init() { Register(skillValidator{}) }

// skillValidator reads where attributed tokens went: which skills and which sub-agent
// types the window's spend concentrated in.
type skillValidator struct{}

func (skillValidator) Name() string     { return skillName }
func (skillValidator) Title() string    { return skillTitle }
func (skillValidator) Describe() string { return skillDescribe }

// WindowScoped: store.Attribution pools skills and sub-agents across every project.
func (skillValidator) WindowScoped() {}

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (skillValidator) Analyze(in Input) Result {
	r := Result{Name: skillName, Title: skillTitle, Describe: skillDescribe, HowToRead: skillHowToRead}
	if len(in.Skills) == 0 && len(in.Agents) == 0 {
		r.Read = noDataRead
		r.Purity = 0.5
		r.Bars = []Bar{}
		r.Takeaway = attributionEmptyTakeaway(len(in.Usage))
		r.Caveats = append(r.Caveats, skillCoverageCaveat)
		return r
	}
	// The share and the total it is a share of must come from the same dimension: reading
	// "80% of 500K" when the 80% is 4K of 5K in the other dimension puts two numbers side by
	// side that cannot both be true, and points the reader at a skill worth 1% of the window.
	topShareOf, attributed, comparable := topAttributionShare(in.Skills, in.Agents)
	// Concentration needs something to concentrate against: with one label its share is
	// 100% by construction, which is arithmetic, not a finding.
	measurable := attributed >= skillMinTokens && comparable
	spread := measurable && topShareOf <= skillConcentrationCeiling

	r.Read = skillRead(measurable, spread)
	r.Purity = skillPurity(topShareOf, measurable)
	r.Figures = []Figure{
		{Label: "skills seen", Value: strconv.Itoa(len(in.Skills))},
		{Label: "sub-agent types", Value: strconv.Itoa(len(in.Agents))},
		{Label: "attributed tokens", Value: compactCount(attributed), Note: "in the dimension below"},
		{Label: "largest single share", Value: formatPercent(topShareOf, 0), Note: "of that dimension"},
	}
	r.Bars = attributionBars(in.Skills, in.Agents, skillTopN)
	r.BarsPseudonym = PseudonymSkill
	r.Takeaway = skillTakeaway(measurable, spread)
	r.Caveats = append(r.Caveats, skillCoverageCaveat)
	return r
}

// skillCoverageCaveat states the single-tool coverage limit this metric carries.
const skillCoverageCaveat = "Prov.: only Claude Code labels turns with a skill or sub-agent today; usage from other tools is absent from this split, not zero."

// attributionTokens sums one attribution dimension's tokens.
func attributionTokens(rows []store.AttributionRow) int64 {
	var sum int64
	for i := range rows {
		sum += rows[i].Tokens
	}
	return sum
}

// topAttributionShare is the largest single skill's or sub-agent's share of its own
// dimension -- the concentration this metric reads -- together with that dimension's own
// token total, so a caller can never render the share against the other dimension's sum.
// Only a dimension holding at least two entries counts: with one, the share is 100%
// whatever happened. ok is false when neither dimension has two, so the caller reports no
// verdict instead of a guaranteed one. Rows arrive sorted by Tokens descending.
func topAttributionShare(skills, agents []store.AttributionRow) (share float64, total int64, ok bool) {
	for _, rows := range [][]store.AttributionRow{skills, agents} {
		if len(rows) < skillMinEntries {
			continue
		}
		dimension := attributionTokens(rows)
		if s := shareOf(rows[0].Tokens, dimension); !ok || s > share {
			share, total, ok = s, dimension, true
		}
	}
	return share, total, ok
}

// attributionBars ranks skills and sub-agents together, each labelled by dimension so a
// skill is never mistaken for an agent. topN caps each dimension separately.
func attributionBars(skills, agents []store.AttributionRow, topN int) []Bar {
	bars := make([]Bar, 0, 2*topN)
	bars = append(bars, dimensionBars(skills, "skill", topN)...)
	bars = append(bars, dimensionBars(agents, "agent", topN)...)
	return bars
}

// dimensionBars renders one attribution dimension's top entries, scaled against that
// dimension's own largest so the two dimensions stay independently readable. The name is
// the Label alone -- skill and sub-agent names are user-authored, so --anonymize replaces
// the whole Label; carrying the dimension in Value keeps it legible once that happens.
func dimensionBars(rows []store.AttributionRow, kind string, topN int) []Bar {
	kept := rows
	if topN > 0 && len(kept) > topN {
		kept = kept[:topN]
	}
	var maxTokens int64
	if len(kept) > 0 {
		maxTokens = kept[0].Tokens
	}
	total := attributionTokens(rows)
	out := make([]Bar, 0, len(kept))
	for i := range kept {
		out = append(out, Bar{
			Label: kept[i].Name,
			Value: kind + " · " + compactCount(kept[i].Tokens) + " tokens · " + formatPercent(shareOf(kept[i].Tokens, total), 0) + " · " + strconv.FormatInt(kept[i].Lines, 10) + " lines",
			Frac:  fracOf(kept[i].Tokens, maxTokens),
		})
	}
	return out
}

// skillRead reports the neutral no-verdict Read while the window cannot support a
// concentration verdict, rather than one earned from nearly nothing or from a single label.
func skillRead(measurable, spread bool) Read {
	if !measurable {
		return noDataRead
	}
	return readFor(spread, "Spread")
}

// skillPurity falls as one skill or sub-agent takes more of the attributed tokens; a window
// that cannot support the verdict yields the neutral 0.5 used for "not enough evidence yet".
func skillPurity(topShare float64, measurable bool) float64 {
	if !measurable {
		return 0.5
	}
	return clamp01(1 - topShare)
}

// attributionEmptyTakeaway distinguishes "nothing ran" from "what ran carried no labels",
// which are different facts and would otherwise read the same.
func attributionEmptyTakeaway(usageRows int) string {
	if usageRows == 0 {
		return "No usage in this window."
	}
	return "No usage in this window carried a skill or sub-agent label."
}

func skillTakeaway(measurable, spread bool) string {
	switch {
	case !measurable:
		return "Not enough attributed usage this window to say whether spend concentrates: that needs meaningful volume and more than one skill or sub-agent to compare."
	case spread:
		return "Attributed spend is spread across skills and sub-agents rather than concentrated in one."
	default:
		return "One skill or sub-agent takes most of the attributed tokens -- worth confirming that is deliberate."
	}
}
