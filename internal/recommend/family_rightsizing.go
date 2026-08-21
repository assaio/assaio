package recommend

import (
	"strconv"
	"strings"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/humanize"
)

// rightSizeMinCandidates is how many small-output premium turns a window needs before naming
// them is worth a person's time. It is a sample floor, not a quality line: the family says
// "here are turns worth opening", never "this share is too high" -- assaio withdrew that
// verdict for want of a source (B177) and a recommendation may not smuggle it back.
const rightSizeMinCandidates = 25

func init() { Register(premiumSmallTurns{}) }

// premiumSmallTurns names premium-model turns that produced very little, as candidates to try
// on a cheaper model. It proposes an experiment over a population, not a judgement about it:
// a short answer can legitimately need the strong model, which is why the action is "try these
// and remeasure" and the follow-up is named before the result is known.
type premiumSmallTurns struct{}

func (premiumSmallTurns) Name() string { return "premium-small-turns" }

func (premiumSmallTurns) Propose(e *Evidence) []Record {
	verdict := e.Result("model-right-sizing")
	if !enough(verdict) {
		return nil
	}
	candidates := figureCount(verdict, "downgrade candidates")
	if candidates < rightSizeMinCandidates {
		return nil
	}
	return []Record{{
		ID:    "premium-small-turns/window",
		Title: "Try a cheaper model on the premium turns that produced almost nothing",
		Evidence: []string{
			humanize.Int(candidates) + " premium-model turns produced under the small-output ceiling",
			"share of premium turns: " + figureValue(verdict, "small-output premium"),
			"read on " + humanize.Int(int64(verdict.Confidence.Samples)) + " " + verdict.Confidence.Unit +
				" at " + verdict.Confidence.Label + " confidence",
		},
		Confidence: verdict.Confidence.Label,
		Scope:      "premium-model turns in this window",
		Action: "Route one routine kind of work -- the one you would least defend on the strong model -- to a cheaper " +
			"one for a fortnight. Label those sessions with `assaio-agent mark` so the comparison is per kind of work rather than an average.",
		Prerequisites: []string{
			"A kind of work you can name and route separately; a blanket model switch is not this experiment",
			"Session labels on both the before and after window, or the comparison averages research against greenfield",
		},
		Effect: "Directionally lower spend on that slice, at unknown cost to quality. assaio has no counterfactual and " +
			"will not predict a number; what it can do is tell you afterwards whether output and rework moved with it.",
		Risks: []string{
			"Task difficulty is invisible in a log: a short answer can be the hard part of the work, and the cheaper model may take more turns to reach it -- which costs more, not less.",
			"Judging the result on cost alone hides a quality regression. Read rework and survival on the same slice.",
		},
		Rollback:     "Route the work back to the premium model; nothing stored changes and no figure is lost.",
		ReviewWindow: "14d",
		FollowUp:     "`assaio-agent effectiveness --by task --compare` on the labelled slice: cost per accepted line, beside rework on the same sessions",
		Status:       StatusProposed,
	}}
}

// figureCount reads a whole-number figure off a verdict, or 0. Recommendations quote what a
// validator published rather than recomputing it, so the two can never disagree.
func figureCount(r *analyze.Result, label string) int64 {
	n, err := strconv.ParseInt(strings.ReplaceAll(figureValue(r, label), ",", ""), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// figureValue reads a figure's rendered value off a verdict, or "".
func figureValue(r *analyze.Result, label string) string {
	for i := range r.Figures {
		if r.Figures[i].Label == label {
			return r.Figures[i].Value
		}
	}
	return ""
}
