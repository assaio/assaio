package analyze

import (
	"strconv"

	"github.com/assaio/assaio/internal/layer"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/report"
)

const (
	adoptionName     = "adoption"
	adoptionTitle    = "Adoption & Usage Breadth"
	adoptionDescribe = "Sessions, active days, and project/tool breadth: how broad AI usage is, and whether it's growing."
	// adoptionHowToRead is Result.HowToRead for this validator -- see its doc comment.
	adoptionHowToRead = "Breadth and growth show how far AI usage has spread across projects, not how good any of it is -- a narrow or flat trend is a cue to invest in onboarding, not a mark against anyone."
	// adoptionBreadthTarget is the project count at which Purity's breadth component
	// saturates to 1.
	adoptionBreadthTarget = 5
	// adoptionMinSessionsForBroad is the minimum total session count "broad" (more than
	// one project) requires before it can carry a favorable Strong read on its own --
	// 2 projects with 1 session each is not yet broad usage, just two data points.
	// Growing (a real week-over-week trend) is not subject to this floor: trendOK
	// already requires a nonzero prior window, its own evidence-of-activity guard.
	adoptionMinSessionsForBroad = 3
)

func init() { Register(adoptionValidator{}) }

// adoptionValidator reads how broadly AI tools are used: session volume, active days,
// project/tool breadth, and whether usage is growing or shrinking week over week.
type adoptionValidator struct{}

func (adoptionValidator) Name() string       { return adoptionName }
func (adoptionValidator) Title() string      { return adoptionTitle }
func (adoptionValidator) Describe() string   { return adoptionDescribe }
func (adoptionValidator) Layer() layer.Layer { return layer.Activity } // sessions, active days and breadth

// Trending: both figures below compare the recent span against the one before it, so the history
// behind that earlier span is part of the claim (analyze.Trending).
func (adoptionValidator) Trending() {}

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (adoptionValidator) Analyze(in Input) Result {
	r := Result{Name: adoptionName, Title: adoptionTitle, Describe: adoptionDescribe, HowToRead: adoptionHowToRead}
	if len(in.Usage) == 0 && len(in.Sessions) == 0 {
		r.noData("active days", "No usage in this window.")
		return r
	}
	// topN=0: dormant counts must reflect every stale project/tool, never a top-N cap.
	r.restsOn(activeDays(&in), "active days")
	insights := report.BuildInsights(in.Usage, in.Prices, in.Now, in.Recent, 0)
	recent, prior, changePct, trendOK := weekOverWeek(in.Usage, in.Now, in.Recent)
	inv := insights.Inventory

	growing := trendOK && changePct > 0
	// Breadth is a count of projects, and a project comes from a session's working directory.
	// A source that records none -- Antigravity CLI writes no cwd anywhere -- leaves every row
	// unattributed, and "0 projects" then reads as usage that has spread nowhere rather than a
	// breadth nothing measured. The trend half still stands: it counts sessions, not projects.
	breadthKnown := inv.Unattributed < len(in.Usage)
	broadSignal := breadthKnown && inv.Projects > 1
	sufficientSample := len(in.Sessions) >= adoptionMinSessionsForBroad
	broad := broadSignal && sufficientSample
	strong := growing || broad

	r.Read = readFor(strong, "Strong")
	if !breadthKnown && !trendOK {
		r.Read = noDataRead
	}
	r.Purity = adoptionPurity(inv.Projects, changePct, trendOK, breadthKnown)
	r.Figures = []Figure{
		{Label: "sessions", Value: humanize.Int(int64(len(in.Sessions)))},
		{Label: "active days", Value: strconv.Itoa(inv.Days)},
		projectsFigure(inv.Projects, breadthKnown),
		{Label: "sessions/active-day", Value: perActiveDay(int64(len(in.Sessions)), int64(inv.Days))},
		dormantFigure(len(insights.GoingStale), len(insights.DormantTools), breadthKnown),
		trendFigure(recent, prior, changePct, trendOK),
	}
	r.Takeaway = adoptionTakeaway(strong, growing, broadSignal, sufficientSample, breadthKnown)
	return r
}

// dormantFigure counts the projects that have gone quiet, and withholds that half where no row
// names a project at all: "0 dormant" over a window with no projects is a clean bill drawn from
// the same silence the breadth figure withholds on. The unused-tool count beside it is read from
// the tool column and stands either way.
func dormantFigure(stale, unusedTools int, breadthKnown bool) Figure {
	f := Figure{Label: "dormant projects", Value: "—", Note: strconv.Itoa(unusedTools) + " tools unused"}
	if breadthKnown {
		f.Value = strconv.Itoa(stale)
	}
	return f
}

// projectsFigure prints "—" where no row in the window names a project at all. Zero projects
// and a window whose source cannot name one are different facts, and only the first is a
// finding about how far AI use has spread.
func projectsFigure(projects int, breadthKnown bool) Figure {
	if !breadthKnown {
		return Figure{Label: "projects", Value: "—", Note: "no source here records one"}
	}
	return Figure{Label: "projects", Value: strconv.Itoa(projects)}
}

func adoptionTakeaway(strong, growing, broadSignal, sufficientSample, breadthKnown bool) string {
	switch {
	case strong && growing:
		return "Usage is broad and trending up week over week."
	case strong:
		return "Usage is broad across projects."
	case broadSignal && !sufficientSample:
		return "Usage spans more than one project, but too few sessions this window to call it broad with confidence."
	case !breadthKnown && growing:
		return "Usage is trending up week over week. No source here records a working directory, so how broadly it has spread cannot be read."
	case !breadthKnown:
		return "No source in this window records a working directory, so breadth cannot be read from it."
	default:
		return "Usage is narrow and flat -- see the dormant projects/tools below."
	}
}

// adoptionPurity blends breadth (projects, saturating at adoptionBreadthTarget) and
// trend into a 0..1 "how broadly and actively AI is used" score. An unknowable breadth is the
// neutral half rather than the worst one: averaging a zero in would score a window down for a
// field its source never wrote.
func adoptionPurity(projects int, changePct float64, trendKnown, breadthKnown bool) float64 {
	breadth := neutralPurity
	if breadthKnown {
		breadth = clamp01(float64(projects) / adoptionBreadthTarget)
	}
	return clamp01((breadth + trendPurity(changePct, trendKnown)) / 2)
}
