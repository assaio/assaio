package analyze

import "github.com/assaio/assaio/internal/pricing"

const (
	tierPremium = "premium"
	tierCheaper = "cheaper"
	tierUnknown = "unknown"
)

// premiumOutputPriceFloor is the output $/token at or above which a model classifies
// "premium". Checked against internal/pricing/litellm.json: every real Claude
// opus/fable-tier model prices at $25/1M output tokens or higher, every sonnet/haiku-tier
// model at $18.75/1M or lower -- $20/1M sits cleanly between the two clusters.
const premiumOutputPriceFloor = 20e-6 // $20 per 1M output tokens

// modelTier classifies model as "premium", "cheaper", or "unknown" from its ACTUAL
// output price in prices, never from its name. This is the only place assaio decides
// model tier: a newly released model auto-classifies the moment its price lands in the
// table, with no hand-maintained name list to keep in sync. A model absent from prices
// is "unknown" -- excluded from both shares honestly, never guessed from its name.
func modelTier(model string, prices pricing.Table) string {
	p, ok := prices[model]
	if !ok {
		p, ok = prices[pricing.NormalizeModel(model)]
	}
	if !ok {
		return tierUnknown
	}
	if p.Output >= premiumOutputPriceFloor {
		return tierPremium
	}
	return tierCheaper
}
