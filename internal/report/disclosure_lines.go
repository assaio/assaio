package report

import (
	"strings"

	"github.com/assaio/assaio/internal/humanize"
)

// The sentences a figure travels with. They live apart from the renderers because more than one
// surface prints the same figure, and a disclosure written twice is two answers to one question.

// statusCaveat states that the dashboard's efficiency signal is directional and scoped to
// projects, never a per-person performance metric -- the deliberate difference from a named team
// leaderboard.
const statusCaveat = "Efficiency is directional and shown per project only -- never a per-person metric."

// lineRateNote states what a $/100-lines figure covers, or why there is none, under the "†"
// ratioCell puts on a ratio that leaves usage out. place names the set its shares are of ("this
// window", "this table"): a share printed under rows drawn from a narrower set reads as their
// own. `status` and `effectiveness` both print it, so one ratio never carries two answers.
func lineRateNote(b *LineRateBasis, place string) string {
	note := lineRateDisclosure(b, place)
	if note != "" && b.Partial() {
		return "† " + note
	}
	return note
}

// lineRateDisclosure keeps "no source records lines" apart from "the line-recording usage has no
// price" and from "every line came from usage the ratio cannot pair", and quantifies what a ratio
// leaves out in tokens -- a line-blind source running an unpriced model has no cost share
// to state, and a sentence triggered on cost would print nothing over it.
func lineRateDisclosure(b *LineRateBasis, place string) string {
	switch {
	case b.LineRows == 0 && b.LineBlindRows == 0:
		return ""
	case b.LineRows == 0:
		return "No source in " + place + " records changed lines, so the AI-line count and $/100 lines are withheld -- absent, not zero."
	case b.Partial():
		return lineRateLeftOut(b, place)
	case b.Per100() != nil:
		return ""
	case b.Tokens == 0 && b.SharedTokens > 0 && b.UnpricedLineTokens > 0:
		return "All line-recording usage in " + place + " is either " + SharedUsage + " or on models with no known price, so $/100 lines is withheld."
	case b.Tokens == 0 && b.SharedTokens > 0:
		return "All line-recording usage in " + place + " is " + SharedUsage + ", so $/100 lines is withheld."
	case b.Tokens == 0 && b.UnpricedLineTokens > 0:
		return "The sources that record changed lines in " + place + " ran only models with no known price, so $/100 lines is withheld rather than priced from other sources' cost."
	case b.Lines == 0 && b.SharedLines > 0 && b.UnpricedLines > 0:
		return "Every AI line in " + place + " came from " + SharedUsage + " or from usage with no known price, so $/100 lines is withheld rather than set against the cost of usage that recorded none."
	case b.Lines == 0 && b.SharedLines > 0:
		return "Every AI line in " + place + " came from " + SharedUsage + ", so $/100 lines is withheld rather than set against the cost of usage that recorded none."
	case b.Lines == 0 && b.UnpricedLines > 0:
		return "Every AI line in " + place + " came from usage with no known price, so $/100 lines is withheld rather than set against the cost of usage that recorded none."
	}
	return ""
}

// SharedUsage names LineRateBasis.SharedTokens by what the credit group knows: that several
// models ran on that day and project, not that one session ran them.
const SharedUsage = "usage from a source that counts lines once per session, on a day and project where several models ran"

// lineRateLeftOut quantifies a partial ratio's exclusions: line-blind usage as a token share and,
// when priced, a cost share; line-recording usage the ratio cannot pair as a token share and in
// lines, because the AI-line count printed beside the ratio includes them.
func lineRateLeftOut(b *LineRateBasis, place string) string {
	total := float64(b.Tokens + b.LineBlindTokens + b.UnpricedLineTokens + b.SharedTokens)
	lines := humanize.Int(b.Lines + b.UnpricedLines + b.SharedLines)
	var parts []string
	if b.LineBlindTokens > 0 {
		part := humanize.Percent(float64(b.LineBlindTokens)/total) + " from sources that record no lines"
		if b.LineBlindCost > 0 {
			part += " (" + humanize.Percent(b.LineBlindCost/(b.Cost+b.LineBlindCost)) + " of the priced cost)"
		}
		parts = append(parts, part)
	}
	if b.UnpricedLineTokens > 0 {
		parts = append(parts, humanize.Percent(float64(b.UnpricedLineTokens)/total)+
			" on line-recording models with no known price ("+humanize.Int(b.UnpricedLines)+" of "+lines+" AI lines)")
	}
	if b.SharedTokens > 0 {
		parts = append(parts, humanize.Percent(float64(b.SharedTokens)/total)+
			" in "+SharedUsage+" ("+humanize.Int(b.SharedLines)+" of "+lines+" AI lines)")
	}
	return "$/100 lines divides only priced usage from sources that record changed lines. Of the tokens in " +
		place + ", it leaves out " + strings.Join(parts, " and ") + "."
}
