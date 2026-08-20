package analyze

import (
	"strconv"
	"strings"

	"github.com/assaio/assaio/internal/layer"

	"github.com/assaio/assaio/internal/humanize"
)

const (
	skillName     = "skill-economics"
	skillTitle    = "Skill & Agent Economics"
	skillDescribe = "Which skills and sub-agents the tokens went to, and how much code each produced."
	// skillHowToRead is Result.HowToRead for this validator -- see its doc comment.
	skillHowToRead = "Shared skills and sub-agents are where a team's AI spend quietly concentrates. A skill burning a large share of the tokens is not a fault -- research and review legitimately read a lot without writing -- but it should be a deliberate choice rather than a surprise."

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

func (skillValidator) Name() string       { return skillName }
func (skillValidator) Title() string      { return skillTitle }
func (skillValidator) Describe() string   { return skillDescribe }
func (skillValidator) Layer() layer.Layer { return layer.Activity } // how the attributed tokens spread across skills

// WindowScoped: store.Attribution pools skills and sub-agents across every project.
func (skillValidator) WindowScoped() {}

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (skillValidator) Analyze(in Input) Result {
	r := Result{Name: skillName, Title: skillTitle, Describe: skillDescribe, HowToRead: skillHowToRead}
	if len(in.Skills) == 0 && len(in.Agents) == 0 {
		// A window can be full of usage and still carry no label: the reach is zero, which is
		// a different statement from "the window was empty" and has to read as one.
		r.covering(0)
		r.noData("labeled skills and sub-agents", attributionEmptyTakeaway(len(in.Usage)))
		r.Purity = neutralPurity
		r.Bars = []Bar{}
		r.Caveats = append(r.Caveats, skillCoverageCaveat())
		return r
	}
	// The share and the total it is a share of must come from the same dimension: reading
	// "80% of 500K" when the 80% is 4K of 5K in the other dimension puts two numbers side by
	// side that cannot both be true, and points the reader at a skill worth 1% of the window.
	r.restsOn(len(in.Skills)+len(in.Agents), "labeled skills and sub-agents")
	topShareOf, attributed, comparable := topAttributionShare(in.Skills, in.Agents)
	// The tokens the share is a share *of*, which is the dimension topAttributionShare picked
	// and not necessarily the larger one. With no comparable dimension there is no share to
	// stand beside, so the figure falls back to every labeled token seen.
	basis := attributedTokens(in.Skills, in.Agents)
	if comparable {
		basis = attributed
	}
	// The share is of attributed tokens, which are a slice of the window: an 80% share of 3%
	// of the tokens is a real figure about almost nothing, and only this metric can say so.
	r.covering(fracOf(basis, in.Totals.Tokens))
	// Concentration needs something to concentrate against: with one label its share is
	// 100% by construction, which is arithmetic, not a finding.
	measurable := attributed >= skillMinTokens && comparable

	r.Read = skillRead(measurable)
	r.Purity = neutralPurity
	r.Figures = []Figure{
		{Label: "skills seen", Value: strconv.Itoa(len(in.Skills))},
		{Label: "sub-agent types", Value: strconv.Itoa(len(in.Agents))},
		attributedFigure(basis, comparable),
		skillShareFigure(topShareOf, comparable),
	}
	r.Bars = attributionBars(in.Skills, in.Agents, skillTopN)
	r.BarsPseudonym = PseudonymSkill
	r.Takeaway = skillTakeaway(measurable, topShareOf)
	r.Caveats = append(r.Caveats, skillCoverageCaveat())
	if measurable {
		r.Caveats = append(r.Caveats, unsourcedLine("a single skill's share of attributed spend", ownHistoryWouldSettleIt))
	}
	return r
}

// attributedFigure renders the token total the share below it is a share of. Its Note only
// claims to be that dimension's total when there *is* a share -- otherwise the number is
// every labeled token seen, and saying "in the dimension below" would point at a "—".
func attributedFigure(basis int64, comparable bool) Figure {
	note := "all labeled turns"
	if comparable {
		note = "in the dimension below"
	}
	return Figure{Label: "attributed tokens", Value: humanize.Count(basis), Note: note}
}

// skillShareFigure renders the largest single share, "—" when no dimension holds two entries
// to compare: a lone label owns 100% of itself, and printing that as a finding is arithmetic
// dressed as evidence.
func skillShareFigure(topShare float64, comparable bool) Figure {
	if !comparable {
		return Figure{Label: "largest single share", Value: "—", Note: "needs two labels to compare"}
	}
	return Figure{Label: "largest single share", Value: humanize.PercentAt(topShare, 0), Note: "of that dimension"}
}

// skillCoverageCaveat names the sources that label a turn at all, read from the depth matrix
// so the sentence cannot outlive the list it used to spell out by hand.
func skillCoverageCaveat() string {
	return "Prov.: turns are labeled with a skill or sub-agent by " + strings.Join(attributionSources(), ", ") +
		"; usage from other sources is absent from this split, not zero."
}

// skillRead reports the neutral no-verdict Read while the window cannot support a
// concentration verdict, rather than one earned from nearly nothing or from a single label.
func skillRead(measurable bool) Read {
	if !measurable {
		return noDataRead
	}
	return reportedRead
}

// attributionEmptyTakeaway distinguishes "nothing ran" from "what ran carried no labels",
// which are different facts and would otherwise read the same.
func attributionEmptyTakeaway(usageRows int) string {
	if usageRows == 0 {
		return "No usage in this window."
	}
	return "No usage in this window carried a skill or sub-agent label."
}

func skillTakeaway(measurable bool, topShare float64) string {
	if !measurable {
		return "Not enough attributed usage this window to read how spend distributes: that needs meaningful volume and more than one skill or sub-agent to compare."
	}
	return "The largest single skill or sub-agent takes " + humanize.PercentAt(topShare, 0) +
		" of the attributed tokens. Concentration is worth confirming is deliberate; it is not by itself a problem, and nothing published says at what share it becomes one."
}
