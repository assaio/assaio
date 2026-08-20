package analyze

import (
	"fmt"
	"strconv"

	"github.com/assaio/assaio/internal/layer"

	"github.com/assaio/assaio/internal/humanize"
)

const (
	modelFitName     = "model-fit"
	modelFitTitle    = "Model Fit"
	modelFitDescribe = "Premium vs. cheaper model token share, lines-per-token contrast, and real sub-agent delegation share."
	// modelFitHowToRead is Result.HowToRead for this validator -- see its doc comment.
	modelFitHowToRead = "High premium-model share isn't wrong, but routine edits and boilerplate are often just as good on cheaper models -- a place to trim spend without losing output."
	// modelFitUnknownWatchCeiling is the unpriced-token-share threshold above which the
	// premium/cheaper split can no longer be read with confidence -- most of the window
	// is invisible to pricing, so a favorable read would be unearned.
	modelFitUnknownWatchCeiling = 0.5
)

func init() { Register(modelFitValidator{}) }

// modelFitValidator reads whether spend concentrates on premium-priced models or spreads
// to cheaper ones (see modelTier), and contrasts AI-line output per token between tiers.
type modelFitValidator struct{}

func (modelFitValidator) Name() string       { return modelFitName }
func (modelFitValidator) Title() string      { return modelFitTitle }
func (modelFitValidator) Describe() string   { return modelFitDescribe }
func (modelFitValidator) Layer() layer.Layer { return layer.Activity } // the premium-model token share; the lines-per-token note is context, not the claim

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (modelFitValidator) Analyze(in Input) Result {
	r := Result{Name: modelFitName, Title: modelFitTitle, Describe: modelFitDescribe, HowToRead: modelFitHowToRead}
	if len(in.Usage) == 0 {
		r.noData("active days", "No usage in this window.")
		return r
	}
	r.restsOn(activeDays(&in), "active days")
	premiumTokens, cheaperTokens, otherTokens, premiumLines, cheaperLines := modelTierTotals(in.ByModel)

	total := premiumTokens + cheaperTokens + otherTokens
	known := premiumTokens + cheaperTokens
	unpriceable := fracOf(otherTokens, total) > modelFitUnknownWatchCeiling

	var premiumShare float64
	if known > 0 {
		premiumShare = float64(premiumTokens) / float64(known)
	}

	r.Read = modelFitRead(unpriceable, known > 0)
	r.Purity = neutralPurity
	r.Figures = []Figure{
		{Label: premiumTierLabel(), Value: humanize.PercentOrDash(premiumTokens, total, 1), Note: linesPerMTok(premiumLines, premiumTokens) + " lines/1M tok"},
		{Label: cheaperTierLabel(), Value: humanize.PercentOrDash(cheaperTokens, total, 1), Note: linesPerMTok(cheaperLines, cheaperTokens) + " lines/1M tok"},
	}
	if otherTokens > 0 {
		r.Figures = append(r.Figures, Figure{
			Label: "unpriced (unknown model)", Value: humanize.PercentOrDash(otherTokens, total, 1),
			Note: "excluded from the premium/cheaper split above",
		})
	}
	r.Figures = append(r.Figures, Figure{
		Label: "sub-agent delegation", Value: humanize.PercentOrDash(in.Delegation.Sub, in.Delegation.Total, 1), Note: "of tokens run inside Task sub-agents",
	})
	r.Bars = modelBars(in.ByModel)
	if unpriceable {
		r.Caveats = []string{"Most tokens this window ran on a model absent from the price table -- the premium/cheaper split above is not a confident read."}
	} else if s, ok := computeModelSavings(in.ByModel, in.Prices, &in); ok {
		r.Figures = append(r.Figures, savingsFigure(s))
		r.Caveats = append(r.Caveats, savingsCaveat(s))
	}
	if !unpriceable && known > 0 {
		r.Caveats = append(r.Caveats, unsourcedLine("a premium-token share", ownHistoryWouldSettleIt))
	}
	r.Takeaway = modelFitTakeaway(unpriceable, known > 0, premiumShare)
	return r
}

// modelFitRead keeps the one judgement this metric can defend and drops the one it cannot.
// Too much of the window on models the price table does not know is a statement about the
// evidence, so it stays a withheld read; the premium/cheaper mix itself is reported and not
// graded, since the 80% ceiling that used to flag it was a number picked once (B177) and
// running everything on the strong model is a choice, not a defect.
func modelFitRead(unpriceable, known bool) Read {
	if unpriceable || !known {
		return noDataRead
	}
	return reportedRead
}

func modelFitTakeaway(unpriceable, known bool, premiumShare float64) string {
	switch {
	case unpriceable:
		return "Most spend this window is on a model with no known price -- add it to the price table before trusting this read."
	case !known:
		return "No token in this window ran on a model the price table places in a tier, so there is no premium/cheaper split to read."
	default:
		return humanize.Percent(premiumShare) + " of the tokens that carry a tier ran on a premium model. " +
			"Routine edits and boilerplate are often just as good on cheaper models, which is where a mix like this gets trimmed -- but running everything on the strong model is a choice, and nothing published says at what share it becomes the wrong one."
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
		bars[i] = Bar{Label: groupLabel(m.Model), Value: humanize.Count(m.Tokens) + " tokens", Frac: fracOf(m.Tokens, maxTokens)}
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
