// Package reprice answers the question about AI spend a vendor will never answer for you:
// which plan, and which model mix, should you be on given what you actually ran.
//
// It performs exactly one operation. It takes turns already in the store and prices them
// against a different entry in the same table. Re-pricing an observed event is arithmetic; a
// counterfactual is a prediction, and this package makes none -- nothing here says what you
// would have saved, because a different model does not emit the same tokens for the same
// request and assaio has never watched one do this work. What it can say is what this same
// observed set of turns costs against another price, under assumptions it states and inside
// the margin its own unpriced share leaves.
package reprice

import (
	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/layer"
)

// Window is one window re-priced: the observed basis, the same premium turns against other
// models, and the projected monthly rate beside each flat plan price the caller supplied.
//
// Assumptions and Refusals are fields rather than renderer prose because they are part of the
// result. A surface that printed the figures without them would be publishing an arbitrage
// this package cannot defend: the conditions are what make the arithmetic true.
type Window struct {
	// Layer is what these figures claim. Cost sits at the activity layer -- it is what the
	// window was billed at, not evidence that anything was produced (ADR 0013) -- and a
	// re-priced cost cannot climb a layer its input never reached.
	Layer layer.Layer `json:"layer"`
	Basis Basis       `json:"basis"`
	// Routes are the same premium turns against another model's table entry, cheapest first.
	Routes []Route `json:"routes"`
	// Unpriceable are the models the caller named that the vendored table carries no rate for.
	// They are reported rather than dropped: a --against the table cannot cost leaves a table
	// that looks like a complete answer to the question the caller asked, and a reader who is
	// not told their target was skipped reads the remaining rows as the ranking they wanted.
	Unpriceable []string `json:"unpriceable"`
	// Plans are flat monthly prices beside Basis.Monthly.
	Plans []Plan `json:"plans"`
	// Assumptions are what the arithmetic holds fixed; Refusals are the claims it does not
	// make. Both are computed from the window, so a thin one carries fewer of each rather
	// than the same boilerplate.
	Assumptions []string `json:"assumptions"`
	Refusals    []string `json:"refusals"`
}

// Options are what the caller supplies that the store cannot: candidate plan prices, which
// assaio deliberately does not vendor, and extra models to re-price onto.
type Options struct {
	// Plans are candidate flat prices, already parsed by ParsePlan.
	Plans []Plan
	// Against are model names to re-price the premium slice onto in addition to the ones this
	// window ran, so the comparison can reach a model the user has never tried -- including a
	// competitor's.
	Against []string
}

// Compute re-prices in against opts. It reads the prepared per-model view analyze already
// built, so a re-pricing and the metric verdicts over the same window cannot disagree about
// which model is the expensive one or what it cost.
func Compute(in *analyze.Input, opts Options) Window {
	b := basis(in)
	candidates := plans(&b, in.PlanMonthlyCost, opts.Plans)
	found, unpriceable := routes(&b, in, opts.Against)
	return Window{
		Layer:       layer.Activity,
		Basis:       b,
		Routes:      found,
		Unpriceable: unpriceable,
		Plans:       candidates,
		Assumptions: assumptions(&b, in.PlanMonthlyCost),
		Refusals:    refusals(len(candidates) > 0),
	}
}

// share is a/b, 0 when b is zero. Every caller guards on the denominator before rendering
// what this returns; the branch exists so a division by zero cannot reach a printed figure
// as NaN.
func share(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}
