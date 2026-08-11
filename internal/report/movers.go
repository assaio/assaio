package report

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	prettytable "github.com/jedib0t/go-pretty/v6/table"

	"github.com/assaio/assaio/internal/humanize"
)

// MoverRow is one group's change in cost and AI lines between two adjacent equal windows:
// the recent window vs. the prior one of the same length. HasUnpricedNow/Prior mark a
// window in which some of the group's usage had no price, so its cost (and the delta) is a
// floor, not a full figure -- surfaced with a "*" and a footnote rather than a bare number
// a reader would mistake for the whole cost.
type MoverRow struct {
	Group            string
	CostNow          float64
	CostPrior        float64
	DeltaCost        float64
	LinesNow         int64
	LinesPrior       int64
	DeltaLines       int64
	HasUnpricedNow   bool
	HasUnpricedPrior bool
	// The unpriced condition is kept per window, never pooled. Every marked figure here --
	// "COST $ NOW" and the delta it feeds -- is anchored on one of the two, so a share over
	// their sum would legend a column with an error bar computed from rows outside it: a
	// recent window that is wholly unpriced beside a large priced prior one reads as a few
	// percent missing, which is the understatement the marker exists to end.
	UnpricedNow, TokensNow     int64
	UnpricedPrior, TokensPrior int64
}

// hasUnpriced reports whether either window excluded some of the group's cost.
func (m *MoverRow) hasUnpriced() bool { return m.HasUnpricedNow || m.HasUnpricedPrior }

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
		m.HasUnpricedNow = recent[i].HasUnpriced
		m.UnpricedNow += recent[i].UnpricedTokens
		m.TokensNow += recent[i].TokensTotal
	}
	for i := range prior {
		m := at(prior[i].Group)
		m.CostPrior = costOrZero(prior[i].Cost)
		m.LinesPrior = prior[i].LinesAdded
		m.HasUnpricedPrior = prior[i].HasUnpriced
		m.UnpricedPrior += prior[i].UnpricedTokens
		m.TokensPrior += prior[i].TokensTotal
	}
	out := make([]MoverRow, 0, len(byGroup))
	for _, m := range byGroup {
		m.DeltaCost = m.CostNow - m.CostPrior
		m.DeltaLines = m.LinesNow - m.LinesPrior
		out = append(out, *m)
	}
	// Stable + name tiebreak so tied cost deltas (common when groups are all unpriced,
	// making every delta 0) keep a deterministic, diffable order across runs.
	sort.SliceStable(out, func(i, j int) bool {
		di, dj := absFloat(out[i].DeltaCost), absFloat(out[j].DeltaCost)
		if di != dj {
			return di > dj
		}
		return out[i].Group < out[j].Group
	})
	return out
}

// RenderMovers writes the period-over-period top-movers table: each group's signed cost and
// AI-line change vs. the prior window of the same length, deltas first so the change leads.
func RenderMovers(w io.Writer, movers []MoverRow, by string) error {
	tw := prettytable.NewWriter()
	tw.SetOutputMirror(w)
	tw.AppendHeader(prettytable.Row{strings.ToUpper(by), "Δ COST $", "COST $ NOW", "Δ AI LINES", "AI LINES NOW"})
	tw.SetColumnConfigs(rightAlignFrom(1, 4))
	var now, prior Unpriced
	for i := range movers {
		m := &movers[i]
		label := m.Group
		if label == "" {
			label = emptyDimLabel(by)
		}
		mark := ""
		if m.hasUnpriced() {
			mark = "*"
			now.Rows++
		}
		now.Tokens, now.Total = now.Tokens+m.UnpricedNow, now.Total+m.TokensNow
		prior.Tokens, prior.Total = prior.Tokens+m.UnpricedPrior, prior.Total+m.TokensPrior
		tw.AppendRow(prettytable.Row{
			label, signedFloat(m.DeltaCost) + mark, humanize.USDCell(m.CostNow) + mark,
			signedInt(m.DeltaLines), m.LinesNow,
		})
	}
	tw.Render()
	if note := UnpricedDisclosure(worseWindow(&now, &prior)); note != "" {
		if _, err := fmt.Fprintln(w, note); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w, CostEstimateDisclosure)
	return err
}

// worseWindow picks the window whose cost is most blind, and names it. A comparison of two
// windows has two error bars and one footnote; reporting the smaller would tell the reader
// the marked figures are more complete than one of them is.
func worseWindow(now, prior *Unpriced) (*Unpriced, string) {
	if prior.Share() > now.Share() {
		return prior, "the prior window's tokens"
	}
	return now, "the recent window's tokens"
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
	s := humanize.USDCell(f)
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
