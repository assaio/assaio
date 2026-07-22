package analyze

import "strconv"

const (
	rightSizeName      = "model-right-sizing"
	rightSizeTitle     = "Model Right-Sizing"
	rightSizeDescribe  = "Turns on a premium model that produced very little output -- small tasks a cheaper, faster model might handle just as well."
	rightSizeHowToRead = "A premium-model turn that emits only a handful of output tokens is often boilerplate or a quick answer a cheaper model handles as well. On a flat plan this is about speed and rate limits, not dollars. Task difficulty is invisible, so this is a prompt to look at a few, never a verdict."
	// RightSizeSmallOutput is the output-token ceiling below which a premium turn is a
	// downgrade candidate; exported so the CLI passes it to store.TurnSizing.
	RightSizeSmallOutput = 300
	// rightSizeWatchShare is the small-premium-turn share above which the read flags over-powering.
	rightSizeWatchShare = 0.4
	// rightSizeMinTurns is the premium-turn floor below which the share is too thin to report.
	rightSizeMinTurns = 20
)

func init() { Register(rightSizeValidator{}) }

// rightSizeValidator flags premium-model turns that produced little output -- small tasks a
// cheaper or faster model might have handled -- as candidates to review, not a verdict.
type rightSizeValidator struct{}

func (rightSizeValidator) Name() string     { return rightSizeName }
func (rightSizeValidator) Title() string    { return rightSizeTitle }
func (rightSizeValidator) Describe() string { return rightSizeDescribe }

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
		r.Read = noDataRead
		r.Takeaway = "No premium-model turns in this window."
		return r
	}
	enough := premiumTurns >= rightSizeMinTurns
	smallShare := fracOf(smallPremium, premiumTurns)

	if enough {
		r.Read = readFor(smallShare < rightSizeWatchShare, "Right-sized")
	} else {
		r.Read = noDataRead
	}
	r.Purity = clamp01(1 - smallShare)
	r.Figures = []Figure{
		{Label: "premium turns", Value: strconv.FormatInt(premiumTurns, 10)},
		{Label: "small-output premium", Value: shareOrDash(smallPremium, premiumTurns, 0), Note: "<" + strconv.Itoa(RightSizeSmallOutput) + " output tokens"},
		{Label: "downgrade candidates", Value: strconv.FormatInt(smallPremium, 10), Note: "worth a cheaper/faster model"},
	}
	r.Takeaway = rightSizeTakeaway(enough, smallShare)
	r.Caveats = []string{
		"Task difficulty is invisible -- a short answer can still need the strong model. A prompt to review a few, not a verdict.",
		"On a flat subscription this is about speed and rate limits, not dollar savings.",
	}
	return r
}

func rightSizeTakeaway(enough bool, smallShare float64) string {
	switch {
	case !enough:
		return "Too few premium-model turns this window to judge right-sizing."
	case smallShare >= rightSizeWatchShare:
		return "A large share of premium-model turns produced little output -- worth trying a cheaper or faster model on routine work."
	default:
		return "Most premium-model turns produced substantial output -- the strong model is earning its use."
	}
}
