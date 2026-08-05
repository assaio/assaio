package report

import (
	"time"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/store"
)

// SessionStats summarizes session-grain behavior honestly: conversation depth (turns),
// work produced (output tokens), how large contexts actually got (peak context), how
// much focused time a session took (resume-safe active minutes), and what share of
// sessions produce code versus stay conversational. Every field is a signal the
// day/project aggregate throws away; none is a misleading cumulative-cache or
// resume-polluted wall-clock number.
//
// Each group is computed over the sessions whose source can record it, never over all of
// them: a whole-session record has no second turn to measure a gap against, and a source
// that never marks a context overflow reports no compaction. The four basis counts below
// say how many sessions each group rests on, so a caller states the reach rather than
// implying the window.
type SessionStats struct {
	// Count is the number of sessions the stats were computed from.
	Count int
	// Turned, Paced, Edited and Compacting are how many of Count could answer each group:
	// turn depth and peak context, focused minutes, edits, and context compaction.
	Turned, Paced, Edited, Compacting int
	// MedianTurns and P90Turns are conversation depth: the typical and tail turn count.
	MedianTurns, P90Turns int64
	// MedianOutputTokens is the median output tokens per session: work produced, not
	// cumulative cache reads.
	MedianOutputTokens int64
	// MedianPeakContextTokens is the median of each session's largest single-turn context
	// window: the honest answer to "how large do contexts get".
	MedianPeakContextTokens int64
	// MedianActiveMinutes is the median resume-safe focused work time per session.
	MedianActiveMinutes float64
	// CodeSessionShare is the share (0-1) of sessions that made at least one edit: how
	// many sessions actually produce code versus stay conversational.
	CodeSessionShare float64
	// CompactionRate is the share (0-1) of sessions with at least one context compaction.
	CompactionRate float64
	// SessionsPerActiveDay is Count over the number of distinct UTC calendar days that had
	// at least one session.
	SessionsPerActiveDay float64
}

// BuildSessionStats computes SessionStats from rows (see store.Store.Sessions), one entry
// per session. Pure and deterministic: the same input always yields the same output, and
// empty input returns a zero-value SessionStats rather than dividing by zero. now is
// accepted for consistency with the other time-windowed Build* analyzers; every field
// here derives from the rows' own per-session values.
func BuildSessionStats(rows []store.SessionRow, now time.Time) SessionStats {
	_ = now
	if len(rows) == 0 {
		return SessionStats{}
	}
	turned := SessionsAnswering(rows, parser.SignalTurnsCount)
	paced := SessionsAnswering(rows, parser.SignalActiveMinutes)
	edited := SessionsAnswering(rows, parser.SignalEditsCount)
	compacting := SessionsAnswering(rows, parser.SignalCompactionsCount)
	turns := fieldInt64(turned, func(r *store.SessionRow) int64 { return r.Turns })

	return SessionStats{
		Count:                   len(rows),
		Turned:                  len(turned),
		Paced:                   len(paced),
		Edited:                  len(edited),
		Compacting:              len(compacting),
		MedianTurns:             percentileInt64(turns, 50),
		P90Turns:                percentileInt64(turns, 90),
		MedianOutputTokens:      percentileInt64(fieldInt64(rows, func(r *store.SessionRow) int64 { return r.OutputTokens }), 50),
		MedianPeakContextTokens: percentileInt64(fieldInt64(turned, func(r *store.SessionRow) int64 { return r.PeakContextTokens }), 50),
		MedianActiveMinutes:     medianFloat(fieldFloat(paced, func(r *store.SessionRow) float64 { return r.ActiveMinutes })),
		CodeSessionShare:        shareWhere(edited, func(r *store.SessionRow) bool { return r.Edits > 0 }),
		CompactionRate:          shareWhere(compacting, func(r *store.SessionRow) bool { return r.Compactions > 0 }),
		SessionsPerActiveDay:    float64(len(rows)) / float64(activeDayCount(rows)),
	}
}
