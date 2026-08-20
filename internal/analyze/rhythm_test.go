package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// rhythmSessionsAt builds n sessions starting on consecutive weekdays at the given local
// hour, each with the given focused-work minutes. Times are constructed in time.Local
// because the validator reads hours in the machine's own zone, so a fixed-offset literal
// would classify differently depending on where the test runs.
func rhythmSessionsAt(n, hour int, activeMinutes float64) []store.SessionRow {
	sessions := make([]store.SessionRow, 0, n)
	for i := range n {
		day := 13 + i%5 // 2026-07-13 is a Monday, so 13..17 are weekdays
		start := time.Date(2026, 7, day, hour, 0, 0, 0, time.Local)
		sessions = append(sessions, store.SessionRow{
			SessionID: "s" + string(rune('a'+i)), Project: "web", Tool: "claude-code",
			FirstTs: start, LastTs: start.Add(time.Hour), Turns: 4, ActiveMinutes: activeMinutes,
		})
	}
	return sessions
}

func rhythmInput(sessions []store.SessionRow) Input {
	return BuildInput(nil, sessions, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
}

// TestRhythmNeverJudgesWhenTheWorkHappened is B178's regression: on a local store every
// session is one person's, so mid-morning weekday work and midnight work get the same read.
// The shares are reported; the judgement is refused.
func TestRhythmNeverJudgesWhenTheWorkHappened(t *testing.T) {
	for _, tc := range []struct {
		name    string
		hour    int
		minutes float64
		share   string
	}{
		{"weekday mid-morning", 10, 30, "0%"},
		{"every session at 23:00", 23, 30, "100%"},
		{"every session a marathon", 10, rhythmMarathonMinutes + 30, "0%"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := mustGet(t, rhythmName).Analyze(rhythmInput(rhythmSessionsAt(6, tc.hour, tc.minutes)))

			if got.Read != reportedRead {
				t.Fatalf("Read = %+v, want the descriptive read: a verdict here judges one person's hours", got.Read)
			}
			if got.Purity != neutralPurity {
				t.Fatalf("Purity = %v, want the neutral gauge", got.Purity)
			}
			if !strings.Contains(figureValues(got.Figures), tc.share) {
				t.Fatalf("Figures = %q, want a %s off-hours share", figureValues(got.Figures), tc.share)
			}
		})
	}
}

// TestRhythmPrintsItsOwnDayBand is the other half of B178: the boundary a reader is measured
// against was never on the surface, so nobody could disagree with it.
func TestRhythmPrintsItsOwnDayBand(t *testing.T) {
	got := mustGet(t, rhythmName).Analyze(rhythmInput(rhythmSessionsAt(6, 10, 30)))

	rendered := figureValues(got.Figures) + " " + strings.Join(got.Caveats, " ") + " " + got.Takeaway
	if !strings.Contains(rendered, "8:00-18:00") {
		t.Fatalf("rendered result never states the ordinary-hours band it counts against:\n%s", rendered)
	}
}

// TestRhythmTooFewSessionsIsNotConfidentlySteady asserts an ordinary-looking but thin
// window does not earn the favorable read off two data points.
func TestRhythmTooFewSessionsIsNotConfidentlySteady(t *testing.T) {
	got := mustGet(t, rhythmName).Analyze(rhythmInput(rhythmSessionsAt(2, 10, 20)))

	if got.Read.Label == "STEADY" {
		t.Fatalf("Read = %q, want no confident steady read from %d sessions", got.Read.Label, 2)
	}
	if !strings.Contains(got.Takeaway, "Too few sessions") {
		t.Fatalf("Takeaway = %q, want the thin-sample explanation", got.Takeaway)
	}
}

// TestRhythmThinWindowMakesNoWorkloadJudgement is the honesty guard on the pair: below the
// session floor the Read withholds its verdict, so the takeaway printed beside it must not
// assert the off-hours finding the Read just refused to make about an individual.
func TestRhythmThinWindowMakesNoWorkloadJudgement(t *testing.T) {
	for _, tt := range []struct {
		name  string
		hour  int
		mins  float64
		count int
	}{
		{"all off-hours", 23, 20, 2},
		{"all marathons", 10, rhythmMarathonMinutes + 30, 2},
		{"off-hours marathons", 23, rhythmMarathonMinutes + 30, rhythmMinSessions - 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := mustGet(t, rhythmName).Analyze(rhythmInput(rhythmSessionsAt(tt.count, tt.hour, tt.mins)))

			if got.Read != noDataRead {
				t.Fatalf("Read = %+v, want the neutral read below the %d-session floor", got.Read, rhythmMinSessions)
			}
			if !strings.Contains(got.Takeaway, "Too few sessions") {
				t.Fatalf("Takeaway = %q, want the thin-sample wording, not a verdict the Read withheld", got.Takeaway)
			}
		})
	}
}

// TestRhythmIgnoresUntimedSessions asserts sessions carrying no start timestamp are
// dropped rather than silently counted as midnight, which would invent off-hours work.
func TestRhythmIgnoresUntimedSessions(t *testing.T) {
	sessions := append(
		rhythmSessionsAt(5, 10, 20),
		store.SessionRow{SessionID: "untimed", Project: "web", Tool: "claude-code", Turns: 2},
	)
	got := mustGet(t, rhythmName).Analyze(rhythmInput(sessions))

	if !strings.Contains(figureValues(got.Figures), "5") {
		t.Fatalf("Figures = %q, want only the 5 timed sessions counted", figureValues(got.Figures))
	}
	if !strings.Contains(figureValues(got.Figures), "0%") {
		t.Fatalf("Figures = %q, want the untimed session not to invent night work", figureValues(got.Figures))
	}
}

// TestRhythmBandsCoverTheWholeDay asserts every time-of-day band renders in chronological
// order, so the bars read as a day rather than as a ranking that hides empty parts of it.
func TestRhythmBandsCoverTheWholeDay(t *testing.T) {
	got := mustGet(t, rhythmName).Analyze(rhythmInput(rhythmSessionsAt(6, 10, 20)))

	if len(got.Bars) != len(rhythmBandOrder) {
		t.Fatalf("Bars = %d, want one per band (%d)", len(got.Bars), len(rhythmBandOrder))
	}
	for i, want := range rhythmBandOrder {
		if got.Bars[i].Label != want {
			t.Fatalf("Bars[%d].Label = %q, want %q -- bands must stay chronological", i, got.Bars[i].Label, want)
		}
	}
	if got.BarsPseudonym != "" {
		t.Fatal("BarsPseudonym must be empty: these bars label time bands, not names a person chose")
	}
}

// TestRhythmEmptyInputSafe asserts the zero-value Input renders the honest no-data block.
func TestRhythmEmptyInputSafe(t *testing.T) {
	got := mustGet(t, rhythmName).Analyze(Input{})
	if got.Read != noDataRead || got.Takeaway != "No usage in this window." {
		t.Fatalf("empty Input = %+v, want the no-data read and takeaway", got)
	}
}
