package analyze

import (
	"strconv"

	"github.com/assaio/assaio/internal/layer"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/trace"
)

const (
	recoveryName     = "recovery"
	recoveryTitle    = "Failure Recovery"
	recoveryDescribe = "What a failed call, a refusal or a lost context costs next: the turns that follow one, against what a turn costs anywhere else in the window."
	// recoveryHowToRead is Result.HowToRead for this validator -- see its doc comment.
	recoveryHowToRead = "Failure is normal; an agent probes and adapts. This asks whether recovering from one is expensive here -- whether the turns after a failure cost more than the turns that do not follow one -- and names the sessions that stopped on one instead of getting past it."
	// recoveryMinTurns is the turn floor below which the ratio is arithmetic rather than evidence.
	recoveryMinTurns = 30
)

func init() { Register(recoveryValidator{}) }

// recoveryValidator reads what happens after something goes wrong in a sequence.
type recoveryValidator struct{}

func (recoveryValidator) Name() string       { return recoveryName }
func (recoveryValidator) Title() string      { return recoveryTitle }
func (recoveryValidator) Describe() string   { return recoveryDescribe }
func (recoveryValidator) Layer() layer.Layer { return layer.Activity } // what a turn costs after a failed one

// TraceScope declares the population: a person's own sessions. A sub-agent cannot abandon a run
// on its own account -- it returns to whoever launched it -- and an SDK caller's one-shot has no
// aftermath to measure.
func (recoveryValidator) TraceScope() string { return trace.Interactive }

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (v recoveryValidator) Analyze(in Input) Result {
	r := Result{Name: recoveryName, Title: recoveryTitle, Describe: recoveryDescribe, HowToRead: recoveryHowToRead}
	view := in.Trace.Scope(v.TraceScope())
	scoped := &view
	if view.Empty() {
		r.noData("sessions", noSequencesTakeaway(&in.Trace))
		return r
	}
	a := buildAftermath(scoped, in.Trace.Newest)
	r.overScope(&in, scoped, "sessions")

	ratio, rated := a.CostRatio()
	enough := rated && a.AfterTurns >= recoveryMinTurns
	r.Read = recoveryRead(enough)
	r.Purity = neutralPurity
	r.Figures = recoveryFigures(&a, ratio, rated)
	r.Takeaway = recoveryTakeaway(enough, ratio, &a)
	r.Caveats = append(r.Caveats, recoveryCannotDistinguish, recoveryCompositionCaveat)
	if enough {
		r.Caveats = append(r.Caveats, unsourcedLine("the cost of recovering from a failure", ownHistoryWouldSettleIt))
	}
	if a.Open > 0 {
		r.Caveats = append(r.Caveats, recoveryOpenCaveat(&a))
	}
	return r
}

func recoveryFigures(a *aftermath, ratio float64, rated bool) []Figure {
	return []Figure{
		recoveryCostFigure(a, ratio, rated),
		{
			Label: "sessions that stopped on a failure",
			Value: humanize.PercentOrDash(int64(len(a.Abandoned)), int64(a.Sequences-a.Open), 1),
			Note:  humanize.Int(int64(len(a.Abandoned))) + " of " + humanize.Int(int64(a.Sequences-a.Open)) + " whose ending is settled",
		},
		{
			Label: "steps run on a summarized context",
			Value: humanize.PercentOrDash(int64(a.StepsAfterCompaction), int64(a.Steps), 1),
			Note:  humanize.Int(int64(a.Compactions)) + " compaction(s) in this scope",
		},
	}
}

// recoveryCostFigure renders the aftermath's cost as a multiple of the window's own turn, and a
// dash when there were no turns on one side of it: a ratio against nothing is not 1.0.
func recoveryCostFigure(a *aftermath, ratio float64, rated bool) Figure {
	if !rated {
		return Figure{
			Label: "cost of the turns after a failure", Value: "—",
			Note: "no assistant turn follows a failure in this window",
		}
	}
	return Figure{
		Label: "cost of the turns after a failure",
		Value: ratioLabel(ratio) + " the window's own turn",
		Note: humanize.Int(a.AfterTurns) + " of " + humanize.Int(a.Turns) + " turns follow one of " +
			humanize.Int(int64(a.Failures)) + " failed or declined call(s) and " +
			humanize.Int(int64(a.Compactions)) + " context loss(es)",
	}
}

// recoveryRead reports the ratio and does not grade it. The 1.5x line that used to decide
// "contained" was a number picked once (B177): the only non-arbitrary point on this scale is
// 1.0, and with thousands of turns on each side a 3% difference clears any noise test while
// meaning nothing anyone would act on. An abandoned session is not folded in either -- it is a
// handful of sessions to open, not a rate.
func recoveryRead(enough bool) Read {
	if !enough {
		return noDataRead
	}
	return reportedRead
}

// ratioLabel renders a multiple the way the heaviest-day figure does, at the two decimals this
// metric turns on: 1.04 and 1.37 are the difference between a finding and an artifact.
func ratioLabel(ratio float64) string {
	return strconv.FormatFloat(ratio, 'f', 2, 64) + "\u00d7"
}

func recoveryTakeaway(enough bool, ratio float64, a *aftermath) string {
	switch {
	case !enough && a.Failures+a.Compactions == 0:
		return "Nothing failed, was declined, or lost its context in this scope, so there was no recovery to measure."
	case !enough:
		return "Too few turns follow a failure in this window to say what recovering from one costs."
	default:
		return "Turns following a failure cost " + ratioLabel(ratio) +
			" what a turn that follows none costs here." + recoveryAbandonedSentence(a)
	}
}

// recoveryAbandonedSentence adds the sessions worth opening, and says nothing when there are none
// -- an absent sentence is the honest form of "this did not happen".
func recoveryAbandonedSentence(a *aftermath) string {
	if len(a.Abandoned) == 0 {
		return ""
	}
	return " " + humanize.Int(int64(len(a.Abandoned))) + " session(s) stopped on a failed or declined call and never got past it."
}

const (
	// recoveryCannotDistinguish is this detector's refusal. Everything below looks the same on a
	// timeline as the thing it might actually be.
	recoveryCannotDistinguish = cannotDistinguish + ": a failure the agent expected -- probing for a file, testing a guess -- from one that cost it the thread; a session that stopped because the work was done from one that gave up; or a compaction at a natural boundary from one that lost something needed. The last visible step is also only the last one *stored*: a session whose transcript was deleted or whose steps fell past the horizon can end anywhere."
	// recoveryCompositionCaveat states the wrong answer, because the wrong answer is the tempting
	// one and it was measured on the way here.
	recoveryCompositionCaveat = "Read over turns, not steps, on purpose. A tool call carries no tokens, and the steps after a failure hold more assistant turns than the window does -- 49.2% against 47.7% inside this detector's ten-step window, 62.2% inside a three-step one. On the audited corpus that makes the per-step figure 1.06× where the per-turn figure is 1.03×, and at three steps 1.37× against 1.04×: the shorter the window, the more of it is the turn that answered the failure, and the more the per-step number measures the sample's composition rather than a cost."
)

func recoveryOpenCaveat(a *aftermath) string {
	return "Prov.: " + humanize.Int(int64(a.Open)) + " session(s) were still running when this was read and are excluded from the ending figure rather than counted as finished -- their last step is whatever they were doing, not how they turned out."
}
