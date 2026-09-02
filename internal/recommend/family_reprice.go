package recommend

import (
	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/reprice"
)

// FamilyCheaperRoute is the name `reprice` addresses this family by when it renders the
// experiment its own arithmetic supports.
const FamilyCheaperRoute = "cheaper-route"

// repriceMinSpread is how much of the window's priced cost a route has to move before the
// experiment is worth a fortnight of somebody's attention. It is a materiality floor, not a
// price verdict: this family says "here is a spread large enough to be worth measuring", never
// "this window is too expensive" -- nothing published sets that second line, and a
// recommendation may not invent one. The number is assaio's own and the record says so.
const repriceMinSpread = 0.20

func init() { Register(cheaperRoute{}) }

// cheaperRoute proposes running one slice of premium work on a cheaper model, on the
// arithmetic that the same observed turns priced against that model's table entry cost
// materially less. The arithmetic is about price, not about quality: assaio has never seen the
// target model do this work, which is why the action is an experiment with a follow-up named
// before the result is known rather than a switch.
type cheaperRoute struct{}

func (cheaperRoute) Name() string { return FamilyCheaperRoute }

func (cheaperRoute) Propose(e *Evidence) []Record {
	if e.Input == nil {
		return nil
	}
	w := reprice.Compute(e.Input, reprice.Options{})
	// A re-priced total drawn over a window whose own cost basis has a hole is advice built on
	// a figure assaio itself discloses as incomplete -- pricing-coverage is the recommendation
	// that window earns, and it is ordered ahead of this one for exactly that reason.
	if !w.Basis.Priced || !w.Basis.Trustworthy() || len(w.Routes) == 0 {
		return nil
	}
	best := &w.Routes[0]
	if best.Share < repriceMinSpread {
		return nil
	}
	return []Record{{
		ID:    "cheaper-route/window",
		Title: "Try one slice of premium work on " + best.Target + ", then read what it cost and what it produced",
		Evidence: []string{
			"observed: " + humanize.USDCompact(w.Basis.Cost) + " priced this window, " +
				humanize.USDCompact(w.Basis.Premium.Cost) + " of it (" +
				humanize.Percent(w.Basis.PremiumShare()) + ") on premium-tier models",
			"re-priced: the same premium turns against " + best.Target + "'s table entry cost " +
				humanize.USDCompact(best.Premium) + ", leaving the window at " + humanize.USDCompact(best.Window) +
				" -- a spread of " + humanize.USDCompact(best.Delta) + " (" + humanize.Percent(best.Share) + ")",
			"that is a re-pricing of tokens held exactly as observed, and says nothing about what " +
				best.Target + " would have produced for the same requests",
		},
		// The condition is arithmetic over the whole window rather than a sampled verdict, so it
		// is as confident as the window's pricing is complete -- which the gate above is what
		// establishes.
		Confidence: analyze.ConfidenceHigh,
		Scope:      "premium-tier turns in this window",
		Action: "Route one kind of work you can name -- the one you would least defend on the strong model -- to " +
			best.Target + " for a fortnight, and label those sessions with `assaio-agent mark` so the comparison is per kind of work rather than an average.",
		Prerequisites: []string{
			"A kind of work you can name and route separately. The spread above is arithmetic over the whole premium slice; moving all of it is not this experiment.",
			"Usage billed per token. On a flat subscription the spread is plan-value arithmetic and not money you would save -- what a cheaper model buys there is speed and rate-limit headroom.",
			"A target that can hold this context: the re-pricing keeps the observed cache mix, and a model starting cold has to write that cache before it can read it.",
		},
		Effect: "Directionally lower spend on the slice you move, at unknown cost to quality. assaio has no counterfactual " +
			"and predicts no number; what it can do is tell you afterwards what moved.",
		Risks: []string{
			"The spread is price for the same tokens, not price for the same result. A cheaper model may need more turns to reach the same place, which costs more rather than less.",
			"The " + humanize.Percent(repriceMinSpread) + " floor this fired at is assaio's own, not a published one. It is the size at which a fortnight of measuring is worth it, never a verdict that this window is too expensive.",
			"Judging the result on cost alone hides a quality regression. Read rework and survival on the same slice.",
		},
		Rollback:     "Route the work back to the premium model; nothing stored changes and no figure is lost.",
		ReviewWindow: "14d",
		FollowUp: "`assaio-agent reprice --since 14d` on the labelled slice, beside `assaio-agent effectiveness --by task --compare`: " +
			"what the moved work actually cost, next to rework on the same sessions",
		Status: StatusProposed,
	}}
}
