package analyze

import (
	"sort"
	"strconv"

	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

const (
	contextName     = "context"
	contextTitle    = "Context Health"
	contextDescribe = "Conversation depth, peak context size, active time, and how often sessions hit compaction."
	// contextHowToRead is Result.HowToRead for this validator -- see its doc comment.
	contextHowToRead = "Frequent compaction means sessions are outgrowing their context window mid-task -- worth trying shorter, more focused sessions rather than reading it as a quality problem."
	// contextWatchCeiling is the compaction-rate threshold above which sessions are
	// running out of context often enough to flag.
	contextWatchCeiling = 0.2
	// contextMinSessionsForHealthy is the minimum session count before a favorable Healthy
	// read is trustworthy: one session with zero compactions is a single data point, not
	// evidence of healthy context sizing. Matches adoption's session floor.
	contextMinSessionsForHealthy = 3
)

func init() { Register(contextValidator{}) }

// contextValidator reads session-grain context health: conversation depth, how large
// contexts got, focused active time, and how often sessions had to compact.
type contextValidator struct{}

func (contextValidator) Name() string     { return contextName }
func (contextValidator) Title() string    { return contextTitle }
func (contextValidator) Describe() string { return contextDescribe }

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (contextValidator) Analyze(in Input) Result {
	r := Result{Name: contextName, Title: contextTitle, Describe: contextDescribe, HowToRead: contextHowToRead}
	if len(in.Sessions) == 0 {
		r.Read = noDataRead
		r.Takeaway = "No usage in this window."
		return r
	}
	stats := report.BuildSessionStats(in.Sessions, in.Now)
	compactionOK := stats.CompactionRate <= contextWatchCeiling
	sufficientSample := stats.Count >= contextMinSessionsForHealthy
	healthy := compactionOK && sufficientSample
	codeMedian, codeMedianOK := medianActiveMinutesForCodeSessions(in.Sessions)

	r.Read = readFor(healthy, "Healthy")
	r.Purity = contextPurity(stats.CompactionRate, sufficientSample)
	r.Figures = []Figure{
		{Label: "sessions", Value: strconv.Itoa(stats.Count)},
		{Label: "median turns", Value: strconv.FormatInt(stats.MedianTurns, 10)},
		{Label: "peak context", Value: strconv.FormatInt(stats.MedianPeakContextTokens, 10) + " tokens"},
		activeWorkFigure(stats.MedianActiveMinutes, codeMedian, codeMedianOK),
		{Label: "compaction rate", Value: formatPercent(stats.CompactionRate, 0)},
		{
			Label: "code sessions", Value: formatPercent(stats.CodeSessionShare, 0),
			Note: formatPercent(1-stats.CodeSessionShare, 0) + " conversational",
		},
	}
	r.Takeaway = contextTakeaway(healthy, compactionOK, sufficientSample)
	return r
}

func contextTakeaway(healthy, compactionOK, sufficientSample bool) string {
	switch {
	case healthy:
		return "Sessions rarely hit context compaction -- context sizing looks fine."
	case compactionOK && !sufficientSample:
		return "Compaction is rare so far, but too few sessions this window to call context health confidently."
	default:
		return "Sessions hit context compaction often -- consider shorter sessions or more aggressive summarization."
	}
}

// contextPurity is high when compaction is rare, but only once there are enough sessions
// to trust the rate; too small a sample yields the neutral 0.5 the other validators use
// for "not enough evidence yet" rather than a confident gauge off a single session.
func contextPurity(compactionRate float64, sufficientSample bool) float64 {
	if !sufficientSample {
		return 0.5
	}
	return clamp01(1 - compactionRate)
}

// activeWorkFigure reports both the overall median active minutes and, when at least one
// session made an edit, the median for code sessions alone. The overall median is
// usually pulled down near zero by many quick, non-code sessions, so showing it bare,
// without the code-session contrast, misrepresents how long focused work sessions run.
func activeWorkFigure(overallMedian, codeMedian float64, codeMedianOK bool) Figure {
	f := Figure{Label: "active work", Value: minutesLabel(overallMedian) + " median"}
	if codeMedianOK {
		f.Note = "code sessions: ~" + minutesLabel(codeMedian)
	}
	return f
}

func minutesLabel(m float64) string {
	return strconv.FormatFloat(m, 'f', 0, 64) + " min"
}

// medianActiveMinutesForCodeSessions returns the median ActiveMinutes across sessions
// with at least one edit -- the code-producing subset, isolated from the many quick,
// non-code sessions that pull the overall median down near zero. ok is false when no
// session in sessions made an edit.
func medianActiveMinutesForCodeSessions(sessions []store.SessionRow) (median float64, ok bool) {
	var actives []float64
	for i := range sessions {
		if sessions[i].Edits > 0 {
			actives = append(actives, sessions[i].ActiveMinutes)
		}
	}
	if len(actives) == 0 {
		return 0, false
	}
	sort.Float64s(actives)
	return medianAt50(actives), true
}

// medianAt50 returns the median of sorted (ascending), via the same linear-interpolation
// method report.BuildSessionStats uses for its own medians, so this figure is directly
// comparable to SessionStats.MedianActiveMinutes.
func medianAt50(sorted []float64) float64 {
	n := len(sorted)
	switch n {
	case 0:
		return 0
	case 1:
		return sorted[0]
	}
	mid := float64(n-1) / 2
	lo := int(mid)
	frac := mid - float64(lo)
	return sorted[lo] + frac*(sorted[lo+1]-sorted[lo])
}
