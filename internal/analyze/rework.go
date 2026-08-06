package analyze

import (
	"strconv"
	"strings"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/parser"

	"github.com/assaio/assaio/internal/report"
)

const (
	reworkName     = "rework"
	reworkTitle    = "Rework & Rejection"
	reworkDescribe = "Within-session code churn and human tool-call rejections -- a directional friction proxy."
	// reworkHowToRead is Result.HowToRead for this validator -- see its doc comment.
	reworkHowToRead = "Elevated rework or rejection flags friction worth a closer look at specific sessions, but the link between AI churn and real bugs is still contested, so treat it as a lead, not a verdict."
	// reworkWatchCeiling is the rework-rate/rejection-rate threshold above which
	// friction is flagged for a closer look.
	reworkWatchCeiling = 0.15
)

func init() { Register(reworkValidator{}) }

// reworkValidator reads within-session churn and rejection friction: AI-added code that
// got removed again in the same transcript, and tool proposals the human declined.
type reworkValidator struct{}

func (reworkValidator) Name() string     { return reworkName }
func (reworkValidator) Title() string    { return reworkTitle }
func (reworkValidator) Describe() string { return reworkDescribe }

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (reworkValidator) Analyze(in Input) Result {
	r := Result{Name: reworkName, Title: reworkTitle, Describe: reworkDescribe, HowToRead: reworkHowToRead}
	if len(in.Usage) == 0 {
		r.noData("active days", "No usage in this window.")
		return r
	}
	r.restsOn(activeDays(&in), "active days")
	// BuildChurn keeps only the sources that record an undone line: one recording changed
	// lines but never a rework line would put its whole output in the denominator against a
	// structural zero, lowering the rate with code nobody watched being undone.
	churn := report.BuildChurn(in.Usage)
	capable := churn.Rows > 0
	r.covering(shareOf(churn.Tokens, in.Totals.Tokens))
	// Refusals have their own capable subset, counted once in buildFriction so this figure
	// and friction's cannot disagree about the same rate.
	f := buildFriction(in.Usage)
	rejectionKnown := f.Refusable > 0
	rejectionRate := shareOf(f.Rejected, f.Refusable)
	low := churn.ReworkRate <= reworkWatchCeiling && rejectionLow(rejectionRate, rejectionKnown)

	r.Read = reworkRead(low, capable)
	r.Purity = reworkPurity(churn.ReworkRate, rejectionRate, rejectionKnown)
	r.Figures = []Figure{
		reworkFigure(&churn),
		{
			Label: "rejection rate", Value: humanize.PercentOrDash(f.Rejected, f.Refusable, 1),
			Note: strconv.FormatInt(f.Rejected, 10) + " of " + strconv.FormatInt(f.Refusable, 10) + " calls that record a refusal",
		},
	}
	r.Caveats = reworkCaveats(rejectionKnown, churn.Rows < len(in.Usage))
	r.Takeaway = reworkTakeaway(low, capable)
	return r
}

// reworkFigure withholds the rate when no source in the window records an undone line: the
// churn proxy would otherwise print the 0% its own empty denominator produces.
func reworkFigure(churn *report.ChurnStat) Figure {
	if churn.Rows == 0 {
		return Figure{Label: "rework", Value: "—", Note: "no source here records an undone line"}
	}
	return Figure{
		Label: "rework", Value: humanize.PercentOrDash(churn.ReworkLines, churn.LinesAdded, 0),
		Note: strconv.FormatInt(churn.ReworkLines, 10) + " lines, within-session thrash proxy",
	}
}

// reworkRead withholds the verdict when neither half could be measured, rather than
// certifying a window as low-friction from two silences.
func reworkRead(low, capable bool) Read {
	if !capable {
		return noDataRead
	}
	return readFor(low, "Low")
}

// rejectionLow reports whether the rejection rate is confirmed within the watch ceiling.
// Unknown (no tool calls recorded this window) never reads as low: there is no rejection
// signal to confirm, so the pair can't be certified low on a fabricated zero.
func rejectionLow(rejectionRate float64, known bool) bool {
	return known && rejectionRate <= reworkWatchCeiling
}

// reworkPurity averages known rates only: an unknown rejection rate (no tool calls) is
// excluded rather than folded in as a fabricated zero, which would inflate purity above
// what the actually-observed rework signal supports.
func reworkPurity(reworkRate, rejectionRate float64, rejectionKnown bool) float64 {
	if !rejectionKnown {
		return clamp01(1 - reworkRate)
	}
	return clamp01(1 - (reworkRate+rejectionRate)/2)
}

func reworkCaveats(rejectionKnown, partialChurn bool) []string {
	caveats := []string{"Evidence on AI churn's real-world impact is contested; bug/survival impact needs the server stage."}
	if !rejectionKnown {
		caveats = append(caveats, "No source in this window records a declined tool call -- the rejection rate is unconfirmed, not zero.")
	}
	if partialChurn {
		caveats = append(caveats, "Prov.: rework reads the sources that record an undone line ("+
			strings.Join(sourcesAnswering(parser.SignalReworkLines), ", ")+
			"); lines from other sources are absent from the rate, not churn-free.")
	}
	return caveats
}

func reworkTakeaway(low, capable bool) string {
	switch {
	case !capable:
		return "No source in this window records a line undone later in the same session, so churn cannot be read from it."
	case low:
		return "Rework and rejection are both low."
	default:
		return "Rework or rejection is elevated or unconfirmed -- worth a closer look."
	}
}
