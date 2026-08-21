package digest

import (
	"fmt"
	"strings"
	"time"
)

// Whether two runs may be compared at all is a different question from what moved between
// them, and it is the one that decides whether the rest of the digest means anything. It lives
// apart from the diff because it never touches a Mover: it reads the two snapshots' windows,
// their parsing builds and their coverage, and answers before any figure is subtracted.

// comparabilityCaveats names every reason this comparison is weaker than it looks. Each one
// is a fact about the two runs, not a guess about the data.
func comparabilityCaveats(now, prev *Snapshot) []string {
	var out []string
	switch {
	case prev.ParsedBy == "" || now.ParsedBy == "":
		// Unknown is not the same as unchanged: silence here would let a comparison across a
		// parser change read as one that was checked and cleared.
		out = append(out, "One of these runs did not record which build read the data, so whether a "+
			"parser change lies between them could not be checked.")
	case now.ParsedBy != prev.ParsedBy:
		out = append(out, fmt.Sprintf(
			"The reader changed between these runs (%s → %s). A parser fix corrects history, "+
				"so a movement below may be the correction rather than a change in how the tools were used.",
			prev.ParsedBy, now.ParsedBy,
		))
	}
	if note := unpricedCaveat(now, prev); note != "" {
		out = append(out, note)
	}
	if note := staleBasisCaveat(now, prev); note != "" {
		out = append(out, note)
	}
	if now.Window != prev.Window {
		out = append(out, fmt.Sprintf(
			"The windows differ (%s → %s), so these totals cover spans of different lengths and "+
				"their difference is not a rate of change.", prev.Window, now.Window,
		))
	}
	if overlap := windowOverlap(now, prev); overlap > 0 {
		out = append(out, fmt.Sprintf(
			"The two windows overlap by about %s: the last run was more recent than the window is long, "+
				"so part of what moved is the same data counted twice.", roundDuration(overlap),
		))
	}
	return out
}

// unpricedCaveat states the blinder of the two windows, because a cost movement between a
// complete window and a partial one is partly a change in what could be seen rather than in
// what was spent -- the line report.RenderMovers already holds for the same comparison.
// unpricedCaveat fires on the share of tokens that carry no price, never on Priced alone: a
// row with no price and no tokens leaves the total complete, and its disclosure says exactly
// that -- so treating the two as one produced a caveat that contradicted its own quote. The
// blinder window is named, and its own sentence is quoted rather than composed around.
func unpricedCaveat(now, prev *Snapshot) string {
	blind, whose := now, "This window"
	if prev.UnpricedShare > now.UnpricedShare {
		blind, whose = prev, "The previous window"
	}
	if blind.UnpricedShare <= 0 || blind.UnpricedNote == "" {
		return ""
	}
	return whose + ": " + trimLeadingMarker(blind.UnpricedNote) +
		". The cost movement above is therefore partly a change in what could be priced."
}

// trimLeadingMarker drops the footnote marker the cost tables prefix their disclosure with;
// the digest has no asterisked column for it to refer to.
func trimLeadingMarker(note string) string {
	return strings.TrimPrefix(strings.TrimPrefix(note, "*"), " ")
}

// staleBasisCaveat fires when the last run is far enough back that "since last time" spans a
// gap rather than a cadence. The windows are still each other's length, so the comparison is
// arithmetically fine -- what it stops being is a report on a continuous period.
func staleBasisCaveat(now, prev *Snapshot) string {
	span, ok := windowSpan(now.Window)
	if !ok || now.Window != prev.Window {
		return ""
	}
	if gap := now.TakenAt.Sub(prev.TakenAt); gap > 2*span {
		return fmt.Sprintf(
			"The last digest was %s ago, more than twice this window's length: the period between them "+
				"is not covered by either window, so this is a comparison of two separate weeks rather "+
				"than a report on what happened since.", roundDuration(gap),
		)
	}
	return ""
}

// windowOverlap is how much of the previous window this one still contains. A weekly digest
// over a 7d window run every 7 days overlaps by nothing, which is the intended cadence.
func windowOverlap(now, prev *Snapshot) time.Duration {
	span, ok := windowSpan(now.Window)
	if !ok || now.Window != prev.Window {
		return 0
	}
	if gap := now.TakenAt.Sub(prev.TakenAt); gap < span {
		return span - gap
	}
	return 0
}

// windowSpan reads the day-count spellings assaio's --since accepts. Anything else returns
// false, and the overlap simply goes unstated rather than guessed.
func windowSpan(window string) (time.Duration, bool) {
	var days int
	if n, err := fmt.Sscanf(window, "%dd", &days); n != 1 || err != nil || days <= 0 {
		return 0, false
	}
	return time.Duration(days) * 24 * time.Hour, true
}

func roundDuration(d time.Duration) string {
	if d >= 24*time.Hour {
		return fmt.Sprintf("%.0f day(s)", d.Hours()/24)
	}
	return fmt.Sprintf("%.0f hour(s)", d.Hours())
}
