package report

// CostEstimateDisclosure is the one-line honesty note attached to every rendered cost.
// assaio prices usage from a public API rate table (the vendored LiteLLM snapshot), which
// is not what a subscription user actually pays -- flat-rate plans (Claude Pro/Max,
// ChatGPT Plus/Pro) make the effective cost per token entirely different. One canonical
// wording, referenced by every cost surface: the CLI cost tables, the analyze litmus, and
// the HTML dashboard colophon, so the basis reads identically everywhere.
const CostEstimateDisclosure = "Cost is an estimate at public pay-as-you-go API prices -- not your actual spend; subscription plans bill a flat rate and differ."

// unpricedFootnote is the one-line legend for the "*" marker every cost surface appends to
// a row whose total excludes some usage priced from an unknown model -- one wording, so a
// starred cost reads the same in report, effectiveness, and movers.
const unpricedFootnote = "* group contains unpriced usage excluded from cost"

// The granularity footnotes name the unit behind a row, so a session aggregate can never
// be read as a per-turn figure. Sources differ here -- most emit one record per request,
// some only a per-session total -- and a mixed row is the case worth calling out loudest,
// because its numbers answer neither question cleanly.
const (
	sessionGranularityFootnote = "‡ session-granularity rows: one record covers a whole session, not one turn"
	mixedGranularityFootnote   = "‡ mixed-granularity total: blends per-turn and whole-session records"
)
