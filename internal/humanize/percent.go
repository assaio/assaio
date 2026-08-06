package humanize

import (
	"math"
	"strconv"
)

// Percent renders a 0-1 share as a whole-number percentage without the two dishonest
// rounding edges. See PercentAt: this is the precision every headline share uses.
func Percent(share float64) string { return PercentAt(share, 0) }

// PercentAt renders a 0-1 share at the given decimal precision, refusing the two roundings
// that state something the data does not: a small but nonzero share must not read "0%",
// which looks like a signal that is absent rather than one that is small (102 real tool-call
// refusals rounded to "0%"), and a share just under whole must not read "100%", which hides
// a real remainder. Both edges scale with the precision asked for, so a figure printed to
// one decimal says "<0.1%" and ">99.9%".
func PercentAt(share float64, decimals int) string {
	step := math.Pow10(-decimals)
	pct := share * 100
	switch {
	case pct > 0 && pct < step/2:
		return "<" + strconv.FormatFloat(step, 'f', decimals, 64) + "%"
	case pct < 100 && pct >= 100-step/2:
		return ">" + strconv.FormatFloat(100-step, 'f', decimals, 64) + "%"
	default:
		return strconv.FormatFloat(pct, 'f', decimals, 64) + "%"
	}
}

// PercentOrDash divides num by den at the given precision, "—" when den is zero: a ratio
// with no denominator is undefined, never the zero the arithmetic would hand it.
func PercentOrDash(num, den int64, decimals int) string {
	if den == 0 {
		return "—"
	}
	return PercentAt(float64(num)/float64(den), decimals)
}
