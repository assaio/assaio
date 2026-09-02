package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/humanize"
)

// unpricedSection prints how much of this store the price table cannot cost, and returns the
// sentence --strict fails on when that share passes maxShare.
//
// "N models, snapshot <date>" is a fact about the table, never about the reader's own data:
// a table that has fallen behind the models in use is indistinguishable from a complete one
// from the inside, which is how five weeks of drift left a window's estimate 45.5% short with
// no surface saying so. This is the line that says it, and --strict is what makes a cron job
// notice before a human eventually does.
func unpricedSection(cmd *cobra.Command, c *storeContents, maxShare float64) string {
	u := &c.Unpriced
	switch {
	case u.Missing():
	case u.Rows > 0 && u.Rows == u.Untokened:
		// A source that publishes no token counter has nothing to price, and no refreshed
		// price table will change that. Reporting it as a missing price would send a reader
		// after a fix that does not exist.
		cmd.Printf("unpriced:     not priceable — %d row(s) in the last %s come from a source that publishes no token counter\n",
			u.Rows, c.Window)
		return ""
	case u.Rows > 0:
		// The middle case report and check both print: rows on a model with no price that
		// carry no token. Collapsing it into "everything is priced" would have this line
		// contradict the "*" the same store's tables show.
		cmd.Printf("unpriced:     no tokens — %d row(s) in the last %s are on a model with no price but carry none\n",
			u.Rows, c.Window)
		return ""
	default:
		cmd.Printf("unpriced:     none — every model used in the last %s has a price\n", c.Window)
		return ""
	}
	share := humanize.PercentAt(u.Share(), 1)
	cmd.Printf("unpriced:     %s of the last %s (%s of %s tokens) on %s\n",
		share, c.Window, humanize.Count(u.Tokens), humanize.Count(u.Total), modelsPhrase(c.Models))
	cmd.Println("              Cost excludes them entirely. Upgrade assaio for a refreshed price table.")
	if u.Untokened > 0 {
		// The two reasons coexist, and only the first switch case above used to be reachable when
		// they did: a store holding both an unlisted model and a counter-less source printed the
		// upgrade line alone, which is a fix for part of the gap presented as the fix for all of it.
		cmd.Printf("              %d of the unpriced row(s) publish no token counter at all, which no refresh changes.\n", u.Untokened)
	}
	if maxShare <= 0 || u.Share() <= maxShare {
		return ""
	}
	return fmt.Sprintf("pricing: %s of the last %s carries no price, over the %s ceiling (pricing.max_unpriced_share)",
		share, c.Window, humanize.PercentAt(maxShare, 1))
}

// modelsPhrase names the models a refresh has to cover, capped so a store that lost a whole
// vendor does not print a hundred names into a diagnostic line.
func modelsPhrase(models []string) string {
	const shown = 3
	noun := fmt.Sprintf("%d models with no price", len(models))
	if len(models) == 1 {
		noun = "1 model with no price"
	}
	if len(models) <= shown {
		return noun + ": " + strings.Join(models, ", ")
	}
	return fmt.Sprintf("%s: %s and %d more", noun, strings.Join(models[:shown], ", "), len(models)-shown)
}
