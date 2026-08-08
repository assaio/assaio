package humanize

import (
	"math"
	"strconv"
)

// scaleTo divides v by base until the value that will actually be *printed* is below base,
// and returns it with the index of the unit it reached. The rounding is what makes this
// worth a function: choosing the unit from the raw value renders 999,999,999 as "1000.0M"
// and 1,048,575 bytes as "1024.0 KB" -- a unit's ceiling stated in the unit below it.
// steps bounds how far it may climb, so the caller never asks for a unit it has no name for.
func scaleTo(v, base float64, decimals, steps int) (scaled float64, step int) {
	for step < steps && roundTo(v, decimals) >= base {
		v /= base
		step++
	}
	return v, step
}

// roundTo rounds v to decimals places, matching what strconv.FormatFloat will print.
func roundTo(v float64, decimals int) float64 {
	p := math.Pow10(decimals)
	return math.Round(v*p) / p
}

// fixed formats v at decimals places.
func fixed(v float64, decimals int) string {
	return strconv.FormatFloat(v, 'f', decimals, 64)
}
