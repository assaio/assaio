package analyze

import (
	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/layer"
	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/report"
)

const (
	throughputName     = "throughput"
	throughputTitle    = "Throughput"
	throughputDescribe = "Total AI-added lines, lines per active day, top projects by lines, and the week-over-week trend."
	// throughputHowToRead is Result.HowToRead for this validator -- see its doc comment.
	throughputHowToRead = "Lines added is an output-volume signal and nothing more. More lines is not better work and fewer is not worse, so this reports the count and its direction and grades neither -- read it beside rework and model fit before deciding what a change in it means."
	// throughputTopN caps the top-projects bars shown in the report.
	throughputTopN = 5
	// throughputMinLinesForTrend is the line count either side of the comparison must reach
	// before a week-over-week change is worth stating -- a change from 1 to 2 lines is a 100%
	// swing on a trivial sample, not a direction. Either side, not the recent one: output
	// collapsing from thousands of lines to none is exactly the direction worth reading, and
	// requiring the recent side to be large would report it as unreadable.
	throughputMinLinesForTrend = 20
)

func init() { Register(throughputValidator{}) }

// throughputValidator reads raw AI-line output: how much code got added, where, and
// whether the pace is picking up or cooling off week over week.
type throughputValidator struct{}

func (throughputValidator) Name() string       { return throughputName }
func (throughputValidator) Title() string      { return throughputTitle }
func (throughputValidator) Describe() string   { return throughputDescribe }
func (throughputValidator) Layer() layer.Layer { return layer.Output } // the verdict is a line count and its trend

// Trending: both figures below compare the recent span against the one before it, so the history
// behind that earlier span is part of the claim (analyze.Trending).
func (throughputValidator) Trending() {}

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (throughputValidator) Analyze(in Input) Result {
	r := Result{Name: throughputName, Title: throughputTitle, Describe: throughputDescribe, HowToRead: throughputHowToRead}
	if len(in.Usage) == 0 {
		r.noData("active days", "No usage in this window.")
		return r
	}
	// The verdict is a line count, so a window whose sources record no line has no verdict to
	// give. Summing them anyway printed "AI lines total: 0" under an output-layer read, which
	// says the AI produced nothing rather than that nothing here counts what it produced.
	if !report.AnySourceAnswers(in.Usage, parser.SignalLinesAdded) {
		r.noData("active days", "No source in this window records a changed line, so output cannot be read from it.")
		return r
	}
	// inv is still needed for Days (Totals carries no distinct-day count); TotalLinesAdded
	// is the same sum as the prepared in.Totals.Lines, so figures read that instead.
	r.restsOn(activeDays(&in), "active days")
	inv := report.BuildInventory(in.Usage, in.Prices)
	recent, prior, changePct, trendOK := weekOverWeek(in.Usage, in.Now, in.Recent)
	sufficientVolume := max(recent, prior) >= throughputMinLinesForTrend

	r.Read = reportedRead
	r.Purity = neutralPurity
	r.Figures = []Figure{
		{Label: "AI lines total", Value: humanize.Int(in.Totals.Lines)},
		{Label: "lines/active-day", Value: perActiveDay(in.Totals.Lines, int64(inv.Days))},
		trendFigure(recent, prior, changePct, trendOK),
	}
	r.Bars = topProjectBars(in.ByProject, throughputTopN)
	r.BarsPseudonym = PseudonymProject
	r.Takeaway = throughputTakeaway(in.Totals.Lines, changePct, trendOK && sufficientVolume)
	r.Caveats = append(r.Caveats,
		"No verdict on purpose: a rising line count is an output measure, and promoting one to a claim about value is the most likely way this project starts lying (B180). The direction is here; what it is worth is not something a line count knows.")
	return r
}

// throughputTakeaway states the count and its direction without colouring either. A trend read
// off too few lines is reported as unreadable rather than as flat: those are different facts.
func throughputTakeaway(lines int64, changePct float64, readable bool) string {
	head := humanize.Int(lines) + " AI-added lines in this window. "
	if !readable {
		return head + "Too few recent lines to read a week-over-week direction from them."
	}
	return head + "Week over week the count is " + trendDirection(changePct) + "."
}

// trendDirection names the direction in words that carry no verdict: "up" and "down" describe a
// count, where "ramping" and "cooling" would describe a team. changePct is a fraction
// (recent-prior)/prior despite its name, so it is rendered as a share and never divided again.
func trendDirection(changePct float64) string {
	switch {
	case changePct > 0:
		return "up " + humanize.Percent(changePct)
	case changePct < 0:
		return "down " + humanize.Percent(-changePct)
	default:
		return "unchanged"
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
			Value: humanize.Int(kept[i].Lines) + " lines",
			Frac:  fracOf(kept[i].Lines, maxLines),
		}
	}
	return bars
}
