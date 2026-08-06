package report

import (
	"sort"

	"github.com/assaio/assaio/internal/store"
)

func fieldInt64(rows []store.SessionRow, pick func(*store.SessionRow) int64) []int64 {
	out := make([]int64, len(rows))
	for i := range rows {
		out[i] = pick(&rows[i])
	}
	return out
}

func fieldFloat(rows []store.SessionRow, pick func(*store.SessionRow) float64) []float64 {
	out := make([]float64, len(rows))
	for i := range rows {
		out[i] = pick(&rows[i])
	}
	return out
}

// medianFloat sorts a copy and returns its median, 0 for an empty slice.
func medianFloat(vals []float64) float64 {
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	return Percentile(sorted, 50)
}

// shareWhere is the share of rows satisfying want, 0 when there are none to ask.
func shareWhere(rows []store.SessionRow, want func(*store.SessionRow) bool) float64 {
	if len(rows) == 0 {
		return 0
	}
	n := 0
	for i := range rows {
		if want(&rows[i]) {
			n++
		}
	}
	return float64(n) / float64(len(rows))
}

// activeDayCount is the number of distinct UTC calendar days rows started on, at least 1 so
// a per-day rate never divides by zero.
func activeDayCount(rows []store.SessionRow) int {
	days := make(map[string]struct{}, len(rows))
	for i := range rows {
		days[rows[i].FirstTs.UTC().Format("2006-01-02")] = struct{}{}
	}
	if len(days) == 0 {
		return 1
	}
	return len(days)
}
