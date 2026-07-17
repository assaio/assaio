package pricing

import (
	"regexp"
	"strings"
)

var (
	bracketSuffix = regexp.MustCompile(`\[[^\]]*\]$`)
	dateSuffix    = regexp.MustCompile(`-\d{8}$`)
)

// NormalizeModel strips a trailing "[...]" tag (e.g. the "[1m]" context-window suffix),
// drops a trailing -YYYYMMDD date stamp, and lower-cases, so log model strings resolve
// to a price key.
func NormalizeModel(m string) string {
	m = bracketSuffix.ReplaceAllString(strings.TrimSpace(m), "")
	return dateSuffix.ReplaceAllString(strings.ToLower(m), "")
}
