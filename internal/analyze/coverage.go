package analyze

import (
	"sort"

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

	activityTokens, byTool := tokensByTool(in.Usage)
	activityShare := fracOf(activityTokens, in.Totals.Tokens)
	pricedShare := fracOf(pricedTokenSum(in.ByModel), in.Totals.Tokens)
	solid := activityShare >= coverageStrongFloor && pricedShare >= coverageStrongFloor

	r.Read = readFor(solid, "Solid")
	r.Purity = clamp01(min(activityShare, pricedShare))
	r.Figures = []Figure{
		{Label: "activity coverage", Value: honestPercent(activityShare), Note: "lines/edits captured"},
		{Label: "priced coverage", Value: honestPercent(pricedShare), Note: "cost known"},
		{Label: "cost-only tokens", Value: honestPercent(1 - activityShare), Note: "no line signals"},
	}
	r.Bars = toolCoverageBars(byTool, in.Totals.Tokens)
	r.Takeaway = coverageTakeaway(activityShare, pricedShare)
	r.Caveats = []string{
		"Cost-only tools (Gemini CLI, Cline, plugins) contribute tokens and cost but no line or edit signals -- see ROADMAP.",
		"Per-record granularity (turn vs session) is not yet surfaced, so a mix isn't flagged here.",
	}
	return r
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

// activityCapableTool reports whether tool's parser extracts line and edit activity, not
// just tokens and cost. Update this when a cost-only parser (Gemini CLI, Cline) gains
// activity extraction (BACKLOG B39).
func activityCapableTool(tool string) bool {
	return tool == "claude-code" || tool == "codex"
}

// rowTokens is a row's billable token total, matching Totals.Tokens -- reasoning tokens
// are a subset of output (usage.Record) and are never re-added.
func rowTokens(r *store.UsageRow) int64 {
	return r.In + r.Out + r.CacheRead + r.CacheWrite
}

// tokensByTool sums tokens per tool and, separately, the subtotal for activity-capable
// tools.
func tokensByTool(rows []store.UsageRow) (activity int64, byTool map[string]int64) {
	byTool = make(map[string]int64)
	for i := range rows {
		t := rowTokens(&rows[i])
		byTool[rows[i].Tool] += t
		if activityCapableTool(rows[i].Tool) {
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
// tools, never projects, so they are never pseudonymized (BarsAreProjects stays false).
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
		if !activityCapableTool(t) {
			label += " (cost only)"
		}
		bars[i] = Bar{Label: label, Value: honestPercent(fracOf(byTool[t], total)), Frac: fracOf(byTool[t], maxTok)}
	}
	return bars
}

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
