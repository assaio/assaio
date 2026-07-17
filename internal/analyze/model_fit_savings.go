package analyze

import (
	"strconv"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// modelSavings is the upper-bound counterfactual: this window's premium-tier tokens cost at
// API rates vs. the same tokens repriced on the cheapest cheaper-tier model actually in use.
// It is deliberately an upper bound -- it assumes every premium token could move with no
// quality loss, which is rarely true -- so it is only ever a directional prompt to review,
// never a switch recommendation.
type modelSavings struct {
	TargetModel  string
	MonthlyUpper float64
}

// computeModelSavings returns the estimate and whether one could be computed at all. It
// requires priced premium usage, at least one cheaper-tier model in use to reprice onto, a
// positive difference, and at least one active day to project a monthly figure.
func computeModelSavings(models []ModelStat, prices pricing.Table, activeDays int) (modelSavings, bool) {
	var pIn, pOut, pCacheRead, pCacheWrite int64
	var premiumCost float64
	var premiumPriced bool
	for i := range models {
		m := &models[i]
		if m.Tier != tierPremium {
			continue
		}
		pIn += m.Input
		pOut += m.Output
		pCacheRead += m.CacheRead
		pCacheWrite += m.CacheWrite
		if m.Priced && m.Cost != nil {
			premiumCost += *m.Cost
			premiumPriced = true
		}
	}
	if !premiumPriced || activeDays <= 0 {
		return modelSavings{}, false
	}
	target, counterfactual, ok := cheapestCheaperCost(models, prices, pIn, pOut, pCacheWrite, pCacheRead)
	if !ok {
		return modelSavings{}, false
	}
	windowSaving := premiumCost - counterfactual
	if windowSaving <= 0 {
		return modelSavings{}, false
	}
	return modelSavings{TargetModel: target, MonthlyUpper: windowSaving / float64(activeDays) * 30}, true
}

// cheapestCheaperCost reprices the premium token bundle onto every cheaper-tier model in use
// and returns the cheapest option's model and cost.
func cheapestCheaperCost(models []ModelStat, prices pricing.Table, in, out, cacheWrite, cacheRead int64) (model string, cost float64, ok bool) {
	for i := range models {
		m := &models[i]
		if m.Tier != tierCheaper {
			continue
		}
		c, priced := prices.CostTokens(m.Model, in, out, cacheWrite, cacheRead)
		if !priced {
			continue
		}
		if !ok || c < cost {
			model, cost, ok = m.Model, c, true
		}
	}
	return model, cost, ok
}

// distinctDays counts calendar days with any usage, the denominator for projecting a
// per-window figure to a monthly one at the observed active-day pace.
func distinctDays(rows []store.UsageRow) int {
	seen := make(map[string]struct{}, len(rows))
	for i := range rows {
		seen[rows[i].Day] = struct{}{}
	}
	return len(seen)
}

// savingsFigure renders the upper-bound saving with the target model in its note.
func savingsFigure(s modelSavings) Figure {
	return Figure{
		Label: "est. savings (upper bound)",
		Value: "~$" + strconv.FormatFloat(s.MonthlyUpper, 'f', 0, 64) + "/mo",
		Note:  "premium-tier tokens repriced on " + s.TargetModel,
	}
}

// savingsCaveat is the honesty framing every rendered saving carries.
func savingsCaveat(s modelSavings) string {
	return "Est. savings is an upper bound -- it assumes every premium-tier token could move to " +
		s.TargetModel + " with no quality loss (rarely true) and projects your active-day pace to 30 days; " +
		"directional, a prompt to review, not a recommendation to switch."
}
