package analyze

// ModelStat is one model's usage across the queried window: token/line totals, its
// already-classified tier, and its priced cost. BuildInput computes ByModel once, so a
// validator reads this directly instead of grouping Usage by model and calling modelTier
// itself. Sorted by Tokens descending.
type ModelStat struct {
	// Model is the model name as recorded by the tool (store.UsageRow.Model).
	Model string
	// Tier is "premium", "cheaper", or "unknown" -- see modelTier. A model priced but
	// below premiumOutputPriceFloor is "cheaper"; a model absent from Prices is
	// "unknown", never guessed from its name.
	Tier string
	// Tokens is In+Output+CacheRead+CacheWrite+Reasoning summed across this model's usage.
	Tokens int64
	// Input, Output, CacheRead, CacheWrite are the same usage summed per token type.
	Input, Output, CacheRead, CacheWrite int64
	// Lines is AI-added code lines summed across this model's usage.
	Lines int64
	// Cost is USD cost priced from Prices; nil when Priced is false.
	Cost *float64
	// Priced is false when Model has no known price in Prices -- Cost is then unknown,
	// not a real zero.
	Priced bool
	// TokenShare is this model's share of every ModelStat's Tokens in ByModel, 0..1.
	TokenShare float64
}

// ProjectStat is one project's AI-line output and cost across the queried window.
// BuildInput computes ByProject once, so a validator reads this directly instead of
// grouping Usage by project itself. Sorted by Lines descending.
type ProjectStat struct {
	// Project is the project name (store.UsageRow.Project); "" for unattributed usage.
	Project string
	// Lines is AI-added code lines summed across this project's usage.
	Lines int64
	// Cost is USD cost priced from Prices, summed from this project's priced usage only;
	// nil when none of its usage priced.
	Cost *float64
	// Priced is false when at least one contributing usage row's model has no known
	// price in Prices -- Cost then undercounts this project's real spend.
	Priced bool
	// TokenShare is this project's share of Totals.Tokens, 0..1.
	TokenShare float64
}

// Totals is the queried window's grand totals across every model and project, computed
// once by BuildInput so no validator re-sums Usage itself.
type Totals struct {
	// Tokens is In+Output+CacheRead+CacheWrite+Reasoning summed across all Usage.
	Tokens int64
	// Input, Output, CacheRead, CacheWrite are the same usage summed per token type.
	Input, Output, CacheRead, CacheWrite int64
	// Lines is AI-added code lines summed across all Usage.
	Lines int64
	// Cost is USD cost priced from Prices, summed from priced usage only; nil when
	// nothing priced.
	Cost *float64
	// Priced is false when at least one usage row's model has no known price in Prices
	// -- Cost then undercounts real spend.
	Priced bool
	// CacheEfficiency is CacheRead / (CacheRead + Input), 0 when that sum is zero.
	CacheEfficiency float64
}
