package humanize

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

// compactRenderers is every tiered renderer this package publishes, with the base it climbs
// by, the point its unscaled tier ends at, and the units it can reach. Adding a renderer here
// is what puts it under the class test below; adding one without is how B127 came back one
// renderer later.
// USD's climbAt is its own: whole dollars run to $9,999 on purpose, because $9,999 and "10.0K"
// are different answers to a question about money.
var compactRenderers = []struct {
	name    string
	base    float64
	climbAt float64
	units   []string
	render  func(float64) string
}{
	{"Count", 1000, 1000, countUnits[:], func(v float64) string { return Count(int64(v)) }},
	{"Bytes", 1024, 1024, byteUnits[:], func(v float64) string { return Bytes(int64(v)) }},
	{"USD", 1000, 10_000, usdUnits[:], USD},
	{"USDCompact", 1000, 1000, usdUnits[:], USDCompact},
}

// TestEveryCompactRendererPicksItsUnitAfterRounding holds the class B127 closed for Count and
// Bytes in v0.13 and USD reopened at v0.21 by never having joined them: a value must not
// print its own ceiling stated in the unit below it. Written over the renderer list rather
// than per function, because the defect returned by way of the one renderer that had no such
// test, not by way of a changed one.
func TestEveryCompactRendererPicksItsUnitAfterRounding(t *testing.T) {
	for _, r := range compactRenderers {
		top := r.units[len(r.units)-1]
		t.Run(r.name, func(t *testing.T) {
			for _, v := range tierProbes(r.base, len(r.units)) {
				out := r.render(v)
				num, unit := splitRendered(t, out)
				ceiling := r.base
				if unit == r.units[0] {
					ceiling = r.climbAt
				}
				if unit == top || math.Abs(num) < ceiling {
					continue
				}
				t.Errorf("%s(%g) = %q: %g reaches %g in unit %q, so it states its own ceiling in the unit below %q",
					r.name, v, out, num, ceiling, unit, top)
			}
		})
	}
}

// tierProbes are the values that sit on and just under each tier's ceiling -- including the
// one that rounds up into the next unit only once printed, which is the whole of this class.
func tierProbes(base float64, units int) []float64 {
	var out []float64
	for k := 1; k < units+1; k++ {
		b := math.Pow(base, float64(k))
		out = append(out, b, b-1, b*(1-5e-5), b*(1-1e-7), b/2, b*9.9)
	}
	return out
}

// splitRendered separates a rendered figure into its numeric part and its unit, dropping the
// "$" and "<" a money renderer may lead with.
func splitRendered(t *testing.T, s string) (float64, string) {
	t.Helper()
	trimmed := strings.TrimLeft(s, "<$")
	cut := len(trimmed)
	for i, c := range trimmed {
		if (c < '0' || c > '9') && c != '.' && c != '-' {
			cut = i
			break
		}
	}
	n, err := strconv.ParseFloat(trimmed[:cut], 64)
	if err != nil {
		t.Fatalf("cannot read a number out of %q: %v", s, err)
	}
	return n, trimmed[cut:]
}
