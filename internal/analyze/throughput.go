package analyze

import (
	"sort"
	"strconv"
	"time"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
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
	r.Bars = topProjectsByLines(in.Usage, in.Prices, in.Now, in.Recent, throughputTopN)
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

// topProjectsByLines ranks the recent-window projects by AI-added lines descending, not
// cost, so the bars' visual order matches the lines each one shows -- a project with
// fewer lines never outranks one with more just because it cost more. topN <= 0 is
// unlimited.
func topProjectsByLines(rows []store.UsageRow, prices pricing.Table, now time.Time, window time.Duration, topN int) []Bar {
	// "project" is a hardcoded valid dimension; the error return is unreachable here.
	projects, _ := report.BuildEffectiveness(recentRows(rows, now, window), prices, "project")
	sort.SliceStable(projects, func(i, j int) bool { return projects[i].LinesAdded > projects[j].LinesAdded })
	if topN > 0 && len(projects) > topN {
		projects = projects[:topN]
	}

	var maxLines int64
	if len(projects) > 0 {
		maxLines = projects[0].LinesAdded
	}
	bars := make([]Bar, len(projects))
	for i, p := range projects {
		bars[i] = Bar{Label: groupLabel(p.Group), Value: strconv.FormatInt(p.LinesAdded, 10) + " lines", Frac: fracOf(p.LinesAdded, maxLines)}
	}
	return bars
}
