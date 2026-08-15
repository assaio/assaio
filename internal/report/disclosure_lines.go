package report

import "github.com/assaio/assaio/internal/humanize"

// The sentences a figure travels with. They live apart from the renderers because more than one
// surface prints the same figure and a disclosure written twice is two answers to one question --
// the reason `$/100 lines` was disclosed on `effectiveness` and bare on `status`.

// statusCaveat states that the dashboard's efficiency signal is directional and scoped to
// projects, never a per-person performance metric -- the deliberate difference from a named team
// leaderboard.
const statusCaveat = "Efficiency is directional and shown per project only -- never a per-person metric."

// LineCoverageDisclosure states how far a line figure reaches on this window, and what that
// makes of a ratio built on one. `effectiveness` discloses it and the dashboard colophon
// discloses it; `status` printed the same two figures with neither disclosure.
func LineCoverageDisclosure(inv *Inventory) string {
	switch {
	case inv.Days == 0:
		// An empty window says nothing about what any source records. Two different absences
		// given one answer is the confusion ADR 0011 exists to separate.
		return ""
	case inv.LineCapableRows == 0:
		return "No source in this window records changed lines, so the AI-line count and $/100 lines are withheld -- absent, not zero."
	case inv.LineCapableTokens < inv.TotalTokens:
		return "AI lines come only from the sources that record them (" +
			humanize.Percent(float64(inv.LineCapableTokens)/float64(inv.TotalTokens)) +
			" of this window's tokens), while the cost in $/100 lines is the whole window's."
	}
	return ""
}
