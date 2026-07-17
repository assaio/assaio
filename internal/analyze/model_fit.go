package analyze

import (
	"fmt"
	"strconv"
)

const (
	modelFitName     = "model-fit"
	modelFitTitle    = "Model Fit"
	modelFitDescribe = "Premium vs. cheaper model token share, lines-per-token contrast, and real sub-agent delegation share."
	// modelFitHowToRead is Result.HowToRead for this validator -- see its doc comment.
	modelFitHowToRead = "High premium-model share isn't wrong, but routine edits and boilerplate are often just as good on cheaper models -- a place to trim spend without losing output."
	// modelFitWatchCeiling is the premium-token-share threshold above which the model
	// mix is flagged for a closer look.
	modelFitWatchCeiling = 0.8
	// modelFitUnknownWatchCeiling is the unpriced-token-share threshold above which the
	// premium/cheaper split can no longer be read with confidence -- most of the window
	// is invisible to pricing, so a favorable read would be unearned.
	modelFitUnknownWatchCeiling = 0.5
)

func init() { Register(modelFitValidator{}) }

// modelFitValidator reads whether spend concentrates on premium-priced models or spreads
// to cheaper ones (see modelTier), and contrasts AI-line output per token between tiers.
type modelFitValidator struct{}

func (modelFitValidator) Name() string     { return modelFitName }
func (modelFitValidator) Title() string    { return modelFitTitle }
func (modelFitValidator) Describe() string { return modelFitDescribe }

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (modelFitValidator) Analyze(in Input) Result {
	r := Result{Name: modelFitName, Title: modelFitTitle, Describe: modelFitDescribe, HowToRead: modelFitHowToRead}
	if len(in.Usage) == 0 {
		r.Read = noDataRead
		r.Takeaway = "No usage in this window."
		return r
	}
	premiumTokens, cheaperTokens, otherTokens, premiumLines, cheaperLines := modelTierTotals(in.ByModel)

	total := premiumTokens + cheaperTokens + otherTokens
	known := premiumTokens + cheaperTokens
	unpriceable := fracOf(otherTokens, total) > modelFitUnknownWatchCeiling

	var premiumShare float64
	if known > 0 {
		premiumShare = float64(premiumTokens) / float64(known)
	}
	watch := unpriceable || premiumShare > modelFitWatchCeiling

	r.Read = readFor(!watch, "Healthy")
	r.Purity = modelFitPurity(premiumShare, known > 0)
	r.Figures = []Figure{
		{Label: premiumTierLabel(), Value: shareOrDash(premiumTokens, total, 1), Note: linesPerMTok(premiumLines, premiumTokens) + " lines/1M tok"},
		{Label: cheaperTierLabel(), Value: shareOrDash(cheaperTokens, total, 1), Note: linesPerMTok(cheaperLines, cheaperTokens) + " lines/1M tok"},
	}
	if otherTokens > 0 {
		r.Figures = append(r.Figures, Figure{
			Label: "unpriced (unknown model)", Value: shareOrDash(otherTokens, total, 1),
			Note: "excluded from the premium/cheaper split above",
		})
	}
	r.Figures = append(r.Figures, Figure{
		Label: "sub-agent delegation", Value: shareOrDash(in.Delegation.Sub, in.Delegation.Total, 1), Note: "of tokens run inside Task sub-agents",
	})
	r.Bars = modelBars(in.ByModel)
	if unpriceable {
		r.Caveats = []string{"Most tokens this window ran on a model absent from the price table -- the premium/cheaper split above is not a confident read."}
	} else if s, ok := computeModelSavings(in.ByModel, in.Prices, distinctDays(in.Usage)); ok {
		r.Figures = append(r.Figures, savingsFigure(s))
		r.Caveats = append(r.Caveats, savingsCaveat(s))
	}
	r.Takeaway = modelFitTakeaway(watch, unpriceable)
	return r
}

// modelFitPurity scores the premium/cheaper mix 0..1, honestly: known is false when
// every token this window is on an unpriced model, so there is no tier signal at all to
// score -- neutral 0.5, never a confident 1 computed from an empty split.
func modelFitPurity(premiumShare float64, known bool) float64 {
	if !known {
		return 0.5
	}
	return clamp01(1 - premiumShare)
}

func modelFitTakeaway(watch, unpriceable bool) string {
	switch {
	case unpriceable:
		return "Most spend this window is on a model with no known price -- add it to the price table before trusting this read."
	case watch:
		return "Nearly all tokens run on premium models -- consider delegating routine work to cheaper models or sub-agents."
	default:
		return "Model mix looks balanced between premium and cheaper models."
	}
}

// modelTierTotals sums Tokens/Lines per ModelStat.Tier across models -- isolated from
// Result-building so the tier accounting itself is directly unit-testable.
func modelTierTotals(models []ModelStat) (premiumTokens, cheaperTokens, otherTokens, premiumLines, cheaperLines int64) {
	for i := range models {
		m := &models[i]
		switch m.Tier {
		case tierPremium:
			premiumTokens += m.Tokens
			premiumLines += m.Lines
		case tierCheaper:
			cheaperTokens += m.Tokens
			cheaperLines += m.Lines
		default:
			otherTokens += m.Tokens
		}
	}
	return premiumTokens, cheaperTokens, otherTokens, premiumLines, cheaperLines
}

// premiumTierLabel and cheaperTierLabel document the live price threshold in the label
// itself, so the text stays accurate if premiumOutputPriceFloor ever changes.
func premiumTierLabel() string {
	return fmt.Sprintf("premium (>=$%.0f/1M out)", premiumOutputPriceFloor*1e6)
}

func cheaperTierLabel() string {
	return fmt.Sprintf("cheaper (<$%.0f/1M out)", premiumOutputPriceFloor*1e6)
}

// modelBars ranks models by token usage descending, for the dashboard's model-split
// visualization. No re-sort needed here: BuildInput already sorts ByModel by Tokens desc.
func modelBars(models []ModelStat) []Bar {
	var maxTokens int64
	if len(models) > 0 {
		maxTokens = models[0].Tokens
	}
	bars := make([]Bar, len(models))
	for i, m := range models {
		bars[i] = Bar{Label: groupLabel(m.Model), Value: strconv.FormatInt(m.Tokens, 10) + " tokens", Frac: fracOf(m.Tokens, maxTokens)}
	}
	return bars
}

// linesPerMTok renders AI lines per 1M tokens, "—" when tokens is zero.
func linesPerMTok(lines, tokens int64) string {
	if tokens == 0 {
		return "—"
	}
	return strconv.FormatFloat(float64(lines)*1_000_000/float64(tokens), 'f', 1, 64)
}
