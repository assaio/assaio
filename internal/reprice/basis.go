package reprice

import (
	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/report"
)

// tierPremium is the value analyze.ModelStat.Tier carries for a model priced at or above the
// premium output floor. The classification itself is deliberately not repeated here: this
// package reads the tier BuildInput already computed, so the binary keeps one authority on
// which model is the expensive one and a re-pricing can never disagree with model-fit about
// where the split falls.
const tierPremium = "premium"

// trustworthyCeiling is the unpriced token share above which a re-priced total is a figure to
// disclose but not to act on: below it the gap is a rounding error on a comparison, above it
// the comparison is missing a slice nobody can size. It is the same number `doctor --strict`
// fails on, so one condition is not held to two ceilings.
const trustworthyCeiling = config.DefaultMaxUnpricedShare

// Basis is the window as observed, priced by the table this binary ships. It is the control
// figure every re-priced total is a delta against: a re-priced number with nothing to compare
// it to is not a comparison.
type Basis struct {
	// Days is the span a monthly projection divides by -- the same span subscription-fit
	// states, read from the same helper, so the two surfaces project at one pace. Zero means
	// the window holds no usage: the projection helper answers a full month where there is
	// nothing to project, and printing that as evidence would state a span nothing rests on.
	Days float64 `json:"days"`
	// Cost is the observed priced cost of the window. Priced is false when nothing in it had
	// a known price at all, where a rendered zero would read as free rather than as unknown.
	Cost   float64 `json:"cost"`
	Priced bool    `json:"priced"`
	// Monthly is Cost projected onto a calendar month at the pace of Days.
	Monthly float64 `json:"monthly"`
	// Premium is the slice a route moves: the priced models the table places in the premium
	// tier. Everything outside it stays exactly as observed.
	Premium Slice `json:"premium"`
	// CacheReadShare is the observed cache-read share of billable tokens. It is an assumption
	// rather than a figure: re-pricing carries it over unchanged, and a target model that has
	// never held this context would not have it.
	CacheReadShare float64 `json:"cacheReadShare"`
	// Unpriced is the margin on everything above.
	Unpriced Unpriced `json:"unpriced"`
	// premium is the same slice as a token bundle, kept for re-pricing rather than rendering.
	premium pricing.Tokens
}

// Trustworthy reports whether the unpriced share is small enough for a re-priced total to be
// acted on rather than only disclosed.
func (b *Basis) Trustworthy() bool { return b.Unpriced.Share < trustworthyCeiling }

// PremiumShare is how much of the observed priced cost the slice a route moves accounts for.
// One definition serves the rendered figure and the recommendation that quotes it, so the two
// can never state different sizes for the same slice.
func (b *Basis) PremiumShare() float64 { return share(b.Premium.Cost, b.Cost) }

// Slice is a set of models and the observed tokens and cost they carry.
type Slice struct {
	Models []string `json:"models"`
	Tokens int64    `json:"tokens"`
	Cost   float64  `json:"cost"`
}

// Unpriced is the margin on every figure in a Window: tokens the price table could not cost,
// of the window's total, and the models a refresh would have to cover. It restates
// report.BuildUnpriced's counts rather than embedding that type, because this document is
// published as JSON and the shared type carries no field names for it.
type Unpriced struct {
	Tokens int64    `json:"tokens"`
	Total  int64    `json:"total"`
	Share  float64  `json:"share"`
	Models []string `json:"models,omitempty"`
}

// Missing reports whether a cost here is understated -- unpriced usage carrying tokens.
func (u *Unpriced) Missing() bool { return u.Tokens > 0 }

func basis(in *analyze.Input) Basis {
	b := Basis{Premium: Slice{Models: []string{}}}
	if len(in.Usage) > 0 {
		b.Days = analyze.ProjectionSpan(in)
	}
	if in.Totals.Cost != nil {
		b.Cost, b.Priced = *in.Totals.Cost, true
		b.Monthly = analyze.MonthlyRate(b.Cost, in)
	}
	b.CacheReadShare = share(float64(in.Totals.CacheRead), float64(in.Totals.Tokens))
	for i := range in.ByModel {
		m := &in.ByModel[i]
		// An unpriced premium model cannot be in the slice: there is no rate to subtract when
		// the route moves it, and including its tokens would price them twice.
		if m.Tier != tierPremium || !m.Priced {
			continue
		}
		b.Premium.Models = append(b.Premium.Models, m.Model)
		b.Premium.Tokens += m.Tokens
		b.premium.In += m.Input
		b.premium.Out += m.Output
		b.premium.CacheRead += m.CacheRead
		b.premium.CacheWrite += m.CacheWrite
		b.premium.CacheWrite1h += m.CacheWrite1h
		if m.Cost != nil {
			b.Premium.Cost += *m.Cost
		}
	}
	u := report.BuildUnpriced(report.Build(in.Usage, in.Prices))
	b.Unpriced = Unpriced{
		Tokens: u.Tokens, Total: u.Total, Share: u.Share(),
		Models: report.UnpricedModels(in.Usage, in.Prices),
	}
	return b
}
