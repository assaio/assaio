package report

import (
	"fmt"
	"io"
	"strconv"

	"github.com/assaio/assaio/internal/humanize"
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
	return "peak context ~" + humanize.Count(s.MedianPeakContextTokens) + " tokens"
}

// codePhrase renders both sides of the split through the honest share formatter, and
// renders them separately rather than subtracting one from a hundred: rounding the code
// share up to whole would otherwise print the conversational share as a flat "0%", which
// is the sentence ADR 0011 exists to keep off this line.
func codePhrase(s *SessionStats) string {
	if s.Edited == 0 {
		return "edits not recorded"
	}
	return fmt.Sprintf("%s produced code, %s conversational",
		humanize.Percent(s.CodeSessionShare), humanize.Percent(1-s.CodeSessionShare))
}

func compactionPhrase(s *SessionStats) string {
	if s.Compacting == 0 {
		return "compaction not recorded"
	}
	return humanize.Percent(s.CompactionRate) + " hit context compaction"
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
