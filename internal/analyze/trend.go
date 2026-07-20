package analyze

import (
	"strconv"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// weekOverWeekLabel is the shared Figure label for the recent-vs-prior AI-lines trend --
// used identically by adoption and throughput so the same signal reads the same way in
// both reports.
const weekOverWeekLabel = "week-over-week AI lines"

// recentCutoff is the "YYYY-MM-DD" day string marking the start of the window ending at
// now: a row's Day >= recentCutoff falls inside it. A window of N days spans exactly N
// day-buckets ending today (today-(N-1) .. today), so the cutoff is today-(N-1), not
// today-N -- the latter admits N+1 buckets and made the recent window one day wider than
// the prior one it is compared against. Rows compare Day as a string, so no per-row time
// parsing is needed.
func recentCutoff(now time.Time, window time.Duration) string {
	days := int(window.Hours() / 24)
	if days < 1 {
		days = 1
	}
	return now.UTC().AddDate(0, 0, -(days - 1)).Format("2006-01-02")
}

// weekOverWeek splits rows into two equal, adjacent windows ending at now -- recent is
// [now-window, now) and prior is [now-2*window, now-window) -- and sums LinesAdded in
// each. changePct is recent/prior - 1; ok is false when prior has zero lines, so a trend
// is never computed against a zero base.
func weekOverWeek(rows []store.UsageRow, now time.Time, window time.Duration) (recent, prior int64, changePct float64, ok bool) {
	recentFrom := recentCutoff(now, window)
	priorFrom := recentCutoff(now, 2*window)
	for i := range rows {
		switch day := rows[i].Day; {
		case day >= recentFrom:
			recent += rows[i].LinesAdded
		case day >= priorFrom:
			prior += rows[i].LinesAdded
		}
	}
	if prior == 0 {
		return recent, prior, 0, false
	}
	return recent, prior, float64(recent-prior) / float64(prior), true
}

// recentRows returns the rows on or after now-window -- the same recent sub-window
// weekOverWeek sums -- for a validator that needs the recent rows themselves, not just
// their LinesAdded total.
func recentRows(rows []store.UsageRow, now time.Time, window time.Duration) []store.UsageRow {
	cutoff := recentCutoff(now, window)
	out := make([]store.UsageRow, 0, len(rows))
	for i := range rows {
		if rows[i].Day >= cutoff {
			out = append(out, rows[i])
		}
	}
	return out
}

// trendLabel renders a week-over-week change as "+35%", "-12%", or "—" when undefined.
func trendLabel(changePct float64, ok bool) string {
	if !ok {
		return "—"
	}
	sign := "+"
	if changePct < 0 {
		sign = ""
	}
	return sign + strconv.FormatFloat(changePct*100, 'f', 0, 64) + "%"
}

// trendFigure builds the shared week-over-week AI-lines Figure that adoption and
// throughput both render, so the same trend reads identically in each report.
func trendFigure(recent, prior int64, changePct float64, ok bool) Figure {
	return Figure{
		Label: weekOverWeekLabel,
		Value: trendLabel(changePct, ok),
		Note:  strconv.FormatInt(recent, 10) + " recent vs " + strconv.FormatInt(prior, 10) + " prior",
	}
}

// trendPurity scores a week-over-week change 0..1 for the dashboard gauge: 0.5 (neutral)
// when the trend is unknown, saturating toward 1 at +100% or more and 0 at -100% or less.
func trendPurity(changePct float64, ok bool) float64 {
	if !ok {
		return 0.5
	}
	return clamp01(0.5 + changePct/2)
}
