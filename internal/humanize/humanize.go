// Package humanize renders counts and amounts at glance legibility, so the same figure
// reads identically wherever it appears.
package humanize

import "strconv"

// countUnits, byteUnits and usdUnits name the tiers each renderer climbs through; the
// first entry is the unscaled value.
var (
	countUnits = [...]string{"", "K", "M", "B"}
	byteUnits  = [...]string{" B", " KB", " MB", " GB", " TB"}
	usdUnits   = [...]string{"", "K", "M"}
)

// Count renders a large count compactly, e.g. 33_400_000_000 -> "33.4B",
// 1_500_000 -> "1.5M", 2_300 -> "2.3K". Real cache-read totals reach billions.
func Count(n int64) string {
	v, step := scaleTo(float64(n), 1000, 1, len(countUnits)-1)
	if step == 0 {
		return strconv.FormatInt(n, 10)
	}
	return fixed(v, 1) + countUnits[step]
}

// Bytes renders an on-disk size in binary units, e.g. 35_131_392 -> "33.5 MB". Sizes are
// compared against what a file manager reports, so the units are powers of 1024 -- unlike
// Count, which renders quantities nobody measures in kibibytes.
func Bytes(n int64) string {
	v, step := scaleTo(float64(n), 1024, 1, len(byteUnits)-1)
	if step == 0 {
		return strconv.FormatInt(n, 10) + byteUnits[0]
	}
	return fixed(v, 1) + byteUnits[step]
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

// USDCompact renders a dollar amount with its sign for a space-constrained surface:
// 31500 -> "$31.5K", 750 -> "$750", 0.40 -> "$0.40". Whole dollars drop their cents, but an
// amount below one never rounds to "$0" -- a real cost rendered as no cost at all is the
// one thing a cost figure may not do, so a sub-cent amount says it is one.
func USDCompact(v float64) string {
	scaled, step := scaleTo(v, 1000, 1, len(usdUnits)-1)
	switch {
	case step > 0:
		return "$" + fixed(scaled, 1) + usdUnits[step]
	case v >= 1 || v <= 0:
		return "$" + fixed(v, 0)
	case v >= 0.005:
		return "$" + fixed(v, 2)
	default:
		return "<$0.01"
	}
}
