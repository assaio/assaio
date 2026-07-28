// Package humanize renders counts and amounts at glance legibility, so the same figure
// reads identically wherever it appears.
package humanize

import "strconv"

// Count renders a large count compactly, e.g. 33_400_000_000 -> "33.4B",
// 1_500_000 -> "1.5M", 2_300 -> "2.3K". Real cache-read totals reach billions.
func Count(n int64) string {
	switch {
	case n >= 1_000_000_000:
		return strconv.FormatFloat(float64(n)/1_000_000_000, 'f', 1, 64) + "B"
	case n >= 1_000_000:
		return strconv.FormatFloat(float64(n)/1_000_000, 'f', 1, 64) + "M"
	case n >= 1_000:
		return strconv.FormatFloat(float64(n)/1_000, 'f', 1, 64) + "K"
	default:
		return strconv.FormatInt(n, 10)
	}
}

// USD renders a dollar amount compactly: 195 -> "195", 26004 -> "26.0K". Cents are shown
// only below $100, where they still carry meaning.
func USD(v float64) string {
	switch {
	case v >= 10_000:
		return strconv.FormatFloat(v/1000, 'f', 1, 64) + "K"
	case v >= 100:
		return strconv.FormatFloat(v, 'f', 0, 64)
	default:
		return strconv.FormatFloat(v, 'f', 2, 64)
	}
}
