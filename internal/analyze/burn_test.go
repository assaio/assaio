package analyze

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// burnInput seeds one usage row per day, each carrying the given token count, so the
// window's daily burn is exactly the slice passed in.
func burnInput(dailyTokens []int64) Input {
	usage := make([]store.UsageRow, 0, len(dailyTokens))
	for i, tokens := range dailyTokens {
		usage = append(usage, store.UsageRow{
			Day: fmt.Sprintf("2026-07-%02d", i+1), Tool: "claude-code",
			Model: "claude-sonnet-4-5", Project: "web", In: tokens,
		})
	}
	return BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
}

// flatDays returns n days all burning the same amount -- the baseline a spike stands out
// from, and the case that drives the median absolute deviation to zero.
func flatDays(n int, tokens int64) []int64 {
	out := make([]int64, n)
	for i := range out {
		out[i] = tokens
	}
	return out
}

// TestBurnDetectsSpikeAgainstFlatBaseline is the metric's core case and a regression
// guard: when more than half the days share one value the median absolute deviation is
// zero, and a median-only implementation silently reports no spike at all -- exactly when
// the one runaway day matters most.
func TestBurnDetectsSpikeAgainstFlatBaseline(t *testing.T) {
	got := mustGet(t, burnName).Analyze(burnInput(append(flatDays(7, 1000), 100_000)))

	if got.Read.Label != "WATCH" {
		t.Fatalf("Read = %q, want WATCH for a 100× day against a flat baseline", got.Read.Label)
	}
	if len(got.Bars) != 1 || !strings.Contains(got.Bars[0].Value, "100.0×") {
		t.Fatalf("Bars = %+v, want the single spike day at 100× the typical day", got.Bars)
	}
	if !strings.Contains(figureValues(got.Figures), "1") {
		t.Fatalf("Figures = %q, want one spike day counted", figureValues(got.Figures))
	}
}

// TestBurnSteadyWindowHasNoSpikes asserts ordinary day-to-day variation is not reported as
// an anomaly.
func TestBurnSteadyWindowHasNoSpikes(t *testing.T) {
	got := mustGet(t, burnName).Analyze(burnInput([]int64{1000, 1100, 900, 1050, 950, 1000, 1020}))

	if got.Read.Label != "STEADY" {
		t.Fatalf("Read = %q, want STEADY for mild day-to-day variation", got.Read.Label)
	}
	if got.Bars == nil {
		t.Fatal("Bars must be non-nil so a spike-free window renders the honest 'none in this window' line")
	}
	if len(got.Bars) != 0 {
		t.Fatalf("Bars = %+v, want none", got.Bars)
	}
}

// TestBurnIdenticalDaysHaveNoDispersion asserts a window where every day burned exactly the
// same reports no spikes rather than dividing by a zero spread.
func TestBurnIdenticalDaysHaveNoDispersion(t *testing.T) {
	got := mustGet(t, burnName).Analyze(burnInput(flatDays(8, 5000)))

	if got.Read.Label != "STEADY" || len(got.Bars) != 0 {
		t.Fatalf("Read = %q, Bars = %+v; identical days cannot contain an outlier", got.Read.Label, got.Bars)
	}
}

// TestBurnShortWindowIsUndefined asserts a window too short to hold a baseline yields the
// neutral no-verdict read rather than a favorable one earned from too little history.
func TestBurnShortWindowIsUndefined(t *testing.T) {
	got := mustGet(t, burnName).Analyze(burnInput([]int64{1000, 50_000, 1000}))

	if got.Read != noDataRead {
		t.Fatalf("Read = %+v, want the neutral read below the %d-day floor", got.Read, burnMinDays)
	}
	if !strings.Contains(got.Takeaway, "Too few active days") {
		t.Fatalf("Takeaway = %q, want the short-window explanation", got.Takeaway)
	}
}

// TestBurnBelowDayFloorNamesNoSpike is the honesty guard on the pair: below the day floor
// the Read withholds its verdict, so nothing printed beside it may name a calendar date as
// unusual or count spikes the validator refuses to stand behind.
func TestBurnBelowDayFloorNamesNoSpike(t *testing.T) {
	tests := []struct {
		name string
		days []int64
	}{
		{"one runaway day among five", append(flatDays(4, 1000), 500_000)},
		{"one day short of the floor", append(flatDays(burnMinDays-2, 1000), 500_000)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mustGet(t, burnName).Analyze(burnInput(tt.days))

			if got.Read != noDataRead {
				t.Fatalf("Read = %+v, want the neutral read below the %d-day floor", got.Read, burnMinDays)
			}
			if got.Bars == nil {
				t.Fatal("Bars must be non-nil so the window renders the honest 'none in this window' line")
			}
			if len(got.Bars) != 0 {
				t.Fatalf("Bars = %+v, want no day named a spike below the day floor", got.Bars)
			}
			if f := findFigure(t, got.Figures, "spike days"); f.Value != "—" {
				t.Fatalf("spike days = %q, want the dash: a count here claims what the Read withheld", f.Value)
			}
		})
	}
}

// TestBurnQuietDayIsNotAnAnomaly asserts only the high side is reported: a day far below
// the typical one is not something to chase.
func TestBurnQuietDayIsNotAnAnomaly(t *testing.T) {
	got := mustGet(t, burnName).Analyze(burnInput(append(flatDays(7, 100_000), 10)))

	if got.Read.Label != "STEADY" || len(got.Bars) != 0 {
		t.Fatalf("Read = %q, Bars = %+v; an unusually quiet day is not a burn anomaly", got.Read.Label, got.Bars)
	}
}

// TestBurnEmptyInputSafe asserts the zero-value Input renders the honest no-data block.
func TestBurnEmptyInputSafe(t *testing.T) {
	got := mustGet(t, burnName).Analyze(Input{})
	if got.Read != noDataRead || got.Takeaway != "No usage in this window." {
		t.Fatalf("empty Input = %+v, want the no-data read and takeaway", got)
	}
}
