package report

import (
	"fmt"
	"io"
	"strconv"

	"github.com/assaio/assaio/internal/humanize"
)

// emptyStatusHint replaces the dashboard when the store holds no records at all.
const emptyStatusHint = "No usage yet. Run 'assaio-agent backfill' to import months of local session logs, " +
	"then 'assaio-agent status' shows your hottest projects and cost-per-line."

// RenderEmptyStatusHint writes the rich empty-state hint shown when the store has no
// records at all, rather than a terse blank dashboard.
func RenderEmptyStatusHint(w io.Writer) error {
	_, err := fmt.Fprintln(w, emptyStatusHint)
	return err
}

// RenderStatusSummary writes the terminal dashboard for in: an inventory line, a
// cost/lines headline, hot projects, going-stale projects, dormant tools, and the
// honesty caveat. It never renders a named-individual ranking -- project and tool
// signals only.
func RenderStatusSummary(w io.Writer, in *Insights) error {
	for _, section := range []func(io.Writer, *Insights) error{
		writeInventoryLine, writeHeadline, writeHotSection, writeGoingStaleSection, writeDormantSection,
	} {
		if err := section(w, in); err != nil {
			return err
		}
	}
	if note := UnpricedDisclosure(&in.Inventory.Unpriced, "the whole window's tokens"); note != "" {
		if _, err := fmt.Fprintln(w, note); err != nil {
			return err
		}
	}
	if note := LineCoverageDisclosure(&in.Inventory); note != "" {
		if _, err := fmt.Fprintln(w, note); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, statusCaveat); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, CostEstimateDisclosure)
	return err
}

// RenderChurnLine writes the one-line rework/thrash figure: how much AI-added code was
// itself removed again later in the same transcript -- a within-session proxy for "AI
// wrote code that didn't stick," never a per-person metric. A window whose sources never
// record an undone line says so, rather than printing the 0% an empty denominator yields.
func RenderChurnLine(w io.Writer, c ChurnStat) error {
	if c.Rows == 0 {
		_, err := fmt.Fprint(w, "rework: not recorded by any source in this window — absent, not churn-free\n\n")
		return err
	}
	if c.ExceedsItsWhole() {
		_, err := fmt.Fprintf(w, "rework: %s\n\n", ChurnBoundaryNote(&c))
		return err
	}
	_, err := fmt.Fprintf(w, "rework: %d lines reworked (%s of AI-added) — thrash proxy, within-session\n\n",
		c.ReworkLines, humanize.PercentOrDash(c.ReworkLines, c.LinesAdded, 0))
	return err
}

func writeInventoryLine(w io.Writer, in *Insights) error {
	inv := in.Inventory
	_, err := fmt.Fprintf(w, "%d projects · %d models · %d tools · %d days of history\n\n",
		inv.Projects, inv.Models, inv.Tools, inv.Days)
	return err
}

func writeHeadline(w io.Writer, in *Insights) error {
	inv := in.Inventory
	cost := "—"
	if inv.TotalCost != nil {
		cost = "$" + strconv.FormatFloat(*inv.TotalCost, 'f', 2, 64)
		if inv.HasUnpriced {
			cost += "*"
		}
	}
	lines, ratio := lineFigures(&inv)
	_, err := fmt.Fprintf(w, "%s total · %s AI lines · %s/100 lines\n\n", cost, lines, ratio)
	return err
}

// lineFigures renders the AI-line count and the $/100 ratio, withholding both when no source
// in the window records a changed line at all. Summing LinesAdded with no capability gate and
// formatting the result made a structural silence print as "0 AI lines" -- an AI that wrote
// nothing rather than a source that never said (ADR 0011, applied throughout internal/analyze
// and nowhere here). Gemini CLI and Cline answer no line signal, so their users read it daily.
func lineFigures(inv *Inventory) (lines, ratio string) {
	if inv.LineCapableRows == 0 {
		return "—", "—"
	}
	lines = strconv.FormatInt(inv.TotalLinesAdded, 10)
	ratio = "—"
	if r := costPer100Lines(inv.TotalCost, inv.TotalLinesAdded); r != nil {
		ratio = "$" + strconv.FormatFloat(*r, 'f', 4, 64)
	}
	return lines, ratio
}

func writeHotSection(w io.Writer, in *Insights) error {
	// The window is named here because the footnote below quantifies the whole window, and a
	// 7-day row under a 30-day share would read as that share's own error bar otherwise.
	if _, err := fmt.Fprintln(w, "Hot — where spend concentrates (last 7 days)"); err != nil {
		return err
	}
	if len(in.Hot) == 0 {
		return writeLineAndBlank(w, "  No usage in the last 7 days.")
	}
	for _, g := range in.Hot {
		if err := writeGroupCostLine(w, g); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w)
	return err
}

// writeGroupCostLine renders one Hot row: name, cost, and cost-per-100-lines.
func writeGroupCostLine(w io.Writer, g GroupStat) error {
	cost := "—"
	if g.Cost != nil {
		cost = "$" + strconv.FormatFloat(*g.Cost, 'f', 4, 64)
	}
	if g.HasUnpriced {
		cost += "*"
	}
	ratio := "—"
	if g.CostPer100Lines != nil {
		ratio = "$" + strconv.FormatFloat(*g.CostPer100Lines, 'f', 4, 64)
	}
	_, err := fmt.Fprintf(w, "  %s — %s · %s/100 lines\n", groupLabel(g), cost, ratio)
	return err
}

func writeGoingStaleSection(w io.Writer, in *Insights) error {
	if _, err := fmt.Fprintln(w, "Going stale"); err != nil {
		return err
	}
	if len(in.GoingStale) == 0 {
		return writeLineAndBlank(w, "  Nothing has gone quiet.")
	}
	for _, g := range in.GoingStale {
		if _, err := fmt.Fprintf(w, "  %s — last active %s\n", groupLabel(g), g.LastActive); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w)
	return err
}

func writeDormantSection(w io.Writer, in *Insights) error {
	if len(in.DormantTools) == 0 {
		_, err := fmt.Fprintln(w, "All detected tools active.")
		return err
	}
	if _, err := fmt.Fprintln(w, "Dormant tools"); err != nil {
		return err
	}
	for _, g := range in.DormantTools {
		if _, err := fmt.Fprintf(w, "  %s — set up, no recent activity\n", groupLabel(g)); err != nil {
			return err
		}
	}
	return nil
}

// groupLabel substitutes "(unknown)" for an empty group name, mirroring the
// effectiveness table's convention for an unset dimension value.
func groupLabel(g GroupStat) string {
	if g.Name == "" {
		return "(unknown)"
	}
	return g.Name
}

// writeLineAndBlank writes line followed by a blank line, the shared shape for a
// section's empty-state message.
func writeLineAndBlank(w io.Writer, line string) error {
	if _, err := fmt.Fprintln(w, line); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w)
	return err
}
