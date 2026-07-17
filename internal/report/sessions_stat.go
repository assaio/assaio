package report

import (
	"math"
	"sort"
)

// percentile returns the p-th percentile (0-100) of sorted, an ascending-sorted slice,
// via linear interpolation between the two closest ranks (the common "linear" method).
// Returns 0 for an empty slice; never indexes out of range.
func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	switch n {
	case 0:
		return 0
	case 1:
		return sorted[0]
	}
	rank := p / 100 * float64(n-1)
	lo := int(rank)
	if lo >= n-1 {
		return sorted[n-1]
	}
	frac := rank - float64(lo)
	return sorted[lo] + frac*(sorted[lo+1]-sorted[lo])
}

// percentileInt64 returns the rounded p-th percentile of vals, sorting a copy so the
// caller's slice order is untouched. Returns 0 for an empty slice.
func percentileInt64(vals []int64, p float64) int64 {
	if len(vals) == 0 {
		return 0
	}
	floats := make([]float64, len(vals))
	for i, v := range vals {
		floats[i] = float64(v)
	}
	sort.Float64s(floats)
	return int64(math.Round(percentile(floats, p)))
}
