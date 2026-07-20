package report

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	prettytable "github.com/jedib0t/go-pretty/v6/table"
)

// effCaveat states that efficiency is a diagnostic signal, never a performance metric.
const effCaveat = "Efficiency is directional: task type (greenfield vs. debugging) drives lines-per-cost; this is a diagnostic per project, never a performance metric."

// effCoverageNote discloses which tools' AI-line counts are real today.
const effCoverageNote = "AI-line activity is captured for Claude Code and Codex today; Gemini CLI and Cline contribute cost but not line counts (see ROADMAP)."

// RenderEffectivenessTable writes rows to w as a human-readable efficiency table
// grouped by by, with a totals footer and the honesty caveats every effectiveness view
// must carry: efficiency is directional, and line-count coverage still varies by tool.
func RenderEffectivenessTable(w io.Writer, rows []EffRow, by string) error {
	tw := prettytable.NewWriter()
	tw.SetOutputMirror(w)
	tw.AppendHeader(prettytable.Row{strings.ToUpper(by), "AI LINES", "EDITS", "REJ", "COST $", "$/100 LINES"})
	tw.SetColumnConfigs(rightAlignFrom(1, 5))

	var totalLines, totalEdits, totalRejected int64
	var totalCost float64
	var anyUnpriced bool
	for i := range rows {
		r := &rows[i]
		cost, priced := formatEffCost(r)
		totalCost += priced
		totalLines += r.LinesAdded
		totalEdits += r.Edits
		totalRejected += r.Rejected
		if r.HasUnpriced {
			anyUnpriced = true
		}
		tw.AppendRow(effTableRow(r, by, cost))
	}
	tw.AppendFooter(prettytable.Row{
		"TOTAL", totalLines, totalEdits, totalRejected,
		strconv.FormatFloat(totalCost, 'f', 4, 64), footerRatio(totalCost, totalLines),
	})
	tw.Render()

	if anyUnpriced {
		if _, err := fmt.Fprintln(w, unpricedFootnote); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, effCaveat); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, effCoverageNote); err != nil {
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
		cell = strconv.FormatFloat(*r.Cost, 'f', 4, 64)
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
	cell := strconv.FormatFloat(*r.CostPer100Lines, 'f', 4, 64)
	if r.HasUnpriced {
		cell += "*"
	}
	return cell
}

// effTableRow builds one data row, substituting a placeholder for an empty group label.
// by isn't otherwise available on EffRow (Group already holds the resolved value), so the
// caller passes it through just for emptyDimLabel's per-dimension placeholder choice.
func effTableRow(r *EffRow, by, cost string) prettytable.Row {
	label := r.Group
	if label == "" {
		label = emptyDimLabel(by)
	}
	return prettytable.Row{label, r.LinesAdded, r.Edits, r.Rejected, cost, formatCostPer100(r)}
}

// footerRatio recomputes $/100 lines from column totals rather than averaging each
// row's ratio, and is "—" when no group in the report has any AI lines.
func footerRatio(totalCost float64, totalLines int64) string {
	if totalLines == 0 {
		return "—"
	}
	return strconv.FormatFloat(totalCost/(float64(totalLines)/100), 'f', 4, 64)
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
