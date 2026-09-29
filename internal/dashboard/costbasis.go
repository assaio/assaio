package dashboard

import (
	"strconv"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/report"
)

// costBasis renders the footnote's "$31.5K / last 30 days · $750 per active day" line:
// the report's cost denominator, honestly dashed when cost or active days are unknown --
// never a fabricated ratio. A day on which only a source with no token counter ran could
// carry no cost, so the per-day figure divides by the other days and names how many it skipped.
func costBasis(inv *report.Inventory, window string) string {
	total := "—"
	if inv.TotalCost != nil {
		total = humanize.USDCompact(*inv.TotalCost)
		if inv.HasUnpriced {
			total += "*"
		}
	}
	if inv.TotalCost == nil || inv.TokenDays == 0 {
		return total + " / " + window + " · — per active day"
	}
	perDay := humanize.USDCompact(*inv.TotalCost / float64(inv.TokenDays))
	if inv.HasUnpriced {
		perDay += "*"
	}
	line := total + " / " + window + " · " + perDay + " per active day"
	if skipped := inv.Days - inv.TokenDays; skipped > 0 {
		line += " (" + strconv.Itoa(skipped) + " day(s) with no token counter left out)"
	}
	return line
}
