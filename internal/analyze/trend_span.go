package analyze

import (
	"time"
)

// span is an inclusive range of UTC day buckets, "YYYY-MM-DD" at both ends, so a row's Day
// compares against it as a string.
type span struct{ from, to string }

func (s span) holds(day string) bool { return day >= s.from && day <= s.to }

// trendSpans are the two equal, adjacent spans every week-over-week figure compares: recent
// is the `window` complete days ending yesterday, prior the same number of days before it.
// Today belongs to neither because it is not over: a partial day set against a whole one reads
// as a fall of up to a seventh that nothing in the work caused.
func trendSpans(now time.Time, window time.Duration) (recent, prior span) {
	days := int(window.Hours() / 24)
	if days < 1 {
		days = 1
	}
	today := day(now)
	back := func(n int) string { return today.AddDate(0, 0, -n).Format(time.DateOnly) }
	return span{from: back(days), to: back(1)}, span{from: back(2 * days), to: back(days + 1)}
}

// label dates a span the way a reader says it: "Sep 5–11", or "Aug 29–Sep 4" across a month.
// It carries no year and no zone; the spans are UTC days, which the surfaces state once.
func (s span) label() string {
	from, errFrom := time.Parse(time.DateOnly, s.from)
	to, errTo := time.Parse(time.DateOnly, s.to)
	if errFrom != nil || errTo != nil {
		return s.from + "–" + s.to
	}
	if from.Month() == to.Month() {
		return from.Format("Jan 2") + "–" + to.Format("2")
	}
	return from.Format("Jan 2") + "–" + to.Format("Jan 2")
}

// endsAt is the midnight after the span's last day: the moment the span is over.
func (s span) endsAt() time.Time {
	t, err := time.Parse(time.DateOnly, s.to)
	if err != nil {
		return time.Time{}
	}
	return t.AddDate(0, 0, 1)
}

// firstDay is the span's first day the way label prints it, "Sep 5".
func (s span) firstDay() string { return s.startsAt().Format("Jan 2") }

// startsAt is the midnight the span's first day begins at.
func (s span) startsAt() time.Time {
	t, err := time.Parse(time.DateOnly, s.from)
	if err != nil {
		return time.Time{}
	}
	return t
}
