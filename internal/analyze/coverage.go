package analyze

import (
	"sort"
	"strings"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/store"
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
		r.Read = noDataRead
		r.Takeaway = "No usage in this window."
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

// turnGranularityShare returns the share of the window's tokens that came from per-turn
// records, and whether the window mixes granularities at all. An all-per-turn window --
// every built-in source today -- reports nothing: a figure that always reads 100% teaches
// a reader to skip it, which is how the one time it does not gets missed.
//
// The denominator counts only records whose granularity is known, so an unlabeled row
// lowers nothing; it is absent from the comparison rather than counted against per-turn.
func turnGranularityShare(rows []store.UsageRow) (share float64, mixed bool) {
	var turn, known int64
	for i := range rows {
		switch rows[i].Granularity {
		case "turn":
			tokens := rowTokens(&rows[i])
			turn += tokens
			known += tokens
		case "session":
			known += rowTokens(&rows[i])
		}
	}
	if known == 0 || turn == known {
		return 1, false
	}
	return fracOf(turn, known), true
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

// rowTokens is a row's billable token total, matching Totals.Tokens -- reasoning tokens
// are a subset of output (usage.Record) and are never re-added.
func rowTokens(r *store.UsageRow) int64 {
	return r.In + r.Out + r.CacheRead + r.CacheWrite
}

// TokensByTool totals the window's tokens per tool. Exported so the signal catalog's coverage
// command reads the same arithmetic this validator does rather than keeping its own.
func TokensByTool(rows []store.UsageRow) map[string]int64 { return tokensByTool(rows) }

// tokensByTool sums tokens per tool.
func tokensByTool(rows []store.UsageRow) map[string]int64 {
	byTool := make(map[string]int64)
	for i := range rows {
		byTool[rows[i].Tool] += rowTokens(&rows[i])
	}
	return byTool
}

// capableTokens totals the tokens from tools that satisfy can, so every coverage share is one
// question asked of the depth matrix rather than a bit each caller interprets for itself.
func capableTokens(byTool map[string]int64, can func(string) bool) int64 {
	var sum int64
	for tool, n := range byTool {
		if can(tool) {
			sum += n
		}
	}
	return sum
}

// pricedTokenSum totals the tokens on priced models across ByModel.
func pricedTokenSum(models []ModelStat) int64 {
	var sum int64
	for i := range models {
		if models[i].Priced {
			sum += models[i].Tokens
		}
	}
	return sum
}

// toolCoverageBars renders each tool's token share, marking cost-only tools. Bars label
// tools, never projects, so they are never pseudonymized (BarsPseudonym stays "").
func toolCoverageBars(byTool map[string]int64, total int64) []Bar {
	tools := make([]string, 0, len(byTool))
	for t := range byTool {
		tools = append(tools, t)
	}
	sort.Slice(tools, func(i, j int) bool {
		if byTool[tools[i]] != byTool[tools[j]] {
			return byTool[tools[i]] > byTool[tools[j]]
		}
		return tools[i] < tools[j]
	})
	var maxTok int64
	if len(tools) > 0 {
		maxTok = byTool[tools[0]]
	}
	bars := make([]Bar, len(tools))
	for i, t := range tools {
		label := t
		if !parser.HasLineOutput(t) {
			label += " (cost only)"
		}
		bars[i] = Bar{Label: label, Value: honestPercent(fracOf(byTool[t], total)), Frac: fracOf(byTool[t], maxTok)}
	}
	return bars
}

// HonestPercent is honestPercent for callers outside this package -- the signal catalog's
// coverage command renders shares through it so a partial verdict can never print "100%".
// (Folding this and the rest of the formatters into internal/humanize is B75.)
func HonestPercent(share float64) string { return honestPercent(share) }

// honestPercent renders a share without the two dishonest rounding edges: a small but
// nonzero share reads "<1%" (not "0%", which looks absent -- e.g. a few real Codex sessions
// dwarfed by Claude's cache-read volume), and a share just under whole reads ">99%" (not
// "100%", which would hide a real gap). The share formatter for honesty-first figures.
func honestPercent(share float64) string {
	switch {
	case share > 0 && share < 0.005:
		return "<1%"
	case share < 1 && share >= 0.995:
		return ">99%"
	default:
		return formatPercent(share, 0)
	}
}
