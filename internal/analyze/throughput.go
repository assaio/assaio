package analyze

import (
	"strconv"

	"github.com/assaio/assaio/internal/report"
)

const (
	throughputName     = "throughput"
	throughputTitle    = "Throughput"
	throughputDescribe = "Total AI-added lines, lines per active day, top projects by lines, and the week-over-week trend."
	// throughputHowToRead is Result.HowToRead for this validator -- see its doc comment.
	throughputHowToRead = "Lines added is an output-volume signal, not a quality score -- read the trend alongside rework and model fit before deciding more code means more value."
	// throughputTopN caps the top-projects bars shown in the report.
	throughputTopN = 5
	// throughputMinLinesForRamping is the minimum recent-window AI-line count a positive
	// trend requires before it can carry a favorable Ramping read -- a change from 1 to
	// 2 lines is a 100% swing on a trivial sample, not a real ramp.
	throughputMinLinesForRamping = 20
)

func init() { Register(throughputValidator{}) }

// throughputValidator reads raw AI-line output: how much code got added, where, and
// whether the pace is picking up or cooling off week over week.
type throughputValidator struct{}

func (throughputValidator) Name() string     { return throughputName }
func (throughputValidator) Title() string    { return throughputTitle }
func (throughputValidator) Describe() string { return throughputDescribe }

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (throughputValidator) Analyze(in Input) Result {
	r := Result{Name: throughputName, Title: throughputTitle, Describe: throughputDescribe, HowToRead: throughputHowToRead}
	if len(in.Usage) == 0 {
		r.Read = noDataRead
		r.Takeaway = "No usage in this window."
		return r
	}
	// inv is still needed for Days (Totals carries no distinct-day count); TotalLinesAdded
	// is the same sum as the prepared in.Totals.Lines, so figures read that instead.
	inv := report.BuildInventory(in.Usage, in.Prices)
	recent, prior, changePct, trendOK := weekOverWeek(in.Usage, in.Now, in.Recent)
	growthSignal := trendOK && changePct > 0
	sufficientVolume := recent >= throughputMinLinesForRamping
	ramping := growthSignal && sufficientVolume

	r.Read = readFor(ramping, "Ramping")
	r.Purity = trendPurity(changePct, trendOK)
	r.Figures = []Figure{
		{Label: "AI lines total", Value: strconv.FormatInt(in.Totals.Lines, 10)},
		{Label: "lines/active-day", Value: perActiveDay(in.Totals.Lines, int64(inv.Days))},
		trendFigure(recent, prior, changePct, trendOK),
	}
	r.Bars = topProjectBars(in.ByProject, throughputTopN)
	r.BarsAreProjects = true
	r.Takeaway = throughputTakeaway(ramping, growthSignal && !sufficientVolume)
	return r
}

func throughputTakeaway(ramping, trendUpButTrivial bool) string {
	switch {
	case ramping:
		return "AI-line output is ramping up week over week."
	case trendUpButTrivial:
		return "AI-line output ticked up, but volume is too low this window to call it a ramp."
	default:
		return "AI-line output is flat or down week over week."
	}
}

// topProjectBars renders the whole-window top projects by AI-added lines, so the bars
// break down the same window as the "AI lines total" figure above them rather than a
// recent-only sub-window. in.ByProject is already sorted by Lines descending, so its
// first entry is the max used to scale Frac. topN <= 0 is unlimited.
func topProjectBars(projects []ProjectStat, topN int) []Bar {
	kept := make([]ProjectStat, 0, len(projects))
	for i := range projects {
		if projects[i].Lines > 0 { // a "top projects by lines" list shouldn't pad with 0-line rows
			kept = append(kept, projects[i])
		}
	}
	if topN > 0 && len(kept) > topN {
		kept = kept[:topN]
	}
	var maxLines int64
	if len(kept) > 0 {
		maxLines = kept[0].Lines
	}
	bars := make([]Bar, len(kept))
	for i := range kept {
		bars[i] = Bar{
			Label: groupLabel(kept[i].Project),
			Value: strconv.FormatInt(kept[i].Lines, 10) + " lines",
			Frac:  fracOf(kept[i].Lines, maxLines),
		}
	}
	return bars
}
