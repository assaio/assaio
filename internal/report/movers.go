package report

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	prettytable "github.com/jedib0t/go-pretty/v6/table"
)

// MoverRow is one group's change in cost and AI lines between two adjacent equal windows:
// the recent window vs. the prior one of the same length.
type MoverRow struct {
	Group      string
	CostNow    float64
	CostPrior  float64
	DeltaCost  float64
	LinesNow   int64
	LinesPrior int64
	DeltaLines int64
}

// Movers diffs two windows' per-group effectiveness rows into MoverRows, sorted by the
// magnitude of the cost change (largest movers first) so "what changed" leads. A group
// present in only one window compares against zero, surfacing new and dropped work.
func Movers(recent, prior []EffRow) []MoverRow {
	byGroup := map[string]*MoverRow{}
	at := func(g string) *MoverRow {
		if _, ok := byGroup[g]; !ok {
			byGroup[g] = &MoverRow{Group: g}
		}
		return byGroup[g]
	}
	for i := range recent {
		m := at(recent[i].Group)
		m.CostNow = costOrZero(recent[i].Cost)
		m.LinesNow = recent[i].LinesAdded
	}
	for i := range prior {
		m := at(prior[i].Group)
		m.CostPrior = costOrZero(prior[i].Cost)
		m.LinesPrior = prior[i].LinesAdded
	}
	out := make([]MoverRow, 0, len(byGroup))
	for _, m := range byGroup {
		m.DeltaCost = m.CostNow - m.CostPrior
		m.DeltaLines = m.LinesNow - m.LinesPrior
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool { return absFloat(out[i].DeltaCost) > absFloat(out[j].DeltaCost) })
	return out
}

// RenderMovers writes the period-over-period top-movers table: each group's signed cost and
// AI-line change vs. the prior window of the same length, deltas first so the change leads.
func RenderMovers(w io.Writer, movers []MoverRow, by string) error {
	tw := prettytable.NewWriter()
	tw.SetOutputMirror(w)
	tw.AppendHeader(prettytable.Row{strings.ToUpper(by), "Δ COST $", "COST $ NOW", "Δ AI LINES", "AI LINES NOW"})
	tw.SetColumnConfigs(rightAlignFrom(1, 4))
	for i := range movers {
		m := &movers[i]
		label := m.Group
		if label == "" {
			label = emptyDimLabel(by)
		}
		tw.AppendRow(prettytable.Row{
			label, signedFloat(m.DeltaCost), strconv.FormatFloat(m.CostNow, 'f', 4, 64),
			signedInt(m.DeltaLines), m.LinesNow,
		})
	}
	tw.Render()
	_, err := fmt.Fprintln(w, CostEstimateDisclosure)
	return err
}

func costOrZero(c *float64) float64 {
	if c == nil {
		return 0
	}
	return *c
}

func absFloat(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// signedFloat formats a cost delta with an explicit leading "+" for gains, so a rise reads
// distinctly from a fall at a glance.
func signedFloat(f float64) string {
	s := strconv.FormatFloat(f, 'f', 4, 64)
	if f > 0 {
		return "+" + s
	}
	return s
}

func signedInt(n int64) string {
	s := strconv.FormatInt(n, 10)
	if n > 0 {
		return "+" + s
	}
	return s
}
