package analyze

import "strconv"

const (
	concentrationName     = "concentration"
	concentrationTitle    = "Spend Concentration"
	concentrationDescribe = "How token spend spreads across projects, and whether the heaviest spenders produced output to match."
	// concentrationHowToRead is Result.HowToRead for this validator -- see its doc comment.
	concentrationHowToRead = "Concentration alone is neither good nor bad -- one project can legitimately own the work. Read the gap instead: a project holding a far larger share of the tokens than of the AI-written lines is where spend is not converting into code."
	// concentrationGapCeiling is the token-share-minus-line-share gap above which a
	// project's spend reads as out of step with the output it bought.
	concentrationGapCeiling = 0.2
	// concentrationMinProjects is the floor below which concentration is undefined: a lone
	// project trivially holds 100% of both shares, so no gap can exist to measure.
	concentrationMinProjects = 2
	// concentrationMinTokenShare keeps a trivially small project from carrying the verdict:
	// one on 1% of tokens and no lines is noise, not a spend problem.
	concentrationMinTokenShare = 0.05
	// concentrationUnattributedFloor is the unattributed-token share above which the
	// caveat about tools that log no working directory is worth showing.
	concentrationUnattributedFloor = 0.1
	concentrationTopN              = 5
)

func init() { Register(concentrationValidator{}) }

// concentrationValidator reads how token spend distributes across projects and where it
// diverges most from the AI-line output that spend produced.
type concentrationValidator struct{}

func (concentrationValidator) Name() string     { return concentrationName }
func (concentrationValidator) Title() string    { return concentrationTitle }
func (concentrationValidator) Describe() string { return concentrationDescribe }

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (concentrationValidator) Analyze(in Input) Result {
	r := Result{Name: concentrationName, Title: concentrationTitle, Describe: concentrationDescribe, HowToRead: concentrationHowToRead}
	if len(in.ByProject) == 0 || in.Totals.Tokens == 0 {
		r.Read = noDataRead
		r.Takeaway = "No usage in this window."
		return r
	}
	all := projectShares(in.ByProject, in.Totals.Lines)
	// Every statistic below is over named projects only: the unattributed bucket pools
	// every tool that logs no working directory, so counting it as a project inflates the
	// project count and the concentration score alike. Its size is disclosed as a caveat.
	named := namedProjects(all)
	r.restsOn(len(named), "projects")
	gap, gapFound := widestSpendGap(named)
	// A gap has to have been computed for "aligned" to mean anything: with every project
	// below the size floor there is nothing to align, and calling that a pass is a green
	// check for an examination that never ran.
	measurable := len(named) >= concentrationMinProjects && gapFound
	aligned := measurable && gap <= concentrationGapCeiling

	r.Read = concentrationRead(measurable, aligned)
	r.Purity = concentrationPurity(measurable, gap, gapFound)
	r.Figures = []Figure{
		{Label: "projects", Value: strconv.Itoa(len(named))},
		{Label: "top project", Value: formatPercent(topShare(named), 0), Note: "of tokens"},
		{Label: "top 3", Value: formatPercent(topNShare(named, 3), 0), Note: "of tokens"},
		{Label: "concentration", Value: strconv.FormatFloat(giniOfShares(named), 'f', 2, 64), Note: "0 even · 1 concentrated"},
		concentrationGapFigure(gap, gapFound),
	}
	r.Bars = concentrationBars(named, concentrationTopN)
	r.BarsPseudonym = PseudonymProject
	r.Takeaway = concentrationTakeaway(measurable, aligned, gapFound)
	if unattributed := unattributedShare(all); unattributed >= concentrationUnattributedFloor {
		r.Caveats = append(r.Caveats, "Prov.: "+formatPercent(unattributed, 0)+" of tokens are unattributed -- tools that log no working directory (Gemini CLI, Cline) cannot be assigned to a project.")
	}
	return r
}

// concentrationRead reports the neutral no-verdict Read when a single project makes the
// gap undefined, rather than a favorable one earned by having nothing to compare.
func concentrationRead(measurable, aligned bool) Read {
	if !measurable {
		return noDataRead
	}
	return readFor(aligned, "Aligned")
}

// concentrationPurity gauges how closely spend tracks output: full when the widest gap is
// zero, falling to zero as that gap approaches every token being spent where no line was
// written. An undefined gap yields the neutral 0.5 the other validators use for
// "not enough evidence yet".
func concentrationPurity(measurable bool, gap float64, gapFound bool) float64 {
	if !measurable || !gapFound {
		return 0.5
	}
	return clamp01(1 - gap)
}

// concentrationGapFigure renders the widest gap in share points, never naming the project
// it belongs to: Figures are not pseudonymized under --anonymize, so the project behind
// the gap is disclosed only through Bars, which are.
func concentrationGapFigure(gap float64, found bool) Figure {
	f := Figure{Label: "widest spend gap", Value: "—", Note: "tokens minus lines, share points"}
	if found {
		f.Value = strconv.FormatFloat(gap*100, 'f', 0, 64) + "pp"
	}
	return f
}

// concentrationBars ranks projects by token share, showing the line share beside it so the
// gap driving the verdict is visible per project. topN <= 0 is unlimited.
func concentrationBars(shares []projectShare, topN int) []Bar {
	kept := shares
	if topN > 0 && len(kept) > topN {
		kept = kept[:topN]
	}
	var maxShare float64
	if len(kept) > 0 {
		maxShare = kept[0].TokenShare
	}
	bars := make([]Bar, len(kept))
	for i := range kept {
		frac := 0.0
		if maxShare > 0 {
			frac = kept[i].TokenShare / maxShare
		}
		bars[i] = Bar{
			Label: groupLabel(kept[i].Project),
			Value: formatPercent(kept[i].TokenShare, 0) + " tokens · " + formatPercent(kept[i].LineShare, 0) + " lines",
			Frac:  frac,
		}
	}
	return bars
}

func concentrationTakeaway(measurable, aligned, gapFound bool) string {
	switch {
	case !gapFound:
		return "No project is large enough this window to read a spend-versus-output gap."
	case !measurable:
		return "Only one project has usage this window -- concentration needs at least two to mean anything."
	case aligned:
		return "Token spend tracks AI-line output across projects."
	default:
		return "One project takes a much larger share of tokens than of the lines produced -- worth asking what those tokens bought."
	}
}
