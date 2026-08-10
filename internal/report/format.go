package report

import "github.com/assaio/assaio/internal/humanize"

// formatCommas renders n with thousands separators, e.g. 12345 -> "12,345".
func formatCommas(n int64) string {
	return humanize.Thousands(float64(n), 0)
}
