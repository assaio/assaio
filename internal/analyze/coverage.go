package analyze

import (
	"sort"
	"strings"

	"github.com/assaio/assaio/internal/parser"
)

const (
	coverageName      = "coverage"
	coverageTitle     = "Coverage & Confidence"
	coverageDescribe  = "How much of this window is high-confidence data: the share of tokens from tools with full activity capture, and the share priced."
	coverageHowToRead = "Coverage is the honesty backbone -- it says how much of every other figure rests on complete data. Low activity coverage means line and edit signals cover only part of your usage; low priced coverage means some cost is excluded, never a real zero."
	// coverageStrongFloor is the share both coverage axes must clear for a confident read.
	coverageStrongFloor = 0.8
)

func init() { Register(coverageValidator{}) }

// coverageValidator reports how much of the window rests on high-confidence data: the share
// of tokens from sources answering every activity signal, the share contributing no lines at
// all, and the share on priced models. It is the provenance meter the other validators'
// honesty leans on.
type coverageValidator struct{}

func (coverageValidator) Name() string     { return coverageName }
func (coverageValidator) Title() string    { return coverageTitle }
func (coverageValidator) Describe() string { return coverageDescribe }

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (coverageValidator) Analyze(in Input) Result {
	r := Result{Name: coverageName, Title: coverageTitle, Describe: coverageDescribe, HowToRead: coverageHowToRead}
	if in.Totals.Tokens == 0 {
		r.noData("active days", "No usage in this window.")
		return r
	}

	r.restsOn(activeDays(&in), "active days")
	byTool := tokensByTool(in.Usage)
	activityShare, pricedShare, _ := coverageShares(&in)
	lineShare := fracOf(capableTokens(byTool, parser.HasLineOutput), in.Totals.Tokens)
	solid := activityShare >= coverageStrongFloor && pricedShare >= coverageStrongFloor

	r.Read = readFor(solid, "Solid")
	r.Purity = clamp01(min(activityShare, pricedShare))
	r.Figures = []Figure{
		{Label: "activity coverage", Value: honestPercent(activityShare), Note: "lines/edits captured"},
		{Label: "priced coverage", Value: honestPercent(pricedShare), Note: "cost known"},
		{Label: "cost-only tokens", Value: honestPercent(1 - lineShare), Note: "no line signals"},
	}
	if turnShare, mixed := turnGranularityShare(in.Usage); mixed {
		r.Figures = append(r.Figures, Figure{
			Label: "turn-level records", Value: honestPercent(turnShare), Note: "rest are session aggregates",
		})
		r.Caveats = append(r.Caveats,
			"Session-granularity records cover a whole session, so per-turn figures describe only the turn-level share above.")
	}
	r.Bars = toolCoverageBars(byTool, in.Totals.Tokens)
	r.Takeaway = coverageTakeaway(activityShare, pricedShare, lineShare)
	r.Caveats = append(r.Caveats, sourceGapCaveats(byTool)...)
	return r
}

// sourceGapCaveats names the window's own sources rather than a list written into prose that
// goes stale the next time a parser lands: which of them contribute cost but no lines, and
// which contribute lines but answer no edit, tool-call or rework signal.
func sourceGapCaveats(byTool map[string]int64) []string {
	var out []string
	if costOnly := toolsWhere(byTool, func(t string) bool { return !parser.HasLineOutput(t) }); len(costOnly) > 0 {
		out = append(out, "Cost-only sources ("+strings.Join(costOnly, ", ")+
			") contribute tokens and cost but no line or edit signals.")
	}
	partial := toolsWhere(byTool, func(t string) bool { return parser.HasLineOutput(t) && !parser.HasFullActivity(t) })
	if len(partial) > 0 {
		out = append(out, "Partial activity ("+strings.Join(partial, ", ")+
			"): changed lines but no edit, tool-call or rework counts, so those figures cover less of this window than the line figures.")
	}
	return out
}

// toolsWhere names the window's tools matching want, alphabetically.
func toolsWhere(byTool map[string]int64, want func(string) bool) []string {
	var out []string
	for tool, n := range byTool {
		if n > 0 && want(tool) {
			out = append(out, tool)
		}
	}
	sort.Strings(out)
	return out
}

// coverageTakeaway names the reason activity coverage is thin, which is not one reason: a
// window can be short of line signals entirely, or carry lines from a source that records
// nothing else. Saying "cost-only tools" for the second contradicts the caveat below it.
func coverageTakeaway(activityShare, pricedShare, lineShare float64) string {
	switch {
	case activityShare >= coverageStrongFloor && pricedShare >= coverageStrongFloor:
		return "Most usage carries full activity and price data -- the other figures rest on solid coverage."
	case activityShare < coverageStrongFloor && lineShare < coverageStrongFloor:
		return "A large share of tokens comes from cost-only sources, so line and edit figures cover only part of your usage."
	case activityShare < coverageStrongFloor:
		return "Lines are covered, but much of this window comes from a source recording no edit or tool-call counts, so those figures cover less than the line figures do."
	default:
		return "Some tokens run on unpriced models, so cost is a floor here, not the full total."
	}
}
