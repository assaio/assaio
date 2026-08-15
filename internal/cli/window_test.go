package cli

import (
	"strconv"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/store"
)

// windowClocks are times of day a command can be run at. The defect this file holds shut was
// invisible at midnight and grew with the clock, so every case runs at all of them.
var windowClocks = []time.Time{
	time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC),
	time.Date(2026, 8, 15, 0, 0, 1, 0, time.UTC),
	time.Date(2026, 8, 15, 12, 30, 0, 0, time.UTC),
	time.Date(2026, 8, 15, 18, 0, 0, 0, time.UTC),
	time.Date(2026, 8, 15, 23, 59, 59, 0, time.UTC),
}

var windowSizes = []int{1, 7, 14, 30, 90, 365}

// TestWindowsAreContiguousBetweenRuns is what a recurring command needs and a bucket-aligned
// window cannot give: `sync --since 1d` on a daily cron must cover the time since the previous
// run, or the records in between reach the team server from no run at all.
func TestWindowsAreContiguousBetweenRuns(t *testing.T) {
	for _, now := range windowClocks {
		for _, days := range windowSizes {
			previous := now.AddDate(0, 0, -days)
			start, err := parseSinceAt(windowSpec(days), now)
			if err != nil {
				t.Fatal(err)
			}
			if start.After(previous) {
				t.Errorf("%dd at %s starts %v after the previous run's window ended, leaving a gap",
					days, now.Format(time.TimeOnly), start.Sub(previous))
			}
		}
	}
}

// TestMonthlyRateDividesByTheWindowItWasGiven is B128 at the far end: whatever parseSinceAt
// returns must be what a per-month projection divides by. A window opened at the current time
// of day covers part of its oldest day-bucket, and counting that bucket whole divided a
// 168-hour window by 8. Asserted through the real window parser rather than a hand-written
// start, because the divisor was right in analyze's own tests and wrong in production on
// exactly that difference.
func TestMonthlyRateDividesByTheWindowItWasGiven(t *testing.T) {
	for _, now := range windowClocks {
		for _, days := range windowSizes {
			start, err := parseSinceAt(windowSpec(days), now)
			if err != nil {
				t.Fatal(err)
			}
			in := &analyze.Input{WindowStart: start, Now: now, Usage: everyDayIn(start, now)}
			// A window carrying its own day count projects to exactly 30 a month.
			if got := analyze.MonthlyRate(float64(days), in); got < 29.99 || got > 30.01 {
				t.Errorf("%dd at %s: MonthlyRate = %.4f, want 30 (divided by %.2f days, not %d)",
					days, now.Format(time.TimeOnly), got, float64(days)*30/got, days)
			}
		}
	}
}

// TestZeroDayWindowIsNotInTheFuture: `0d` is a legal window (config.sincePattern documents it),
// and a start after now inverts every range built on it -- reconcile's scope puts the whole
// export outside the window and reports zero overlapping days.
func TestZeroDayWindowIsNotInTheFuture(t *testing.T) {
	for _, now := range windowClocks {
		start, err := parseSinceAt("0d", now)
		if err != nil {
			t.Fatal(err)
		}
		if start.After(now) {
			t.Errorf("0d at %s starts at %s, which is in the future", now.Format(time.TimeOnly), start)
		}
	}
}

func windowSpec(days int) string { return strconv.Itoa(days) + "d" }

// everyDayIn is one usage row per day bucket in the window, so the projection's span is the
// window's rather than the evidence's.
func everyDayIn(start, now time.Time) []store.UsageRow {
	var rows []store.UsageRow
	for d := startOfUTCDay(start); !d.After(startOfUTCDay(now)); d = d.AddDate(0, 0, 1) {
		rows = append(rows, store.UsageRow{Day: d.Format("2006-01-02")})
	}
	return rows
}
