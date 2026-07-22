package dashboard

import (
	"strconv"

	"github.com/assaio/assaio/internal/report"
)

// costBasis renders the footnote's "$31.5K / last 30 days · $750 per active day" line:
// the report's cost denominator, honestly dashed when cost or active days are unknown --
// never a fabricated ratio.
func costBasis(inv report.Inventory, window string) string {
	total := "—"
	if inv.TotalCost != nil {
		total = formatCompactUSD(*inv.TotalCost)
		if inv.HasUnpriced {
			total += "*"
		}
	}
	perDay := "—"
	if inv.TotalCost != nil && inv.Days > 0 {
		perDay = formatCompactUSD(*inv.TotalCost / float64(inv.Days))
	}
	return total + " / " + window + " · " + perDay + " per active day"
}

// formatCompactUSD renders a USD amount compactly for the assay footnote, e.g.
// 31500 -> "$31.5K", 750 -> "$750".
func formatCompactUSD(v float64) string {
	switch {
	case v >= 1_000_000:
		return "$" + strconv.FormatFloat(v/1_000_000, 'f', 1, 64) + "M"
	case v >= 1_000:
		return "$" + strconv.FormatFloat(v/1_000, 'f', 1, 64) + "K"
	default:
		return "$" + strconv.FormatFloat(v, 'f', 0, 64)
	}
}
