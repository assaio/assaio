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
	if _, err := fmt.Fprintf(w, "  %s · %s · %s · %s/active day\n",
		outputPhrase(stats), codePhrase(stats), compactionPhrase(stats),
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
	if s.Contexted == 0 {
		return "peak context not recorded"
	}
	return "peak context ~" + humanize.Count(s.MedianPeakContextTokens) + " tokens"
}

// outputPhrase keeps a token figure off a window whose sources keep no token counter. It is a
// separate basis from the turn count beside it: a source can record every turn of a session and
// no token at all, and "0 output tokens/session" would then be this line stating a fact about
// someone's work that came from the format's silence.
func outputPhrase(s *SessionStats) string {
	if s.Tokened == 0 {
		return "output tokens not recorded"
	}
	return humanize.Int(s.MedianOutputTokens) + " output tokens/session"
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

// renderSessionsBasis names the figure above that rests on the fewest sessions and says how
// many, and stays silent when every one of them covers the whole window. Without it a mix of
// sources prints one line of figures that quietly describe different subsets of it; without
// the name, a reader cannot tell which of the six the warning is about.
func renderSessionsBasis(w io.Writer, s *SessionStats) error {
	name, narrowest := narrowestSessionFigure(s)
	if narrowest >= s.Count {
		return nil
	}
	_, err := fmt.Fprintf(w,
		"  The narrowest figure above, %s, reads %d of %d sessions -- the ones whose source records it; the rest are absent from it, not zero.\n",
		name, narrowest, s.Count)
	return err
}

// narrowestSessionFigure is the figure above resting on the fewest sessions, among those that
// printed a number at all. A basis of zero is skipped rather than reported as the narrowest:
// that figure already replaced itself with "not recorded" on the line above, and counting it
// here made the whole block read "0 of 248 sessions" over figures that never claimed one.
// Ties go to the first in render order, so the sentence is stable across runs.
func narrowestSessionFigure(s *SessionStats) (name string, basis int) {
	for _, f := range []struct {
		name  string
		basis int
	}{
		{"turn depth", s.Turned},
		{"active work", s.Paced},
		{"peak context", s.Contexted},
		{"output tokens", s.Tokened},
		{"the code-session split", s.Edited},
		{"compaction rate", s.Compacting},
	} {
		if f.basis > 0 && (name == "" || f.basis < basis) {
			name, basis = f.name, f.basis
		}
	}
	if name == "" {
		// Every figure withheld itself; there is no narrowest one to name, and the block's
		// own "not recorded" phrases have already said so six times.
		return "", s.Count
	}
	return name, basis
}

// formatWhole renders a float rounded to a whole number, e.g. 11.6 -> "12".
func formatWhole(f float64) string {
	return strconv.FormatFloat(f, 'f', 0, 64)
}
