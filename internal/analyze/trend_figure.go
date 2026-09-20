package analyze

import (
	"strconv"
	"strings"

	"github.com/assaio/assaio/internal/humanize"
)

// figure publishes the trend under q's label. Its Note is written to be quoted whole -- the
// shared card quotes it rather than re-deriving anything (ADR 0014) -- so it carries the dates,
// both sums where they mean something, and the reason when no direction is stated.
func (t *trend) figure(q trendQty) Figure {
	f := Figure{Label: q.label, Value: "—", Note: t.note(q)}
	if t.readable() {
		f.Value = changeLabel(t.now, t.was)
	}
	return f
}

func (t *trend) note(q trendQty) string {
	parts := []string{t.prior.label() + " → " + t.recent.label()}
	switch t.gap {
	case trendRead, gapZeroBase, gapThin:
		parts = append(parts, q.count(t.was)+" → "+q.count(t.now))
	}
	if r := t.reason(); r != "" {
		parts = append(parts, r)
	}
	if t.excluded > 0 && t.gap != gapNoSource {
		parts = append(parts, t.leftOut(q))
	}
	return strings.Join(parts, " · ")
}

// reason names the gap in the words every surface prints, "" when a direction is stated.
func (t *trend) reason() string {
	switch t.gap {
	case gapWindow:
		return "window starts after " + t.prior.firstDay()
	case gapHistory:
		return "store history begins " + t.historyFrom.Format("Jan 2") + ", inside these weeks"
	case gapNoSource:
		return "no source has a row by " + t.prior.firstDay() + " in this window"
	case gapZeroBase:
		return "earlier-week volume is zero; no percentage exists"
	case gapThin:
		return "volume is too small to state a direction"
	default:
		return ""
	}
}

// leftOut states what the sums leave out: how many sources, and what share of the two spans'
// volume they held. The share is the part a reader cannot do without -- a count reads the same
// whether the sources held a sliver or most of the machine.
func (t *trend) leftOut(q trendQty) string {
	sources := strconv.Itoa(t.excluded) + " sources"
	if t.excluded == 1 {
		sources = "1 source"
	}
	return sources + ": no row by " + t.prior.firstDay() + " in window; omitted (" + humanize.Percent(t.leftShare()) + " of both weeks' " + q.unit + ")"
}

// changeLabel renders a direction without rounding it to nothing: equal sums print "0%", any
// other change its sign and humanize.Percent of the magnitude, so a fall of 0.4% reads "-<1%".
// A zero base has no change to render, whatever the caller checked first.
func changeLabel(now, was int64) string {
	switch {
	case was == 0:
		return "—"
	case now == was:
		return "0%"
	}
	change := float64(now-was) / float64(was)
	if change > 0 {
		return "+" + humanize.Percent(change)
	}
	return "-" + humanize.Percent(-change)
}
