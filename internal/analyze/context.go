package analyze

import (
	"github.com/assaio/assaio/internal/layer"
	"github.com/assaio/assaio/internal/report"
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

func (contextValidator) Name() string       { return contextName }
func (contextValidator) Title() string      { return contextTitle }
func (contextValidator) Describe() string   { return contextDescribe }
func (contextValidator) Layer() layer.Layer { return layer.Activity } // how often sessions hit compaction

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (contextValidator) Analyze(in Input) Result {
	r := Result{Name: contextName, Title: contextTitle, Describe: contextDescribe, HowToRead: contextHowToRead}
	if len(in.Sessions) == 0 {
		r.noData("sessions", "No usage in this window.")
		return r
	}
	// The verdict is the compaction rate, so the metric reaches exactly as far as compaction
	// capture does: a source that never marks a context overflow reports none, and averaging
	// that silence in would read as a window whose sessions all sat comfortably inside it.
	stats := report.BuildSessionStats(in.Sessions, in.Now)
	r.restsOn(stats.Compacting, "sessions with compaction capture")
	r.covering(shareOf(int64(stats.Compacting), int64(stats.Count)))
	r.Figures = contextFigures(&stats, in.Sessions)
	if stats.Compacting == 0 {
		r.Read = noDataRead
		r.Purity = neutralPurity
		r.Takeaway = "No source in this window marks a context compaction, so context health cannot be read from it."
		r.Caveats = []string{contextCoverageCaveat(&stats)}
		return r
	}
	compactionOK := stats.CompactionRate <= contextWatchCeiling
	sufficientSample := stats.Compacting >= contextMinSessionsForHealthy
	healthy := compactionOK && sufficientSample

	r.Read = readFor(healthy, "Healthy")
	r.Purity = contextPurity(stats.CompactionRate, sufficientSample)
	r.Takeaway = contextTakeaway(healthy, compactionOK, sufficientSample)
	if narrowestBasis(&stats) < stats.Count {
		r.Caveats = []string{contextCoverageCaveat(&stats)}
	}
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
