package reconcile

import (
	"strconv"
	"strings"
	"time"
)

// dayLayouts are the timestamp shapes an export is read as. The list is short on purpose:
// a layout guessed wrong shifts a row into the wrong day and quietly moves money between
// buckets, so an unrecognized shape is skipped and counted instead.
var dayLayouts = []string{
	"2006-01-02",
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006/01/02",
}

// parseDay normalizes an export's date cell to a YYYY-MM-DD bucket in UTC, matching the
// bucket the store already uses.
func parseDay(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	for _, layout := range dayLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return day(t), true
		}
	}
	return "", false
}

// parseMoney reads an amount that may carry a currency symbol, thousands separators, or
// parentheses for a credit. It does not convert currency: Reconcile refuses a non-USD
// export rather than inventing a rate.
func parseMoney(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	negative := strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")")
	if negative {
		s = strings.TrimSuffix(strings.TrimPrefix(s, "("), ")")
	}
	s = strings.NewReplacer("$", "", ",", "", " ", "", " ", "", "USD", "", "usd", "").Replace(s)
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	if negative {
		v = -v
	}
	return v, true
}

// parseCount reads a whole-number cell, tolerating thousands separators and a decimal part
// that is exactly zero -- exports commonly render counts as floats.
func parseCount(s string) (int64, bool) {
	s = strings.NewReplacer(",", "", " ", "", " ", "").Replace(strings.TrimSpace(s))
	if s == "" {
		return 0, false
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n, true
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f != float64(int64(f)) {
		return 0, false
	}
	return int64(f), true
}
