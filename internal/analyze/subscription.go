package analyze

import (
	"strconv"

	"github.com/assaio/assaio/internal/store"
)

const (
	subscriptionName      = "subscription-fit"
	subscriptionTitle     = "Subscription Fit"
	subscriptionDescribe  = "Whether a flat monthly plan (Claude Max/Pro, ChatGPT Plus/Pro) pays off vs API pay-as-you-go, from your configured plan cost."
	subscriptionHowToRead = "This projects your window's API-equivalent cost to a month at your active-day pace, then compares it against the flat plan price you configured. A high multiple means the plan is a bargain at your volume; below 1x means API pay-as-you-go might be cheaper. The API figure is an estimate at public prices, not your actual bill."
	// monthDays is the active-day count the per-day API-equivalent rate is projected onto.
	monthDays = 30
)

// planUnsetRead is the neutral faceplate shown when usage exists but no plan cost is
// configured -- the validator still appears, prompting the user to set one.
var planUnsetRead = Read{Key: "neutral", Label: "SET PLAN"}

func init() { Register(subscriptionValidator{}) }

// subscriptionValidator answers "is my flat subscription worth it?" by comparing the
// window's API-equivalent estimate (projected to a month) against the configured plan cost.
// It is the honest resolution to the estimate-vs-bill gap: a subscription user's
// API-equivalent $ is meaningless as a spend figure but very meaningful as plan value.
type subscriptionValidator struct{}

func (subscriptionValidator) Name() string     { return subscriptionName }
func (subscriptionValidator) Title() string    { return subscriptionTitle }
func (subscriptionValidator) Describe() string { return subscriptionDescribe }

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (subscriptionValidator) Analyze(in Input) Result {
	r := Result{Name: subscriptionName, Title: subscriptionTitle, Describe: subscriptionDescribe, HowToRead: subscriptionHowToRead}
	if in.Totals.Tokens == 0 {
		r.Read = noDataRead
		r.Takeaway = "No usage in this window."
		return r
	}

	apiMonthly, priced := projectedMonthlyCost(&in)
	if in.PlanMonthlyCost <= 0 {
		r.Read = planUnsetRead
		if priced {
			r.Figures = []Figure{{Label: "API-equivalent", Value: "~$" + money(apiMonthly) + "/mo", Note: "projected estimate"}}
		}
		r.Takeaway = "Set config.pricing.monthly_subscription_cost to see whether your flat plan beats API pricing at this volume."
		r.Caveats = []string{"The API-equivalent $ is an estimate at public pay-as-you-go prices, not your actual bill."}
		return r
	}
	if !priced {
		r.Read = noDataRead
		r.Takeaway = "No priced usage this window to compare against the plan."
		return r
	}

	multiple := apiMonthly / in.PlanMonthlyCost
	payingOff := multiple >= 1
	r.Read = readFor(payingOff, "Paying off")
	r.Purity = clamp01(multiple / 2)
	r.Figures = []Figure{
		{Label: "plan cost", Value: "$" + money(in.PlanMonthlyCost) + "/mo", Note: "configured flat rate"},
		{Label: "API-equivalent", Value: "~$" + money(apiMonthly) + "/mo", Note: "projected estimate"},
		{Label: "value multiple", Value: valueMultiple(multiple), Note: "API-equiv / plan"},
		{Label: "vs API", Value: signedMoney(apiMonthly-in.PlanMonthlyCost) + "/mo", Note: savingsNote(payingOff)},
	}
	r.Takeaway = subscriptionTakeaway(payingOff, multiple)
	r.Caveats = []string{
		"The API-equivalent $ is an estimate at public pay-as-you-go prices, not your actual bill.",
		"Projected to a month from this window's usage, and it excludes any unpriced-model usage.",
	}
	return r
}

// projectedMonthlyCost scales the window's API-equivalent cost to a month at the observed
// per-active-day rate, so a sparse or partial window isn't compared against a full month's
// plan price at face value (which would under-report a heavy-but-recent user). priced is
// false when nothing in the window had a known price. Same active-day-pace convention as
// model-fit savings.
func projectedMonthlyCost(in *Input) (cost float64, priced bool) {
	if in.Totals.Cost == nil {
		return 0, false
	}
	days := distinctActiveDays(in.Usage)
	if days <= 0 {
		return *in.Totals.Cost, true
	}
	return *in.Totals.Cost / float64(days) * monthDays, true
}

// distinctActiveDays counts the distinct UTC calendar days ("YYYY-MM-DD") the window's usage
// spans -- the denominator for the per-active-day run-rate projection.
func distinctActiveDays(rows []store.UsageRow) int {
	seen := make(map[string]struct{})
	for i := range rows {
		if rows[i].Day != "" {
			seen[rows[i].Day] = struct{}{}
		}
	}
	return len(seen)
}

func subscriptionTakeaway(payingOff bool, multiple float64) string {
	switch {
	case multiple >= 3:
		return "Your flat plan returns ~" + valueMultiple(multiple) + " its price in API-equivalent usage -- it is paying off handily."
	case payingOff:
		return "Your flat plan is worth ~" + valueMultiple(multiple) + " its price in API-equivalent usage -- it pays off."
	default:
		return "Your API-equivalent usage is below the plan price this window -- at this volume, API pay-as-you-go could be cheaper."
	}
}

// money renders a dollar amount compactly: 195 -> "195", 26004 -> "26.0K".
func money(v float64) string {
	switch {
	case v >= 10_000:
		return strconv.FormatFloat(v/1000, 'f', 1, 64) + "K"
	case v >= 100:
		return strconv.FormatFloat(v, 'f', 0, 64)
	default:
		return strconv.FormatFloat(v, 'f', 2, 64)
	}
}

func signedMoney(v float64) string {
	if v >= 0 {
		return "+$" + money(v)
	}
	return "-$" + money(-v)
}

// valueMultiple renders the API-equiv/plan ratio: "134x", or "2.3x" below ten.
func valueMultiple(m float64) string {
	if m >= 10 {
		return strconv.FormatFloat(m, 'f', 0, 64) + "x"
	}
	return strconv.FormatFloat(m, 'f', 1, 64) + "x"
}

func savingsNote(payingOff bool) string {
	if payingOff {
		return "saved vs API"
	}
	return "over API cost"
}
