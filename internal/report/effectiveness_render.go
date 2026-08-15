package report

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	prettytable "github.com/jedib0t/go-pretty/v6/table"

	"github.com/assaio/assaio/internal/humanize"
)

// effCaveat states that efficiency is a diagnostic signal, never a performance metric. It
// names the dimension the table is actually grouped by: printed as "per project" over a table
// grouped by something else, the caveat scoped a claim to a dimension the rows never carried.
func effCaveat(by string) string {
	return "Efficiency is directional: task type (greenfield vs. debugging) drives lines-per-cost; this is a diagnostic per " +
		by + ", never a performance metric."
}

// effCoverageNote discloses how far the AI-line column reaches. Quantified rather than
// qualitative: a note that reads the same whether the line-blind share is 0.1% or half the table
// is the failure UnpricedDisclosure already fixed for the cost column.
func effCoverageNote(lineCapableTokens, totalTokens int64) string {
	if totalTokens == 0 || lineCapableTokens == totalTokens {
		return "Every source in this table records changed lines."
	}
	return "AI lines and edits come only from the sources that record them (" +
		humanize.Percent(float64(lineCapableTokens)/float64(totalTokens)) +
		" of this table's tokens); a group showing — contributes cost and no line count. Run `assaio-agent signals coverage` for what your own data supports."
}

// RenderEffectivenessTable writes rows to w as a human-readable efficiency table
// grouped by by, with a totals footer and the honesty caveats every effectiveness view
// must carry: efficiency is directional, and line-count coverage still varies by tool.
func RenderEffectivenessTable(w io.Writer, rows []EffRow, by string) error {
	tw := prettytable.NewWriter()
	tw.SetOutputMirror(w)
	tw.AppendHeader(prettytable.Row{strings.ToUpper(by), "AI LINES", "EDITS", "REJ", "COST $", "$/100 LINES"})
	tw.SetColumnConfigs(rightAlignFrom(1, 5))

	var totalLines, totalEdits, totalRejected, lineCapableTokens int64
	var totalCost float64
	var unpriced Unpriced
	for i := range rows {
		r := &rows[i]
		cost, priced := formatEffCost(r)
		totalCost += priced
		if r.LineCapable {
			totalLines += r.LinesAdded
			totalEdits += r.Edits
			lineCapableTokens += r.TokensTotal
		}
		totalRejected += r.Rejected
		unpriced.Tokens += r.UnpricedTokens
		unpriced.Total += r.TokensTotal
		if r.HasUnpriced {
			unpriced.Rows++
		}
		tw.AppendRow(effTableRow(r, cost))
	}
	tw.AppendFooter(prettytable.Row{
		"TOTAL", humanize.Int(totalLines), humanize.Int(totalEdits), humanize.Int(totalRejected),
		humanize.USDCell(totalCost), footerRatio(totalCost, totalLines),
	})
	tw.Render()

	if note := UnpricedDisclosure(&unpriced, "the tokens in this table"); note != "" {
		if _, err := fmt.Fprintln(w, note); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, effCaveat(by)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, effCoverageNote(lineCapableTokens, unpriced.Total)); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, CostEstimateDisclosure)
	return err
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

// effTableRow builds one data row, substituting a placeholder for an empty group label.
func effTableRow(r *EffRow, cost string) prettytable.Row {
	label := r.Group
	if label == "" {
		label = "(unknown)"
	}
	lines, edits := humanize.Int(r.LinesAdded), humanize.Int(r.Edits)
	if !r.LineCapable {
		lines, edits = "—", "—"
	}
	return prettytable.Row{label, lines, edits, humanize.Int(r.Rejected), cost, formatCostPer100(r)}
}

// footerRatio recomputes $/100 lines from column totals rather than averaging each
// row's ratio, and is "—" when no group in the report has any AI lines.
func footerRatio(totalCost float64, totalLines int64) string {
	if totalLines == 0 {
		return "—"
	}
	return humanize.USDCell(totalCost / (float64(totalLines) / 100))
}

// RenderEffectivenessJSON writes rows to w as indented JSON.
func RenderEffectivenessJSON(w io.Writer, rows []EffRow) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

// RenderEffectivenessCSV writes rows to w as CSV with a header row.
func RenderEffectivenessCSV(w io.Writer, rows []EffRow) error {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"group", "lines_added", "lines_removed", "edits", "tool_calls", "rejected",
		"tokens_total", "cost", "has_unpriced", "cost_per_100_lines",
	})
	for i := range rows {
		r := &rows[i]
		cost, ratio := "", ""
		if r.Cost != nil {
			cost = strconv.FormatFloat(*r.Cost, 'f', 6, 64)
		}
		if r.CostPer100Lines != nil {
			ratio = strconv.FormatFloat(*r.CostPer100Lines, 'f', 6, 64)
		}
		_ = cw.Write([]string{
			r.Group, strconv.FormatInt(r.LinesAdded, 10), strconv.FormatInt(r.LinesRemoved, 10),
			strconv.FormatInt(r.Edits, 10), strconv.FormatInt(r.ToolCalls, 10), strconv.FormatInt(r.Rejected, 10),
			strconv.FormatInt(r.TokensTotal, 10), cost, strconv.FormatBool(r.HasUnpriced), ratio,
		})
	}
	cw.Flush()
	return cw.Error()
}
