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
func medianTokens(days []dayBurn) int64 {
	values := make([]float64, len(days))
	for i := range days {
		values[i] = float64(days[i].Tokens)
	}
	sort.Float64s(values)
	return int64(percentileAt(values, 0.5))
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
		if float64(days[i].Tokens-median)/sigma > burnZThreshold {
			spikes = append(spikes, days[i])
		}
	}
	sort.Slice(spikes, func(i, j int) bool { return spikes[i].Tokens > spikes[j].Tokens })
	return spikes
}

// burnDispersion estimates the window's spread on the standard-deviation scale, preferring
// the median absolute deviation because one runaway day cannot inflate it enough to hide
// itself. It falls back to the mean absolute deviation when that median is zero -- which
// happens whenever more than half the days share one value -- and reports ok=false only
// when every day burned exactly the same, where no day can be an outlier.
func burnDispersion(days []dayBurn, median int64) (sigma float64, ok bool) {
	deviations := absoluteDeviations(days, median)
	sort.Float64s(deviations)
	if mad := percentileAt(deviations, 0.5); mad > 0 {
		return mad * burnMADToSigma, true
	}
	var sum float64
	for _, d := range deviations {
		sum += d
	}
	if sum == 0 {
		return 0, false
	}
	return sum / float64(len(deviations)) * burnMeanADToSigma, true
}

// absoluteDeviations is each day's distance from the median burn.
func absoluteDeviations(days []dayBurn, median int64) []float64 {
	out := make([]float64, len(days))
	for i := range days {
		d := float64(days[i].Tokens - median)
		if d < 0 {
			d = -d
		}
		out[i] = d
	}
	return out
}
