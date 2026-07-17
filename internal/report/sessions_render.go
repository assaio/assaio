package report

import (
	"fmt"
	"io"
	"strconv"
)

// RenderSessionsBlock writes the compact, honest "Sessions" section of the status
// dashboard: conversation depth, focused work time, how large contexts actually got, work
// produced, and how many sessions produce code versus stay conversational. An empty stats
// value (no sessions in the window) renders an honest "no sessions" line, not zeros.
func RenderSessionsBlock(w io.Writer, stats SessionStats) error {
	if _, err := fmt.Fprintln(w, "Sessions"); err != nil {
		return err
	}
	if stats.Count == 0 {
		_, err := fmt.Fprintln(w, "  No sessions in this window.")
		return err
	}
	if _, err := fmt.Fprintf(w, "  %d sessions · median %d turns (p90 %d) · %smin active work · peak context ~%s tokens\n",
		stats.Count, stats.MedianTurns, stats.P90Turns,
		formatWhole(stats.MedianActiveMinutes), formatCompactTokens(stats.MedianPeakContextTokens)); err != nil {
		return err
	}
	codePct := int64(stats.CodeSessionShare*100 + 0.5)
	_, err := fmt.Fprintf(w, "  %s output tokens/session · %d%% produced code, %d%% conversational · %s hit context compaction · %s/active day\n",
		formatCommas(stats.MedianOutputTokens), codePct, 100-codePct,
		formatPercent(stats.CompactionRate), strconv.FormatFloat(stats.SessionsPerActiveDay, 'f', 1, 64))
	return err
}

// formatWhole renders a float rounded to a whole number, e.g. 11.6 -> "12".
func formatWhole(f float64) string {
	return strconv.FormatFloat(f, 'f', 0, 64)
}

// formatPercent renders a 0-1 share as a whole-number percentage, e.g. 0.25 -> "25%".
func formatPercent(share float64) string {
	return strconv.FormatFloat(share*100, 'f', 0, 64) + "%"
}

// formatCompactTokens renders a token count compactly: "85k", "1.2M", or a bare small
// number, for a glance-legible context-size figure.
func formatCompactTokens(n int64) string {
	switch {
	case n >= 1_000_000:
		return strconv.FormatFloat(float64(n)/1_000_000, 'f', 1, 64) + "M"
	case n >= 1000:
		return strconv.FormatInt((n+500)/1000, 10) + "k"
	default:
		return strconv.FormatInt(n, 10)
	}
}
