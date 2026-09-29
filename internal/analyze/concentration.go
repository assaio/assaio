package analyze

import (
	"strconv"

	"github.com/assaio/assaio/internal/layer"

	"github.com/assaio/assaio/internal/humanize"
)

const (
	concentrationName     = "concentration"
	concentrationTitle    = "Spend Concentration"
	concentrationDescribe = "How token spend spreads across projects, and whether the heaviest spenders produced output to match."
	// concentrationHowToRead is Result.HowToRead for this validator -- see its doc comment.
	concentrationHowToRead = "Concentration alone is neither good nor bad -- one project can legitimately own the work. Read the gap instead: a project holding a far larger share of the tokens than of the AI-written lines is where spend is not converting into code."

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

func (concentrationValidator) Name() string       { return concentrationName }
func (concentrationValidator) Title() string      { return concentrationTitle }
func (concentrationValidator) Describe() string   { return concentrationDescribe }
func (concentrationValidator) Layer() layer.Layer { return layer.Output } // the verdict is the gap between a project's spend share and its line share

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (concentrationValidator) Analyze(in Input) Result {
	r := Result{Name: concentrationName, Title: concentrationTitle, Describe: concentrationDescribe, HowToRead: concentrationHowToRead}
	if len(in.ByProject) == 0 || in.Totals.Tokens == 0 {
		r.noData("projects", "No usage in this window.")
		return r
	}
	all := projectShares(in.ByProject, in.Totals.Lines)
	// Every statistic below is over named projects only: the unattributed bucket pools
	// every tool that logs no working directory, so counting it as a project inflates the
	// project count and the concentration score alike. Its size is disclosed as a caveat.
	named := namedProjects(all)
	r.restsOn(len(named), "projects")
	// Tokens from a source that records no lines write zero lines by construction, so counted in
	// a project's token share they widen its spend-to-output gap by the source's silence rather
	// than by anything the project did. The gap compares lines with line-recording tokens only.
	gap, gapFound := widestSpendGap(named)
	// A gap has to have been computed for "aligned" to mean anything: with every project
	// below the size floor there is nothing to align, and calling that a pass is a green
	// check for an examination that never ran.
	measurable := len(named) >= concentrationMinProjects && gapFound

	r.Read = concentrationRead(measurable)
	r.Purity = neutralPurity
	r.Figures = []Figure{
		{Label: "projects", Value: strconv.Itoa(len(named))},
		{Label: "top project", Value: humanize.PercentAt(topShare(named), 0), Note: "of tokens"},
		{Label: "top 3", Value: humanize.PercentAt(topNShare(named, 3), 0), Note: "of tokens"},
		{Label: "concentration", Value: strconv.FormatFloat(giniOfShares(named), 'f', 2, 64), Note: "0 even · 1 concentrated"},
		concentrationGapFigure(gap, gapFound),
	}
	r.Bars = concentrationBars(named, concentrationTopN)
	r.BarsPseudonym = PseudonymProject
	r.Takeaway = concentrationTakeaway(measurable, gapFound, gap)
	if measurable {
		r.Caveats = append(r.Caveats, unsourcedLine("a spend-versus-output gap", ownHistoryWouldSettleIt))
	}
	if unattributed := unattributedShare(all); unattributed >= concentrationUnattributedFloor {
		r.Caveats = append(r.Caveats, "Prov.: "+humanize.PercentAt(unattributed, 0)+" of tokens are unattributed -- a source that logs no working directory cannot be assigned to a project.")
	}
	r.Caveats = append(r.Caveats, lineBlindCaveats(named)...)
	return r
}

// lineBlindCaveats names the projects whose tokens came, wholly or in part, from sources that
// record no lines: the first are left out of the gap, the second enter it on their other tokens.
func lineBlindCaveats(shares []projectShare) []string {
	var whole, part int
	for i := range shares {
		switch {
		case shares[i].lineBlindTokens == 0:
		case shares[i].lineTokens == 0:
			whole++
		default:
			part++
		}
	}
	var out []string
	if whole > 0 {
		out = append(out, strconv.Itoa(whole)+
			" project(s) run entirely on sources that record no lines, so they are excluded from the spend-versus-output gap rather than counted as producing nothing.")
	}
	if part > 0 {
		out = append(out, strconv.Itoa(part)+
			" project(s) run partly on sources that record no lines; the gap counts only their tokens from sources that do.")
	}
	return out
}

// concentrationRead reports the neutral no-verdict Read when a single project makes the
// gap undefined, rather than a favorable one earned by having nothing to compare.
func concentrationRead(measurable bool) Read {
	if !measurable {
		return noDataRead
	}
	return reportedRead
}

// concentrationGapFigure renders the widest gap in share points, never naming the project
// it belongs to: Figures are not pseudonymized under --anonymize, so the project behind
// the gap is disclosed only through Bars, which are.
func concentrationGapFigure(gap float64, found bool) Figure {
	f := Figure{Label: "widest spend gap", Value: "—", Note: "line-recording tokens minus lines, share points"}
	if found {
		f.Value = gapLabel(gap)
	}
	return f
}

// gapLabel renders a difference of two shares in share points. Every surface uses it, so the
// figure and the sentence beside it cannot render the same quantity in different units -- which
// they did, "89pp" above "89%", for one release.
func gapLabel(gap float64) string {
	return strconv.FormatFloat(gap*100, 'f', 0, 64) + "pp"
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
		bars[i] = Bar{Label: groupLabel(kept[i].Project), Value: concentrationBarValue(&kept[i]), Frac: frac}
	}
	return bars
}

// concentrationBarValue shows the token share the bar is ranked by and the line share beside it;
// a project that also ran a source recording no lines shows the share the gap reads as well, so
// the project behind the gap can be found from the bars.
func concentrationBarValue(s *projectShare) string {
	value := humanize.PercentAt(s.TokenShare, 0) + " tokens · "
	if s.lineBlindTokens > 0 {
		value += humanize.PercentAt(s.LineTokenShare, 0) + " of line-recording tokens · "
	}
	return value + humanize.PercentAt(s.LineShare, 0) + " lines"
}

func concentrationTakeaway(measurable, gapFound bool, gap float64) string {
	switch {
	case !gapFound:
		return "No project is large enough this window to read a spend-versus-output gap."
	case !measurable:
		return "Only one project has usage this window -- concentration needs at least two to mean anything."
	default:
		// Share points, not a percent: the gap is a difference of two shares, and rendering it
		// as "89%" invites the relative reading the figure above deliberately avoids.
		return "The widest gap between a project's share of line-recording tokens and its share of AI lines is " +
			gapLabel(gap) + " -- the project worth asking what those tokens bought. Lines are not value, so a gap is a question, not a fault."
	}
}
