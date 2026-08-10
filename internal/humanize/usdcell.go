package humanize

import "math"

// usdCellFloor is the smallest amount USDCell will print; below it a cost is stated as a
// bound rather than rounded into a zero it is not.
const usdCellFloor = 0.00005

// USDCell renders a dollar amount for a table column whose header already carries the
// "$": comma-grouped and rounded to cents from a dollar up, and kept to two significant
// digits below one. Four fixed decimals read the same either way -- "107640.1234" is
// unreadable at the top of the range and "0.0000" is a lie at the bottom.
func USDCell(v float64) string {
	switch {
	case v < 0:
		return "-" + USDCell(-v)
	case v == 0:
		return "0.00"
	case v >= 1:
		return Thousands(v, 2)
	case v < usdCellFloor:
		return "<0.0001"
	default:
		return Thousands(v, subDollarPrec(v))
	}
}

// subDollarPrec is the decimal count that keeps two significant digits of an amount below
// one dollar: 0.14 stays "0.14", 0.0032 stays "0.0032".
func subDollarPrec(v float64) int {
	prec := 1 - int(math.Floor(math.Log10(v)))
	if prec < 2 {
		return 2
	}
	if prec > 4 {
		return 4
	}
	return prec
}
