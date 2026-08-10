package humanize

import (
	"strconv"
	"strings"
)

// Thousands renders f at prec decimal places with comma-grouped thousands, e.g.
// (107640.1, 2) -> "107,640.10". Grouping is what makes a six-figure count readable at a
// glance; the decimals are the caller's call.
func Thousands(f float64, prec int) string {
	s := strconv.FormatFloat(f, 'f', prec, 64)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	intPart, frac, hasFrac := strings.Cut(s, ".")
	grouped := groupDigits(intPart)
	if hasFrac {
		grouped += "." + frac
	}
	if neg {
		grouped = "-" + grouped
	}
	return grouped
}

// groupDigits inserts a comma every three digits from the right of a non-negative integer
// string, e.g. "12345" -> "12,345".
func groupDigits(intPart string) string {
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
