package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/report"
)

// budget holds the optional ceilings check gates on; zero means no budget for that axis.
type budget struct {
	MaxTokens int64
	MaxCost   float64
}

func (b budget) empty() bool { return b.MaxTokens <= 0 && b.MaxCost <= 0 }

// checkTotals rolls the window's usage up to the two axes a budget gates on.
type checkTotals struct {
	Tokens int64
	Cost   float64
	// UnpricedRows and UnpricedTokens are what the cost axis cannot see: usage on a model
	// the price table has no row for. Cost excludes it entirely, so the pair is what stops
	// the gate reporting OK on a window it could not price. They are two facts, not one: a
	// row can be unpriced and carry no token at all.
	UnpricedRows   int
	UnpricedTokens int64
}

// HasUnpriced reports whether any row carried no known price, which is what the "*" marker
// beside a cost means.
func (t checkTotals) HasUnpriced() bool { return t.UnpricedRows > 0 }

// sumCheckTotals adds every priced row's cost and every row's tokens into the window
// totals; HasUnpriced carries the usual "cost excludes unpriced usage". Reasoning tokens
// are a subset of output (usage.Record) and are never added again -- the budget this gates
// on must be the same total report and effectiveness show.
func sumCheckTotals(rows []report.Row) checkTotals {
	var t checkTotals
	for i := range rows {
		r := &rows[i]
		tokens := r.In + r.Out + r.CacheRead + r.CacheWrite
		t.Tokens += tokens
		if r.Priced {
			t.Cost += *r.Cost
		} else {
			t.UnpricedTokens += tokens
		}
		if r.HasUnpriced {
			t.UnpricedRows++
		}
	}
	return t
}

// unpriced is the window's unpriced condition in the shape every surface discloses it in,
// so the gate's footnote is the same sentence report and the dashboard print.
func (t checkTotals) unpriced() *report.Unpriced {
	return &report.Unpriced{Tokens: t.UnpricedTokens, Total: t.Tokens, Rows: t.UnpricedRows}
}

// evaluateBudget returns one breach message per failed axis; empty means within budget.
//
// A cost budget over a window carrying unpriced usage fails rather than passes, for the
// reason the rule gate already states: a gate that could not be evaluated is not a gate that
// passed. Cost excludes a model the price table has no row for, so the comparison would be
// made against a figure missing that model's whole spend -- on the maintainer's own store,
// 45% of the window's tokens before the v0.13 price refresh. The token axis is unaffected:
// tokens are physical, and an unpriced model says nothing about them.
//
// The trigger is unpriced *tokens*, not the HasUnpriced flag beside them: Claude writes
// locally-generated turns as model "<synthetic>" with an all-zero usage block, which is an
// unpriced row carrying no spend to be missing, and gating on the flag failed every run on a
// real store the refresh had left fully priced.
func evaluateBudget(t checkTotals, b budget) []string {
	var breaches []string
	if b.MaxTokens > 0 && t.Tokens > b.MaxTokens {
		breaches = append(breaches, fmt.Sprintf("tokens %d > %d", t.Tokens, b.MaxTokens))
	}
	switch {
	case b.MaxCost <= 0:
	case t.UnpricedTokens > 0:
		breaches = append(breaches, fmt.Sprintf(
			"API-equivalent cost budget cannot be evaluated: %d of %d tokens are on models the price table has no row for, and cost excludes them",
			t.UnpricedTokens, t.Tokens,
		))
	case t.Cost > b.MaxCost:
		breaches = append(breaches, fmt.Sprintf("API-equivalent cost $%.2f > $%.2f", t.Cost, b.MaxCost))
	}
	return breaches
}

// costVerdict is the word the cost line prints, drawn from the same three cases
// evaluateBudget decides on so the rendered verdict and the exit code cannot disagree.
func costVerdict(t checkTotals, b budget) string {
	switch {
	case t.UnpricedTokens > 0:
		return "UNPRICED"
	case t.Cost > b.MaxCost:
		return "EXCEEDED"
	default:
		return "OK"
	}
}

func renderCheck(cmd *cobra.Command, since string, t checkTotals, b budget, p config.Pricing) error {
	lw := &lineWriter{w: cmd.OutOrStdout()}
	lw.printf("budget check · last %s\n", since)
	lw.printf("  total tokens: %d\n", t.Tokens)
	cost := fmt.Sprintf("$%.2f", t.Cost)
	if t.HasUnpriced() {
		cost += "*"
	}
	lw.printf("  total cost:   %s (API-equivalent estimate)\n", cost)
	if note := report.UnpricedDisclosure(t.unpriced(), "this window's tokens"); note != "" {
		lw.printf("  %s\n", note)
	}
	if line, ok := effectiveBasisLine(p, t.Tokens); ok {
		lw.printf("  %s\n", line)
	}
	lw.println("")
	writeBudgetAxis(lw, "token", b.MaxTokens > 0, tokenVerdict(t, b), fmt.Sprintf("%d / %d", t.Tokens, b.MaxTokens))
	writeBudgetAxis(lw, "cost", b.MaxCost > 0, costVerdict(t, b), fmt.Sprintf("$%.2f / $%.2f (API-equivalent)", t.Cost, b.MaxCost))
	if b.empty() {
		lw.println("  no budget set -- pass --max-tokens or --max-cost to gate.")
	}
	lw.println("")
	return lw.err
}

// writeBudgetAxis prints one budget line, or nothing when that axis has no budget set.
func writeBudgetAxis(lw *lineWriter, label string, set bool, verdict, detail string) {
	if !set {
		return
	}
	lw.printf("  %s budget: %s  %s\n", label, detail, verdict)
}

// tokenVerdict is the token axis's word; tokens are physical, so pricing never enters it.
func tokenVerdict(t checkTotals, b budget) string {
	if t.Tokens > b.MaxTokens {
		return "EXCEEDED"
	}
	return "OK"
}

// effectiveBasisLine renders the user's own cost basis when they configured one -- the
// honest counter to the API-equivalent estimate shown above it.
func effectiveBasisLine(p config.Pricing, tokens int64) (string, bool) {
	if !p.Configured() {
		return "", false
	}
	if p.EffectivePerToken > 0 {
		cost, _ := p.EffectiveWindowCost(tokens)
		return fmt.Sprintf("your basis: effective ~$%.2f this window at $%g/token (not the API figure above)", cost, p.EffectivePerToken), true
	}
	if p.MonthlySubscriptionCost > 0 {
		return fmt.Sprintf("your basis: ~$%.2f/mo flat subscription -- the $ above is an API-equivalent estimate, not this bill", p.MonthlySubscriptionCost), true
	}
	return "your basis: flat subscription -- the $ above is an API-equivalent estimate, not your spend", true
}
