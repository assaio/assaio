package analyze

import (
	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/layer"
	"github.com/assaio/assaio/internal/trace"
)

const (
	editLoopsName     = "edit-loops"
	editLoopsTitle    = "Repeat Edits"
	editLoopsDescribe = "How often a file goes back for another pass after a command ran, and which sessions stand far outside this window's own rate."
	// editLoopsHowToRead is Result.HowToRead for this validator -- see its doc comment.
	editLoopsHowToRead = "Going back to a file after running something is how work gets done -- write, run, fix. The rate is a baseline, not a fault, which is why the finding is the session that sits far outside this window's own, not the rate itself."
	// editLoopsMinSequences is the sequence floor below which a window has no typical session for
	// one to stand out from, on the same argument burn-anomaly's day floor makes.
	editLoopsMinSequences = 7
	editLoopsTopProjects  = 5
)

func init() { Register(editLoopsValidator{}) }

// editLoopsValidator reads how often a sequence returns to a file it has already edited, and
// which sequences do it far more than the rest of the window.
type editLoopsValidator struct{}

func (editLoopsValidator) Name() string       { return editLoopsName }
func (editLoopsValidator) Title() string      { return editLoopsTitle }
func (editLoopsValidator) Describe() string   { return editLoopsDescribe }
func (editLoopsValidator) Layer() layer.Layer { return layer.Activity } // how often an edit returns to a file

// TraceScope declares the population: a person's own sessions. Sub-agent sequences are a
// different animal and read differently -- 15.5% against 25.0% on the audited store -- and an
// SDK caller's one-shot run has no repeat behaviour to measure at all.
func (editLoopsValidator) TraceScope() string { return trace.Interactive }

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (v editLoopsValidator) Analyze(in Input) Result {
	r := Result{Name: editLoopsName, Title: editLoopsTitle, Describe: editLoopsDescribe, HowToRead: editLoopsHowToRead}
	view := in.Trace.Scope(v.TraceScope())
	scoped := &view
	if view.Empty() {
		r.noData("sessions", noSequencesTakeaway(&in.Trace))
		return r
	}
	p := buildRepeats(scoped)
	r.overScope(&in, scoped, "sessions")
	if p.Edits == 0 {
		r.noData("sessions", "No session in scope edited a file this window, so nothing could be returned to.")
		return r
	}

	r.Read = editLoopsRead(&p)
	r.Purity = editLoopsPurity(&p)
	r.Figures = editLoopsFigures(&p)
	r.Bars = editLoopsBars(&p)
	r.BarsPseudonym = PseudonymProject
	r.Takeaway = editLoopsTakeaway(&p)
	r.Caveats = append(r.Caveats, editLoopsCannotDistinguish, editLoopsNoCostClaim)
	if p.Untargeted > 0 {
		r.Caveats = append(r.Caveats, editLoopsUntargetedCaveat(&p))
	}
	return r
}

func editLoopsFigures(p *repeatProfile) []Figure {
	figures := []Figure{{
		Label: "repeat-edit rate",
		Value: humanize.PercentOrDash(int64(p.Repeats), int64(p.Edits), 1),
		Note:  humanize.Int(int64(p.Repeats)) + " of " + humanize.Int(int64(p.Edits)) + " edits went back after a command ran",
	}}
	figures = append(figures, editLoopsOutlierFigure(p))
	if worst, ok := p.Worst(); ok {
		figures = append(figures, Figure{
			Label: "heaviest session",
			Value: humanize.PercentAt(worst.Rate(), 0),
			Note:  humanize.Int(int64(worst.Edits)) + " edits, " + humanize.Int(int64(worst.Repeats)) + " of them repeats",
		})
	}
	return figures
}

// editLoopsOutlierFigure states how many sessions stand outside the window, and says so as a
// dash rather than a zero when the window could not be judged: "none stand out" and "there was
// nothing to stand out from" are different answers.
func editLoopsOutlierFigure(p *repeatProfile) Figure {
	if !p.Spread {
		note := humanize.Int(int64(len(p.Judged))) + " session(s) had enough edits to rank, too few to read a spread from"
		if p.Unreachable {
			note = humanize.Int(int64(len(p.Judged))) + " session(s) ranked, but they vary too widely for a line any of them could cross"
		}
		return Figure{Label: "sessions standing out", Value: "—", Note: note}
	}
	return Figure{
		Label: "sessions standing out",
		Value: humanize.Int(int64(len(p.Outliers))),
		Note:  "above " + humanize.PercentAt(p.Floor, 0) + ", of " + humanize.Int(int64(len(p.Judged))) + " with enough edits to rank",
	}
}

