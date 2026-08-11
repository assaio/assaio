package analyze

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/parser"

	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

// contextFigures renders each figure over the sessions that can carry it, and "—" where no
// source in the window records it at all. That is the difference this validator exists to
// keep: a session whose context did not overflow, and a tool that never says whether one did.
func contextFigures(stats *report.SessionStats, sessions []store.SessionRow) []Figure {
	codeMedian, codeMedianOK := medianActiveMinutesForCodeSessions(sessions)
	return []Figure{
		{Label: "sessions", Value: humanize.Int(int64(stats.Count))},
		basisFigure("median turns", strconv.FormatInt(stats.MedianTurns, 10), "", stats.Turned),
		basisFigure("peak context", humanize.Count(stats.MedianPeakContextTokens)+" tokens", "", stats.Turned),
		activeWorkFigure(stats, codeMedian, codeMedianOK),
		basisFigure("compaction rate", humanize.PercentAt(stats.CompactionRate, 0), "", stats.Compacting),
		codeSessionsFigure(stats),
	}
}

// basisFigure prints value only when some session could produce it; a figure computed over
// nothing prints "—" rather than the zero the arithmetic hands it.
func basisFigure(label, value, note string, basis int) Figure {
	if basis == 0 {
		return Figure{Label: label, Value: "—", Note: "no source here records it"}
	}
	return Figure{Label: label, Value: value, Note: note}
}

func codeSessionsFigure(stats *report.SessionStats) Figure {
	if stats.Edited == 0 {
		return Figure{Label: "code sessions", Value: "—", Note: "no source here records edits"}
	}
	return Figure{
		Label: "code sessions", Value: humanize.PercentAt(stats.CodeSessionShare, 0),
		Note: humanize.PercentAt(1-stats.CodeSessionShare, 0) + " conversational",
	}
}

// activeWorkFigure reports both the overall median active minutes and, when at least one
// session made an edit, the median for code sessions alone. The overall median is usually
// pulled down near zero by many quick, non-code sessions, so showing it bare, without the
// code-session contrast, misrepresents how long focused work sessions run.
func activeWorkFigure(stats *report.SessionStats, codeMedian float64, codeMedianOK bool) Figure {
	if stats.Paced == 0 {
		return Figure{Label: "active work", Value: "—", Note: "no source here records focused time"}
	}
	f := Figure{Label: "active work", Value: minutesLabel(stats.MedianActiveMinutes) + " median"}
	if codeMedianOK {
		f.Note = "code sessions: ~" + minutesLabel(codeMedian)
	}
	return f
}

func minutesLabel(m float64) string {
	return strconv.FormatFloat(m, 'f', 0, 64) + " min"
}

// medianActiveMinutesForCodeSessions returns the median ActiveMinutes across sessions with
// at least one edit -- the code-producing subset, isolated from the many quick, non-code
// sessions that pull the overall median down near zero. ok is false when no session made an
// edit.
//
// It reads both capabilities it needs, not one: a source recording no edit would put every
// session in the non-code group, and one recording no focused minutes would contribute a
// structural zero to the median of a figure about how long focused work runs.
func medianActiveMinutesForCodeSessions(sessions []store.SessionRow) (median float64, ok bool) {
	measurable, _ := sessionsAnswering(sessions, parser.SignalEditsCount, parser.SignalActiveMinutes)
	var actives []float64
	for i := range measurable {
		if measurable[i].Edits > 0 {
			actives = append(actives, measurable[i].ActiveMinutes)
		}
	}
	if len(actives) == 0 {
		return 0, false
	}
	sort.Float64s(actives)
	return medianAt50(actives), true
}

// narrowestBasis is the smallest number of sessions any figure above rests on -- what
// decides whether the reach has to be stated at all.
func narrowestBasis(stats *report.SessionStats) int {
	return min(stats.Turned, stats.Paced, stats.Edited, stats.Compacting)
}

// contextCoverageCaveat names how much of the window each group of figures rests on, so a
// mix of sources never prints one row of numbers quietly describing different subsets of it.
func contextCoverageCaveat(stats *report.SessionStats) string {
	return fmt.Sprintf(
		"Prov.: of %d sessions, %d come from a source recording turns and context size, %d recording focused minutes, %d recording edits, and %d marking compaction; the rest are absent from those figures, not zero.",
		stats.Count, stats.Turned, stats.Paced, stats.Edited, stats.Compacting,
	)
}
