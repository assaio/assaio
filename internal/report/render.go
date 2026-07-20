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
	var anyUnpriced bool
	for i := range rows {
		r := &rows[i]
		cost, priced := formatCost(r)
		total += priced
		if r.HasUnpriced {
			anyUnpriced = true
		}
		tw.AppendRow(tableRow(r, grouped, by, cost))
	}

	tw.AppendFooter(append(append(prettytable.Row{}, dimFooter...),
		"", "", "", "", "TOTAL", strconv.FormatFloat(total, 'f', 4, 64)))
	tw.Render()
	if anyUnpriced {
		if _, err := fmt.Fprintln(w, unpricedFootnote); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w, CostEstimateDisclosure)
	return err
}

// tableDimColumns returns the leading header and footer cells for the dimension
// columns: Day/Tool/Model when ungrouped, or a single upper-cased dimension name.
func tableDimColumns(grouped bool, by string) (header, footer []interface{}) {
	if !grouped {
		return []interface{}{"Day", "Tool", "Model"}, []interface{}{"", "", ""}
	}
	return []interface{}{strings.ToUpper(by)}, []interface{}{""}
}

// rightAlignFrom right-aligns numCols numeric columns following dimCols columns.
func rightAlignFrom(dimCols, numCols int) []prettytable.ColumnConfig {
	cols := make([]prettytable.ColumnConfig, 0, numCols)
	for n := dimCols + 1; n <= dimCols+numCols; n++ {
		cols = append(cols, prettytable.ColumnConfig{Number: n, Align: text.AlignRight})
	}
	return cols
}

// formatCost renders r's cost cell (with a trailing "*" when the row has unpriced
// usage) and returns the priced amount to add to the running total.
func formatCost(r *Row) (cell string, priced float64) {
	cell = "—"
	if r.Priced {
		cell = strconv.FormatFloat(*r.Cost, 'f', 4, 64)
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
			label = emptyDimLabel(by)
		}
		return prettytable.Row{label, r.In, r.Out, r.CacheRead, r.CacheWrite, cacheEffStr, cost}
	}
	return prettytable.Row{r.Day, r.Tool, r.Model, r.In, r.Out, r.CacheRead, r.CacheWrite, cacheEffStr, cost}
}

// RenderJSON writes rows to w as indented JSON.
func RenderJSON(w io.Writer, rows []Row) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

// RenderCSV writes rows to w as CSV with a header row.
func RenderCSV(w io.Writer, rows []Row) error {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"day", "tool", "model", "project", "entrypoint", "member", "in", "out",
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
			r.Day, r.Tool, r.Model, r.Project, r.Entrypoint, r.Member,
			strconv.FormatInt(r.In, 10), strconv.FormatInt(r.Out, 10),
			strconv.FormatInt(r.CacheRead, 10), strconv.FormatInt(r.CacheWrite, 10),
			strconv.FormatInt(r.Reasoning, 10), cacheEffStr,
			cost, strconv.FormatBool(r.Priced), strconv.FormatBool(r.HasUnpriced),
		})
	}
	cw.Flush()
	return cw.Error()
}