func editLoopsBars(p *repeatProfile) []Bar {
	projects := p.byProject()
	if len(projects) > editLoopsTopProjects {
		projects = projects[:editLoopsTopProjects]
	}
	if len(projects) == 0 {
		return nil
	}
	// byProject sorts by rate descending, so the first is the maximum this list scales against.
	top := projects[0].Rate()
	out := make([]Bar, 0, len(projects))
	for i := range projects {
		var frac float64
		if top > 0 {
			frac = projects[i].Rate() / top
		}
		out = append(out, Bar{
			Label: projects[i].Project,
			Value: humanize.PercentAt(projects[i].Rate(), 0) + " of " + humanize.Int(int64(projects[i].Edits)) + " edits",
			Frac:  frac,
		})
	}
	return out
}

// editLoopsRead withholds a verdict unless the window had a spread to judge against. A rate on its
// own is not a verdict here: there is no published threshold for it, and inventing one would ship
// the maintainer's 25% as everyone's line.
func editLoopsRead(p *repeatProfile) Read {
	if !p.Spread {
		return noDataRead
	}
	return readFor(len(p.Outliers) == 0, "Even")
}

// editLoopsPurity is how much of the window's editing sits outside the sessions that stand out.
// An unjudgeable window gets the neutral 0.5 rather than a full gauge earned by having too little
// data to find anything.
func editLoopsPurity(p *repeatProfile) float64 {
	if !p.Spread {
		return 0.5
	}
	return clamp01(1 - shareOf(int64(p.OutlierEdits()), int64(p.Edits)))
}

func editLoopsTakeaway(p *repeatProfile) string {
	switch {
	case p.Unreachable:
		return "This window's sessions vary too widely for any one of them to stand outside the rest: the line that would mark an outlier sits above the highest rate " +
			humanize.Int(int64(len(p.Judged))) + " ranked session(s) could reach. The window rate above is still the window's."
	case !p.Spread && len(p.Judged) == 0:
		return "No session in scope reached " + humanize.Int(editLoopsMinEdits) + " edits, which is the fewest a repeat rate can be read from."
	case !p.Spread:
		return "Too few sessions with enough edits to say which of them stands out; the window rate above is still the window's."
	case len(p.Outliers) == 0:
		return "No session returns to its files far more than the rest of this window -- the repeat rate is even across the work."
	default:
		return humanize.Int(int64(len(p.Outliers))) + " session(s) return to the same file far more than the rest of this window, holding " +
			humanize.Percent(shareOf(int64(p.OutlierEdits()), int64(p.Edits))) + " of its edits -- worth opening one to see whether it was a hard change or a loop."
	}
}

const (
	// editLoopsCannotDistinguish is the refusal this detector ships with. A pattern is not a
	// fault, and every reading below is the same shape on the timeline.
	editLoopsCannotDistinguish = cannotDistinguish + ": a hard bug from a loop, a deliberate refactor of one file from thrashing on it, a hub file every change touches from a file being fought, or a red-green test cycle where every pass was intended. A step also carries no command identity, so \"a command ran\" is all this sees -- never that it was a test, and never that it ran anything at all: retrieving a background shell's output (BashOutput) or killing one is classified as a command like any other, so a session polling a long-running server can read as returning to a file it never re-ran anything against (`B170`). Ziegler et al. make the same point about their own headline metric: it \"cannot be recommended as singular and ultimate criterion of quality\" (arXiv:2205.06537)."
	// editLoopsNoCostClaim exists because the obvious next figure is wrong. Measured on the
	// maintainer's store: the stretches between a file's first and last edit hold 70.2% of the
	// window's tokens across 67.5% of its steps -- 1.04x, which is the window's own rate. Naming
	// a cost here would dress proportional spend up as waste the repeats caused.
	editLoopsNoCostClaim = "No cost is claimed. Spend inside these stretches was measured at 1.04x the window's per-step average, i.e. ordinary: a repeat costs what the work around it costs, and calling it waste would be a claim this cannot support."
)

func editLoopsUntargetedCaveat(p *repeatProfile) string {
	return "Prov.: " + humanize.Int(int64(p.Untargeted)) + " edit(s) in scope name no file and are absent from the denominator rather than counted as first passes -- a call whose file this build could not read, or a relative path with no working directory to resolve it against."
}
