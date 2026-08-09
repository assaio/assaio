package analyze

import (
	"strconv"

	"github.com/assaio/assaio/internal/humanize"
)

const (
	subscriptionName      = "subscription-fit"
	subscriptionTitle     = "Subscription Fit"
	subscriptionDescribe  = "Whether a flat monthly plan (Claude Max/Pro, ChatGPT Plus/Pro) pays off vs API pay-as-you-go, from your configured plan cost."
	subscriptionHowToRead = "This projects your window's API-equivalent cost onto a calendar month -- the window's span in real days, not only the days you worked -- and compares it against the flat plan price you configured. A high multiple means the plan is a bargain at your volume; below 1x means API pay-as-you-go might be cheaper. The API figure is an estimate at public prices, not your actual bill."
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

// WindowScoped: the plan price covers the whole window, not one project's share of it.
func (subscriptionValidator) WindowScoped() {}

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (subscriptionValidator) Analyze(in Input) Result {
	r := Result{Name: subscriptionName, Title: subscriptionTitle, Describe: subscriptionDescribe, HowToRead: subscriptionHowToRead}
	if in.Totals.Tokens == 0 {
		r.noData("active days", "No usage in this window.")
		return r
	}

	r.restsOn(activeDays(&in), "active days")
	apiMonthly, priced := projectedMonthlyCost(&in)
	if in.PlanMonthlyCost <= 0 {
		r.Read = planUnsetRead
		if priced {
			r.Figures = []Figure{{Label: "API-equivalent", Value: "~$" + humanize.USD(apiMonthly) + "/mo", Note: "projected estimate"}}
		}
		r.Takeaway = "Set config.pricing.monthly_subscription_cost to see whether your flat plan beats API pricing at this volume."
		r.Caveats = []string{"The API-equivalent $ is an estimate at public pay-as-you-go prices, not your actual bill."}
		return r
	}
	if !priced {
		// Nothing in the window carries a price, so the comparison the verdict is about has
		// no side to stand on -- a declared zero reach, not a thin one.
		r.covering(0)
		r.Read = noDataRead
		r.Takeaway = "No priced usage this window to compare against the plan."
		return r
	}

	multiple := apiMonthly / in.PlanMonthlyCost
	payingOff := multiple >= 1
	r.Read = readFor(payingOff, "Paying off")
	r.Purity = clamp01(multiple / 2)
	r.Figures = []Figure{
		{Label: "plan cost", Value: "$" + humanize.USD(in.PlanMonthlyCost) + "/mo", Note: "configured flat rate"},
		{Label: "API-equivalent", Value: "~$" + humanize.USD(apiMonthly) + "/mo", Note: "projected estimate"},
		{Label: "value multiple", Value: valueMultiple(multiple), Note: "API-equiv / plan"},
		{Label: "vs API", Value: signedMoney(apiMonthly-in.PlanMonthlyCost) + "/mo", Note: savingsNote(payingOff)},
	}
	r.Takeaway = subscriptionTakeaway(payingOff, multiple)
	r.Caveats = []string{
		"The API-equivalent $ is an estimate at public pay-as-you-go prices, not your actual bill.",
		"Projected onto 30 calendar days from this window's span -- the days inside it you did not work are still days the plan was paid for -- and it excludes any unpriced-model usage.",
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
	return MonthlyRate(*in.Totals.Cost, in), true
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

func signedMoney(v float64) string {
	if v >= 0 {
		return "+$" + humanize.USD(v)
	}
	return "-$" + humanize.USD(-v)
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
