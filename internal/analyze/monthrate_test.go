package analyze

import (
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

func spanInput(rows []store.UsageRow, start, now string) *Input {
	in := &Input{Usage: rows}
	if start != "" {
		in.WindowStart, _ = time.Parse("2006-01-02", start)
	}
	in.Now, _ = time.Parse("2006-01-02", now)
	return in
}

// TestMonthlyRateSpansTheWindow is B142's case: five active days inside a seven-day window
// must project as that week repeated, not as six working weeks. Dividing by active days
// turned $50 into $300/mo where the honest figure is about $214 -- enough to flip a $250 plan
// from "not paying off" to "paying off", in the direction that flatters the plan.
func TestMonthlyRateSpansTheWindow(t *testing.T) {
	workWeek := []store.UsageRow{
		{Day: "2026-03-02"},
		{Day: "2026-03-03"},
		{Day: "2026-03-04"},
		{Day: "2026-03-05"},
		{Day: "2026-03-06"},
	}
	got := MonthlyRate(50, spanInput(workWeek, "2026-03-02", "2026-03-08"))
	if got < 214 || got > 215 {
		t.Errorf("MonthlyRate = %.2f, want ~214.29 ($50 over a 7-day window x 30); the active-day reading gives 300", got)
	}
}

// TestMonthlyRateClampsToFirstUsage: a window longer than the evidence divides by the
// evidence. Somebody three days into using assaio who asks for ninety days has three days of
// history, and spreading it over ninety understates the rate as badly as active days
// overstate it.
func TestMonthlyRateClampsToFirstUsage(t *testing.T) {
	rows := []store.UsageRow{{Day: "2026-03-06"}, {Day: "2026-03-07"}, {Day: "2026-03-08"}}
	got := MonthlyRate(50, spanInput(rows, "2025-12-08", "2026-03-08"))
	if got < 499 || got > 501 {
		t.Errorf("MonthlyRate = %.2f, want ~500 ($50 over the 3 days that exist x 30)", got)
	}
}

func TestMonthlyRateEdges(t *testing.T) {
	cases := []struct {
		name string
		in   *Input
		want float64
	}{
		{
			"one day is one day of evidence",
			spanInput([]store.UsageRow{{Day: "2026-03-02"}}, "2026-03-02", "2026-03-02"), 1500,
		},
		{"no usage projects nothing", spanInput(nil, "2026-03-02", "2026-03-08"), 50},
		{
			"an unreadable day is skipped, not read as the epoch",
			spanInput([]store.UsageRow{{Day: "not-a-date"}, {Day: "2026-03-02"}}, "", "2026-03-02"), 1500,
		},
		{
			"no window start falls back to the usage itself",
			spanInput([]store.UsageRow{{Day: "2026-03-02"}}, "", "2026-03-31"), 50,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := MonthlyRate(50, c.in); got < c.want-0.5 || got > c.want+0.5 {
				t.Errorf("MonthlyRate = %.2f, want %.2f", got, c.want)
			}
		})
	}
}
