package analyze

import "sort"

// The scale a window is judged against its own typical value on. Shared rather than restated per
// metric: two validators asking "how far from typical is this" with two sets of constants drift
// apart into two different meanings of "far", and a reader has no way to see it happen.
const (
	// zThreshold is the modified z-score above which a value counts as standing outside the
	// window. 3.5 is the conventional cutoff for the MAD-based score.
	zThreshold = 3.5
	// madToSigma scales a median absolute deviation onto the standard-deviation scale for
	// normally distributed data, so zThreshold keeps its conventional meaning.
	madToSigma = 1 / 0.6745
	// meanADToSigma scales a mean absolute deviation the same way, for the fallback below.
	meanADToSigma = 1.253314
)

// medianOf is the middle value, 0 for an empty input. values is sorted in place.
func medianOf(values []float64) float64 {
	sort.Float64s(values)
	return percentileAt(values, 0.5)
}

// dispersion estimates the spread of values around med on the standard-deviation scale,
// preferring the median absolute deviation because one runaway value cannot inflate it enough to
// hide itself. It falls back to the mean absolute deviation when that median is zero -- which
// happens whenever more than half the values are identical -- and reports ok=false only when
// every value equals med, where nothing can be an outlier.
func dispersion(values []float64, med float64) (sigma float64, ok bool) {
	deviations := make([]float64, len(values))
	for i, v := range values {
		if d := v - med; d < 0 {
			deviations[i] = -d
		} else {
			deviations[i] = d
		}
	}
	sort.Float64s(deviations)
	if mad := percentileAt(deviations, 0.5); mad > 0 {
		return mad * madToSigma, true
	}
	var sum float64
	for _, d := range deviations {
		sum += d
	}
	if sum == 0 {
		return 0, false
	}
	return sum / float64(len(deviations)) * meanADToSigma, true
}

// outlierFloor is the value above which a member of this window stands outside it, and whether
// the window has enough spread for the question to mean anything.
func outlierFloor(values []float64) (floor float64, ok bool) {
	med := medianOf(values)
	sigma, ok := dispersion(values, med)
	if !ok {
		return 0, false
	}
	return med + zThreshold*sigma, true
}
