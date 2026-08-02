package analyze

import (
	"sort"

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

// coverageValidator reports how much of the window rests on high-confidence data: the
// share of tokens from tools with full activity extraction (Claude Code, Codex) versus
// cost-only sources, and the share of tokens on priced models. It is the provenance meter
// the other validators' honesty leans on.
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
	_, byTool := tokensByTool(in.Usage)
	activityShare, pricedShare, _ := coverageShares(&in)
	solid := activityShare >= coverageStrongFloor && pricedShare >= coverageStrongFloor

	r.Read = readFor(solid, "Solid")
	r.Purity = clamp01(min(activityShare, pricedShare))
	r.Figures = []Figure{
		{Label: "activity coverage", Value: honestPercent(activityShare), Note: "lines/edits captured"},
		{Label: "priced coverage", Value: honestPercent(pricedShare), Note: "cost known"},
		{Label: "cost-only tokens", Value: honestPercent(1 - activityShare), Note: "no line signals"},
	}
	if turnShare, mixed := turnGranularityShare(in.Usage); mixed {
		r.Figures = append(r.Figures, Figure{
			Label: "turn-level records", Value: honestPercent(turnShare), Note: "rest are session aggregates",
		})
		r.Caveats = append(r.Caveats,
			"Session-granularity records cover a whole session, so per-turn figures describe only the turn-level share above.")
	}
	r.Bars = toolCoverageBars(byTool, in.Totals.Tokens)
	r.Takeaway = coverageTakeaway(activityShare, pricedShare)
	r.Caveats = append(r.Caveats,
		"Cost-only tools (Gemini CLI, Cline, plugins) contribute tokens and cost but no line or edit signals -- see ROADMAP.")
	return r
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

func coverageTakeaway(activityShare, pricedShare float64) string {
	switch {
	case activityShare >= coverageStrongFloor && pricedShare >= coverageStrongFloor:
		return "Most usage carries full activity and price data -- the other figures rest on solid coverage."
	case activityShare < coverageStrongFloor:
		return "A large share of tokens comes from cost-only tools, so line and edit figures cover only part of your usage."
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
func TokensByTool(rows []store.UsageRow) map[string]int64 {
	_, byTool := tokensByTool(rows)
	return byTool
}

// tokensByTool sums tokens per tool and, separately, the subtotal for activity-capable
// tools.
func tokensByTool(rows []store.UsageRow) (activity int64, byTool map[string]int64) {
	byTool = make(map[string]int64)
	for i := range rows {
		t := rowTokens(&rows[i])
		byTool[rows[i].Tool] += t
		if parser.HasActivity(rows[i].Tool) {
			activity += t
		}
	}
	return activity, byTool
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
		if !parser.HasActivity(t) {
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
