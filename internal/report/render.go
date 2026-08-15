package report

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	prettytable "github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/assaio/assaio/internal/humanize"
)

// RenderTable writes rows to w as a human-readable table with a totals footer. by is
// the --by dimension used to produce rows: "day" (or "") renders the current
// Day/Tool/Model columns unchanged; any other dimension renders a single leading
// column labeled with the upper-cased dimension name holding that dimension's value,
// with an empty value shown as "(unknown)". Rows whose cost excludes unpriced usage
// are marked with a trailing "*" and a footnote.
func RenderTable(w io.Writer, rows []Row, by string) error {
	tw := prettytable.NewWriter()
	tw.SetOutputMirror(w)

	grouped := by != "" && by != "day"
	dimCols, dimFooter := tableDimColumns(grouped, by)
	tw.AppendHeader(append(append(prettytable.Row{}, dimCols...),
		"In", "Out", "Cache R", "Cache W", "Cache%", "Cost $"))
	tw.SetColumnConfigs(rightAlignFrom(len(dimCols), 6))

	var total float64
	for i := range rows {
		r := &rows[i]
		cost, priced := formatCost(r)
		total += priced
		tw.AppendRow(tableRow(r, grouped, by, cost))
	}

	tw.AppendFooter(append(append(prettytable.Row{}, dimFooter...),
		"", "", "", "", "TOTAL", humanize.USDCell(total)))
	tw.Render()
	unpriced := BuildUnpriced(rows)
	if note := UnpricedDisclosure(&unpriced, "the tokens in this table"); note != "" {
		if _, err := fmt.Fprintln(w, note); err != nil {
			return err
		}
	}
	if note, ok := granularityFootnote(rows); ok {
		if _, err := fmt.Fprintln(w, note); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w, CostEstimateDisclosure)
	return err
}

// granularityFootnote returns the legend for the "‡" marker when any row is not plain
// per-turn data. A mixed row wins over a merely session-level one: it is the harder case
// to read, so it is the one named.
func granularityFootnote(rows []Row) (string, bool) {
	note := ""
	for i := range rows {
		switch rows[i].Granularity {
		case GranularityMixed:
			return mixedGranularityFootnote, true
		case "session":
			note = sessionGranularityFootnote
		}
	}
	return note, note != ""
}

// granularityMark is the "‡" a row carries when its unit is not one turn.
func granularityMark(g string) string {
	if g == "session" || g == GranularityMixed {
		return " ‡"
	}
	return ""
}

// tableDimColumns returns the leading header and footer cells for the dimension
// columns: Day/Tool/Model when ungrouped, or a single upper-cased dimension name.
func tableDimColumns(grouped bool, by string) (header, footer []interface{}) {
	if !grouped {
		return []interface{}{"Day", "Tool", "Model"}, []interface{}{"", "", ""}
	}
	return []interface{}{strings.ToUpper(by)}, []interface{}{""}
}

// rightAlignFrom right-aligns numCols numeric columns following dimCols columns. The
// header goes with the digits: a comma-grouped count is a string, which the table would
// otherwise left-align away from the column it names.
func rightAlignFrom(dimCols, numCols int) []prettytable.ColumnConfig {
	cols := make([]prettytable.ColumnConfig, 0, numCols)
	for n := dimCols + 1; n <= dimCols+numCols; n++ {
		cols = append(cols, prettytable.ColumnConfig{
			Number: n, Align: text.AlignRight, AlignHeader: text.AlignRight, AlignFooter: text.AlignRight,
		})
	}
	return cols
}

// formatCost renders r's cost cell (with a trailing "*" when the row has unpriced
// usage) and returns the priced amount to add to the running total.
func formatCost(r *Row) (cell string, priced float64) {
	cell = "—"
	if r.Priced {
		cell = humanize.USDCell(*r.Cost)
		priced = *r.Cost
	}
	if r.HasUnpriced {
		cell += "*"
	}
	return cell, priced
}

// tableRow builds one data row, using the single grouped-dimension layout or the
// default Day/Tool/Model layout.
func tableRow(r *Row, grouped bool, by, cost string) prettytable.Row {
	cacheEffStr := "—"
	if r.CacheEff != nil {
		cacheEffStr = strconv.FormatFloat(*r.CacheEff*100, 'f', 1, 64)
	}
	if grouped {
		label := dimValue(r, by)
		if label == "" {
			label = "(unknown)"
		}
		return prettytable.Row{
			label + granularityMark(r.Granularity),
			humanize.Int(r.In), humanize.Int(r.Out), humanize.Int(r.CacheRead), humanize.Int(r.CacheWrite),
			cacheEffStr, cost,
		}
	}
	return prettytable.Row{
		r.Day, r.Tool + granularityMark(r.Granularity), r.Model,
		humanize.Int(r.In), humanize.Int(r.Out), humanize.Int(r.CacheRead), humanize.Int(r.CacheWrite),
		cacheEffStr, cost,
	}
}

// RenderJSON writes rows to w as indented JSON.
func RenderJSON(w io.Writer, rows []Row) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

// RenderCSV writes rows to w as CSV with a header row. The three annotation columns are
// part of the fixed shape rather than added per dimension: `--by task|outcome|difficulty`
// stamps the group key into one of them and leaves every other identity column empty, so a
// header without them emitted rows nothing could tell apart.
func RenderCSV(w io.Writer, rows []Row) error {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"day", "tool", "model", "project", "entrypoint", "member", "granularity",
		"task", "outcome", "difficulty", "in", "out",
		"cache_read", "cache_write", "reasoning", "cache_eff", "cost", "priced", "has_unpriced",
	})
	for i := range rows {
		r := &rows[i]
		cost := ""
		if r.Priced {
			cost = strconv.FormatFloat(*r.Cost, 'f', 6, 64)
		}
		cacheEffStr := ""
		if r.CacheEff != nil {
			cacheEffStr = strconv.FormatFloat(*r.CacheEff, 'f', 6, 64)
		}
		_ = cw.Write([]string{
			r.Day, r.Tool, r.Model, r.Project, r.Entrypoint, r.Member, r.Granularity,
			r.Task, r.Outcome, r.Difficulty,
			strconv.FormatInt(r.In, 10), strconv.FormatInt(r.Out, 10),
			strconv.FormatInt(r.CacheRead, 10), strconv.FormatInt(r.CacheWrite, 10),
			strconv.FormatInt(r.Reasoning, 10), cacheEffStr,
			cost, strconv.FormatBool(r.Priced), strconv.FormatBool(r.HasUnpriced),
		})
	}
	cw.Flush()
	return cw.Error()
}
