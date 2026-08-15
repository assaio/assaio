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
	v, step := scaleTo(float64(n), 1000, len(countUnits)-1)
	if step == 0 {
		return strconv.FormatInt(n, 10)
	}
	return fixed(v, compactDecimals) + countUnits[step]
}

// Bytes renders an on-disk size in binary units, e.g. 35_131_392 -> "33.5 MB". Sizes are
// compared against what a file manager reports, so the units are powers of 1024 -- unlike
// Count, which renders quantities nobody measures in kibibytes.
func Bytes(n int64) string {
	v, step := scaleTo(float64(n), 1024, len(byteUnits)-1)
	if step == 0 {
		return strconv.FormatInt(n, 10) + byteUnits[0]
	}
	return fixed(v, compactDecimals) + byteUnits[step]
}

// USD renders a dollar amount compactly: 195 -> "195", 26004 -> "26.0K",
// 33_500_000 -> "33.5M". Cents are shown only below $100, where they still carry meaning.
// Every tier is chosen from the value that will actually be printed, so an amount never
// states its own ceiling in the tier below it (scaleTo's doc names the failure).
func USD(v float64) string {
	switch {
	case roundTo(v, 0) >= 10_000:
		scaled, step := scaleTo(v, 1000, len(usdUnits)-1)
		return fixed(scaled, compactDecimals) + usdUnits[step]
	case roundTo(v, 2) >= 100:
		return fixed(v, 0)
	case v > 0 && v < 0.005:
		// A real cost rendered as no cost at all is the one thing a money figure may not do --
		// the rule USDCompact already held and this one did not.
		return "<0.01"
	default:
		return fixed(v, 2)
	}
}

// USDCompact renders a dollar amount with its sign for a space-constrained surface:
// 31500 -> "$31.5K", 750 -> "$750", 0.40 -> "$0.40". Whole dollars drop their cents, but an
// amount below one never rounds to "$0" -- a real cost rendered as no cost at all is the
// one thing a cost figure may not do, so a sub-cent amount says it is one.
func USDCompact(v float64) string {
	scaled, step := scaleTo(v, 1000, len(usdUnits)-1)
	switch {
	case step > 0:
		return "$" + fixed(scaled, compactDecimals) + usdUnits[step]
	case v >= 1 || v <= 0:
		return "$" + fixed(v, 0)
	case v >= 0.005:
		return "$" + fixed(v, 2)
	default:
		return "<$0.01"
	}
}

// Days renders a day count for a span a figure rests on: whole days below one decimal's worth of
// difference, one decimal otherwise, because "5 days" and "5.8 days" are different evidence.
func Days(d float64) string {
	if d == roundTo(d, 0) {
		return fixed(d, 0) + " days"
	}
	return fixed(d, 1) + " days"
}
