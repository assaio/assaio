package reprice

import (
	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/report"
)

// The conditions every figure in a Window is true under. They are computed from the window
// rather than written into the renderer so that a thin one carries only the conditions it
// actually has, and so no surface can publish the arithmetic without them.

// assumptions are what the re-pricing holds fixed. Each names a place a re-priced total can be
// wrong in a direction nobody can size, which is why they travel with the figures instead of
// living in documentation a reader of the output never opens.
// planMonthlyCost is the configured flat subscription price, zero when nobody configured one.
// It is an assumption rather than a figure here: it changes what every delta in this window
// means without changing any of them.
func assumptions(b *Basis, planMonthlyCost float64) []string {
	out := []string{
		// The basis itself is an estimate, and it is the same estimate every other cost surface
		// discloses -- quoted from the one canonical wording so a re-priced dollar and a reported
		// one cannot read as two different kinds of dollar. It is an assumption rather than a
		// renderer footer so the JSON document carries it too.
		report.CostEstimateDisclosure,
		"Token counts are held exactly as observed. A different model does not emit the same tokens for the " +
			"same request, so a re-priced total is this window's tokens at another rate -- not a bill anyone would have received.",
	}
	if planMonthlyCost > 0 {
		// The reader has told this binary they are on a flat plan. Every delta above is then
		// arithmetic about plan value, not about a bill that would shrink, and saying so is not
		// optional once assaio knows: a "-$29.5K" under a column headed `vs observed` is money to
		// anyone who is not told otherwise. This lived as a recommendation prerequisite, which
		// only fires when three other conditions hold; the condition it states holds regardless.
		out = append(out, "A flat subscription of $"+humanize.USD(planMonthlyCost)+
			"/mo is configured, so no delta here is spend that would stop: on a flat plan this is plan-value "+
			"arithmetic, and what a cheaper model buys is speed and rate-limit headroom rather than a smaller bill.")
	}
	if b.CacheReadShare > 0 {
		out = append(out, "The observed cache mix carries over unchanged: "+humanize.Percent(b.CacheReadShare)+
			" of these tokens were cache reads. A model that has never held this context must write that cache "+
			"before it can read it, so its first window costs more than this arithmetic shows.")
	}
	if b.Unpriced.Missing() {
		// One decimal, the precision report's own disclosure states this share at, so the same
		// condition does not read as two different sizes on two surfaces.
		out = append(out, humanize.PercentAt(b.Unpriced.Share, 1)+" of the window's tokens ("+
			humanize.Count(b.Unpriced.Tokens)+" of "+humanize.Count(b.Unpriced.Total)+
			") carry no known price and are excluded from both sides. Every total here is a floor, and the spread "+
			"is the spread of the priced part only.")
	}
	if !b.Trustworthy() {
		out = append(out, "That unpriced share is above the "+humanize.Percent(trustworthyCeiling)+
			" `doctor --strict` fails at, which is enough of the window to change the answer. Price those models before acting on anything here.")
	}
	return out
}

// refusals are the claims this arithmetic does not make. They are rendered, not documented,
// because each one is a sentence a reader would otherwise supply for themselves.
func refusals(hasPlans bool) []string {
	out := []string{
		"No claim about another model's output. This compares price for the same observed tokens, never quality " +
			"for the same work: assaio has never seen the target model do this work and holds no evidence about what it would produce.",
		"No counterfactual. Nothing here says what you would have saved -- only what this same observed set of " +
			"turns costs against another table, which is arithmetic rather than a prediction.",
	}
	if hasPlans {
		out = append(out, "No claim about entitlements. A plan's rate limits and quotas are invisible to assaio: "+
			"it reads tokens, not whether a cheaper plan would have let you run them.")
	}
	return out
}
