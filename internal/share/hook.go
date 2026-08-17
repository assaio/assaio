package share

import (
	"fmt"
	"math"

	"github.com/assaio/assaio/internal/humanize"
)

// rule is one candidate first frame. threshold is published rather than tuned in private:
// it is the number a reader can hold the line to. A rule fires only when its own figure
// exists and clears that threshold, and the window picks the rule whose figure clears it
// by the widest margin -- so the opening line is chosen by the data, never by whoever is
// posting.
type rule struct {
	family    string
	threshold float64
	at        func(f *facts) num
	line      func(f *facts) string
	sub       func(f *facts) string
}

// pick resolves the window's first frame. It returns a universal hook whenever no rule
// clears its threshold, which is what makes an empty first frame unreachable -- including
// on the first run, where almost nothing has accumulated yet.
func pick(f *facts, sample bool) Hook {
	if sample {
		return Hook{
			Family: "universal", Universal: true,
			Line: "Sample data. This is what yours would look like.",
			// Not "no logs were found": --demo is a deliberate flag, so this card is just as
			// likely to have been asked for on a machine full of real usage.
			Sub: "Bundled sample usage, not this machine's.",
		}
	}
	best, bestScore := -1, 0.0
	for i := range rules {
		got := rules[i].at(f)
		if !got.ok || got.v < rules[i].threshold {
			continue
		}
		if s := clearance(got.v, rules[i].threshold); s > bestScore {
			best, bestScore = i, s
		}
	}
	if best >= 0 {
		r := &rules[best]
		return Hook{Family: r.family, Line: r.line(f), Sub: r.sub(f)}
	}
	if f.historyDay > 0 && f.historyDay < freshWindow.Hours()/24 {
		return Hook{
			Family: "universal", Universal: true,
			Line: fmt.Sprintf("Day one with assaio: %s sessions, %s tokens.", humanize.Int(f.sessions), humanize.Count(f.tokens)),
			Sub:  "Come back in a month and this line will say something else.",
		}
	}
	return Hook{
		Family: "universal", Universal: true,
		Line: "Fame and shame: my last window of AI.",
		Sub:  scaleSub(f),
	}
}

// clearance scores how far a figure cleared its own threshold, on a scale the families can
// share. A raw ratio cannot: a token count clears 1e9 by 42x while a percentage is capped by
// 100, so every count beat every share and the self-critical family was unreachable -- the
// fame/shame balance the post promises would have been an artifact of unit mismatch rather
// than of the data. The log keeps the ordering inside a family and makes the families
// comparable across it.
func clearance(v, threshold float64) float64 {
	if threshold <= 0 || v < threshold {
		return 0
	}
	return math.Log1p(v/threshold - 1)
}

func scaleSub(f *facts) string {
	return fmt.Sprintf("%s sessions · %s tokens · %s lines", humanize.Int(f.sessions), humanize.Count(f.tokens), humanize.Int(f.lines))
}

// outputSub is scaleSub for a hook that already said the token count, so the support line
// adds what the headline did not rather than repeating it.
func outputSub(f *facts) string {
	return fmt.Sprintf("%s sessions · %s lines the AI wrote", humanize.Int(f.sessions), humanize.Int(f.lines))
}

// moneySub deliberately does not put the premium share next to a dollar figure. "$4.19 per
// 100 lines · 92.6% on premium models" reads the second number as a share of spend; it is a
// share of tokens, and the adjacency alone was enough to make the swap.
func moneySub(f *facts) string {
	if f.perHundred == "" {
		return scaleSub(f)
	}
	return fmt.Sprintf("%s per 100 lines · %s sessions", f.perHundred, humanize.Int(f.sessions))
}

func costOf(f *facts) float64 {
	if f.cost == nil {
		return 0
	}
	return *f.cost
}

func costNum(f *facts) num { return num{v: costOf(f), ok: f.cost != nil} }

// planMultiple scores the plan hook by the plan's own price, so a configured plan always
// clears its threshold of 1 and competes on the strength of the rest rather than winning
// automatically on an unbounded multiple.
func planMultiple(f *facts) float64 {
	if f.planCost <= 0 {
		return 0
	}
	return 1
}
