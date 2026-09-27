package report

import (
	"github.com/assaio/assaio/internal/label"
	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// EffRow is one usage group's effectiveness signals under a single --by dimension:
// AI-line output, edit/rejection activity, and cost efficiency.
type EffRow struct {
	// Group is this row's label for the chosen --by dimension, e.g. a project name.
	Group string `json:"group"`
	// LinesAdded is AI-added code lines, the primary effect proxy.
	LinesAdded int64 `json:"lines_added"`
	// LineCapable reports whether any row in this group came from a source that records a
	// changed line. False means the group's line, edit and $/100-lines cells are absent rather
	// than zero -- the whole point of this table is comparing groups by lines per cost, so a
	// structural zero here invites dropping the tool that "produces nothing" (ADR 0011).
	LineCapable bool `json:"line_capable"`
	// LinesRemoved is AI-removed code lines.
	LinesRemoved int64 `json:"lines_removed"`
	// Edits is the count of productive edit tool-calls (Edit/Write/NotebookEdit/MultiEdit).
	Edits int64 `json:"edits"`
	// EditCapable reports whether any row in this group came from a source that records an
	// edit. It is a separate bit from LineCapable because the two do not travel together:
	// Copilot CLI reports changed lines and no edit count, Antigravity CLI the reverse, and
	// gating the edit column on the line capability withheld a figure one of them measured
	// while the note beside it called that figure absent.
	EditCapable bool `json:"edit_capable"`
	// ToolCalls is the count of all tool-use calls in the group, including edits.
	ToolCalls int64 `json:"tool_calls"`
	// Rejected is tool-use proposals the human declined: a friction signal.
	Rejected int64 `json:"rejected"`
	// Refusable reports whether any row in this group came from a source that records a
	// human declining a tool call. Claude Code is the only one today, so an ungated "0
	// rejected" is a friction reading taken from a field nobody wrote.
	Refusable bool `json:"refusable"`
	// TokensTotal sums input, output, cache-read, and cache-write tokens. Reasoning
	// tokens are a subset of output (usage.Record) and are never added again here.
	TokensTotal int64 `json:"tokens_total"`
	// Cost is USD cost summed from priced usage only; nil when nothing in the group priced.
	Cost *float64 `json:"cost"`
	// HasUnpriced reports whether this group excludes some usage's cost because its
	// model has no known price.
	HasUnpriced bool `json:"has_unpriced"`
	// UnpricedTokens is how much of TokensTotal carried no known price -- by how much the
	// cost above is short, which HasUnpriced alone never said.
	UnpricedTokens int64 `json:"unpriced_tokens"`
	// CostPer100Lines is LineCapableCost per 100 of the same usage's AI lines; nil when that
	// usage has no line or no price -- never a divide-by-zero substitute, and never the group's
	// whole cost over lines only some of its sources could record.
	CostPer100Lines *float64 `json:"cost_per_100_lines"`
	// LineCapableCost is the priced cost of this group's line-recording usage, the numerator
	// CostPer100Lines divides; nil when there is no rate.
	LineCapableCost *float64 `json:"line_capable_cost"`
	// LineCapableUnpricedTokens is line-recording usage with no known price, left out of the
	// ratio on both sides.
	LineCapableUnpricedTokens int64 `json:"line_capable_unpriced_tokens"`
	// lineRate is the population CostPer100Lines divides and what it leaves out.
	lineRate LineRateBasis
	// editBlindRows counts this group's usage rows from sources recording no edit.
	editBlindRows int
	// Tokened reports whether any row in this group came from a source that counts tokens at
	// all. It is LineCapable's counterpart on the cost half: false means this group's cost is
	// unknowable rather than merely unpriced.
	Tokened bool `json:"tokened"`
}

// BuildEffectiveness groups rows by the chosen --by dimension (day, project, tool,
// model, or entrypoint) and computes each group's AI-line output against its cost. An
// unknown dimension returns an error listing the valid ones.
func BuildEffectiveness(rows []store.UsageRow, t pricing.Table, by string) ([]EffRow, error) {
	if err := DimError(by); err != nil {
		return nil, err
	}

	out := groupBy(
		len(rows),
		func(i int) string { return usageDimValue(&rows[i], by) },
		func(key string) EffRow { return EffRow{Group: key} },
		func(g *EffRow, i int) { accumulateEff(g, &rows[i], t) },
	)
	for i := range out {
		if out[i].LineCapable {
			out[i].CostPer100Lines = out[i].lineRate.Per100()
			out[i].LineCapableCost = out[i].lineRate.LineCost()
		}
		out[i].LineCapableUnpricedTokens = out[i].lineRate.UnpricedLineTokens
	}
	return out, nil
}

// usageDimValue returns u's value for one of validDims.
func usageDimValue(u *store.UsageRow, by string) string {
	switch by {
	case "day":
		return u.Day
	case "project":
		return u.Project
	case "tool":
		return u.Tool
	case "model":
		return u.Model
	case "entrypoint":
		return u.Entrypoint
	case label.Task:
		return orUnlabeled(u.Task)
	case label.Outcome:
		return orUnlabeled(u.Outcome)
	case label.Difficulty:
		return orUnlabeled(u.Difficulty)
	default:
		return ""
	}
}

// accumulateEff folds u's activity counts and, when u.Model is priced, its cost into g.
func accumulateEff(g *EffRow, u *store.UsageRow, t pricing.Table) {
	g.LineCapable = g.LineCapable || parser.RecordsLines(u.Tool)
	if parser.Answers(u.Tool, parser.SignalEditsCount) {
		g.EditCapable = true
	} else {
		g.editBlindRows++
	}
	g.Refusable = g.Refusable || parser.Answers(u.Tool, parser.SignalRejectedCount)
	g.Tokened = g.Tokened || parser.Answers(u.Tool, parser.SignalTokensTotal)
	g.LinesAdded += u.LinesAdded
	g.LinesRemoved += u.LinesRemoved
	g.Edits += u.Edits
	g.ToolCalls += u.ToolCalls
	g.Rejected += u.Rejected
	g.TokensTotal += RowTokens(u)

	cost, ok := t.CostTokens(u.Model, pricing.Tokens{In: u.In, Out: u.Out, CacheWrite: u.CacheWrite, CacheRead: u.CacheRead, CacheWrite1h: u.CacheWrite1h})
	g.lineRate.Add(u.Tool, u.LinesAdded, RowTokens(u), cost, ok)
	if !ok {
		g.HasUnpriced = true
		g.UnpricedTokens += RowTokens(u)
		return
	}
	if g.Cost == nil {
		zero := 0.0
		g.Cost = &zero
	}
	*g.Cost += cost
}
