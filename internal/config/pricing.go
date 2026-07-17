package config

import "fmt"

// pricingModeAPI prices usage at the vendored public API rate table (the default).
// pricingModeSubscription tells assaio the user pays a flat subscription instead, so the
// API-derived cost is only an equivalent estimate, not their actual spend.
const (
	pricingModeAPI          = "api"
	pricingModeSubscription = "subscription"
)

// Pricing lets a user whose real cost is not the public API rate -- a Claude Pro/Max or
// ChatGPT Plus/Pro subscriber, or a negotiated-rate customer -- tell assaio their actual
// basis. Unset (mode "" / "api", zero rates) keeps the default: every cost is a public
// pay-as-you-go API estimate, disclosed as such. This never changes measured token counts,
// only how their monetary equivalent is framed.
type Pricing struct {
	// Mode is "" / "api" (default) or "subscription". Subscription marks the API cost as an
	// equivalent estimate and surfaces the configured rate / monthly cost as the truer figure.
	Mode string `koanf:"mode"`
	// EffectivePerToken is the user's real blended $/token (input+output combined), e.g. a
	// subscription's monthly cost divided by tokens used. Zero (default) means unset.
	EffectivePerToken float64 `koanf:"effective_per_token"`
	// MonthlySubscriptionCost is the flat monthly plan price in dollars, for comparison
	// against API-equivalent spend. Zero (default) means unset.
	MonthlySubscriptionCost float64 `koanf:"monthly_subscription_cost"`
}

// Validate checks the pricing mode and that configured rates are non-negative.
func (p Pricing) Validate() error {
	switch p.Mode {
	case "", pricingModeAPI, pricingModeSubscription:
	default:
		return fmt.Errorf("invalid pricing.mode %q (want api|subscription)", p.Mode)
	}
	if p.EffectivePerToken < 0 {
		return fmt.Errorf("pricing.effective_per_token must be >= 0, got %g", p.EffectivePerToken)
	}
	if p.MonthlySubscriptionCost < 0 {
		return fmt.Errorf("pricing.monthly_subscription_cost must be >= 0, got %g", p.MonthlySubscriptionCost)
	}
	return nil
}

// IsSubscription reports whether the user opted into subscription-basis framing.
func (p Pricing) IsSubscription() bool {
	return p.Mode == pricingModeSubscription
}

// Configured reports whether the user supplied any pricing basis at all -- a mode other
// than the API default, or a non-zero effective rate / monthly cost.
func (p Pricing) Configured() bool {
	return p.IsSubscription() || p.EffectivePerToken > 0 || p.MonthlySubscriptionCost > 0
}

// EffectiveWindowCost returns the user's configured effective cost for a token total and
// whether an effective rate is set. It never consults the API table: this is the user's own
// blended rate, the honest counter to the API estimate.
func (p Pricing) EffectiveWindowCost(tokens int64) (cost float64, ok bool) {
	if p.EffectivePerToken <= 0 {
		return 0, false
	}
	return float64(tokens) * p.EffectivePerToken, true
}
