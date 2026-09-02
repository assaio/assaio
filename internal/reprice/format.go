package reprice

import (
	"strconv"
	"strings"

	"github.com/assaio/assaio/internal/humanize"
)

// The value formatters this package's figures are rendered through. They sit apart from the
// block layout because each carries a rule of its own about what a number may not say.

// namesShown is how many models a field lists before it says how many it left out. A
// truncated list that reads as complete is the shape a reader acts on and finds short.
const namesShown = 4

// names lists models and says how many it did not name. It truncates only when the tail is
// worth a phrase: "(and 1 more)" is longer than the single name it would stand in for.
func names(all []string) string {
	if len(all) <= namesShown+1 {
		return strings.Join(all, ", ")
	}
	return strings.Join(all[:namesShown], ", ") + " (and " + strconv.Itoa(len(all)-namesShown) + " more)"
}

// windowSpan renders the span the figures rest on, and refuses to name one where the window
// holds no usage: the projection helper answers a full calendar month there, and printing that
// as evidence would state a span behind figures that have none.
func windowSpan(b *Basis) string {
	if b.Days <= 0 {
		return "—"
	}
	return humanize.Days(b.Days)
}

// signedMoney renders a change in cost with its direction. humanize scales from the magnitude
// only, so the sign is applied here rather than passed through: a negative amount would
// otherwise skip every unit tier and print in full dollars beside its scaled neighbours.
func signedMoney(v float64) string {
	if v < 0 {
		return "-" + humanize.USDCompact(-v)
	}
	return "+" + humanize.USDCompact(v)
}

// signedPercent renders a signed share, applying humanize's small-share guard to the
// magnitude. Without it a real but small spread renders "-0.0%", which reads as no change
// rather than a change too small to matter.
func signedPercent(v float64) string {
	if v < 0 {
		return "-" + humanize.PercentAt(-v, 1)
	}
	return "+" + humanize.PercentAt(v, 1)
}

// multipleNote states what a plan multiple means in the direction it points. A multiple is a
// ratio of an estimate to a real price and is never a saving: the plan's own bill is the only
// figure in the comparison that is neither an estimate nor a projection.
func multipleNote(m *float64) string {
	switch {
	case m == nil:
		return "no priced usage this window to compare against the plan"
	case *m < 1:
		return "the plan costs more than this window's API-equivalent at this volume"
	default:
		return multiple(*m) + " the plan price in API-equivalent usage"
	}
}

// multiple renders a ratio the way subscription-fit does, so the same window read on two
// surfaces states one number: whole above ten, one decimal below it.
func multiple(m float64) string {
	if m >= 10 {
		return strconv.FormatFloat(m, 'f', 0, 64) + "x"
	}
	return strconv.FormatFloat(m, 'f', 1, 64) + "x"
}
