package analyze

import (
	"sort"

	"github.com/assaio/assaio/internal/store"
)

// dayBurn is one active day's total token burn.
type dayBurn struct {
	Day    string
	Tokens int64
}

// tokensPerDay collapses the daily usage rows -- one per tool/model/project combination --
// into one total per calendar day, sorted by day ascending.
func tokensPerDay(rows []store.UsageRow) []dayBurn {
	byDay := make(map[string]int64, len(rows))
	for i := range rows {
		byDay[rows[i].Day] += rowTokens(&rows[i])
	}
	out := make([]dayBurn, 0, len(byDay))
	for day, tokens := range byDay {
		out = append(out, dayBurn{Day: day, Tokens: tokens})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Day < out[j].Day })
	return out
}

// medianTokens is the middle day's burn, the baseline every spike is measured against.
func medianTokens(days []dayBurn) int64 { return int64(medianOf(burnValues(days))) }

func burnValues(days []dayBurn) []float64 {
	out := make([]float64, len(days))
	for i := range days {
		out[i] = float64(days[i].Tokens)
	}
	return out
}

// burnSpikes returns the days burning far above the median, by modified z-score. Only the
// high side is reported: a quiet day is not an anomaly worth chasing. A window with no
// dispersion at all -- every day identical -- yields no spikes rather than dividing by
// nothing. Sorted by burn descending.
//
// Below burnMinDays it names none: the same floor the Read withholds its verdict at, held
// here so no caller can print a spike the validator refuses to stand behind.
func burnSpikes(days []dayBurn, median int64) []dayBurn {
	if len(days) < burnMinDays {
		return nil
	}
	sigma, ok := burnDispersion(days, median)
	if !ok {
		return nil
	}
	var spikes []dayBurn
	for i := range days {
		if float64(days[i].Tokens-median)/sigma > zThreshold {
			spikes = append(spikes, days[i])
		}
	}
	sort.Slice(spikes, func(i, j int) bool { return spikes[i].Tokens > spikes[j].Tokens })
	return spikes
}

// burnDispersion is the window's spread on the standard-deviation scale, from the shared rule
// every "far from typical" question in this package is answered with (dispersion.go).
func burnDispersion(days []dayBurn, median int64) (sigma float64, ok bool) {
	return dispersion(burnValues(days), float64(median))
}
