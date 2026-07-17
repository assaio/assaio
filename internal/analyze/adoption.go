package analyze

import (
	"strconv"

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

func (adoptionValidator) Name() string     { return adoptionName }
func (adoptionValidator) Title() string    { return adoptionTitle }
func (adoptionValidator) Describe() string { return adoptionDescribe }

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (adoptionValidator) Analyze(in Input) Result {
	r := Result{Name: adoptionName, Title: adoptionTitle, Describe: adoptionDescribe, HowToRead: adoptionHowToRead}
	if len(in.Usage) == 0 && len(in.Sessions) == 0 {
		r.Read = noDataRead
		r.Takeaway = "No usage in this window."
		return r
	}
	// topN=0: dormant counts must reflect every stale project/tool, never a top-N cap.
	insights := report.BuildInsights(in.Usage, in.Prices, in.Now, in.Recent, 0)
	recent, prior, changePct, trendOK := weekOverWeek(in.Usage, in.Now, in.Recent)
	inv := insights.Inventory

	growing := trendOK && changePct > 0
	broadSignal := inv.Projects > 1
	sufficientSample := len(in.Sessions) >= adoptionMinSessionsForBroad
	broad := broadSignal && sufficientSample
	strong := growing || broad

	r.Read = readFor(strong, "Strong")
	r.Purity = adoptionPurity(inv.Projects, changePct, trendOK)
	r.Figures = []Figure{
		{Label: "sessions", Value: strconv.Itoa(len(in.Sessions))},
		{Label: "active days", Value: strconv.Itoa(inv.Days)},
		{Label: "projects", Value: strconv.Itoa(inv.Projects)},
		{Label: "sessions/active-day", Value: perActiveDay(int64(len(in.Sessions)), int64(inv.Days))},
		{
			Label: "dormant projects", Value: strconv.Itoa(len(insights.GoingStale)),
			Note: strconv.Itoa(len(insights.DormantTools)) + " tools unused",
		},
		trendFigure(recent, prior, changePct, trendOK),
	}
	r.Takeaway = adoptionTakeaway(strong, growing, broadSignal, sufficientSample)
	return r
}

func adoptionTakeaway(strong, growing, broadSignal, sufficientSample bool) string {
	switch {
	case strong && growing:
		return "Usage is broad and trending up week over week."
	case strong:
		return "Usage is broad across projects."
	case broadSignal && !sufficientSample:
		return "Usage spans more than one project, but too few sessions this window to call it broad with confidence."
	default:
		return "Usage is narrow and flat -- see the dormant projects/tools below."
	}
}

// adoptionPurity blends breadth (projects, saturating at adoptionBreadthTarget) and
// trend into a 0..1 "how broadly and actively AI is used" score.
func adoptionPurity(projects int, changePct float64, trendKnown bool) float64 {
	breadth := clamp01(float64(projects) / adoptionBreadthTarget)
	return clamp01((breadth + trendPurity(changePct, trendKnown)) / 2)
}
