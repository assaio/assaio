package cli

import (
	"fmt"
	"time"

	"github.com/assaio/assaio/internal/i18n"
)

// agoLabel renders an elapsed duration at the coarsest useful unit. Shared with doctor's
// per-source freshness line so both read the same way.
func agoLabel(since time.Duration, l *i18n.CLI) string {
	switch {
	case since < time.Minute:
		return l.AgeJustNow
	case since < time.Hour:
		return fmt.Sprintf(l.AgeMinutes, int(since.Minutes()))
	case since < 24*time.Hour:
		return fmt.Sprintf(l.AgeHours, int(since.Hours()))
	default:
		return fmt.Sprintf(l.AgeDays, int(since.Hours()/24))
	}
}
