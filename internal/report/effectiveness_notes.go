package report

import "github.com/assaio/assaio/internal/humanize"

// effCaveat states that efficiency is a diagnostic signal, never a performance metric. It
// names the dimension the table is actually grouped by: printed as "per project" over a table
// grouped by something else, the caveat scoped a claim to a dimension the rows never carried.
func effCaveat(by string) string {
	return "Efficiency is directional: task type (greenfield vs. debugging) drives lines-per-cost; this is a diagnostic per " +
		by + ", never a performance metric."
}

// effCoverage is what the table's disclosure is computed from: how many rows come from a
// source that can answer each withholdable column, and how far the AI-line answer reaches
// measured in tokens.
//
// Capability is decided on rows and only quantified on tokens. A source that records neither
// lines nor tokens contributes nothing to either side of the token share, so the share alone
// read 100% -- "every source in this table records changed lines" printed over a table whose
// only source records none.
type effCoverage struct {
	Rows, LineCapableRows, EditCapableRows int
	LineCapableTokens, TotalTokens         int64
}

// effCoverageNote discloses which columns this table withholds and how far the AI-line column
// reaches. Quantified rather than qualitative: a note that reads the same whether the
// line-blind share is 0.1% or half the table is the failure UnpricedDisclosure already fixed
// for the cost column.
//
// Lines and edits are disclosed apart because they are recorded apart -- Copilot CLI records
// one, Antigravity CLI the other -- and a note that withholds both because one is missing
// calls a measured figure absent, which is the same fabrication as a fabricated zero with the
// sign flipped.
func effCoverageNote(c *effCoverage) string {
	switch {
	case c.Rows == 0:
		return ""
	case c.LineCapableRows == 0 || c.EditCapableRows == 0:
		return effWithheldNote(c)
	case c.LineCapableRows == c.Rows && c.EditCapableRows == c.Rows &&
		(c.TotalTokens == 0 || c.LineCapableTokens == c.TotalTokens):
		return "Every source in this table records changed lines and edits."
	case c.TotalTokens == 0:
		return "AI lines and edits come only from the sources that record them; a group showing — in a column contributes nothing to it. Run `assaio-agent signals coverage` for what your own data supports."
	}
	return "AI lines come only from the sources that record them (" +
		humanize.Percent(float64(c.LineCapableTokens)/float64(c.TotalTokens)) +
		" of this table's tokens); a group showing — in a column contributes cost and nothing to that column. Run `assaio-agent signals coverage` for what your own data supports."
}

// effWithheldNote names the columns no source in the table can fill, and only those. The
// $/100-lines column follows the AI-line column because it is built from it; the edit column
// stands on its own capability, and the sentence says so rather than sweeping it in.
func effWithheldNote(c *effCoverage) string {
	switch {
	case c.LineCapableRows == 0 && c.EditCapableRows == 0:
		return "No source in this table records a changed line or an edit, so the AI-line, edit and $/100-lines columns are withheld -- absent, not zero."
	case c.LineCapableRows == 0:
		return "No source in this table records a changed line, so the AI-line and $/100-lines columns are withheld -- absent, not zero; the edit column beside them reads the sources that do record one."
	}
	return "No source in this table records an edit, so the edit column is withheld -- absent, not zero; AI lines come only from the sources that record them."
}
