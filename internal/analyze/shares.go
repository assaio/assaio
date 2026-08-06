package analyze

// The shared token-share arithmetic: one place where a "share of the window" is computed,
// so the coverage validator, every confidence envelope and the signal catalog's coverage
// command can never disagree about the same fraction.

import (
	"sort"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/report"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/store"
)

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

// rowTokens is a row's billable token total, matching Totals.Tokens. The sum itself lives
// in report so no package keeps a second opinion about what a window cost.
func rowTokens(r *store.UsageRow) int64 { return report.RowTokens(r) }

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
		bars[i] = Bar{Label: label, Value: humanize.Percent(fracOf(byTool[t], total)), Frac: fracOf(byTool[t], maxTok)}
	}
	return bars
}
