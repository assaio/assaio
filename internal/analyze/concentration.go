package analyze

import (
	"strconv"

	"github.com/assaio/assaio/internal/layer"

	"github.com/assaio/assaio/internal/humanize"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/store"
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
	// A project whose tokens all come from a source that records no lines writes zero lines by
	// construction, so its whole token share would read as the widest spend-to-output gap in
	// the window. That is the source's silence, not a project failing to convert spend.
	lineBlind := lineBlindProjects(in.Usage)
	gap, gapFound := widestSpendGap(named, lineBlind)
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
	if len(lineBlind) > 0 {
		r.Caveats = append(r.Caveats, strconv.Itoa(len(lineBlind))+
			" project(s) run entirely on sources that record no lines, so they are excluded from the spend-versus-output gap rather than counted as producing nothing.")
	}
	return r
}

// lineBlindProjects names the projects whose every token came from a source that records no
// changed lines. Read from the window's own rows, since the prepared per-project view carries
// no tool and so cannot tell a project that wrote nothing from one nobody was watching.
func lineBlindProjects(rows []store.UsageRow) map[string]bool {
	seen := make(map[string]bool)
	for i := range rows {
		if rowTokens(&rows[i]) == 0 {
			continue
		}
		project := rows[i].Project
		if parser.HasLineOutput(rows[i].Tool) {
			seen[project] = false
			continue
		}
		if _, ok := seen[project]; !ok {
			seen[project] = true
		}
	}
	blind := make(map[string]bool, len(seen))
	for project, isBlind := range seen {
		if isBlind {
			blind[project] = true
		}
	}
	return blind
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
	f := Figure{Label: "widest spend gap", Value: "—", Note: "tokens minus lines, share points"}
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
		bars[i] = Bar{
			Label: groupLabel(kept[i].Project),
			Value: humanize.PercentAt(kept[i].TokenShare, 0) + " tokens · " + humanize.PercentAt(kept[i].LineShare, 0) + " lines",
			Frac:  frac,
		}
	}
	return bars
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
		return "The widest gap between a project's share of tokens and its share of AI lines is " +
			gapLabel(gap) + " -- the project worth asking what those tokens bought. Lines are not value, so a gap is a question, not a fault."
	}
}
