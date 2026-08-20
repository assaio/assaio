package analyze

import (
	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/layer"
	"github.com/assaio/assaio/internal/report"
)

const (
	contextName     = "context"
	contextTitle    = "Context Health"
	contextDescribe = "Conversation depth, peak context size, active time, and how often sessions hit compaction."
	// contextHowToRead is Result.HowToRead for this validator -- see its doc comment.
	contextHowToRead = "Frequent compaction means sessions are outgrowing their context window mid-task -- worth trying shorter, more focused sessions rather than reading it as a quality problem."
	// contextMinSessionsForHealthy is the minimum session count before the compaction rate is
	// worth printing at all: one session with zero compactions is a single data point, not a
	// rate. Matches adoption's session floor.
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
	sufficientSample := stats.Compacting >= contextMinSessionsForHealthy

	r.Read = contextRead(sufficientSample)
	r.Purity = neutralPurity
	r.Takeaway = contextTakeaway(sufficientSample, stats.CompactionRate)
	if narrowestBasis(&stats) < stats.Count {
		r.Caveats = []string{contextCoverageCaveat(&stats)}
	}
	if sufficientSample {
		r.Caveats = append(r.Caveats, unsourcedLine("a compaction rate", ownHistoryWouldSettleIt))
	}
	return r
}

// contextRead reports the compaction rate without grading it. The 20% ceiling that used to
// decide "healthy" was a number picked once (B177), and compaction is a property of task size
// against a model's context window as much as of how anyone works.
func contextRead(sufficientSample bool) Read {
	if !sufficientSample {
		return noDataRead
	}
	return reportedRead
}

func contextTakeaway(sufficientSample bool, compactionRate float64) string {
	if !sufficientSample {
		return "Too few sessions with compaction capture this window to read a compaction rate."
	}
	return humanize.Percent(compactionRate) + " of the sessions that record it hit context compaction. " +
		"A high rate points at sessions outgrowing their context window mid-task, which shorter, more focused sessions address -- it is not a quality problem, and nothing published says where it becomes one."
}
