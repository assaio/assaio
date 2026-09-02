package report

import (
	"fmt"
	"io"
	"strings"

	prettytable "github.com/jedib0t/go-pretty/v6/table"

	"github.com/assaio/assaio/internal/humanize"
)

// RenderEffectivenessTable writes rows to w as a human-readable efficiency table
// grouped by by, with a totals footer and the honesty caveats every effectiveness view
// must carry: efficiency is directional, and line-count coverage still varies by tool.
func RenderEffectivenessTable(w io.Writer, rows []EffRow, by string) error {
	tw := prettytable.NewWriter()
	tw.SetOutputMirror(w)
	tw.AppendHeader(prettytable.Row{strings.ToUpper(by), "AI LINES", "EDITS", "REJ", "COST $", "$/100 LINES"})
	tw.SetColumnConfigs(rightAlignFrom(1, 5))

	totals := effTotals{Coverage: effCoverage{Rows: len(rows)}}
	var unpriced Unpriced
	for i := range rows {
		r := &rows[i]
		cost, priced := formatEffCost(r)
		totals.add(r, priced)
		unpriced.Tokens += r.UnpricedTokens
		unpriced.Total += r.TokensTotal
		if r.HasUnpriced {
			unpriced.Rows++
			if !r.Tokened {
				unpriced.Untokened++
			}
		}
		tw.AppendRow(effTableRow(r, cost))
	}
	totals.Coverage.TotalTokens = unpriced.Total
	tw.AppendFooter(effTotalRow(&totals))
	tw.Render()

	if note := UnpricedDisclosure(&unpriced, "the tokens in this table"); note != "" {
		if _, err := fmt.Fprintln(w, note); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, effCaveat(by)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, effCoverageNote(&totals.Coverage)); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, CostEstimateDisclosure)
	return err
}

// effTotals is the footer's running sum together with the capability counts that decide which
// of its cells may be printed at all.
type effTotals struct {
	Lines, Edits, Rejected int64
	Cost                   float64
	AnyPriced              bool
	// RefusableRows is how many groups came from a source that records a human declining a
	// call; Coverage carries the same count for the line and edit columns.
	RefusableRows int
	Coverage      effCoverage
}

// add folds one group into the footer, each column under its own capability. One condition for
// all of them dropped what a source did measure: a group recording edits and no changed line
// contributed nothing to the edit total it could answer.
func (t *effTotals) add(r *EffRow, priced float64) {
	t.Cost += priced
	t.AnyPriced = t.AnyPriced || r.Cost != nil
	if r.LineCapable {
		t.Coverage.LineCapableRows++
		t.Lines += r.LinesAdded
		t.Coverage.LineCapableTokens += r.TokensTotal
	}
	if r.EditCapable {
		t.Coverage.EditCapableRows++
		t.Edits += r.Edits
	}
	if r.Refusable {
		t.RefusableRows++
		t.Rejected += r.Rejected
	}
}

// effTotalRow sums the columns that have something to sum. A total is the cell a reader trusts
// most and checks least, so each cell withholds on the same condition its rows do: no group
// recording a changed line totals no lines, no group whose source records a refusal totals no
// rejections, and no group priced totals no cost -- "0" and "$0.00" would each be a
// measurement nobody made.
func effTotalRow(t *effTotals) prettytable.Row {
	lineCapable := t.Coverage.LineCapableRows > 0
	return prettytable.Row{
		"TOTAL",
		capableCell(humanize.Int(t.Lines), lineCapable),
		capableCell(humanize.Int(t.Edits), t.Coverage.EditCapableRows > 0),
		capableCell(humanize.Int(t.Rejected), t.RefusableRows > 0),
		capableCell(humanize.USDCell(t.Cost), t.AnyPriced),
		capableCell(footerRatio(t.Cost, t.Lines), lineCapable && t.AnyPriced),
	}
}

// capableCell prints value only where some source behind the cell records what it counts, and
// the dash otherwise -- never the zero the arithmetic hands back for a field nobody wrote.
func capableCell(value string, capable bool) string {
	if !capable {
		return "—"
	}
	return value
}

// formatEffCost renders r's cost cell (with a trailing "*" when the row has unpriced
// usage) and returns the priced amount to add to the running total.
func formatEffCost(r *EffRow) (cell string, priced float64) {
	cell = "—"
	if r.Cost != nil {
		cell = humanize.USDCell(*r.Cost)
		priced = *r.Cost
	}
	if r.HasUnpriced {
		cell += "*"
	}
	return cell, priced
}

// formatCostPer100 renders r's $/100-lines cell, "—" when the ratio is undefined.
func formatCostPer100(r *EffRow) string {
	if r.CostPer100Lines == nil {
		return "—"
	}
	cell := humanize.USDCell(*r.CostPer100Lines)
	if r.HasUnpriced {
		cell += "*"
	}
	return cell
}

// effTableRow builds one data row, substituting a placeholder for an empty group label. Each
// activity cell answers for its own source capability: the three are recorded apart and a
// group can answer any one of them without the others.
func effTableRow(r *EffRow, cost string) prettytable.Row {
	label := r.Group
	if label == "" {
		label = "(unknown)"
	}
	return prettytable.Row{
		label,
		capableCell(humanize.Int(r.LinesAdded), r.LineCapable),
		capableCell(humanize.Int(r.Edits), r.EditCapable),
		capableCell(humanize.Int(r.Rejected), r.Refusable),
		cost,
		formatCostPer100(r),
	}
}

// footerRatio recomputes $/100 lines from column totals rather than averaging each
// row's ratio, and is "—" when no group in the report has any AI lines.
func footerRatio(totalCost float64, totalLines int64) string {
	if totalLines == 0 {
		return "—"
	}
	return humanize.USDCell(totalCost / (float64(totalLines) / 100))
}
