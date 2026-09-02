package analyze

import (
	"strings"

	"github.com/assaio/assaio/internal/layer"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/threshold"

	"github.com/assaio/assaio/internal/report"
)

const (
	reworkName     = "rework"
	reworkTitle    = "Rework & Rejection"
	reworkDescribe = "Within-session code churn and human tool-call rejections -- a directional friction proxy."
	// reworkHowToRead is Result.HowToRead for this validator -- see its doc comment.
	reworkHowToRead = "Rework and rejection are a directional friction proxy: how much AI-added code was undone inside the same transcript, and how often a human declined a proposed call. The link between AI churn and real bugs is contested, and nothing published says where either rate becomes a problem, so these are two figures to compare against your own work -- not a grade."
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
	complete := reworkKnown && rejectionKnown

	r.Read = reworkRead(complete)
	r.Purity = neutralPurity
	r.Figures = []Figure{
		reworkFigure(&churn),
		{
			Label: "rejection rate", Value: humanize.PercentOrDash(f.Rejected, f.Refusable, 1),
			Note: humanize.Int(f.Rejected) + " of " + humanize.Int(f.Refusable) + " calls that record a refusal",
		},
	}
	r.Caveats = reworkCaveats(rejectionKnown, churn.ExceedsItsWhole(), churn.Rows < len(in.Usage))
	if complete {
		r.Caveats = append(r.Caveats, unsourcedLine("a rework or rejection rate", ownHistoryWouldSettleIt))
	}
	// The published churn figure a reader will hold this rate up against, named rather than
	// left as a silence they fill with it (internal/threshold).
	r.Caveats = append(r.Caveats, citationLines(threshold.For(reworkName), in.Now)...)
	r.Takeaway = reworkTakeaway(complete, churn.ReworkRate, reworkKnown, rejectionRate)
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

// reworkRead separates having both rates from missing one. It grades neither: the 15% ceiling
// that used to decide "elevated" applied one picked number to two different quantities (B177),
// and a pair cannot be certified low from one of its halves either -- which is why a Codex
// store, recording an undone line but never a declined call, once read "worth a closer look" on
// every window forever and the ordering promoted it.
func reworkRead(complete bool) Read {
	if !complete {
		return noDataRead
	}
	return reportedRead
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

func reworkTakeaway(complete bool, reworkRate float64, reworkKnown bool, rejectionRate float64) string {
	if !complete {
		return "One half of this pair could not be measured here, so neither is read as the window's whole story -- see the caveats for which."
	}
	rework := "—"
	if reworkKnown {
		rework = humanize.Percent(reworkRate)
	}
	return "AI-added lines undone inside the same session: " + rework + "; proposed calls a human declined: " +
		humanize.Percent(rejectionRate) + ". Both are directional proxies for friction, and neither has a published line that would make one of them a problem."
}
