package report

import (
	"fmt"
	"io"
	"strconv"
)

// RenderSessionsBlock writes the compact, honest "Sessions" section of the status
// dashboard: conversation depth, focused work time, how large contexts actually got, work
// produced, and how many sessions produce code versus stay conversational. An empty stats
// value (no sessions in the window) renders an honest "no sessions" line, not zeros, and a
// figure no source in the window records says so rather than printing the zero it would
// otherwise average to.
func RenderSessionsBlock(w io.Writer, stats *SessionStats) error {
	if _, err := fmt.Fprintln(w, "Sessions"); err != nil {
		return err
	}
	if stats.Count == 0 {
		_, err := fmt.Fprintln(w, "  No sessions in this window.")
		return err
	}
	if _, err := fmt.Fprintf(w, "  %d sessions · %s · %s · %s\n",
		stats.Count, turnsPhrase(stats), activePhrase(stats), contextPhrase(stats)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  %s output tokens/session · %s · %s · %s/active day\n",
		formatCommas(stats.MedianOutputTokens), codePhrase(stats), compactionPhrase(stats),
		strconv.FormatFloat(stats.SessionsPerActiveDay, 'f', 1, 64)); err != nil {
		return err
	}
	return renderSessionsBasis(w, stats)
}

func turnsPhrase(s *SessionStats) string {
	if s.Turned == 0 {
		return "turn depth not recorded"
	}
	return fmt.Sprintf("median %d turns (p90 %d)", s.MedianTurns, s.P90Turns)
}

func activePhrase(s *SessionStats) string {
	if s.Paced == 0 {
		return "focused time not recorded"
	}
	return formatWhole(s.MedianActiveMinutes) + "min active work"
}

func contextPhrase(s *SessionStats) string {
	if s.Turned == 0 {
		return "peak context not recorded"
	}
	return "peak context ~" + formatCompactTokens(s.MedianPeakContextTokens) + " tokens"
}

func codePhrase(s *SessionStats) string {
	if s.Edited == 0 {
		return "edits not recorded"
	}
	codePct := int64(s.CodeSessionShare*100 + 0.5)
	return fmt.Sprintf("%d%% produced code, %d%% conversational", codePct, 100-codePct)
}

func compactionPhrase(s *SessionStats) string {
	if s.Compacting == 0 {
		return "compaction not recorded"
	}
	return formatPercent(s.CompactionRate) + " hit context compaction"
}

// renderSessionsBasis states how many sessions the narrowest figure above rests on, and
// stays silent when every one of them covers the whole window. Without it a mix of sources
// prints one line of figures that quietly describe different subsets of it.
func renderSessionsBasis(w io.Writer, s *SessionStats) error {
	narrowest := min(s.Turned, s.Paced, s.Edited, s.Compacting)
	if narrowest >= s.Count {
		return nil
	}
	_, err := fmt.Fprintf(w,
		"  Turn, edit and compaction figures read the %d of %d sessions whose source records them; the rest are absent from those, not zero.\n",
		narrowest, s.Count)
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
