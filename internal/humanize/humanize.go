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

// Bytes renders an on-disk size in binary units, e.g. 35_131_392 -> "33.5 MB". Sizes are
// compared against what a file manager reports, so the units are powers of 1024 -- unlike
// Count, which renders quantities nobody measures in kibibytes.
func Bytes(n int64) string {
	const unit = 1024
	if n < unit {
		return strconv.FormatInt(n, 10) + " B"
	}
	v, exp := float64(n)/unit, 0
	for v >= unit && exp < 3 {
		v /= unit
		exp++
	}
	return strconv.FormatFloat(v, 'f', 1, 64) + " " + [...]string{"KB", "MB", "GB", "TB"}[exp]
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
