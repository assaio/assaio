package analyze

import (
	"strings"

	"github.com/assaio/assaio/internal/layer"

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

func (reworkValidator) Name() string       { return reworkName }
func (reworkValidator) Title() string      { return reworkTitle }
func (reworkValidator) Describe() string   { return reworkDescribe }
func (reworkValidator) Layer() layer.Layer { return layer.Output } // the verdict is a share of AI-added lines

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
	r.covering(shareOf(churn.Tokens, in.Totals.Tokens))
	// Refusals have their own capable subset, counted once in buildFriction so this figure
	// and friction's cannot disagree about the same rate.
	f := buildFriction(in.Usage)
	rejectionKnown := f.Refusable > 0
	rejectionRate := shareOf(f.Rejected, f.Refusable)
	reworkKnown := reworkMeasurable(&churn)
	elevated := (reworkKnown && churn.ReworkRate > reworkWatchCeiling) ||
		(rejectionKnown && rejectionRate > reworkWatchCeiling)

	r.Read = reworkRead(elevated, reworkKnown && rejectionKnown)
	r.Purity = reworkPurity(churn.ReworkRate, reworkKnown, rejectionRate, rejectionKnown)
	if r.Read == noDataRead {
		r.Purity = neutralPurity
	}
	r.Figures = []Figure{
		reworkFigure(&churn),
		{
			Label: "rejection rate", Value: humanize.PercentOrDash(f.Rejected, f.Refusable, 1),
			Note: humanize.Int(f.Rejected) + " of " + humanize.Int(f.Refusable) + " calls that record a refusal",
		},
	}
	r.Caveats = reworkCaveats(rejectionKnown, churn.ExceedsItsWhole(), churn.Rows < len(in.Usage))
	r.Takeaway = reworkTakeaway(elevated, reworkKnown && rejectionKnown)
	return r
}

// reworkFigure withholds the rate whenever it is not a share of this window: the churn proxy
// would otherwise print the 0% its own empty denominator produces, or the 8000% a window that
// opened mid-session produces from removals it never counted the additions for.
func reworkFigure(churn *report.ChurnStat) Figure {
	switch {
	case churn.Rows == 0:
		return Figure{Label: "rework", Value: "—", Note: "no source here records an undone line"}
	case churn.ExceedsItsWhole():
		return Figure{Label: "rework", Value: "—", Note: report.ChurnBoundaryNote(churn)}
	}
	return Figure{
		Label: "rework", Value: humanize.PercentOrDash(churn.ReworkLines, churn.LinesAdded, 0),
		Note: humanize.Int(churn.ReworkLines) + " lines, within-session thrash proxy",
	}
}

// reworkMeasurable reports whether the churn rate is a share of this window: a source that
// records an undone line, a denominator to divide by, and no removal of a line the window
// never counted as added.
func reworkMeasurable(c *report.ChurnStat) bool {
	return c.Rows > 0 && c.LinesAdded > 0 && !c.ExceedsItsWhole()
}

// reworkRead gives the three answers this pair of rates can support. A measured rate above its
// ceiling is a finding and always shows. Everything measured and low is a clean bill. Anything
// else -- a half nothing in the window could measure -- is withheld: a pair cannot be certified
// low from one of its halves, and it cannot be flagged from a silence either. Flagging was the
// old behaviour, so a Codex store, which records an undone line but never a declined call, read
// "worth a closer look" on every window forever and the ordering promoted it.
func reworkRead(elevated, complete bool) Read {
	switch {
	case elevated:
		return readFor(false, "Low")
	case complete:
		return readFor(true, "Low")
	default:
		return noDataRead
	}
}

// reworkPurity averages the known rates only: folding an unmeasured one in as a zero credits the
// window for friction nobody could observe.
func reworkPurity(reworkRate float64, reworkKnown bool, rejectionRate float64, rejectionKnown bool) float64 {
	var sum float64
	known := 0
	if reworkKnown {
		sum, known = sum+reworkRate, known+1
	}
	if rejectionKnown {
		sum, known = sum+rejectionRate, known+1
	}
	if known == 0 {
		return neutralPurity
	}
	return clamp01(1 - sum/float64(known))
}

func reworkCaveats(rejectionKnown, cutWindow, partialChurn bool) []string {
	caveats := []string{"Evidence on AI churn's real-world impact is contested; bug/survival impact needs the server stage."}
	if !rejectionKnown {
		caveats = append(caveats, "No source in this window records a declined tool call -- the rejection rate is unconfirmed, not zero.")
	}
	if cutWindow {
		caveats = append(caveats, "The rework rate is withheld: this window holds removals of lines it never counted as added, so the ratio is not a share of it. A longer --since covers the sessions whole.")
	}
	if partialChurn {
		caveats = append(caveats, "Prov.: rework reads the sources that record an undone line ("+
			strings.Join(sourcesAnswering(parser.SignalReworkLines), ", ")+
			"); lines from other sources are absent from the rate, not churn-free.")
	}
	return caveats
}

func reworkTakeaway(elevated, complete bool) string {
	switch {
	case elevated:
		return "A measured friction rate is elevated -- worth a closer look."
	case complete:
		return "Rework and rejection are both low."
	default:
		return "One half of this pair could not be measured here, so the window is not certified low -- see the caveats for which."
	}
}
