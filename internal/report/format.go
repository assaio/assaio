package report

import (
	"strconv"
	"strings"
)

// formatCommas renders n with thousands separators, e.g. 12345 -> "12,345".
func formatCommas(n int64) string {
	return formatThousands(float64(n), 0)
}

// formatThousands renders f with prec decimal places and comma-grouped thousands.
func formatThousands(f float64, prec int) string {
	s := strconv.FormatFloat(f, 'f', prec, 64)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	intPart, frac, hasFrac := strings.Cut(s, ".")
	grouped := groupThousands(intPart)
	if hasFrac {
		grouped += "." + frac
	}
	if neg {
		grouped = "-" + grouped
	}
	return grouped
}

// groupThousands inserts a comma every three digits from the right of a non-negative
// integer string, e.g. "12345" -> "12,345".
func groupThousands(intPart string) string {
	n := len(intPart)
	if n <= 3 {
		return intPart
	}
	var b strings.Builder
	lead := n % 3
	if lead > 0 {
		b.WriteString(intPart[:lead])
		b.WriteByte(',')
	}
	for i := lead; i < n; i += 3 {
		b.WriteString(intPart[i : i+3])
		if i+3 < n {
			b.WriteByte(',')
		}
	}
	return b.String()
}
