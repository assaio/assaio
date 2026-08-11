package dashboard

import (
	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/report"
)

// costBasis renders the footnote's "$31.5K / last 30 days · $750 per active day" line:
// the report's cost denominator, honestly dashed when cost or active days are unknown --
// never a fabricated ratio.
func costBasis(inv *report.Inventory, window string) string {
	total := "—"
	if inv.TotalCost != nil {
		total = humanize.USDCompact(*inv.TotalCost)
		if inv.HasUnpriced {
			total += "*"
		}
	}
	perDay := "—"
	if inv.TotalCost != nil && inv.Days > 0 {
		perDay = humanize.USDCompact(*inv.TotalCost / float64(inv.Days))
	}
	return total + " / " + window + " · " + perDay + " per active day"
}
