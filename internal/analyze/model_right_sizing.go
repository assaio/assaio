package analyze

import (
	"strconv"

	"github.com/assaio/assaio/internal/layer"

	"github.com/assaio/assaio/internal/humanize"
)

const (
	rightSizeName      = "model-right-sizing"
	rightSizeTitle     = "Model Right-Sizing"
	rightSizeDescribe  = "Turns on a premium model that produced very little output -- small tasks a cheaper, faster model might handle just as well."
	rightSizeHowToRead = "A premium-model turn that emits only a handful of output tokens is often boilerplate or a quick answer a cheaper model handles as well. On a flat plan this is about speed and rate limits, not dollars. Task difficulty is invisible, so this is a prompt to look at a few, never a verdict."
	// RightSizeSmallOutput is the output-token ceiling below which a premium turn is a
	// downgrade candidate; exported so the CLI passes it to store.TurnSizing.
	RightSizeSmallOutput = 300

	// rightSizeMinTurns is the premium-turn floor below which the share is too thin to report.
	rightSizeMinTurns = 20
)

func init() { Register(rightSizeValidator{}) }

// rightSizeValidator flags premium-model turns that produced little output -- small tasks a
// cheaper or faster model might have handled -- as candidates to review, not a verdict.
type rightSizeValidator struct{}

func (rightSizeValidator) Name() string       { return rightSizeName }
func (rightSizeValidator) Title() string      { return rightSizeTitle }
func (rightSizeValidator) Describe() string   { return rightSizeDescribe }
func (rightSizeValidator) Layer() layer.Layer { return layer.Activity } // premium turns whose output tokens were small

// Needs declares the per-model turn counts this metric reads at a grain the daily usage
// aggregate hides, and the price table that decides which model is the premium one.
func (rightSizeValidator) Needs() []Capability { return []Capability{CapTurnSizing, CapPrices} }

// WindowScoped: TurnSizing is per-model across the window, with no project dimension.
func (rightSizeValidator) WindowScoped() {}

//nolint:gocritic // Input is required by the Validator interface; analyzed once per run, not a hot path.
func (rightSizeValidator) Analyze(in Input) Result {
	r := Result{Name: rightSizeName, Title: rightSizeTitle, Describe: rightSizeDescribe, HowToRead: rightSizeHowToRead}
	var premiumTurns, smallPremium int64
	for i := range in.TurnSizing {
		m := &in.TurnSizing[i]
		if modelTier(m.Model, in.Prices) != tierPremium {
			continue
		}
		premiumTurns += m.Turns
		smallPremium += m.SmallTurns
	}
	if premiumTurns == 0 {
		r.noData("premium-model turns", "No premium-model turns in this window.")
		return r
	}
	r.restsOn(int(premiumTurns), "premium-model turns")
	enough := premiumTurns >= rightSizeMinTurns
	smallShare := fracOf(smallPremium, premiumTurns)

	if enough {
		r.Read = reportedRead
	} else {
		r.Read = noDataRead
	}
	r.Purity = neutralPurity
	r.Figures = []Figure{
		{Label: "premium turns", Value: humanize.Int(premiumTurns)},
		{Label: "small-output premium", Value: humanize.PercentOrDash(smallPremium, premiumTurns, 0), Note: "<" + strconv.Itoa(RightSizeSmallOutput) + " output tokens"},
		{Label: "downgrade candidates", Value: humanize.Int(smallPremium), Note: "worth a cheaper/faster model"},
	}
	r.Takeaway = rightSizeTakeaway(enough, smallShare, smallPremium)
	r.Caveats = []string{
		"Task difficulty is invisible -- a short answer can still need the strong model. A prompt to review a few, not a verdict.",
		"On a flat subscription this is about speed and rate limits, not dollar savings.",
		"\"Small output\" is assaio's own definition -- under " + strconv.Itoa(RightSizeSmallOutput) + " output tokens -- not a vendor's, and it counts what a turn produced rather than what it was asked to do.",
	}
	if enough {
		r.Caveats = append(r.Caveats, unsourcedLine("a small-output share", ownHistoryWouldSettleIt))
	}
	return r
}

func rightSizeTakeaway(enough bool, smallShare float64, smallPremium int64) string {
	if !enough {
		return "Too few premium-model turns this window to read a small-output share."
	}
	return humanize.Percent(smallShare) + " of premium-model turns produced under " +
		strconv.Itoa(RightSizeSmallOutput) + " output tokens -- " + humanize.Int(smallPremium) +
		" turns worth opening to see whether a cheaper or faster model would have done. A short answer can still need the strong model, so this names candidates and grades nothing."
}
