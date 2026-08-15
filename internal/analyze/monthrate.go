package analyze

import (
	"time"

	"github.com/assaio/assaio/internal/store"
)

// monthDays is the calendar month a run-rate is projected onto.
const monthDays = 30

// MonthlyRate projects a window's figure onto a month, dividing by the window's span in
// *real* days rather than by the days that carried usage.
//
// The difference is not cosmetic. Somebody who codes five days a week has five active days
// in every seven, so dividing by active days and multiplying by 30 prices six working weeks
// as a month: $50 a week becomes $300/mo where repeating that week for a calendar month is
// about $214. A $250 plan flips from "not paying off" to "paying off" on arithmetic alone,
// in the direction that flatters the plan -- which is the last direction this project should
// round in. A flat plan is billed per calendar month, so the comparison has to be one too.
//
// The span starts at the window's own start or at the first day carrying usage, whichever is
// later: somebody three days into using assaio has three days of evidence, and stretching it
// across a requested 90-day window would understate the rate as badly as active days
// overstate it.
func MonthlyRate(value float64, in *Input) float64 {
	return value / projectionSpan(in) * monthDays
}

// ProjectionSpan is the number of days a per-month projection divides by, so a surface can state
// it rather than leave a reader unable to tell a 30-day projection from a 5-day one.
func ProjectionSpan(in *Input) float64 { return projectionSpan(in) }

func projectionSpan(in *Input) float64 {
	days := spanDays(in.Usage, in.WindowStart, in.Now)
	if days <= 0 {
		return monthDays
	}
	return days
}

// spanDays is what the projection divides by: the window's own length when the window opens
// after the evidence does, and the evidence's day-bucket span when it does not.
//
// The two are different quantities on purpose. A window is a duration and is exact at any time
// of day; counting the day-buckets it touches instead credited a partial oldest bucket and gave
// a 168-hour window a divisor of 8. Usage, by contrast, is stored per day, so three days into
// using assaio there are three buckets of evidence and no finer answer exists.
func spanDays(rows []store.UsageRow, windowStart, now time.Time) float64 {
	first, ok := firstUsageDay(rows)
	if !ok {
		return 0
	}
	if !windowStart.IsZero() && !windowStart.Before(first) {
		if span := now.Sub(windowStart).Hours() / 24; span >= 1 {
			return span
		}
		return 1
	}
	end := day(now)
	if end.Before(first) {
		return 1
	}
	return float64(int(end.Sub(first)/(24*time.Hour)) + 1)
}

// firstUsageDay is the earliest day bucket carrying usage. A row whose day cannot be read is
// skipped rather than treated as the epoch, which would stretch the span across fifty years
// and report a monthly rate of nearly zero.
func firstUsageDay(rows []store.UsageRow) (time.Time, bool) {
	var first time.Time
	ok := false
	for i := range rows {
		d, err := time.Parse("2006-01-02", rows[i].Day)
		if err != nil {
			continue
		}
		if !ok || d.Before(first) {
			first, ok = d, true
		}
	}
	return first, ok
}

func day(t time.Time) time.Time {
	if t.IsZero() {
		return time.Time{}
	}
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}
