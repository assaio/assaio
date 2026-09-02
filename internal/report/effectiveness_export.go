package report

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"strconv"
)

// The machine formats of `effectiveness`. They live apart from the table because they answer a
// different reader: the table prints a dash where a source records nothing, and these two have
// to carry that same absence as a field a program can branch on.

// RenderEffectivenessJSON writes rows to w as indented JSON.
func RenderEffectivenessJSON(w io.Writer, rows []EffRow) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

// RenderEffectivenessCSV writes rows to w as CSV with a header row. The four capability
// columns carry what the table renders as a dash: a numeric format has no dash to print, so
// without them a group whose source keeps no counter exports the same "0" as one that did the
// work and produced nothing. They are appended rather than placed beside the columns they
// qualify, so a reader's existing column positions still hold.
func RenderEffectivenessCSV(w io.Writer, rows []EffRow) error {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"group", "lines_added", "lines_removed", "edits", "tool_calls", "rejected",
		"tokens_total", "cost", "has_unpriced", "cost_per_100_lines",
		"line_capable", "edit_capable", "refusable", "tokened",
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
			strconv.FormatBool(r.LineCapable), strconv.FormatBool(r.EditCapable),
			strconv.FormatBool(r.Refusable), strconv.FormatBool(r.Tokened),
		})
	}
	cw.Flush()
	return cw.Error()
}
