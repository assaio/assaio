package report

import "github.com/assaio/assaio/internal/humanize"

// CostEstimateDisclosure is the one-line honesty note attached to every rendered cost.
// assaio prices usage from a public API rate table (the vendored LiteLLM snapshot), which
// is not what a subscription user actually pays -- flat-rate plans (Claude Pro/Max,
// ChatGPT Plus/Pro) make the effective cost per token entirely different. One canonical
// wording, referenced by every cost surface: the CLI cost tables, the analyze litmus, and
// the HTML dashboard colophon, so the basis reads identically everywhere.
const CostEstimateDisclosure = "Cost is an estimate at public pay-as-you-go API prices -- not your actual spend; subscription plans bill a flat rate and differ."

// UnpricedDisclosure is the legend for the "*" marker every cost surface appends to a row
// whose total excludes usage on a model with no known price -- one wording, so a starred
// cost reads the same in report, effectiveness, movers and the dashboard colophon. It states
// the share, because a marker that reads the same at 0.1% and at 45% tells the reader only
// that something is missing, never whether it is worth acting on.
// scope names the set the share was computed over. It is not decoration: a legend printed
// under figures drawn from a different set reads as their error bar, and understating it
// there is the exact failure the quantified marker exists to end.
func UnpricedDisclosure(u *Unpriced, scope string) string {
	switch {
	case u.Missing():
		return "* cost excludes " + humanize.PercentAt(u.Share(), 1) + " of " + scope + " (" +
			humanize.Count(u.Tokens) + " of " + humanize.Count(u.Total) +
			" tokens) on models the price table does not carry -- a refreshed table ships with each release"
	case u.Rows > 0:
		return "* marks rows on a model with no known price; they carry no tokens, so the total above is complete"
	default:
		return ""
	}
}

// The granularity footnotes name the unit behind a row, so a session aggregate can never
// be read as a per-turn figure. Sources differ here -- most emit one record per request,
// some only a per-session total -- and a mixed row is the case worth calling out loudest,
// because its numbers answer neither question cleanly.
const (
	sessionGranularityFootnote = "‡ session-granularity rows: one record covers a whole session, not one turn"
	mixedGranularityFootnote   = "‡ mixed-granularity total: blends per-turn and whole-session records"
)
