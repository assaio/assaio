package recommend

import (
	"strconv"
	"strings"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/report"
)

// pricingMinShare is the unpriced share below which naming the models is noise rather than
// advice: a rounding error on the cost figure, not a hole in it. It matches the default
// `doctor --strict` gate so a reader is not asked to hold two numbers for the same condition.
const pricingMinShare = 0.05

func init() { Register(pricingCoverage{}) }

// pricingCoverage proposes the one action that makes every cost figure in this store more
// correct: teach the price table the models the window actually ran on. It is first among the
// families on purpose -- an engine that recommends changes to how someone works, while its own
// cost basis is missing a fifth of the window, has the order backwards.
type pricingCoverage struct{}

func (pricingCoverage) Name() string { return "pricing-coverage" }

func (pricingCoverage) Propose(e *Evidence) []Record {
	if e.Input == nil {
		return nil
	}
	rows := report.Build(e.Input.Usage, e.Input.Prices)
	unpriced := report.BuildUnpriced(rows)
	if !unpriced.Missing() || unpriced.Share() < pricingMinShare {
		return nil
	}
	models := report.UnpricedModels(e.Input.Usage, e.Input.Prices)
	if len(models) == 0 {
		return nil
	}
	named := models
	if len(named) > 5 {
		named = named[:5]
	}
	return []Record{{
		ID:    "pricing-coverage/window",
		Title: "Price the models this window ran on, before trusting any cost figure it produced",
		Evidence: []string{
			humanize.Percent(unpriced.Share()) + " of the window's billable tokens carry no known price (" +
				humanize.Count(unpriced.Tokens) + " of " + humanize.Count(unpriced.Total) + ")",
			modelLine(models, named),
		},
		// The condition is arithmetic over the whole window, not a sampled verdict, so it is
		// as confident as the store is complete.
		Confidence: analyze.ConfidenceHigh,
		Scope:      "this window's cost basis",
		Action: "Add a price for " + named[0] + " (and the others listed) to the vendored table, or set " +
			"`pricing.effective_per_token` if you are on a negotiated rate. `assaio-agent doctor` lists every model a refresh must cover.",
		Prerequisites: []string{"The published per-token price for each model, from the vendor's own pricing page"},
		// The share above is a share of *tokens*. What it costs is unknown by construction --
		// the models are unpriced, so their rate is exactly what nobody here knows -- and
		// quoting the token share as the size of the cost error would be the predicted number
		// Record.Effect forbids.
		Effect: "Cost, `$`/100 lines and subscription-fit stop understating. The direction is certain; the size is not, " +
			"and cannot be until the prices are in: the share above is a share of tokens, and an unpriced model can bill " +
			"well above or well below this window's average rate.",
		Risks: []string{
			"A wrong price is worse than no price: it turns a disclosed gap into a confident wrong number. Copy the figure, do not estimate it.",
		},
		Rollback:     "Remove the added price entry; the window returns to reporting the gap rather than a cost.",
		ReviewWindow: "immediately -- rerun `assaio-agent doctor` after the change",
		FollowUp:     "the unpriced share on the same window, which should reach 0%",
		Status:       StatusProposed,
	}}
}

// modelLine names the heaviest unpriced models, and says how many it did not name. A truncated
// list that reads as complete is the shape of an evidence line a reader acts on and finds short.
func modelLine(all, named []string) string {
	line := "heaviest unpriced model(s): " + strings.Join(named, ", ")
	if len(all) > len(named) {
		line += " (and " + strconv.Itoa(len(all)-len(named)) + " more)"
	}
	return line
}
