package report

import (
	"sort"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// SessionStats summarizes session-grain behavior honestly: conversation depth (turns),
// work produced (output tokens), how large contexts actually got (peak context), how
// much focused time a session took (resume-safe active minutes), and what share of
// sessions produce code versus stay conversational. Every field is a signal the
// day/project aggregate throws away; none is a misleading cumulative-cache or
// resume-polluted wall-clock number.
type SessionStats struct {
	// Count is the number of sessions the stats were computed from.
	Count int
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

	turns := make([]int64, len(rows))
	outputs := make([]int64, len(rows))
	peaks := make([]int64, len(rows))
	actives := make([]float64, len(rows))
	codeSessions, compacted := 0, 0
	activeDays := make(map[string]struct{}, len(rows))
	for i := range rows {
		r := &rows[i]
		turns[i] = r.Turns
		outputs[i] = r.OutputTokens
		peaks[i] = r.PeakContextTokens
		actives[i] = r.ActiveMinutes
		if r.Edits > 0 {
			codeSessions++
		}
		if r.Compactions > 0 {
			compacted++
		}
		activeDays[r.FirstTs.UTC().Format("2006-01-02")] = struct{}{}
	}
	sort.Float64s(actives)
	n := float64(len(rows))

	return SessionStats{
		Count:                   len(rows),
		MedianTurns:             percentileInt64(turns, 50),
		P90Turns:                percentileInt64(turns, 90),
		MedianOutputTokens:      percentileInt64(outputs, 50),
		MedianPeakContextTokens: percentileInt64(peaks, 50),
		MedianActiveMinutes:     percentile(actives, 50),
		CodeSessionShare:        float64(codeSessions) / n,
		CompactionRate:          float64(compacted) / n,
		SessionsPerActiveDay:    n / float64(len(activeDays)),
	}
}
